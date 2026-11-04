package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/joe--cool/pccli/v2/internal/output"
	"github.com/joe--cool/pccli/v2/internal/planningcenter"
	"github.com/joe--cool/pccli/v2/internal/services"
)

type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, context.Canceled) {
		return 130
	}
	var u usageError
	var s *services.Error
	if errors.As(err, &u) || errors.As(err, &s) && s.Code == "invalid_input" {
		return 2
	}
	return 1
}
func printError(out io.Writer, j bool, err error) {
	if err == nil {
		return
	}
	code := "api_error"
	if ExitCode(err) == 2 {
		code = "invalid_input"
	}
	if errors.Is(err, context.Canceled) {
		code = "cancelled"
	}
	var s *services.Error
	var transport *planningcenter.TransportError
	var response *planningcenter.ResponseError
	var committed *committedOutputError
	if errors.As(err, &transport) && transport.UnknownOutcome {
		code = "outcome_unknown"
	} else if errors.As(err, &response) {
		code = "invalid_response"
	}
	if errors.As(err, &committed) {
		code = "partial_success"
	}
	body := map[string]any{"code": code, "message": output.SafeText(err.Error())}
	if errors.As(err, &s) {
		body["code"] = s.Code
		if s.Candidates != nil {
			body["candidates"] = s.Candidates
			body["candidates_complete"] = false
		}
		if s.Recovery != nil {
			body["recovery"] = s.Recovery
		}
	}
	if errors.Is(err, context.Canceled) {
		body["code"] = "cancelled"
	}
	var api planningcenter.APIError
	if errors.As(err, &api) {
		body["http_status"] = api.StatusCode
	}
	if committed != nil {
		body["result"] = committed.Result
	}
	if j {
		_ = writeJSON(out, map[string]any{"schema": "pccli/v1", "error": body})
		return
	}
	_, _ = fmt.Fprintf(out, "Error: %s\n", output.SafeText(err.Error()))
	if s != nil && s.Candidates != nil {
		b, _ := json.Marshal(s.Candidates)
		_, _ = fmt.Fprintf(out, "Candidates: %s\n", output.SafeText(string(b)))
	}
	if s != nil && s.Recovery != nil {
		b, _ := json.Marshal(s.Recovery)
		_, _ = fmt.Fprintf(out, "Recovery receipt: %s\nInspect the destination before retrying. Reuse --upload-id only after confirming no attachment was created.\n", output.SafeText(string(b)))
	}
}
