package planningcenter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joe--cool/pccli/v2/internal/config"
)

func testClient(transport http.RoundTripper) *Client {
	return NewClient(config.Config{BaseURL: "https://api.example.test", UploadURL: "https://upload.example.test", ClientID: "synthetic-client", ClientSecret: "synthetic-secret", Timeout: time.Second}, transport)
}

func TestMalformedSuccessfulResponses(t *testing.T) {
	for _, body := range []string{"", `{}`, `null`, `{"data":null}`, `{"data":{}}`, `{"data":[{"id":"1","type":"Song"}]}`, `{"data":[]} trailing`, `{"data":5}`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			c := testClient(roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return response(200, body), nil }))
			var dst Collection[struct{}]
			var invalid *ResponseError
			if err := c.Get(context.Background(), "/services/v2/songs", nil, &dst); !errors.As(err, &invalid) {
				t.Fatalf("expected invalid response: %v", err)
			}
			if calls != 1 {
				t.Fatal("malformed response retried")
			}
		})
	}
}
func TestWriteMalformedResponseUnknown(t *testing.T) {
	c := testClient(roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, `{}`), nil }))
	var dst Single[struct{}]
	err := c.Post(context.Background(), "/services/v2/songs", nil, map[string]string{"title": "x"}, &dst)
	var unknown *TransportError
	if !errors.As(err, &unknown) || !unknown.UnknownOutcome {
		t.Fatalf("expected unknown outcome: %v", err)
	}
}
func TestHeadersAreProductScoped(t *testing.T) {
	c := testClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		want := ""
		if strings.HasPrefix(r.URL.Path, "/services/") {
			want = ServicesAPIVersion
		}
		if got := r.Header.Get("X-PCO-API-Version"); got != want {
			t.Fatalf("header %q, want %q", got, want)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "synthetic-client" || pass != "synthetic-secret" {
			t.Fatal("missing auth")
		}
		return response(200, `{"data":[]}`), nil
	}))
	for _, path := range []string{"/services/v2/songs", "/people/v2/people"} {
		var dst Collection[struct{}]
		if err := c.Get(context.Background(), path, nil, &dst); err != nil {
			t.Fatal(err)
		}
	}
}
func TestEndpointRejectsHostileLinks(t *testing.T) {
	c := testClient(nil)
	for _, path := range []string{"https://evil.test/services/v2/songs", "//evil.test/x", "https://user:pass@api.example.test/x", "/services/../people", "/services/%2e%2e/people", "/services/v2/songs#fragment", "/services\\evil"} {
		if _, err := c.endpoint(path, nil); err == nil {
			t.Errorf("accepted %s", path)
		}
	}
	for _, path := range []string{"/services/v2/songs?offset=100", "https://api.example.test/services/v2/songs?offset=100"} {
		if _, err := c.endpoint(path, nil); err != nil {
			t.Fatal(err)
		}
	}
}
func TestRedirectNeverSendsCredentials(t *testing.T) {
	hits := 0
	c := testClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		hits++
		if r.URL.Host != "api.example.test" {
			t.Fatal("credentials followed redirect")
		}
		resp := response(302, "")
		resp.Header.Set("Location", "https://evil.example.test/secret")
		return resp, nil
	}))
	if err := c.Get(context.Background(), "/services/v2/songs", nil, nil); err == nil {
		t.Fatal("accepted redirect")
	}
	if hits != 1 {
		t.Fatalf("unexpected requests: %d", hits)
	}
}
func TestBoundedGetRetriesAndNoWriteReplay(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			calls := 0
			c := testClient(roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return response(503, `secret`), nil }))
			c.wait = func(context.Context, time.Duration) error { return nil }
			err := c.do(context.Background(), method, "/services/v2/songs", nil, nil, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			want := 1
			if method == http.MethodGet {
				want = 3
			}
			if calls != want {
				t.Fatalf("calls %d, want %d", calls, want)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("remote body leaked")
			}
		})
	}
}
func TestNetworkRetryAndCancellation(t *testing.T) {
	calls := 0
	c := testClient(roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, io.EOF }))
	c.wait = func(context.Context, time.Duration) error { return nil }
	if err := c.Get(context.Background(), "/services/v2/songs", nil, nil); err == nil || calls != 3 {
		t.Fatalf("err %v calls %d", err, calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls = 0
	c = testClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		cancel()
		resp := response(429, `{}`)
		resp.Header.Set("Retry-After", "20")
		return resp, nil
	}))
	if err := c.Get(ctx, "/services/v2/songs", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	if calls != 1 {
		t.Fatal("retried after cancellation")
	}
}
func TestRetryAfter(t *testing.T) {
	if retryAfter("12") != 12*time.Second {
		t.Fatal("seconds")
	}
	d := retryAfter(time.Now().Add(5 * time.Second).UTC().Format(http.TimeFormat))
	if d <= 3*time.Second || d > 5*time.Second {
		t.Fatalf("date %s", d)
	}
	if retryAfter("99999999999999") <= maxRetryWait {
		t.Fatal("unbounded header wrapped")
	}
	if retryAfter("-1") != 0 {
		t.Fatal("negative")
	}
}
func TestUploadStreamsWithLength(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chart.pdf")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(64 << 20); err != nil {
		t.Fatal(err)
	}
	f.Close()
	c := testClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.ContentLength < 64<<20 || r.ContentLength > (64<<20)+4096 {
			t.Fatalf("length %d", r.ContentLength)
		}
		if r.Header.Get("X-PCO-API-Version") != "" {
			t.Fatal("services version sent to upload")
		}
		if r.GetBody != nil {
			t.Fatal("upload is buffered/replayable")
		}
		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		part, err := mr.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		if part.FileName() != "chart.pdf" {
			t.Fatal(part.FileName())
		}
		n, err := io.Copy(io.Discard, part)
		if err != nil || n != 64<<20 {
			t.Fatalf("copy %d %v", n, err)
		}
		if _, err = mr.NextPart(); err != io.EOF {
			t.Fatal(err)
		}
		return response(200, `{"data":[{"id":"upload-id","type":"File","attributes":{}}]}`), nil
	}))
	var dst Collection[struct{}]
	if err = c.UploadFile(context.Background(), path, &dst); err != nil {
		t.Fatal(err)
	}
}
func TestErrorsExcludeRemoteSecrets(t *testing.T) {
	e := APIError{StatusCode: 400, Body: "synthetic-secret", Errors: []ErrorObject{{Detail: "\x1b[31msynthetic-secret"}}}
	if strings.Contains(e.Error(), "secret") || strings.Contains(e.Error(), "\x1b") {
		t.Fatal(e.Error())
	}
}
