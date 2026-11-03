package planningcenter

import (
	"encoding/json"
	"fmt"
)

type Resource[T any] struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Attributes    T               `json:"attributes"`
	Relationships json.RawMessage `json:"relationships,omitempty"`
	Links         json.RawMessage `json:"links,omitempty"`
}

type Collection[T any] struct {
	Data  []Resource[T] `json:"data"`
	Meta  Meta          `json:"meta,omitempty"`
	Links Links         `json:"links,omitempty"`
}

type Single[T any] struct {
	Data Resource[T] `json:"data"`
}

type Meta struct {
	TotalCount int `json:"total_count,omitempty"`
	Count      int `json:"count,omitempty"`
}

type Links struct {
	Self string `json:"self,omitempty"`
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

type ErrorResponse struct {
	Errors []ErrorObject `json:"errors"`
}

type ErrorObject struct {
	Status string `json:"status,omitempty"`
	Code   string `json:"code,omitempty"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type APIError struct {
	StatusCode int
	Errors     []ErrorObject
	Body       string
}

// Error deliberately excludes remote detail/body text: servers and proxies may
// echo request secrets or terminal escape sequences in either field.
func (e APIError) Error() string {
	return fmt.Sprintf("Planning Center API returned HTTP %d", e.StatusCode)
}

type ResponseError struct{ Reason string }

func (e *ResponseError) Error() string { return "invalid Planning Center response: " + e.Reason }

type TransportError struct {
	Operation      string
	UnknownOutcome bool
	Cause          error
}

func (e *TransportError) Error() string {
	if e.UnknownOutcome {
		return "Planning Center " + e.Operation + " outcome is unknown; do not replay automatically"
	}
	return "Planning Center " + e.Operation + " failed"
}
func (e *TransportError) Unwrap() error { return e.Cause }

func validateResource(data json.RawMessage) error {
	var resource struct {
		ID         string          `json:"id"`
		Type       string          `json:"type"`
		Attributes json.RawMessage `json:"attributes"`
	}
	if json.Unmarshal(data, &resource) != nil || resource.ID == "" || resource.Type == "" || len(resource.Attributes) == 0 || resource.Attributes[0] != '{' {
		return &ResponseError{Reason: "resource requires identity, type and attributes"}
	}
	return nil
}
func (c *Collection[T]) UnmarshalJSON(data []byte) error {
	type plain Collection[T]
	var shape struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &shape) != nil || len(shape.Data) == 0 || shape.Data[0] != '[' {
		return &ResponseError{Reason: "collection data must be an array"}
	}
	var resources []json.RawMessage
	if err := json.Unmarshal(shape.Data, &resources); err != nil {
		return err
	}
	for _, resource := range resources {
		if err := validateResource(resource); err != nil {
			return err
		}
	}
	return json.Unmarshal(data, (*plain)(c))
}
func (s *Single[T]) UnmarshalJSON(data []byte) error {
	type plain Single[T]
	var shape struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &shape) != nil || len(shape.Data) == 0 || shape.Data[0] != '{' {
		return &ResponseError{Reason: "single data must be an object"}
	}
	if err := validateResource(shape.Data); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(s))
}
