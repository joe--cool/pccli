package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

// PlanningRecovery retains validated remote identities after a later failure.
type PlanningReceipt struct {
	Resource      string `json:"resource"`
	ID            string `json:"id"`
	ServiceTypeID string `json:"service_type_id"`
	PlanID        string `json:"plan_id"`
}
type PlanningRecovery struct {
	Operation     string            `json:"operation"`
	Resource      string            `json:"resource"`
	ServiceTypeID string            `json:"service_type_id"`
	PlanID        string            `json:"plan_id"`
	TargetID      string            `json:"target_id"`
	Stage         string            `json:"stage"`
	Created       []PlanningReceipt `json:"created"`
}

func canonicalPlanSeries(id *string) *string {
	if id == nil {
		return nil
	}
	canonical := strings.TrimLeft(*id, "0")
	return &canonical
}
func planRecovery(operation, serviceTypeID, id string) *PlanningRecovery {
	recovery := &PlanningRecovery{Operation: operation, Resource: "Plan", ServiceTypeID: serviceTypeID, PlanID: id, TargetID: id, Stage: "readback", Created: []PlanningReceipt{}}
	if operation == "create" {
		recovery.Created = append(recovery.Created, PlanningReceipt{Resource: "Plan", ID: id, ServiceTypeID: serviceTypeID, PlanID: id})
	}
	return recovery
}
func (l *Library) checkPlanSeries(ctx context.Context, id *string) error {
	if id == nil {
		return nil
	}
	r, e := l.api.get(ctx, "/services/v2/series/"+*id, "Series", *id)
	if e != nil {
		return e
	}
	// Validate the known series title independently; a malformed lookup cannot
	// authorize attaching a plan to an untrustworthy identity.
	if raw, ok := r.attrs["title"]; ok {
		var title *string
		if json.Unmarshal(raw, &title) != nil {
			return fail("invalid_response", "invalid series title")
		}
	}
	return nil
}
func (l *Library) writePlan(ctx context.Context, method, path, id string, attrs map[string]any, seriesID *string) (Plan, error) {
	var m Plan
	submitted, e := l.api.mutate(ctx, method, path, "Plan", attrs)
	if e != nil {
		return m, e
	}
	if id != "" && submitted.id != id {
		return m, fail("outcome_unknown", "write submitted but response identified a different plan")
	}
	if method == "POST" {
		path += "/" + submitted.id
	}
	id = submitted.id
	operation := "update"
	if method == "POST" {
		operation = "create"
	}
	recovery := planRecovery(operation, submitted.planning.ServiceType, id)
	failed := func(message string, cause error) (Plan, error) {
		return m, &Error{Code: "verification_failed", Message: message, PlanningRecovery: recovery, Cause: cause}
	}
	read, e := l.api.get(ctx, path, "Plan", id)
	if e != nil {
		return failed(fmt.Sprintf("plan %s write submitted but readback failed; inspect the plan before retrying", id), e)
	}
	verifyAttrs := make(map[string]any, len(attrs))
	for k, v := range attrs {
		if k != "series_id" {
			verifyAttrs[k] = v
		}
	}
	if e = verify(read, verifyAttrs); e != nil {
		return failed(e.Error(), e)
	}
	if seriesID != nil {
		got, e := read.relatedID("series", "Series")
		if e != nil {
			return failed("plan write submitted but series relationship was invalid", e)
		}
		if got == nil || *got != *seriesID {
			return failed("plan write submitted but readback did not confirm series relationship", nil)
		}
	}
	if e = read.model(&m); e != nil {
		return failed("plan write submitted but readback model was invalid", e)
	}
	return m, nil
}
func (l *Library) CreatePlan(ctx context.Context, s PlanningScope, p PlanCreate) (Plan, error) {
	var m Plan
	if e := ValidatePlanCreate(p); e != nil {
		return m, e
	}
	p.SeriesID = canonicalPlanSeries(p.SeriesID)
	s, e := l.planningParents(ctx, s, false)
	if e != nil {
		return m, e
	}
	if e = l.checkPlanSeries(ctx, p.SeriesID); e != nil {
		return m, e
	}
	return l.writePlan(ctx, "POST", serviceTypePath(s.ServiceType)+"/plans", "", fields(p), p.SeriesID)
}

// DeletePlan verifies absence after the accepted DELETE, without retrying writes.
func (l *Library) DeletePlan(ctx context.Context, s PlanningScope, target string) error {
	if _, _, _, e := ParseSelector(target); e != nil {
		return e
	}
	s, e := l.planningParents(ctx, s, false)
	if e != nil {
		return e
	}
	r, e := l.selectPlanning(ctx, serviceTypePath(s.ServiceType)+"/plans", "Plan", target, true, "title", s.IncludeArchived)
	if e != nil {
		return e
	}
	path := serviceTypePath(s.ServiceType) + "/plans/" + r.id
	current, e := l.api.get(ctx, path, "Plan", r.id)
	if e != nil {
		return e
	}
	if !reflect.DeepEqual(r.attrs, current.attrs) {
		return changed("Plan")
	}
	if e = l.api.http.Delete(ctx, path, nil); e != nil {
		return e
	}
	_, e = l.api.get(ctx, path, "Plan", r.id)
	var api planningcenter.APIError
	if errors.As(e, &api) && api.StatusCode == 404 {
		return nil
	}
	return &Error{Code: "verification_failed", Message: fmt.Sprintf("plan %s deletion was accepted but absence could not be verified; inspect before retrying", r.id), PlanningRecovery: planRecovery("delete", s.ServiceType, r.id), Cause: e}
}
