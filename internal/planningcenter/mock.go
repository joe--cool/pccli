package planningcenter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type MockTransport struct {
	fixture fixture
}

type fixture struct {
	Routes map[string]json.RawMessage `json:"routes"`
}

func NewMockTransport(path string) (*MockTransport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mock fixture %s: %w", path, err)
	}
	var fx fixture
	if err := json.Unmarshal(data, &fx); err != nil {
		return nil, fmt.Errorf("parse mock fixture %s: %w", path, err)
	}
	return &MockTransport{fixture: fx}, nil
}

func (t *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.Method + " " + req.URL.Path
	body, ok := t.fixture.Routes[key]
	if !ok {
		body = mockNotFound(key)
		return jsonResponse(req, http.StatusNotFound, body), nil
	}
	return jsonResponse(req, http.StatusOK, body), nil
}

func jsonResponse(req *http.Request, status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": []string{"application/vnd.api+json"}},
		Body:       ioNopCloser{bytes.NewReader(body)},
		Request:    req,
	}
}

func mockNotFound(route string) []byte {
	body, _ := json.Marshal(ErrorResponse{
		Errors: []ErrorObject{{
			Status: "404",
			Code:   "MOCK_ROUTE_NOT_FOUND",
			Title:  "Mock route not found",
			Detail: fmt.Sprintf("No mock route is registered for %s", route),
		}},
	})
	return body
}

type ioNopCloser struct {
	*bytes.Reader
}

func (c ioNopCloser) Close() error {
	return nil
}
