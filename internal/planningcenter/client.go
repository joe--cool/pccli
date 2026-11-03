package planningcenter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joe--cool/pccli/internal/config"
)

const ServicesAPIVersion = "2018-11-01"
const maxRetryWait = 30 * time.Second
const maxResponseBytes = 32 << 20

type Client struct {
	baseURL, uploadURL, clientID, clientSecret, userAgent string
	httpClient                                            *http.Client
	wait                                                  func(context.Context, time.Duration) error
}

func NewClient(cfg config.Config, transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{baseURL: cfg.BaseURL, uploadURL: cfg.UploadURL, clientID: cfg.ClientID, clientSecret: cfg.ClientSecret, userAgent: cfg.UserAgent,
		httpClient: &http.Client{Timeout: cfg.Timeout, Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("Planning Center redirects are not permitted")
		}}, wait: waitContext}
}

func (c *Client) Get(ctx context.Context, path string, query url.Values, dst any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, dst)
}
func (c *Client) Post(ctx context.Context, path string, query url.Values, body any, dst any) error {
	return c.do(ctx, http.MethodPost, path, query, body, dst)
}
func (c *Client) Patch(ctx context.Context, path string, query url.Values, body any, dst any) error {
	return c.do(ctx, http.MethodPatch, path, query, body, dst)
}
func (c *Client) Delete(ctx context.Context, path string, query url.Values) error {
	return c.do(ctx, http.MethodDelete, path, query, nil, nil)
}

// UploadFile uses a small multipart prefix and suffix around the file reader. Its
// known Content-Length avoids chunked-transfer requirements without buffering files.
func (c *Client) UploadFile(ctx context.Context, path string, dst any) error {
	endpoint, err := safeEndpoint(c.uploadURL, "/v2/files", nil)
	if err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open upload file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect upload file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("upload requires a regular file")
	}
	var framing bytes.Buffer
	writer := multipart.NewWriter(&framing)
	if _, err = writer.CreateFormFile("file", filepath.Base(path)); err != nil {
		return errors.New("create upload form failed")
	}
	prefix := append([]byte(nil), framing.Bytes()...)
	framing.Reset()
	if err = writer.Close(); err != nil {
		return errors.New("finalize upload form failed")
	}
	suffix := append([]byte(nil), framing.Bytes()...)
	reader := io.MultiReader(bytes.NewReader(prefix), &contextReader{ctx: ctx, reader: io.LimitReader(file, info.Size())}, bytes.NewReader(suffix))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, reader)
	if err != nil {
		return errors.New("create upload request failed")
	}
	req.ContentLength = int64(len(prefix)) + info.Size() + int64(len(suffix))
	c.headers(req, false)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return requestFailure(ctx, "upload", true, err)
	}
	return classifyResponse("upload", true, decodeResponse(resp, dst))
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func (c *Client) headers(req *http.Request, services bool) {
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("User-Agent", c.userAgent)
	if services {
		req.Header.Set("X-PCO-API-Version", ServicesAPIVersion)
	}
	if c.clientID != "" || c.clientSecret != "" {
		req.SetBasicAuth(c.clientID, c.clientSecret)
	}
}
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, dst any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return errors.New("encode request body failed")
		}
	}
	endpoint, err := c.endpoint(path, query)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
		if err != nil {
			return errors.New("create request failed")
		}
		c.headers(req, strings.HasPrefix(req.URL.Path, "/services/"))
		if body != nil {
			req.Header.Set("Content-Type", "application/vnd.api+json")
		}
		resp, requestErr := c.httpClient.Do(req)
		var result error
		var delay time.Duration
		retry := false
		if requestErr != nil {
			result = requestFailure(ctx, "request", method != http.MethodGet, requestErr)
			retry = ctx.Err() == nil && transientNetwork(requestErr)
		} else {
			result = classifyResponse("request", method != http.MethodGet, decodeResponse(resp, dst))
			retry = resp.StatusCode == 429 || resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504
			delay = retryAfter(resp.Header.Get("Retry-After"))
		}
		if result == nil {
			return nil
		}
		if method != http.MethodGet || !retry || attempt == 2 {
			return result
		}
		if delay > maxRetryWait {
			return result
		}
		if delay == 0 {
			delay = time.Duration(200*(1<<attempt))*time.Millisecond + time.Duration(rand.IntN(100))*time.Millisecond
		}
		if err = c.wait(ctx, delay); err != nil {
			return err
		}
	}
	return errors.New("Planning Center request failed")
}
func transientNetwork(err error) bool {
	var u *url.Error
	if errors.As(err, &u) {
		err = u.Err
	}
	var n interface{ Timeout() bool }
	if errors.As(err, &n) {
		return true
	}
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}
func requestFailure(ctx context.Context, op string, write bool, cause error) error {
	if ctx.Err() != nil {
		cause = ctx.Err()
	} else { // Do not retain URL/remote transport text in an error reachable by formatters.
		cause = errors.New("connection failed")
	}
	return &TransportError{Operation: op, UnknownOutcome: write, Cause: cause}
}
func classifyResponse(op string, write bool, err error) error {
	if err == nil {
		return nil
	}
	var invalid *ResponseError
	if write && errors.As(err, &invalid) {
		return &TransportError{Operation: op, UnknownOutcome: true, Cause: err}
	}
	var api APIError
	if write && errors.As(err, &api) && api.StatusCode >= 500 {
		return &TransportError{Operation: op, UnknownOutcome: true, Cause: err}
	}
	return err
}
func (c *Client) endpoint(path string, query url.Values) (string, error) {
	return safeEndpoint(c.baseURL, path, query)
}

// ResolveEndpoint validates API-origin links before callers inspect cursor policy.
func (c *Client) ResolveEndpoint(path string) (*url.URL, error) {
	value, err := c.endpoint(path, nil)
	if err != nil {
		return nil, err
	}
	return url.Parse(value)
}
func safeEndpoint(base, path string, query url.Values) (string, error) {
	invalid := errors.New("invalid Planning Center endpoint")
	origin, err := url.Parse(base)
	if err != nil || origin.Host == "" || origin.User != nil || origin.Fragment != "" || origin.RawQuery != "" || (origin.Scheme != "https" && origin.Scheme != "http") {
		return "", invalid
	}
	ref, err := url.Parse(path)
	if err != nil || ref.User != nil || ref.Fragment != "" || strings.Contains(path, "\\") {
		return "", invalid
	}
	if ref.IsAbs() || ref.Host != "" {
		if !strings.EqualFold(ref.Scheme, origin.Scheme) || !strings.EqualFold(ref.Host, origin.Host) {
			return "", invalid
		}
	} else if !strings.HasPrefix(ref.Path, "/") {
		ref.Path = "/" + ref.Path
	}
	for _, segment := range strings.Split(ref.Path, "/") {
		if segment == ".." || segment == "." {
			return "", invalid
		}
	}
	endpoint := origin.ResolveReference(ref)
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}
	return endpoint.String(), nil
}
func decodeResponse(resp *http.Response, dst any) error {
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Remote bodies can contain credentials, request content, or terminal controls.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		return APIError{StatusCode: resp.StatusCode}
	}
	if dst == nil {
		_, err := io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		if err != nil {
			return &ResponseError{Reason: "could not read response"}
		}
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return &ResponseError{Reason: "could not read response"}
	}
	if len(data) > maxResponseBytes {
		return &ResponseError{Reason: "response exceeds size limit"}
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || len(envelope.Data) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) {
		return &ResponseError{Reason: "missing or invalid data"}
	}
	if envelope.Data[0] != '{' && envelope.Data[0] != '[' {
		return &ResponseError{Reason: "data must be a resource or collection"}
	}
	if err = json.Unmarshal(data, dst); err != nil {
		return &ResponseError{Reason: "invalid response shape"}
	}
	return nil
}
func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func retryAfter(value string) time.Duration {
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		if seconds > int64(maxRetryWait/time.Second) {
			return maxRetryWait + time.Second
		}
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		delay := time.Until(date)
		if delay > 0 {
			return delay
		}
	}
	return 0
}
