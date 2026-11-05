package services

import (
	"context"
	"fmt"
	"net/url"
)

// HTTPAPI is the transport seam. Library owns product policy above it.
type HTTPAPI interface {
	Get(context.Context, string, url.Values, any) error
	Post(context.Context, string, url.Values, any, any) error
	Patch(context.Context, string, url.Values, any, any) error
	Delete(context.Context, string, url.Values) error
	UploadFile(context.Context, string, any) error
}
type Library struct{ api *adapter }

func NewLibrary(http HTTPAPI) *Library { return &Library{api: &adapter{http: http}} }

type Visibility string

const (
	Visible Visibility = "visible"
	Hidden  Visibility = "hidden"
	All     Visibility = "all"
)

type Scope struct {
	ServiceTypeID   string     `json:"service_type_id,omitempty"`
	Song            string     `json:"song"`
	Arrangement     string     `json:"arrangement"`
	Key             string     `json:"key"`
	Visibility      Visibility `json:"visibility"`
	IncludeArchived bool       `json:"include_archived"`
}
type ListOptions struct {
	Limit int
	All   bool
}
type SongListOptions struct {
	ListOptions
	Title, Search, Author, Themes, Order string
	CCLINumber                           int
	Visibility                           Visibility
}
type ListMeta struct {
	Returned   int  `json:"returned"`
	HasMore    bool `json:"has_more"`
	Complete   bool `json:"complete"`
	TotalCount *int `json:"total_count,omitempty"`
}
type ListResult[T any] struct {
	Data []T      `json:"data"`
	Meta ListMeta `json:"meta"`
}
type SongShowOptions struct {
	Visibility      Visibility
	Arrangement     string
	IncludeArchived bool
}
type SongDetail struct {
	Song         Song                    `json:"song"`
	Arrangements ListResult[Arrangement] `json:"arrangements"`
	Arrangement  *ArrangementDetail      `json:"arrangement"`
}
type ArrangementDetail struct {
	Arrangement Arrangement     `json:"arrangement"`
	Keys        ListResult[Key] `json:"keys"`
}
type Error struct {
	Code             string            `json:"code"`
	Message          string            `json:"message"`
	Candidates       any               `json:"candidates,omitempty"`
	Recovery         *AttachmentResult `json:"recovery,omitempty"`
	PlanningRecovery *PlanningRecovery `json:"-"`
	Cause            error             `json:"-"`
}

func (e *Error) Error() string     { return e.Message }
func (e *Error) Unwrap() error     { return e.Cause }
func fail(code, msg string) *Error { return &Error{Code: code, Message: msg} }
func visible(v Visibility) (Visibility, error) {
	if v == "" {
		v = Visible
	}
	if v != Visible && v != Hidden && v != All {
		return v, fail("invalid_input", "visibility must be visible, hidden, or all")
	}
	return v, nil
}
func bound[T any](data []T, o ListOptions) (ListResult[T], error) {
	if o.Limit < 0 || o.All && o.Limit != 0 {
		return ListResult[T]{}, fail("invalid_input", "limit must be positive and cannot be combined with all")
	}
	if data == nil {
		data = []T{}
	}
	n := o.Limit
	if n == 0 {
		n = 25
	}
	more := !o.All && len(data) > n
	if more {
		data = data[:n]
	}
	return ListResult[T]{data, ListMeta{Returned: len(data), HasMore: more, Complete: !more}}, nil
}
func required(s, name string) error {
	if s == "" {
		return fail("invalid_input", name+" is required")
	}
	return nil
}
func incomplete(err error) error {
	return &Error{Code: "incomplete_lookup", Message: "lookup could not be completed: " + err.Error(), Cause: err}
}
func changed(kind string) error {
	return fail("concurrent_change", fmt.Sprintf("%s changed during selection; retry using its ID", kind))
}

func displayLimit(o ListOptions) int {
	if o.All {
		return -1
	}
	if o.Limit == 0 {
		return 25
	}
	return o.Limit
}
