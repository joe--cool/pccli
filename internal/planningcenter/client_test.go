package planningcenter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/joe--cool/pccli/internal/config"
)

func TestClientGetSendsBasicAuthAndDecodesJSONAPI(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "client" || pass != "secret" {
			t.Fatalf("unexpected auth: %q %q %v", user, pass, ok)
		}
		if got := r.URL.Query().Get("where[title]"); got != "Amazing%" {
			t.Fatalf("unexpected query: %q", got)
		}
		return response(http.StatusOK, `{"data":{"id":"1001","type":"Song","attributes":{"title":"Amazing Grace"}}}`), nil
	})

	client := NewClient(config.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		BaseURL:      "https://example.test",
		Timeout:      time.Second,
		UserAgent:    "pccli-test",
	}, transport)

	var response Single[struct {
		Title string `json:"title"`
	}]
	query := url.Values{"where[title]": []string{"Amazing%"}}
	if err := client.Get(context.Background(), "/services/v2/songs/1001", query, &response); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if response.Data.ID != "1001" || response.Data.Attributes.Title != "Amazing Grace" {
		t.Fatalf("unexpected response: %#v", response.Data)
	}
}

func TestClientReturnsAPIErrorDetails(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return response(http.StatusForbidden, `{"errors":[{"status":"403","detail":"permission denied"}]}`), nil
	})

	client := NewClient(config.Config{
		BaseURL:   "https://example.test",
		Timeout:   time.Second,
		UserAgent: "pccli-test",
	}, transport)

	err := client.Get(context.Background(), "/services/v2/songs", nil, nil)
	apiErr, ok := err.(APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", apiErr.StatusCode)
	}
	if got := apiErr.Error(); got != "Planning Center API returned HTTP 403: permission denied" {
		t.Fatalf("unexpected error string: %q", got)
	}
}

func TestClientRetriesGetAfterRateLimit(t *testing.T) {
	calls := 0
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			resp := response(http.StatusTooManyRequests, `{"errors":[{"status":"429","detail":"rate limited"}]}`)
			resp.Header.Set("Retry-After", "0")
			return resp, nil
		}
		return response(http.StatusOK, `{"data":[]}`), nil
	})

	client := NewClient(config.Config{
		BaseURL:   "https://example.test",
		Timeout:   time.Second,
		UserAgent: "pccli-test",
	}, transport)
	client.sleep = func(time.Duration) {}

	var response Collection[struct{}]
	if err := client.Get(context.Background(), "/services/v2/songs", nil, &response); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/vnd.api+json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
