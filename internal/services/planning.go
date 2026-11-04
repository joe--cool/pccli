package services

import (
	"context"
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
)

// PlanningScope names explicit parents. It is independent of song-library scope.
type PlanningScope struct {
	ServiceType     string `json:"service_type_id"`
	Plan            string `json:"plan_id"`
	IncludeArchived bool   `json:"include_archived"`
}

type ServiceType struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Frequency  *string `json:"frequency"`
	ArchivedAt *string `json:"archived_at"`
}
type Plan struct {
	ID                   string  `json:"id"`
	ServiceTypeID        string  `json:"service_type_id"`
	Title                *string `json:"title"`
	SeriesTitle          *string `json:"series_title"`
	Dates                *string `json:"dates"`
	ShortDates           *string `json:"short_dates"`
	SortDate             *string `json:"sort_date"`
	LastTimeAt           *string `json:"last_time_at"`
	ItemsCount           *int    `json:"items_count"`
	PlanPeopleCount      *int    `json:"plan_people_count"`
	NeededPositionsCount *int    `json:"needed_positions_count"`
	TotalLength          *int    `json:"total_length"`
	Public               *bool   `json:"public"`
	PlanningCenterURL    *string `json:"planning_center_url"`
}
type Item struct {
	SongID          *string `json:"song_id"`
	ArrangementID   *string `json:"arrangement_id"`
	KeyID           *string `json:"key_id"`
	ID              string  `json:"id"`
	ServiceTypeID   string  `json:"service_type_id"`
	PlanID          string  `json:"plan_id"`
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	ItemType        *string `json:"item_type"`
	ServicePosition *string `json:"service_position"`
	Sequence        *int    `json:"sequence"`
	Length          *int    `json:"length"`
	KeyName         *string `json:"key_name"`
}
type Team struct {
	ID            string  `json:"id"`
	ServiceTypeID string  `json:"service_type_id"`
	Name          string  `json:"name"`
	ArchivedAt    *string `json:"archived_at"`
	ScheduleTo    *string `json:"schedule_to"`
	RehearsalTeam *bool   `json:"rehearsal_team"`
}
type PlanTime struct {
	ID            string  `json:"id"`
	ServiceTypeID string  `json:"service_type_id"`
	PlanID        string  `json:"plan_id"`
	Name          *string `json:"name"`
	StartsAt      *string `json:"starts_at"`
	EndsAt        *string `json:"ends_at"`
	TimeType      *string `json:"time_type"`
}
type ScheduledPerson struct {
	PersonID         *string `json:"person_id"`
	TeamID           *string `json:"team_id"`
	ID               string  `json:"id"`
	ServiceTypeID    string  `json:"service_type_id"`
	PlanID           string  `json:"plan_id"`
	Name             *string `json:"name"`
	Status           *string `json:"status"`
	TeamPositionName *string `json:"team_position_name"`
}
type ServiceTypeListOptions struct {
	ListOptions
	IncludeArchived bool
}
type PlanListOptions struct {
	ListOptions
	Filter string
}
type PlanPatch struct {
	Title       *string `json:"title,omitempty"`
	SeriesTitle *string `json:"series_title,omitempty"`
}

func ValidatePlanPatch(p PlanPatch) error {
	if p.Title == nil && p.SeriesTitle == nil {
		return fail("invalid_input", "at least one of title or series-title is required")
	}
	return nil
}
func planningType(kind string) any {
	switch kind {
	case "ServiceType":
		return &ServiceType{}
	case "Plan":
		return &Plan{}
	case "Item":
		return &Item{}
	case "Team":
		return &Team{}
	case "PlanTime":
		return &PlanTime{}
	case "PlanPerson":
		return &ScheduledPerson{}
	}
	return nil
}

// Nullable provider text stays nullable in the machine contract. Known numeric
// and boolean fields cannot silently accept null as a fabricated zero value.
func validatePlanningAttributes(kind string, attrs map[string]json.RawMessage) error {
	model := planningType(kind)
	if model == nil {
		return nil
	}
	typ := reflect.TypeOf(model).Elem()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := field.Tag.Get("json")
		raw, ok := attrs[name]
		if !ok {
			continue
		}
		if name == "id" || name == "service_type_id" || name == "plan_id" {
			continue
		}
		if string(raw) == "null" && (field.Type.Kind() != reflect.Pointer || field.Type.Elem().Kind() != reflect.String) {
			return fail("invalid_response", "invalid null "+kind+" attribute "+name)
		}
	}
	raw, _ := json.Marshal(attrs)
	if json.Unmarshal(raw, model) != nil {
		return fail("invalid_response", "invalid typed "+kind+" attributes")
	}
	if kind == "ServiceType" || kind == "Team" {
		var name string
		if json.Unmarshal(attrs["name"], &name) != nil || strings.TrimSpace(name) == "" {
			return fail("invalid_response", "resource name missing or invalid")
		}
	}
	return nil
}
func serviceTypePath(id string) string { return "/services/v2/service_types/" + id }
func planPath(s PlanningScope) string  { return serviceTypePath(s.ServiceType) + "/plans/" + s.Plan }
func validatePlanningScope(s PlanningScope, needPlan bool) error {
	if s.ServiceType == "" {
		return fail("invalid_input", "--service-type is required")
	}
	if _, _, _, err := ParseSelector(s.ServiceType); err != nil {
		return err
	}
	if needPlan && s.Plan == "" {
		return fail("invalid_input", "--plan is required")
	}
	if s.Plan != "" {
		_, _, _, err := ParseSelector(s.Plan)
		return err
	}
	return nil
}
func (l *Library) selectPlanning(ctx context.Context, path, kind, target string, exact bool, field string, includeArchived bool) (record, error) {
	id, _, _, err := ParseSelector(target)
	if err != nil {
		return record{}, err
	}
	if id != "" {
		r, e := l.api.get(ctx, path+"/"+id, kind, id)
		if e == nil && !planningVisible(r, includeArchived) {
			e = fail("not_found", "record is archived; use --include-archived")
		}
		return r, e
	}
	choice, err := newRecordSelection(target, exact, field)
	if err != nil {
		return record{}, err
	}
	if err = l.api.walk(ctx, path, kind, nil, func(r record) error {
		if !planningVisible(r, includeArchived) {
			return nil
		}
		if kind == "Plan" || kind == "Item" {
			if raw, ok := r.attrs[field]; !ok || string(raw) == "null" {
				return nil
			}
		}
		return choice.add(r)
	}); err != nil {
		return record{}, incomplete(err)
	}
	return choice.result()
}
func (l *Library) planningParents(ctx context.Context, s PlanningScope, needPlan bool) (PlanningScope, error) {
	if err := validatePlanningScope(s, needPlan); err != nil {
		return s, err
	}
	st, err := l.selectPlanning(ctx, "/services/v2/service_types", "ServiceType", s.ServiceType, true, "name", s.IncludeArchived)
	if err != nil {
		return s, err
	}
	s.ServiceType = st.id
	if needPlan {
		plan, err := l.selectPlanning(ctx, serviceTypePath(st.id)+"/plans", "Plan", s.Plan, true, "title", s.IncludeArchived)
		if err != nil {
			return s, err
		}
		s.Plan = plan.id
	}
	return s, nil
}
func planningList[T any](ctx context.Context, l *Library, path, kind string, q url.Values, o ListOptions, includeArchived bool) (ListResult[T], error) {
	if _, err := bound([]T{}, o); err != nil {
		return ListResult[T]{}, err
	}
	limit := o.Limit
	if limit == 0 {
		limit = 25
	}
	if o.All {
		limit = -1
	}
	records, err := l.api.collectLimited(ctx, path, kind, q, limit, func(r record) (bool, error) { return planningVisible(r, includeArchived), nil })
	if err != nil {
		return ListResult[T]{}, err
	}
	out := make([]T, 0, len(records))
	for _, r := range records {
		var m T
		if err = r.model(&m); err != nil {
			return ListResult[T]{}, err
		}
		out = append(out, m)
	}
	return bound(out, o)
}
func (l *Library) ListServiceTypes(ctx context.Context, o ServiceTypeListOptions) (ListResult[ServiceType], error) {
	return planningList[ServiceType](ctx, l, "/services/v2/service_types", "ServiceType", nil, o.ListOptions, o.IncludeArchived)
}
func (l *Library) WalkServiceTypes(ctx context.Context, includeArchived bool, emit func(ServiceType) error) (ListMeta, error) {
	return walkModels(ctx, l, "/services/v2/service_types", "ServiceType", nil, func(r record) (bool, error) { return planningVisible(r, includeArchived), nil }, emit)
}
func (l *Library) ShowServiceType(ctx context.Context, target string, includeArchived bool) (ServiceType, error) {
	r, e := l.selectPlanning(ctx, "/services/v2/service_types", "ServiceType", target, false, "name", includeArchived)
	var m ServiceType
	if e == nil {
		e = r.model(&m)
	}
	return m, e
}
func planQuery(filter string) (url.Values, error) {
	if filter == "" {
		return nil, nil
	}
	if !oneOf(filter, "future", "past", "no_dates") {
		return nil, fail("invalid_input", "filter must be future, past, or no_dates")
	}
	return url.Values{"filter": {filter}}, nil
}
func (l *Library) ListPlans(ctx context.Context, s PlanningScope, o PlanListOptions) (ListResult[Plan], error) {
	q, e := planQuery(o.Filter)
	if e != nil {
		return ListResult[Plan]{}, e
	}
	if _, e = bound([]Plan{}, o.ListOptions); e != nil {
		return ListResult[Plan]{}, e
	}
	s, e = l.planningParents(ctx, s, false)
	if e != nil {
		return ListResult[Plan]{}, e
	}
	return planningList[Plan](ctx, l, serviceTypePath(s.ServiceType)+"/plans", "Plan", q, o.ListOptions, s.IncludeArchived)
}
func (l *Library) WalkPlans(ctx context.Context, s PlanningScope, filter string, emit func(Plan) error) (ListMeta, error) {
	q, e := planQuery(filter)
	if e != nil {
		return ListMeta{}, e
	}
	if emit == nil {
		return ListMeta{}, fail("invalid_input", "an export callback is required")
	}
	s, e = l.planningParents(ctx, s, false)
	if e != nil {
		return ListMeta{}, e
	}
	return walkModels(ctx, l, serviceTypePath(s.ServiceType)+"/plans", "Plan", q, nil, emit)
}
func (l *Library) ShowPlan(ctx context.Context, s PlanningScope, target string) (Plan, error) {
	var m Plan
	if _, _, _, e := ParseSelector(target); e != nil {
		return m, e
	}
	s, e := l.planningParents(ctx, s, false)
	if e != nil {
		return m, e
	}
	r, e := l.selectPlanning(ctx, serviceTypePath(s.ServiceType)+"/plans", "Plan", target, false, "title", s.IncludeArchived)
	if e == nil {
		e = r.model(&m)
	}
	return m, e
}
func (l *Library) UpdatePlan(ctx context.Context, s PlanningScope, target string, p PlanPatch) (Plan, error) {
	var m Plan
	if e := ValidatePlanPatch(p); e != nil {
		return m, e
	}
	if _, _, _, e := ParseSelector(target); e != nil {
		return m, e
	}
	s, e := l.planningParents(ctx, s, false)
	if e != nil {
		return m, e
	}
	r, e := l.selectPlanning(ctx, serviceTypePath(s.ServiceType)+"/plans", "Plan", target, true, "title", s.IncludeArchived)
	if e != nil {
		return m, e
	}
	attrs := map[string]any{}
	if p.Title != nil {
		attrs["title"] = *p.Title
	}
	if p.SeriesTitle != nil {
		attrs["series_title"] = *p.SeriesTitle
	}
	path := serviceTypePath(s.ServiceType) + "/plans/" + r.id
	current, e := l.api.get(ctx, path, "Plan", r.id)
	if e != nil {
		return m, e
	}
	if !reflect.DeepEqual(current.attrs, r.attrs) {
		return m, changed("Plan")
	}
	updated, e := l.api.mutate(ctx, "PATCH", path, "Plan", attrs)
	if e != nil {
		return m, e
	}
	if updated.id != r.id {
		return m, fail("outcome_unknown", "write submitted but response identified a different plan")
	}
	updated, e = l.api.get(ctx, path, "Plan", r.id)
	if e != nil {
		return m, &Error{Code: "verification_failed", Message: "plan update submitted but readback failed; inspect the plan before retrying", Cause: e}
	}
	if e = verify(updated, attrs); e != nil {
		return m, e
	}
	e = updated.model(&m)
	return m, e
}

func (l *Library) ListItems(ctx context.Context, s PlanningScope, o ListOptions) (ListResult[Item], error) {
	if _, e := bound([]Item{}, o); e != nil {
		return ListResult[Item]{}, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return ListResult[Item]{}, e
	}
	return planningList[Item](ctx, l, planPath(s)+"/items", "Item", nil, o, s.IncludeArchived)
}
func (l *Library) WalkItems(ctx context.Context, s PlanningScope, emit func(Item) error) (ListMeta, error) {
	if emit == nil {
		return ListMeta{}, fail("invalid_input", "an export callback is required")
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return ListMeta{}, e
	}
	return walkModels(ctx, l, planPath(s)+"/items", "Item", nil, nil, emit)
}
func (l *Library) ShowItem(ctx context.Context, s PlanningScope, target string) (Item, error) {
	var m Item
	if _, _, _, e := ParseSelector(target); e != nil {
		return m, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return m, e
	}
	r, e := l.selectPlanning(ctx, planPath(s)+"/items", "Item", target, false, "title", s.IncludeArchived)
	if e == nil {
		e = r.model(&m)
	}
	return m, e
}

func (l *Library) ListTeams(ctx context.Context, s PlanningScope, o ListOptions) (ListResult[Team], error) {
	if _, e := bound([]Team{}, o); e != nil {
		return ListResult[Team]{}, e
	}
	s, e := l.planningParents(ctx, s, false)
	if e != nil {
		return ListResult[Team]{}, e
	}
	return planningList[Team](ctx, l, serviceTypePath(s.ServiceType)+"/teams", "Team", nil, o, s.IncludeArchived)
}
func (l *Library) WalkTeams(ctx context.Context, s PlanningScope, emit func(Team) error) (ListMeta, error) {
	if emit == nil {
		return ListMeta{}, fail("invalid_input", "an export callback is required")
	}
	s, e := l.planningParents(ctx, s, false)
	if e != nil {
		return ListMeta{}, e
	}
	return walkModels(ctx, l, serviceTypePath(s.ServiceType)+"/teams", "Team", nil, func(r record) (bool, error) { return planningVisible(r, s.IncludeArchived), nil }, emit)
}
func (l *Library) ShowTeam(ctx context.Context, s PlanningScope, target string) (Team, error) {
	var m Team
	if _, _, _, e := ParseSelector(target); e != nil {
		return m, e
	}
	s, e := l.planningParents(ctx, s, false)
	if e != nil {
		return m, e
	}
	r, e := l.selectPlanning(ctx, serviceTypePath(s.ServiceType)+"/teams", "Team", target, false, "name", s.IncludeArchived)
	if e == nil {
		e = r.model(&m)
	}
	return m, e
}

func (l *Library) ListPlanTimes(ctx context.Context, s PlanningScope, o ListOptions) (ListResult[PlanTime], error) {
	if _, e := bound([]PlanTime{}, o); e != nil {
		return ListResult[PlanTime]{}, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return ListResult[PlanTime]{}, e
	}
	return planningList[PlanTime](ctx, l, planPath(s)+"/plan_times", "PlanTime", nil, o, s.IncludeArchived)
}
func (l *Library) WalkPlanTimes(ctx context.Context, s PlanningScope, emit func(PlanTime) error) (ListMeta, error) {
	if emit == nil {
		return ListMeta{}, fail("invalid_input", "an export callback is required")
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return ListMeta{}, e
	}
	return walkModels(ctx, l, planPath(s)+"/plan_times", "PlanTime", nil, nil, emit)
}

func (l *Library) ListScheduledPeople(ctx context.Context, s PlanningScope, o ListOptions) (ListResult[ScheduledPerson], error) {
	if _, e := bound([]ScheduledPerson{}, o); e != nil {
		return ListResult[ScheduledPerson]{}, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return ListResult[ScheduledPerson]{}, e
	}
	return planningList[ScheduledPerson](ctx, l, planPath(s)+"/team_members", "PlanPerson", nil, o, s.IncludeArchived)
}
func (l *Library) WalkScheduledPeople(ctx context.Context, s PlanningScope, emit func(ScheduledPerson) error) (ListMeta, error) {
	if emit == nil {
		return ListMeta{}, fail("invalid_input", "an export callback is required")
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return ListMeta{}, e
	}
	return walkModels(ctx, l, planPath(s)+"/team_members", "PlanPerson", nil, nil, emit)
}

// Teams may be shared by several service types. The singular service_type is
// their original owner, while a plural membership explicitly constrains scope.
func (r record) teamScope() error {
	if len(r.rel) == 0 {
		return nil
	}
	var rel map[string]json.RawMessage
	if json.Unmarshal(r.rel, &rel) != nil || rel == nil {
		return fail("invalid_response", "invalid team relationships")
	}
	if owner, err := r.relatedID("service_type", "ServiceType"); err != nil {
		return err
	} else if _, ok := rel["service_type"]; ok && owner == nil {
		return fail("invalid_response", "invalid team service_type relationship")
	}
	raw, ok := rel["service_types"]
	if !ok {
		return nil
	}
	var association struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &association) != nil || len(association.Data) == 0 || association.Data[0] != '[' {
		return fail("invalid_response", "invalid team service_types relationship")
	}
	var data []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if json.Unmarshal(association.Data, &data) != nil {
		return fail("invalid_response", "invalid team service_types relationship")
	}
	found := false
	seen := map[string]bool{}
	for _, entry := range data {
		if entry.Type != "ServiceType" || !validID(entry.ID) || seen[entry.ID] {
			return fail("invalid_response", "invalid team service type identity")
		}
		seen[entry.ID] = true
		if entry.ID == r.planning.ServiceType {
			found = true
		}
	}
	if r.planning.ServiceType != "" && !found {
		return fail("invalid_response", "team does not belong to requested service type")
	}
	return nil
}

func planningVisible(r record, includeArchived bool) bool {
	if includeArchived || (r.kind != "ServiceType" && r.kind != "Team") {
		return true
	}
	var archived *string
	_ = json.Unmarshal(r.attrs["archived_at"], &archived)
	return archived == nil || *archived == ""
}

// Optional relationship identities preserve assignment context without resolving
// or exposing a global people directory.
func (r record) relatedID(name, kind string) (*string, error) {
	if len(r.rel) == 0 {
		return nil, nil
	}
	var relationships map[string]json.RawMessage
	if json.Unmarshal(r.rel, &relationships) != nil || relationships == nil {
		return nil, fail("invalid_response", "invalid resource relationships")
	}
	raw, ok := relationships[name]
	if !ok {
		return nil, nil
	}
	var rel struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &rel) != nil || len(rel.Data) == 0 {
		return nil, fail("invalid_response", "invalid "+name+" relationship")
	}
	if string(rel.Data) == "null" {
		return nil, nil
	}
	var data struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if json.Unmarshal(rel.Data, &data) != nil || data.Type != kind || !validID(data.ID) {
		return nil, fail("invalid_response", "invalid "+name+" identity")
	}
	return &data.ID, nil
}
