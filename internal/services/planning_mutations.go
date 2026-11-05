package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

func childRecovery(operation, kind string, s PlanningScope, id string) *PlanningRecovery {
	recovery := &PlanningRecovery{Operation: operation, Resource: kind, ServiceTypeID: s.ServiceType, PlanID: s.Plan, TargetID: id, Stage: "readback", Created: []PlanningReceipt{}}
	if operation == "create" {
		recovery.Created = append(recovery.Created, PlanningReceipt{Resource: kind, ID: id, ServiceTypeID: s.ServiceType, PlanID: s.Plan})
	}
	return recovery
}

// Typed workflow callers construct attributes; this helper shares only commit
// classification and independent readback, not an arbitrary mutation interface.
func writePlanningChild[T any](ctx context.Context, l *Library, method, path, kind string, s PlanningScope, id string, attrs map[string]any, relationships map[string]string) (T, error) {
	var out T
	submitted, e := l.api.mutate(ctx, method, path, kind, attrs)
	if e != nil {
		return out, e
	}
	if id != "" && submitted.id != id {
		return out, fail("outcome_unknown", "write submitted but response identified a different "+kind)
	}
	operation := "update"
	if method == "POST" {
		operation = "create"
		path += "/" + submitted.id
	}
	id = submitted.id
	recovery := childRecovery(operation, kind, s, id)
	failed := func(message string, cause error) (T, error) {
		return out, &Error{Code: "verification_failed", Message: message, PlanningRecovery: recovery, Cause: cause}
	}
	read, e := l.api.get(ctx, path, kind, id)
	if e != nil {
		return failed(kind+" write submitted but readback failed", e)
	}
	if kind == "PlanPerson" {
		var partial *bool
		_ = json.Unmarshal(read.attrs["can_accept_partial"], &partial)
		if partial != nil && *partial {
			return failed("assignment write submitted but returned partial or split-time assignment", nil)
		}
	}
	if kind == "NeededPosition" {
		for name, relatedKind := range map[string]string{"time": "PlanTime", "time_preference_option": "TimePreferenceOption"} {
			id, e := read.relatedID(name, relatedKind)
			if e != nil {
				return failed("needed-position write submitted but time scope was invalid", e)
			}
			if id != nil {
				return failed("needed-position write submitted but readback became time-specific", nil)
			}
		}
	}
	verifyAttrs := map[string]any{}
	for name, v := range attrs {
		if !strings.HasSuffix(name, "_id") {
			verifyAttrs[name] = v
		}
	}
	if e = verifyPlanningChildFields(kind, read, verifyAttrs); e != nil {
		return failed(e.Error(), e)
	}
	for name, want := range relationships {
		if name == "item_type" {
			if e = verify(read, map[string]any{name: want}); e != nil {
				return failed(e.Error(), e)
			}
			continue
		}
		relatedKind := map[string]string{"song": "Song", "arrangement": "Arrangement", "key": "Key", "person": "Person", "team": "Team"}[name]
		got, e := read.relatedID(name, relatedKind)
		if e != nil {
			return failed("write submitted but linked identity is invalid", e)
		}
		if got == nil || *got != want {
			return failed("write submitted but linked "+name+" was not confirmed", nil)
		}
	}
	if e = read.model(&out); e != nil {
		return failed("write submitted but readback model was invalid", e)
	}
	return out, nil
}
func (l *Library) deletePlanningChild(ctx context.Context, path, kind string, s PlanningScope, id string) error {
	if e := l.api.http.Delete(ctx, path, nil); e != nil {
		return e
	}
	_, e := l.api.get(ctx, path, kind, id)
	var api planningcenter.APIError
	if errors.As(e, &api) && api.StatusCode == 404 {
		return nil
	}
	return &Error{Code: "verification_failed", Message: kind + " deletion accepted but absence could not be verified", PlanningRecovery: childRecovery("delete", kind, s, id), Cause: e}
}
