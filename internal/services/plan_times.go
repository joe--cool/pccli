package services

import (
	"context"
	"encoding/json"
	"regexp"
	"time"
)

type PlanTimePatch struct {
	Name     *string `json:"name,omitempty"`
	StartsAt *string `json:"starts_at,omitempty"`
	EndsAt   *string `json:"ends_at,omitempty"`
	TimeType *string `json:"time_type,omitempty"`
}
type PlanTimeCreate PlanTimePatch

var planningTimePattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)

func parsePlanningTime(value string) (time.Time, error) {
	if !planningTimePattern.MatchString(value) {
		return time.Time{}, fail("invalid_input", "timestamps must use RFC3339 with an explicit offset and at most nanosecond precision")
	}
	parsed, e := time.Parse(time.RFC3339Nano, value)
	if e != nil || parsed.Year() < 1 || parsed.Year() > 9999 {
		return time.Time{}, fail("invalid_input", "invalid RFC3339 timestamp (year must be 0001 through 9999)")
	}
	return parsed, nil
}
func validateTimeFields(p PlanTimePatch) error {
	for _, value := range []*string{p.StartsAt, p.EndsAt} {
		if value != nil {
			if _, e := parsePlanningTime(*value); e != nil {
				return e
			}
		}
	}
	if p.TimeType != nil && !oneOf(*p.TimeType, "service", "rehearsal", "other") {
		return fail("invalid_input", "time-type must be service, rehearsal, or other")
	}
	return validateTimeRange(p.StartsAt, p.EndsAt)
}
func validateTimeRange(start, end *string) error {
	if start == nil || end == nil {
		return nil
	}
	a, e := parsePlanningTime(*start)
	if e != nil {
		return e
	}
	b, e := parsePlanningTime(*end)
	if e != nil {
		return e
	}
	if b.Before(a) {
		return fail("invalid_input", "ends-at must not be before starts-at")
	}
	return nil
}
func ValidatePlanTimeCreate(p PlanTimeCreate) error {
	if p.StartsAt == nil || p.TimeType == nil {
		return fail("invalid_input", "plan-time creation requires starts-at and time-type")
	}
	return validateTimeFields(PlanTimePatch(p))
}
func ValidatePlanTimePatch(p PlanTimePatch) error {
	if len(fields(p)) == 0 {
		return fail("invalid_input", "at least one plan-time field is required")
	}
	return validateTimeFields(p)
}
func planTimesPath(s PlanningScope) string { return planPath(s) + "/plan_times" }
func (l *Library) ShowPlanTime(ctx context.Context, s PlanningScope, target string) (PlanTime, error) {
	var out PlanTime
	if _, _, _, e := ParseSelector(target); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	row, e := l.selectPlanning(ctx, planTimesPath(s), "PlanTime", target, false, "name", s.IncludeArchived)
	if e == nil {
		e = row.model(&out)
	}
	return out, e
}
func (l *Library) CreatePlanTime(ctx context.Context, s PlanningScope, p PlanTimeCreate) (PlanTime, error) {
	var out PlanTime
	if e := ValidatePlanTimeCreate(p); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	return writePlanningChild[PlanTime](ctx, l, "POST", planTimesPath(s), "PlanTime", s, "", fields(p), nil)
}
func (l *Library) UpdatePlanTime(ctx context.Context, s PlanningScope, target string, p PlanTimePatch) (PlanTime, error) {
	var out PlanTime
	if e := ValidatePlanTimePatch(p); e != nil {
		return out, e
	}
	if _, _, _, e := ParseSelector(target); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	row, e := l.selectPlanning(ctx, planTimesPath(s), "PlanTime", target, true, "name", s.IncludeArchived)
	if e != nil {
		return out, e
	}
	if p.StartsAt != nil || p.EndsAt != nil {
		var current PlanTime
		if e = row.model(&current); e != nil {
			return out, e
		}
		start, end := current.StartsAt, current.EndsAt
		if p.StartsAt != nil {
			start = p.StartsAt
		}
		if p.EndsAt != nil {
			end = p.EndsAt
		}
		if start == nil {
			return out, fail("invalid_response", "existing starts_at is unavailable; provide --starts-at with --ends-at")
		}
		if e = validateTimeRange(start, end); e != nil {
			return out, e
		}
	}
	path := planTimesPath(s) + "/" + row.id
	if e = l.recheckPlanningRecord(ctx, path, row); e != nil {
		return out, e
	}
	return writePlanningChild[PlanTime](ctx, l, "PATCH", path, "PlanTime", s, row.id, fields(p), nil)
}
func (l *Library) DeletePlanTime(ctx context.Context, s PlanningScope, target string) error {
	if _, _, _, e := ParseSelector(target); e != nil {
		return e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return e
	}
	row, e := l.selectPlanning(ctx, planTimesPath(s), "PlanTime", target, true, "name", s.IncludeArchived)
	if e != nil {
		return e
	}
	path := planTimesPath(s) + "/" + row.id
	if e = l.recheckPlanningRecord(ctx, path, row); e != nil {
		return e
	}
	return l.deletePlanningChild(ctx, path, "PlanTime", s, row.id)
}
func verifyPlanningChildFields(kind string, row record, attrs map[string]any) error {
	if kind == "PlanPerson" {
		others := map[string]any{}
		for name, value := range attrs {
			if name != "status" {
				others[name] = value
				continue
			}
			var status string
			if json.Unmarshal(row.attrs[name], &status) != nil {
				return fail("verification_failed", "assignment status was not confirmed")
			}
			got, e := canonicalScheduleStatus(status)
			if e != nil || got != value {
				return fail("verification_failed", "assignment status differs from requested value")
			}
		}
		return verify(row, others)
	}
	if kind != "PlanTime" {
		return verify(row, attrs)
	}
	others := map[string]any{}
	for name, want := range attrs {
		if name != "starts_at" && name != "ends_at" {
			others[name] = want
			continue
		}
		var got *string
		if raw, ok := row.attrs[name]; !ok || json.Unmarshal(raw, &got) != nil || got == nil {
			return fail("verification_failed", "plan-time write submitted but "+name+" was unavailable")
		}
		actual, e := parsePlanningTime(*got)
		if e != nil {
			return fail("verification_failed", "plan-time write submitted but returned timestamp was invalid")
		}
		expected, e := parsePlanningTime(want.(string))
		if e != nil || !actual.Equal(expected) {
			return fail("verification_failed", "plan-time write submitted but "+name+" differs")
		}
	}
	return verify(row, others)
}
