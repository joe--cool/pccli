package planningcenter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

type Client struct {
	baseURL      string
	uploadURL    string
	clientID     string
	clientSecret string
	userAgent    string
	httpClient   *http.Client
	sleep        func(time.Duration)
}

func NewClient(cfg config.Config, transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultTransport
	}

	return &Client{
		baseURL:      cfg.BaseURL,
		uploadURL:    cfg.UploadURL,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		userAgent:    cfg.UserAgent,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		sleep: time.Sleep,
	}
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

func (c *Client) UploadFile(ctx context.Context, path string, dst any) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open upload file %s: %w", path, err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return fmt.Errorf("create upload form: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("read upload file %s: %w", path, err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize upload form: %w", err)
	}

	endpoint, err := url.JoinPath(c.uploadURL, "/v2/files")
	if err != nil {
		return fmt.Errorf("parse Planning Center upload URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", c.userAgent)
	if c.clientID != "" || c.clientSecret != "" {
		req.SetBasicAuth(c.clientID, c.clientSecret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload file to Planning Center: %w", err)
	}
	return decodeResponse(resp, dst)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, dst any) error {
	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		payload = encoded
	}

	endpoint, err := c.endpoint(path, query)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.api+json")
		req.Header.Set("User-Agent", c.userAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/vnd.api+json")
		}
		if c.clientID != "" || c.clientSecret != "" {
			req.SetBasicAuth(c.clientID, c.clientSecret)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request Planning Center: %w", err)
		}

		err = decodeResponse(resp, dst)
		if err == nil {
			return nil
		}
		lastErr = err

		apiErr, ok := err.(APIError)
		if !ok || apiErr.StatusCode != http.StatusTooManyRequests || method != http.MethodGet || attempt == 2 {
			return err
		}

		delay := retryAfter(resp.Header.Get("Retry-After"))
		if delay == 0 {
			delay = time.Second
		}
		c.sleep(delay)
	}

	return lastErr
}

func (c *Client) endpoint(path string, query url.Values) (string, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parsed, err := url.Parse(c.baseURL + path)
	if err != nil {
		return "", fmt.Errorf("parse Planning Center URL: %w", err)
	}
	if len(query) > 0 {
		parsed.RawQuery = query.Encode()
	}
	return parsed.String(), nil
}

func decodeResponse(resp *http.Response, dst any) error {
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		apiErr := APIError{StatusCode: resp.StatusCode, Body: string(data)}
		var errorResponse ErrorResponse
		if err := json.Unmarshal(data, &errorResponse); err == nil {
			apiErr.Errors = errorResponse.Errors
		}
		return apiErr
	}

	if dst == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}
	return nil
}

func retryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
