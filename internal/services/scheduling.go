package services

import (
	"context"
	"encoding/json"
	"strings"
)

type TeamScope struct {
	ServiceType     string
	Team            string
	IncludeArchived bool
}
type TeamPerson struct {
	ID         string  `json:"id"`
	TeamID     string  `json:"team_id"`
	FullName   *string `json:"full_name"`
	Archived   *bool   `json:"archived"`
	ArchivedAt *string `json:"archived_at"`
}
type TeamPosition struct {
	ID       string `json:"id"`
	TeamID   string `json:"team_id"`
	Name     string `json:"name"`
	Sequence *int   `json:"sequence"`
}
type ScheduledPersonCreate struct {
	Team          string  `json:"-"`
	Person        string  `json:"-"`
	Position      string  `json:"-"`
	Status        *string `json:"status,omitempty"`
	Notes         *string `json:"notes,omitempty"`
	DeclineReason *string `json:"decline_reason,omitempty"`
}
type ScheduledPersonPatch struct {
	Team          string  `json:"-"`
	Position      *string `json:"-"`
	Status        *string `json:"status,omitempty"`
	Notes         *string `json:"notes,omitempty"`
	DeclineReason *string `json:"decline_reason,omitempty"`
}

func canonicalScheduleStatus(value string) (string, error) {
	switch strings.ToLower(value) {
	case "u", "unconfirmed":
		return "U", nil
	case "c", "confirmed":
		return "C", nil
	case "d", "declined":
		return "D", nil
	}
	return "", fail("invalid_input", "status must be unconfirmed, confirmed, declined, U, C, or D")
}
func validateSchedulingSelectors(values ...string) error {
	for _, value := range values {
		if _, _, _, e := ParseSelector(value); e != nil {
			return e
		}
	}
	return nil
}
func ValidateScheduledPersonCreate(p ScheduledPersonCreate) error {
	if p.Team == "" || p.Person == "" || p.Position == "" {
		return fail("invalid_input", "scheduling requires team, person, and position")
	}
	if e := validateSchedulingSelectors(p.Team, p.Person, p.Position); e != nil {
		return e
	}
	if p.Status != nil {
		_, e := canonicalScheduleStatus(*p.Status)
		return e
	}
	return nil
}
func ValidateScheduledPersonPatch(p ScheduledPersonPatch) error {
	if len(fields(p)) == 0 && p.Position == nil {
		return fail("invalid_input", "at least one assignment field is required")
	}
	if p.Team != "" {
		if e := validateSchedulingSelectors(p.Team); e != nil {
			return e
		}
	}
	if p.Position != nil {
		if e := validateSchedulingSelectors(*p.Position); e != nil {
			return e
		}
	}
	if p.Status != nil {
		_, e := canonicalScheduleStatus(*p.Status)
		return e
	}
	return nil
}
func (l *Library) teamContext(ctx context.Context, s TeamScope) (TeamScope, record, error) {
	var row record
	if s.Team == "" {
		return s, row, fail("invalid_input", "team is required")
	}
	if e := validateSchedulingSelectors(s.Team); e != nil {
		return s, row, e
	}
	parents, e := l.planningParents(ctx, PlanningScope{ServiceType: s.ServiceType, IncludeArchived: s.IncludeArchived}, false)
	if e != nil {
		return s, row, e
	}
	s.ServiceType = parents.ServiceType
	row, e = l.selectPlanning(ctx, serviceTypePath(s.ServiceType)+"/teams", "Team", s.Team, true, "name", s.IncludeArchived)
	if e != nil {
		return s, row, e
	}
	s.Team = row.id
	return s, row, nil
}
func wholePlanTeam(row record) error {
	var schedule *string
	if json.Unmarshal(row.attrs["schedule_to"], &schedule) != nil || schedule == nil || *schedule != "plan" {
		return fail("unsupported_capability", "scheduling changes require a team with known schedule_to=plan; split-time or unavailable scheduling modes are unsupported")
	}
	return nil
}
func teamPeoplePath(team string) string    { return "/services/v2/teams/" + team + "/people" }
func teamPositionsPath(team string) string { return "/services/v2/teams/" + team + "/team_positions" }
func (l *Library) ListTeamPeople(ctx context.Context, s TeamScope, o ListOptions) (ListResult[TeamPerson], error) {
	if _, e := bound([]TeamPerson{}, o); e != nil {
		return ListResult[TeamPerson]{}, e
	}
	s, _, e := l.teamContext(ctx, s)
	if e != nil {
		return ListResult[TeamPerson]{}, e
	}
	return planningList[TeamPerson](ctx, l, teamPeoplePath(s.Team), "Person", nil, o, s.IncludeArchived)
}
func (l *Library) WalkTeamPeople(ctx context.Context, s TeamScope, emit func(TeamPerson) error) (ListMeta, error) {
	if emit == nil {
		return ListMeta{}, fail("invalid_input", "an export callback is required")
	}
	s, _, e := l.teamContext(ctx, s)
	if e != nil {
		return ListMeta{}, e
	}
	return walkModels(ctx, l, teamPeoplePath(s.Team), "Person", nil, func(r record) (bool, error) { return planningVisible(r, s.IncludeArchived), nil }, emit)
}
func (l *Library) ShowTeamPerson(ctx context.Context, s TeamScope, target string) (TeamPerson, error) {
	var out TeamPerson
	if e := validateSchedulingSelectors(target); e != nil {
		return out, e
	}
	s, _, e := l.teamContext(ctx, s)
	if e != nil {
		return out, e
	}
	row, e := l.selectPlanning(ctx, teamPeoplePath(s.Team), "Person", target, false, "full_name", s.IncludeArchived)
	if e == nil {
		e = row.model(&out)
	}
	return out, e
}
func (l *Library) ListTeamPositions(ctx context.Context, s TeamScope, o ListOptions) (ListResult[TeamPosition], error) {
	if _, e := bound([]TeamPosition{}, o); e != nil {
		return ListResult[TeamPosition]{}, e
	}
	s, _, e := l.teamContext(ctx, s)
	if e != nil {
		return ListResult[TeamPosition]{}, e
	}
	return planningList[TeamPosition](ctx, l, teamPositionsPath(s.Team), "TeamPosition", nil, o, true)
}
func (l *Library) WalkTeamPositions(ctx context.Context, s TeamScope, emit func(TeamPosition) error) (ListMeta, error) {
	if emit == nil {
		return ListMeta{}, fail("invalid_input", "an export callback is required")
	}
	s, _, e := l.teamContext(ctx, s)
	if e != nil {
		return ListMeta{}, e
	}
	return walkModels(ctx, l, teamPositionsPath(s.Team), "TeamPosition", nil, nil, emit)
}
func (l *Library) ShowTeamPosition(ctx context.Context, s TeamScope, target string) (TeamPosition, error) {
	var out TeamPosition
	if e := validateSchedulingSelectors(target); e != nil {
		return out, e
	}
	s, _, e := l.teamContext(ctx, s)
	if e != nil {
		return out, e
	}
	row, e := l.selectPlanning(ctx, teamPositionsPath(s.Team), "TeamPosition", target, false, "name", true)
	if e == nil {
		e = row.model(&out)
	}
	return out, e
}

// Position assignment is a name-valued write. Even an ID cannot distinguish two
// catalog records that serialize to the same name; reject those before writing.
func (l *Library) schedulingPosition(ctx context.Context, team, target string, uniqueName bool) (record, error) {
	rows, e := l.api.collect(ctx, teamPositionsPath(team), "TeamPosition", nil)
	if e != nil {
		return record{}, incomplete(e)
	}
	selected, e := pick(rows, target, true, "name")
	if e != nil {
		return record{}, e
	}
	var name string
	_ = json.Unmarshal(selected.attrs["name"], &name)
	matches := 0
	for _, row := range rows {
		var other string
		_ = json.Unmarshal(row.attrs["name"], &other)
		if strings.EqualFold(name, other) {
			matches++
		}
	}
	if uniqueName && matches != 1 {
		candidates := []map[string]string{}
		for _, row := range rows {
			var other string
			_ = json.Unmarshal(row.attrs["name"], &other)
			if strings.EqualFold(name, other) {
				candidates = append(candidates, map[string]string{"id": row.id, "name": other})
			}
		}
		return record{}, &Error{Code: "ambiguous_target", Message: "position name is duplicated in the team and cannot be assigned unambiguously", Candidates: candidates}
	}
	return selected, nil
}
func (l *Library) member(ctx context.Context, team, target string) (record, error) {
	// Exhaust the association even for an ID: the requested person must belong to
	// the selected team, not merely exist in the organization directory.
	rows, e := l.api.collect(ctx, teamPeoplePath(team), "Person", nil)
	if e != nil {
		return record{}, incomplete(e)
	}
	selection, e := newRecordSelection(target, true, "full_name")
	if e != nil {
		return record{}, e
	}
	for _, candidate := range rows {
		if selection.id != "" {
			if candidate.id == selection.id {
				selection.exactCount++
				selection.firstExact = candidate
			}
			continue
		}
		if raw, ok := candidate.attrs["full_name"]; !ok || string(raw) == "null" {
			continue
		}
		if e = selection.add(candidate); e != nil {
			return record{}, e
		}
	}
	row, e := selection.result()
	if e != nil {
		return record{}, e
	}
	var archived *bool
	_ = json.Unmarshal(row.attrs["archived"], &archived)
	var at *string
	_ = json.Unmarshal(row.attrs["archived_at"], &at)
	_, hasArchived := row.attrs["archived"]
	atRaw, hasAt := row.attrs["archived_at"]
	if !hasArchived && (!hasAt || string(atRaw) != "null") && (at == nil || *at == "") {
		return record{}, fail("invalid_response", "person archive state is unavailable; cannot safely schedule")
	}
	if archived != nil && *archived || at != nil && *at != "" {
		return record{}, fail("invalid_input", "cannot schedule an archived person")
	}
	return row, nil
}
func scheduledPeoplePath(s PlanningScope) string { return planPath(s) + "/team_members" }
func (l *Library) ShowScheduledPerson(ctx context.Context, s PlanningScope, target string) (ScheduledPerson, error) {
	var out ScheduledPerson
	if e := validateSchedulingSelectors(target); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	row, e := l.selectPlanning(ctx, scheduledPeoplePath(s), "PlanPerson", target, false, "name", s.IncludeArchived)
	if e == nil {
		e = row.model(&out)
	}
	return out, e
}
func (l *Library) CreateScheduledPerson(ctx context.Context, s PlanningScope, p ScheduledPersonCreate) (ScheduledPerson, error) {
	var out ScheduledPerson
	if e := ValidateScheduledPersonCreate(p); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	teamScope, team, e := l.teamContext(ctx, TeamScope{ServiceType: s.ServiceType, Team: p.Team, IncludeArchived: s.IncludeArchived})
	if e != nil {
		return out, e
	}
	if e = wholePlanTeam(team); e != nil {
		return out, e
	}
	person, e := l.member(ctx, teamScope.Team, p.Person)
	if e != nil {
		return out, e
	}
	position, e := l.schedulingPosition(ctx, teamScope.Team, p.Position, true)
	if e != nil {
		return out, e
	}
	var name string
	_ = json.Unmarshal(position.attrs["name"], &name)
	attrs := fields(p)
	attrs["person_id"] = person.id
	attrs["team_id"] = team.id
	attrs["team_position_name"] = name
	attrs["prepare_notification"] = false
	status := "U"
	if p.Status != nil {
		status, _ = canonicalScheduleStatus(*p.Status)
	}
	attrs["status"] = status
	return writePlanningChild[ScheduledPerson](ctx, l, "POST", scheduledPeoplePath(s), "PlanPerson", s, "", attrs, map[string]string{"person": person.id, "team": team.id})
}
func (l *Library) assignmentContext(ctx context.Context, s PlanningScope, row record, teamGuard string, requireMembership bool) (TeamScope, error) {
	var scope TeamScope
	teamID, e := row.relatedID("team", "Team")
	if e != nil {
		return scope, e
	}
	personID, e := row.relatedID("person", "Person")
	if e != nil {
		return scope, e
	}
	if teamID == nil || personID == nil {
		return scope, fail("invalid_response", "assignment team and person identities are required for scheduling changes")
	}
	scope, team, e := l.teamContext(ctx, TeamScope{ServiceType: s.ServiceType, Team: *teamID, IncludeArchived: s.IncludeArchived})
	if e != nil {
		return scope, e
	}
	if e = wholePlanTeam(team); e != nil {
		return scope, e
	}
	var partial *bool
	_ = json.Unmarshal(row.attrs["can_accept_partial"], &partial)
	if partial != nil && *partial {
		return scope, fail("unsupported_capability", "partial or split-time assignments cannot be changed by these commands")
	}
	if teamGuard != "" {
		_, guard, e := l.teamContext(ctx, TeamScope{ServiceType: s.ServiceType, Team: teamGuard, IncludeArchived: s.IncludeArchived})
		if e != nil {
			return scope, e
		}
		if guard.id != *teamID {
			return scope, fail("invalid_input", "assignment does not belong to selected team")
		}
	}
	if requireMembership {
		if _, e = l.member(ctx, scope.Team, *personID); e != nil {
			return scope, e
		}
	}
	return scope, nil
}
func (l *Library) UpdateScheduledPerson(ctx context.Context, s PlanningScope, target string, p ScheduledPersonPatch) (ScheduledPerson, error) {
	var out ScheduledPerson
	if e := ValidateScheduledPersonPatch(p); e != nil {
		return out, e
	}
	if e := validateSchedulingSelectors(target); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	row, e := l.selectPlanning(ctx, scheduledPeoplePath(s), "PlanPerson", target, true, "name", s.IncludeArchived)
	if e != nil {
		return out, e
	}
	team, e := l.assignmentContext(ctx, s, row, p.Team, true)
	if e != nil {
		return out, e
	}
	attrs := fields(p)
	if p.Status != nil {
		attrs["status"], _ = canonicalScheduleStatus(*p.Status)
	}
	if p.Position != nil {
		position, e := l.schedulingPosition(ctx, team.Team, *p.Position, true)
		if e != nil {
			return out, e
		}
		var name string
		_ = json.Unmarshal(position.attrs["name"], &name)
		attrs["team_position_name"] = name
	}
	path := scheduledPeoplePath(s) + "/" + row.id
	if e = l.recheckPlanningRecord(ctx, path, row); e != nil {
		return out, e
	}
	// No prepare_notification attribute on updates: omitted means unchanged.
	personID, _ := row.relatedID("person", "Person")
	teamID, _ := row.relatedID("team", "Team")
	return writePlanningChild[ScheduledPerson](ctx, l, "PATCH", path, "PlanPerson", s, row.id, attrs, map[string]string{"person": *personID, "team": *teamID})
}
func (l *Library) DeleteScheduledPerson(ctx context.Context, s PlanningScope, target, teamGuard string) error {
	if e := validateSchedulingSelectors(target); e != nil {
		return e
	}
	if teamGuard != "" {
		if e := validateSchedulingSelectors(teamGuard); e != nil {
			return e
		}
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return e
	}
	row, e := l.selectPlanning(ctx, scheduledPeoplePath(s), "PlanPerson", target, true, "name", s.IncludeArchived)
	if e != nil {
		return e
	}
	if _, e = l.assignmentContext(ctx, s, row, teamGuard, false); e != nil {
		return e
	}
	path := scheduledPeoplePath(s) + "/" + row.id
	if e = l.recheckPlanningRecord(ctx, path, row); e != nil {
		return e
	}
	return l.deletePlanningChild(ctx, path, "PlanPerson", s, row.id)
}
