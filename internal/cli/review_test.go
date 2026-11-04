package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/internal/config"
	"github.com/joe--cool/pccli/internal/planningcenter"
	"github.com/joe--cool/pccli/internal/services"
)

type reviewTransport func(*http.Request) (*http.Response, error)

func (f reviewTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func reviewResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func reviewLibrary(handler reviewTransport) *services.Library {
	return services.NewLibrary(planningcenter.NewClient(config.Config{BaseURL: "https://api.planningcenteronline.com", Color: "never"}, handler))
}

const reviewSong = `{"data":{"id":"1","type":"Song","attributes":{"title":"Grace","hidden":false}}}`

func TestReviewJSONSelectionRespectsValuesAndTerminator(t *testing.T) {
	for _, test := range []struct {
		args []string
		want bool
	}{
		{[]string{"services", "songs", "list", "--title", "--json", "--bad"}, false},
		{[]string{"services", "songs", "list", "-q", "--json", "--bad"}, false},
		{[]string{"--json", "services", "songs", "list", "--title", "--json=false", "--bad"}, true},
		{[]string{"services", "songs", "show", "--", "--json"}, false},
		{[]string{"services", "songs", "list", "--json=1", "--bad"}, true},
		{[]string{"--json", "services", "songs", "list", "--json=FALSE", "--bad"}, false},
		{[]string{"--json", "services", "songs", "list", "--json=false", "--bad"}, false},
	} {
		if got := jsonRequested(test.args); got != test.want {
			t.Fatalf("args %q got %t want %t", test.args, got, test.want)
		}
	}
}
func TestReviewLastScheduledCLIForwardsServiceType(t *testing.T) {
	requests := 0
	library := reviewLibrary(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			t.Fatalf("unexpected write %s", r.Method)
		}
		if r.URL.Path == "/services/v2/songs/1" {
			return reviewResponse(200, reviewSong), nil
		}
		if r.URL.Path != "/services/v2/songs/1/last_scheduled_item" || r.URL.Query().Get("service_type") != "789" {
			t.Fatalf("unexpected request %s", r.URL)
		}
		requests++
		return reviewResponse(200, `{"data":[{"id":"45","type":"Item","attributes":{"title":"Sunday","length":120}}]}`), nil
	})
	out, stderr, err := runCommand(injectedCommand(library), "services", "songs", "last-scheduled", "1", "--service-type", "789", "--json")
	if err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}
	got := decodeOutput(t, out)
	var data []services.AncillaryRecord
	if json.Unmarshal(got["data"], &data) != nil || len(data) != 1 || data[0].ID != "45" || requests != 1 {
		t.Fatalf("unexpected result %s requests %d", out, requests)
	}
}
func TestReviewAttachmentWriteOnlyAcceptancePlainAndJSON(t *testing.T) {
	for _, machine := range []bool{false, true} {
		writes := 0
		library := reviewLibrary(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/services/v2/songs/1" {
				return reviewResponse(200, reviewSong), nil
			}
			if r.URL.Path == "/services/v2/songs/1/attachments" {
				return reviewResponse(200, `{"data":[{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"}}]}`), nil
			}
			if r.URL.Path != "/services/v2/songs/1/attachments/30" {
				t.Fatalf("unexpected %s", r.URL)
			}
			if r.Method == "PATCH" {
				writes++
				body, _ := io.ReadAll(r.Body)
				if !bytes.Contains(body, []byte(`"song_part":"Verse 1"`)) {
					t.Fatal(string(body))
				}
			}
			return reviewResponse(200, `{"data":{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"}}}`), nil
		})
		args := []string{"services", "attachments", "update", "30", "--song", "1", "--song-part", "Verse 1"}
		if machine {
			args = append(args, "--json")
		}
		out, stderr, err := runCommand(injectedCommand(library), args...)
		if err != nil || writes != 1 {
			t.Fatalf("err %v writes %d stderr %s", err, writes, stderr)
		}
		if machine {
			got := decodeOutput(t, out)
			var result services.AttachmentUpdateResult
			if json.Unmarshal(got["data"], &result) != nil || result.Outcome != "accepted" || strings.Join(result.UnverifiedFields, ",") != "song_part" {
				t.Fatal(out)
			}
		} else if !strings.Contains(out, "accepted") || !strings.Contains(out, "song_part") || !strings.Contains(out, "chart.pdf") {
			t.Fatalf("accepted receipt unclear: %s", out)
		}
	}
}
func TestReviewCancelledTransportErrorDoesNotPublishSuccess(t *testing.T) {
	library := reviewLibrary(func(*http.Request) (*http.Response, error) { return nil, context.Canceled })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command := injectedCommand(library)
	ctx = withApp(ctx, appFrom(command))
	var stdout, stderr bytes.Buffer
	err := executeCommand(ctx, command, []string{"services", "songs", "show", "1", "--json"}, &stdout, &stderr)
	if !errors.Is(err, context.Canceled) || ExitCode(err) != 130 || stdout.Len() != 0 {
		t.Fatalf("err %v output %s", err, stdout.String())
	}
	decodeOutput(t, stderr.String())
}

func TestReviewAttachmentOutputFailurePreservesKnownReceipt(t *testing.T) {
	writes := 0
	library := reviewLibrary(func(r *http.Request) (*http.Response, error) {
		if r.Method == "GET" && r.URL.Path == "/services/v2/songs/1" {
			return reviewResponse(200, reviewSong), nil
		}
		if r.Method != "POST" || r.URL.Path != "/services/v2/songs/1/attachments" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		writes++
		return reviewResponse(201, `{"data":{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"}}}`), nil
	})
	command := injectedCommand(library)
	var stderr bytes.Buffer
	err := executeCommand(command.Context(), command, []string{"services", "attachments", "add", "--song", "1", "--upload-id", "known-receipt-uuid", "--filename", "chart.pdf", "--json"}, errorWriter{}, &stderr)
	if err == nil || writes != 1 {
		t.Fatalf("err %v writes %d", err, writes)
	}
	got := decodeOutput(t, stderr.String())
	var body struct {
		Code     string                    `json:"code"`
		Recovery services.AttachmentResult `json:"recovery"`
	}
	if json.Unmarshal(got["error"], &body) != nil || body.Code != "partial_success" || body.Recovery.Upload == nil || body.Recovery.Upload.ID != "known-receipt-uuid" || body.Recovery.Outcome != "created" {
		t.Fatalf("receipt lost: %s", stderr.String())
	}
}
func TestReviewTransportErrorMachineClassification(t *testing.T) {
	for _, test := range []struct {
		err  error
		code string
	}{
		{&planningcenter.TransportError{Operation: "request", UnknownOutcome: true, Cause: io.ErrUnexpectedEOF}, "outcome_unknown"},
		{&planningcenter.ResponseError{Reason: "invalid response shape"}, "invalid_response"},
		{planningcenter.APIError{StatusCode: 403, Body: "secret must never be exposed"}, "api_error"},
	} {
		var stderr bytes.Buffer
		printError(&stderr, true, test.err)
		got := decodeOutput(t, stderr.String())
		var body map[string]any
		if json.Unmarshal(got["error"], &body) != nil || body["code"] != test.code {
			t.Fatalf("error %T: %s", test.err, stderr.String())
		}
		if strings.Contains(stderr.String(), "secret must never") {
			t.Fatal("remote body leaked")
		}
	}
}
