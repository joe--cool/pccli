package planningcenter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/joe--cool/pccli/v2/internal/config"
)

func TestRateLimitDelayAndBoundedAttempts(t *testing.T) {
	for _, tc := range []struct {
		name, method, retryAfter string
		floor                    time.Duration
		attempts                 int
	}{
		{"server delay", http.MethodGet, "19", 19 * time.Second, 3},
		{"use announced window", http.MethodGet, "", 20 * time.Second, 3},
		{"reject long automatic waits", http.MethodGet, "90", 90 * time.Second, 1},
		{"preserve write retry hint", http.MethodPost, "19", 19 * time.Second, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := NewClient(config.Config{BaseURL: "https://example.test"}, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				calls++
				res := response(http.StatusTooManyRequests, `{"errors":[{"detail":"private-data-example"}]}`)
				res.Header.Set("Retry-After", tc.retryAfter)
				res.Header.Set("X-PCO-API-Request-Rate-Period", "20 seconds")
				res.Header.Set("X-PCO-API-Request-Rate-Limit", "100")
				res.Header.Set("X-PCO-API-Request-Rate-Count", "101")
				return res, nil
			}))
			var buffer bytes.Buffer
			client.SetRateLimitLog(&buffer)
			var waits []time.Duration
			client.wait = func(_ context.Context, d time.Duration) error { waits = append(waits, d); return nil }
			err := client.do(context.Background(), tc.method, "/services/v2/songs", nil, nil, nil)
			var apiErr APIError
			if !errors.As(err, &apiErr) || apiErr.RetryAfter != tc.floor || calls != tc.attempts || len(waits) != tc.attempts-1 {
				t.Fatalf("unexpected result: %v, attempts %d, waits %v", err, calls, waits)
			}
			for _, delay := range waits {
				if delay < tc.floor || delay >= tc.floor+time.Second {
					t.Fatalf("delay %s outside provider floor plus jitter", delay)
				}
			}
			log := strings.TrimSpace(buffer.String())
			if strings.Contains(log, "private-data-example") || strings.Contains(log, "/services/v2") {
				t.Fatal("diagnostics included request or response content")
			}
			lines := strings.Split(log, "\n")
			if len(lines) != calls {
				t.Fatalf("logged %d events, expected %d", len(lines), calls)
			}
			for i, line := range lines {
				var event struct {
					Timestamp time.Time `json:"timestamp"`
					Status    int       `json:"http_status"`
					Count     int       `json:"rate_count"`
					Attempt   int       `json:"attempt"`
					Wait      int64     `json:"retry_wait_ms"`
				}
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					t.Fatal(err)
				}
				if event.Timestamp.IsZero() || event.Status != 429 || event.Count != 101 || event.Attempt != i+1 {
					t.Fatalf("unexpected diagnostic event: %s", line)
				}
				if i < len(waits) && event.Wait != waits[i].Milliseconds() || i == len(waits) && event.Wait != 0 {
					t.Fatalf("wrong retry wait in event: %s", line)
				}
			}
		})
	}
}
