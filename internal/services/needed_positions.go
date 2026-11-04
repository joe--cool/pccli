package services

import (
	"context"
	"encoding/json"
)

type NeededPosition struct {
	ID                     string  `json:"id"`
	ServiceTypeID          string  `json:"service_type_id"`
	PlanID                 string  `json:"plan_id"`
	TeamID                 *string `json:"team_id"`
	TimeID                 *string `json:"time_id"`
	TimePreferenceOptionID *string `json:"time_preference_option_id"`
	Quantity               *int    `json:"quantity"`
	TeamPositionName       *string `json:"team_position_name"`
	ScheduledTo            *string `json:"scheduled_to"`
}
type NeededPositionCreate struct {
	Team     string `json:"-"`
	Position string `json:"-"`
	Quantity *int   `json:"quantity,omitempty"`
}
type NeededPositionPatch struct {
	Quantity *int `json:"quantity,omitempty"`
}
type NeededPositionResult struct {
	NeededPosition      NeededPosition `json:"needed_position"`
	Outcome             string         `json:"outcome"`
	UnverifiedFields    []string       `json:"unverified_fields"`
	RequestedPositionID *string        `json:"requested_position_id"`
}

func ValidateNeededPositionCreate(p NeededPositionCreate) error {
	if p.Team == "" || p.Position == "" || p.Quantity == nil {
		return fail("invalid_input", "needed-position creation requires team, position, and quantity")
	}
	if e := validateSchedulingSelectors(p.Team, p.Position); e != nil {
		return e
	}
	return ValidateNeededPositionPatch(NeededPositionPatch{Quantity: p.Quantity})
}
func ValidateNeededPositionPatch(p NeededPositionPatch) error {
	if p.Quantity == nil || *p.Quantity < 0 {
		return fail("invalid_input", "quantity must be supplied and nonnegative")
	}
	return nil
}
func neededPositionsPath(s PlanningScope) string { return planPath(s) + "/needed_positions" }
func (l *Library) ListNeededPositions(ctx context.Context, s PlanningScope, o ListOptions) (ListResult[NeededPosition], error) {
	if _, e := bound([]NeededPosition{}, o); e != nil {
		return ListResult[NeededPosition]{}, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return ListResult[NeededPosition]{}, e
	}
	return planningList[NeededPosition](ctx, l, neededPositionsPath(s), "NeededPosition", nil, o, s.IncludeArchived)
}
func (l *Library) WalkNeededPositions(ctx context.Context, s PlanningScope, emit func(NeededPosition) error) (ListMeta, error) {
	if emit == nil {
		return ListMeta{}, fail("invalid_input", "an export callback is required")
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return ListMeta{}, e
	}
	return walkModels(ctx, l, neededPositionsPath(s), "NeededPosition", nil, nil, emit)
}
func (l *Library) ShowNeededPosition(ctx context.Context, s PlanningScope, target string) (NeededPosition, error) {
	var out NeededPosition
	if e := validateSchedulingSelectors(target); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	row, e := l.selectPlanning(ctx, neededPositionsPath(s), "NeededPosition", target, false, "team_position_name", s.IncludeArchived)
	if e == nil {
		e = row.model(&out)
	}
	return out, e
}
func (l *Library) CreateNeededPosition(ctx context.Context, s PlanningScope, p NeededPositionCreate) (NeededPositionResult, error) {
	var out NeededPositionResult
	if e := ValidateNeededPositionCreate(p); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	scope, team, e := l.teamContext(ctx, TeamScope{ServiceType: s.ServiceType, Team: p.Team, IncludeArchived: s.IncludeArchived})
	if e != nil {
		return out, e
	}
	if e = wholePlanTeam(team); e != nil {
		return out, e
	}
	position, e := l.schedulingPosition(ctx, scope.Team, p.Position, false)
	if e != nil {
		return out, e
	}
	var name string
	_ = json.Unmarshal(position.attrs["name"], &name)
	attrs := fields(p)
	attrs["team_id"] = team.id
	attrs["team_position_id"] = position.id
	created, e := writePlanningChild[NeededPosition](ctx, l, "POST", neededPositionsPath(s), "NeededPosition", s, "", attrs, map[string]string{"team": team.id})
	if e != nil {
		return out, e
	}
	if created.TeamPositionName == nil || *created.TeamPositionName != name {
		return out, &Error{Code: "verification_failed", Message: "needed-position creation accepted but readable position name was not confirmed", PlanningRecovery: childRecovery("create", "NeededPosition", s, created.ID)}
	}
	// The provider does not expose the requested position ID for readback.
	id := position.id
	return NeededPositionResult{NeededPosition: created, Outcome: "accepted", UnverifiedFields: []string{"team_position_id"}, RequestedPositionID: &id}, nil
}
func (l *Library) neededContext(ctx context.Context, s PlanningScope, row record) error {
	team, e := row.relatedID("team", "Team")
	if e != nil {
		return e
	}
	if team == nil {
		return fail("invalid_response", "needed-position team identity is required for changes")
	}
	for name, kind := range map[string]string{"time": "PlanTime", "time_preference_option": "TimePreferenceOption"} {
		id, e := row.relatedID(name, kind)
		if e != nil {
			return e
		}
		if id != nil {
			return fail("unsupported_capability", "time-specific needed positions cannot be changed by whole-plan commands")
		}
	}
	_, selected, e := l.teamContext(ctx, TeamScope{ServiceType: s.ServiceType, Team: *team, IncludeArchived: s.IncludeArchived})
	if e != nil {
		return e
	}
	return wholePlanTeam(selected)
}
func (l *Library) UpdateNeededPosition(ctx context.Context, s PlanningScope, target string, p NeededPositionPatch) (NeededPositionResult, error) {
	var out NeededPositionResult
	if e := ValidateNeededPositionPatch(p); e != nil {
		return out, e
	}
	if e := validateSchedulingSelectors(target); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	row, e := l.selectPlanning(ctx, neededPositionsPath(s), "NeededPosition", target, true, "team_position_name", s.IncludeArchived)
	if e != nil {
		return out, e
	}
	if e = l.neededContext(ctx, s, row); e != nil {
		return out, e
	}
	path := neededPositionsPath(s) + "/" + row.id
	if e = l.recheckPlanningRecord(ctx, path, row); e != nil {
		return out, e
	}
	teamID, _ := row.relatedID("team", "Team")
	updated, e := writePlanningChild[NeededPosition](ctx, l, "PATCH", path, "NeededPosition", s, row.id, fields(p), map[string]string{"team": *teamID})
	if e != nil {
		return out, e
	}
	if updated.TimeID != nil || updated.TimePreferenceOptionID != nil {
		return out, &Error{Code: "verification_failed", Message: "needed-position update accepted but readback became time-specific", PlanningRecovery: childRecovery("update", "NeededPosition", s, row.id)}
	}
	return NeededPositionResult{NeededPosition: updated, Outcome: "verified", UnverifiedFields: []string{}}, nil
}
func (l *Library) DeleteNeededPosition(ctx context.Context, s PlanningScope, target string) error {
	if e := validateSchedulingSelectors(target); e != nil {
		return e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return e
	}
	row, e := l.selectPlanning(ctx, neededPositionsPath(s), "NeededPosition", target, true, "team_position_name", s.IncludeArchived)
	if e != nil {
		return e
	}
	if e = l.neededContext(ctx, s, row); e != nil {
		return e
	}
	path := neededPositionsPath(s) + "/" + row.id
	if e = l.recheckPlanningRecord(ctx, path, row); e != nil {
		return e
	}
	return l.deletePlanningChild(ctx, path, "NeededPosition", s, row.id)
}
