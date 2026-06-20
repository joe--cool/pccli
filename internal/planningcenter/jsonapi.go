package planningcenter

import (
	"encoding/json"
	"fmt"
	"strings"
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

func (e APIError) Error() string {
	if len(e.Errors) == 0 {
		if strings.TrimSpace(e.Body) != "" {
			return fmt.Sprintf("Planning Center API returned HTTP %d: %s", e.StatusCode, strings.TrimSpace(e.Body))
		}
		return fmt.Sprintf("Planning Center API returned HTTP %d", e.StatusCode)
	}

	parts := make([]string, 0, len(e.Errors))
	for _, item := range e.Errors {
		switch {
		case item.Detail != "":
			parts = append(parts, item.Detail)
		case item.Title != "":
			parts = append(parts, item.Title)
		case item.Code != "":
			parts = append(parts, item.Code)
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("Planning Center API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("Planning Center API returned HTTP %d: %s", e.StatusCode, strings.Join(parts, "; "))
}
