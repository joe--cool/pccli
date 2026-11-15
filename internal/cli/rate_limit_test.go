package cli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

func TestRateLimitErrorIncludesRetryHint(t *testing.T) {
	var stderr bytes.Buffer
	printError(&stderr, true, planningcenter.APIError{StatusCode: 429, RetryAfter: 19*time.Second + time.Millisecond})
	var envelope struct {
		Schema string `json:"schema"`
		Error  struct {
			Status int `json:"http_status"`
			Retry  int `json:"retry_after_seconds"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Schema != "pccli/v1" || envelope.Error.Status != 429 || envelope.Error.Retry != 20 {
		t.Fatalf("unexpected error envelope: %s", stderr.String())
	}
}
