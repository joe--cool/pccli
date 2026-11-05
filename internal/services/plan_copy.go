package services

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

type PlanCopyOptions struct {
	Title               *string    `json:"title,omitempty"`
	CopyItems           bool       `json:"copy_items"`
	CopyTimes           bool       `json:"copy_times"`
	CopyPeople          bool       `json:"copy_people"`
	CopyNeededPositions bool       `json:"copy_needed_positions"`
	CopySettings        bool       `json:"copy_settings"`
	TimeOffset          *string    `json:"time_offset"`
	PreserveDates       bool       `json:"preserve_dates"`
	Visibility          Visibility `json:"visibility"`
}
type CopyUnverifiedField struct {
	Resource    string `json:"resource"`
	ID          string `json:"id"`
	Field       string `json:"field"`
	RequestedID string `json:"requested_id"`
}
type PlanCopyResult struct {
	Plan             Plan                  `json:"plan"`
	SourcePlanID     string                `json:"source_plan_id"`
	Components       []string              `json:"components"`
	Created          []PlanningReceipt     `json:"created"`
	Outcome          string                `json:"outcome"`
	UnverifiedFields []CopyUnverifiedField `json:"unverified_fields"`
}

func ValidatePlanCopyOptions(o PlanCopyOptions) error {
	if !o.CopyItems && !o.CopyTimes && !o.CopyPeople && !o.CopyNeededPositions && !o.CopySettings {
		return fail("invalid_input", "select at least one copy component")
	}
	if !o.CopyTimes && (o.TimeOffset != nil || o.PreserveDates) {
		return fail("invalid_input", "time-offset and preserve-dates require copy-times")
	}
	if o.CopyTimes && ((o.TimeOffset == nil && !o.PreserveDates) || (o.TimeOffset != nil && o.PreserveDates)) {
		return fail("invalid_input", "copy-times requires exactly one of time-offset or preserve-dates")
	}
	if o.TimeOffset != nil {
		if _, e := time.ParseDuration(*o.TimeOffset); e != nil {
			return fail("invalid_input", "time-offset must be an elapsed Go duration, such as 168h or -24h")
		}
	}
	_, e := visible(o.Visibility)
	return e
}

type copyItemInput struct {
	sourceID string
	input    ItemCreate
}
type copyTimeInput struct {
	sourceID string
	input    PlanTimeCreate
}
type copyPersonInput struct {
	sourceID string
	input    ScheduledPersonCreate
}
type copyNeedInput struct {
	sourceID string
	input    NeededPositionCreate
}
type copyCollection struct {
	path, kind string
	records    []record
}
type planCopyInput struct {
	scope       PlanningScope
	source      record
	plan        PlanCreate
	settings    *PlanPatch
	existing    map[string]bool
	sourceIDs   map[string]map[string]bool
	collections []copyCollection
	items       []copyItemInput
	times       []copyTimeInput
	people      []copyPersonInput
	needs       []copyNeedInput
	components  []string
}

func copyCanonicalID(id string) string { return strings.TrimLeft(id, "0") }
func copyComponents(o PlanCopyOptions) []string {
	out := []string{}
	for _, component := range []struct {
		name     string
		selected bool
	}{{"settings", o.CopySettings}, {"times", o.CopyTimes}, {"items", o.CopyItems}, {"people", o.CopyPeople}, {"needed_positions", o.CopyNeededPositions}} {
		if component.selected {
			out = append(out, component.name)
		}
	}
	return out
}
func (l *Library) preflightPlanCopy(ctx context.Context, s PlanningScope, target string, o PlanCopyOptions) (planCopyInput, error) {
	input := planCopyInput{existing: map[string]bool{}, sourceIDs: map[string]map[string]bool{}, components: copyComponents(o)}
	if e := ValidatePlanCopyOptions(o); e != nil {
		return input, e
	}
	if e := validateSchedulingSelectors(target); e != nil {
		return input, e
	}
	s, e := l.planningParents(ctx, s, false)
	if e != nil {
		return input, e
	}
	selected, e := l.selectPlanning(ctx, serviceTypePath(s.ServiceType)+"/plans", "Plan", target, true, "title", s.IncludeArchived)
	if e != nil {
		return input, e
	}
	source, e := l.api.get(ctx, serviceTypePath(s.ServiceType)+"/plans/"+selected.id, "Plan", selected.id)
	if e != nil {
		return input, e
	}
	if !samePlanningAttributes(selected, source) {
		return input, changed("source plan")
	}
	var selectedRelationships, freshRelationships map[string]json.RawMessage
	if len(selected.rel) > 0 {
		_ = json.Unmarshal(selected.rel, &selectedRelationships)
	}
	if len(source.rel) > 0 {
		_ = json.Unmarshal(source.rel, &freshRelationships)
	}
	if _, known := selectedRelationships["series"]; known {
		if _, provided := freshRelationships["series"]; !provided {
			return input, changed("source series")
		}
		before, err := selected.relatedID("series", "Series")
		if err != nil {
			return input, err
		}
		after, err := source.relatedID("series", "Series")
		if err != nil {
			return input, err
		}
		if !reflect.DeepEqual(before, after) {
			return input, changed("source series")
		}
	}
	var plan Plan
	if e = source.model(&plan); e != nil {
		return input, e
	}
	s.Plan = source.id
	input.scope = s
	input.source = source
	private := false
	input.plan = PlanCreate{Title: plan.Title, Public: &private}
	if o.Title != nil {
		input.plan.Title = o.Title
	}
	if plan.SeriesID != nil {
		input.plan.SeriesID = plan.SeriesID
		if e = l.checkPlanSeries(ctx, plan.SeriesID); e != nil {
			return input, e
		}
	} else {
		input.plan.SeriesTitle = plan.SeriesTitle
	}
	if o.CopySettings {
		if plan.Public == nil || plan.RemindersDisabled == nil {
			return input, fail("unsupported_capability", "selected source public/reminder settings are unavailable")
		}
		input.plan.Public = plan.Public
		input.settings = &PlanPatch{RemindersDisabled: plan.RemindersDisabled}
	}
	// The complete returned destination collection prevents a contradictory
	// create response from turning the copy into writes to a pre-existing plan.
	existing, e := l.api.collect(ctx, serviceTypePath(s.ServiceType)+"/plans", "Plan", nil)
	if e != nil {
		return input, e
	}
	input.existing[copyCanonicalID(source.id)] = true
	for _, row := range existing {
		input.existing[copyCanonicalID(row.id)] = true
	}
	selections := []struct {
		selected   bool
		kind, path string
	}{{o.CopyTimes, "PlanTime", planTimesPath(s)}, {o.CopyItems, "Item", itemPath(s)}, {o.CopyPeople, "PlanPerson", scheduledPeoplePath(s)}, {o.CopyNeededPositions, "NeededPosition", neededPositionsPath(s)}}
	for _, selection := range selections {
		if !selection.selected {
			continue
		}
		rows, e := l.api.collect(ctx, selection.path, selection.kind, nil)
		if e != nil {
			return input, e
		}
		if selection.kind == "Item" {
			last := map[string]int{}
			for _, row := range rows {
				var item Item
				if e = row.model(&item); e != nil {
					return input, e
				}
				if item.Sequence != nil {
					group := ""
					if item.ServicePosition != nil {
						group = *item.ServicePosition
					}
					if prior, ok := last[group]; ok && *item.Sequence <= prior {
						return input, fail("invalid_response", "source item sequence values contradict returned order")
					}
					last[group] = *item.Sequence
				}
			}
		}
		input.collections = append(input.collections, copyCollection{selection.path, selection.kind, rows})
		input.sourceIDs[selection.kind] = map[string]bool{}
		for _, row := range rows {
			input.sourceIDs[selection.kind][copyCanonicalID(row.id)] = true
			switch selection.kind {
			case "PlanTime":
				prepared, e := prepareCopyTime(row, o)
				if e != nil {
					return input, e
				}
				input.times = append(input.times, copyTimeInput{row.id, prepared})
			case "Item":
				prepared, e := l.prepareCopyItem(ctx, row, s, o)
				if e != nil {
					return input, e
				}
				input.items = append(input.items, copyItemInput{row.id, prepared})
			case "PlanPerson":
				prepared, e := l.prepareCopyPerson(ctx, row, s)
				if e != nil {
					return input, e
				}
				input.people = append(input.people, copyPersonInput{row.id, prepared})
			case "NeededPosition":
				prepared, e := l.prepareCopyNeed(ctx, row, s)
				if e != nil {
					return input, e
				}
				input.needs = append(input.needs, copyNeedInput{row.id, prepared})
			}
		}
	}
	return input, nil
}
func prepareCopyTime(row record, o PlanCopyOptions) (PlanTimeCreate, error) {
	var model PlanTime
	if e := row.model(&model); e != nil {
		return PlanTimeCreate{}, e
	}
	prepared := PlanTimeCreate{Name: model.Name, StartsAt: model.StartsAt, EndsAt: model.EndsAt, TimeType: model.TimeType}
	var offset time.Duration
	if o.TimeOffset != nil {
		offset, _ = time.ParseDuration(*o.TimeOffset)
	}
	for _, field := range []**string{&prepared.StartsAt, &prepared.EndsAt} {
		if *field == nil {
			continue
		}
		parsed, e := parsePlanningTime(**field)
		if e != nil {
			return prepared, fail("invalid_response", "source plan time is invalid")
		}
		shifted := parsed.Add(offset)
		if shifted.Year() < 1 || shifted.Year() > 9999 {
			return prepared, fail("invalid_input", "time offset moves a source timestamp outside years 0001 through 9999")
		}
		value := shifted.Format(time.RFC3339Nano)
		*field = &value
	}
	if e := ValidatePlanTimeCreate(prepared); e != nil {
		return prepared, &Error{Code: "invalid_response", Message: "source time lacks valid start/type or range", Cause: e}
	}
	return prepared, nil
}
func (l *Library) prepareCopyItem(ctx context.Context, row record, s PlanningScope, o PlanCopyOptions) (ItemCreate, error) {
	var model Item
	if e := row.model(&model); e != nil {
		return ItemCreate{}, e
	}
	out := ItemCreate{ItemPatch: ItemPatch{Title: model.Title, Description: model.Description, HTMLDetails: model.HTMLDetails, Length: model.Length, ServicePosition: model.ServicePosition, ChartSequence: model.ChartSequence}}
	if model.ItemType == nil || !oneOf(*model.ItemType, "item", "header", "song") {
		return out, fail("unsupported_capability", "selected source items require known item, header, or song types; media is not copied")
	}
	for name, kind := range map[string]string{"selected_layout": "Layout", "selected_background": "Attachment", "selected_attachment": "Attachment"} {
		id, e := row.relatedID(name, kind)
		if e != nil {
			return out, e
		}
		if id != nil {
			return out, fail("unsupported_capability", "source item has an unsupported "+name+" link")
		}
	}
	if len(row.rel) > 0 {
		var relationships map[string]json.RawMessage
		_ = json.Unmarshal(row.rel, &relationships)
		if raw, ok := relationships["media"]; ok {
			var relationship struct {
				Data json.RawMessage `json:"data"`
			}
			if json.Unmarshal(raw, &relationship) != nil || len(relationship.Data) == 0 {
				return out, fail("invalid_response", "source media relationship is malformed")
			}
			if string(relationship.Data) != "null" {
				var media []json.RawMessage
				if json.Unmarshal(relationship.Data, &media) != nil {
					return out, fail("invalid_response", "source media relationship must be an array")
				}
				if len(media) > 0 {
					return out, fail("unsupported_capability", "source item media links are not copied")
				}
			}
		}
	}
	if *model.ItemType == "song" {
		if model.SongID == nil || model.ArrangementID == nil {
			return out, fail("invalid_response", "source song item requires explicit song and arrangement identities")
		}
		if model.KeyID == nil {
			return out, fail("unsupported_capability", "copying a song item requires a known song, arrangement, and selected key identity")
		}
		out.Song = &Scope{Song: *model.SongID, Arrangement: *model.ArrangementID, Visibility: o.Visibility, IncludeArchived: s.IncludeArchived}
		if model.KeyID != nil {
			out.Song.Key = *model.KeyID
		}
		if _, _, e := l.itemLink(ctx, out.Song); e != nil {
			return out, e
		}
	} else {
		if model.SongID != nil || model.ArrangementID != nil || model.KeyID != nil {
			return out, fail("invalid_response", "source item type contradicts song linkage")
		}
		out.Type = model.ItemType
	}
	if e := ValidateItemCreate(out); e != nil {
		return out, &Error{Code: "invalid_response", Message: "source item cannot be copied with supported fields", Cause: e}
	}
	return out, nil
}
func (l *Library) prepareCopyPerson(ctx context.Context, row record, s PlanningScope) (ScheduledPersonCreate, error) {
	var model ScheduledPerson
	if e := row.model(&model); e != nil {
		return ScheduledPersonCreate{}, e
	}
	if model.PersonID == nil || model.TeamID == nil || model.TeamPositionName == nil {
		return ScheduledPersonCreate{}, fail("invalid_response", "source assignment lacks person, team, or position name")
	}
	team, e := l.assignmentContext(ctx, s, row, "", true)
	if e != nil {
		return ScheduledPersonCreate{}, e
	}
	position, e := l.schedulingPosition(ctx, team.Team, "name:"+*model.TeamPositionName, true)
	if e != nil {
		return ScheduledPersonCreate{}, e
	}
	var expectedName string
	_ = json.Unmarshal(position.attrs["name"], &expectedName)
	return ScheduledPersonCreate{Team: team.Team, Person: *model.PersonID, Position: position.id, Notes: model.Notes, ExpectedPositionName: &expectedName}, nil
}
func (l *Library) prepareCopyNeed(ctx context.Context, row record, s PlanningScope) (NeededPositionCreate, error) {
	var model NeededPosition
	if e := row.model(&model); e != nil {
		return NeededPositionCreate{}, e
	}
	if model.TeamID == nil || model.TeamPositionName == nil || model.Quantity == nil {
		return NeededPositionCreate{}, fail("invalid_response", "source needed position lacks team, position name, or quantity")
	}
	if e := l.neededContext(ctx, s, row); e != nil {
		return NeededPositionCreate{}, e
	}
	// The source position ID is unreadable. A unique current label is the only
	// supported bridge to a selected destination position, and remains unverified.
	position, e := l.schedulingPosition(ctx, *model.TeamID, "name:"+*model.TeamPositionName, true)
	if e != nil {
		return NeededPositionCreate{}, e
	}
	var expectedName string
	_ = json.Unmarshal(position.attrs["name"], &expectedName)
	input := NeededPositionCreate{Team: *model.TeamID, Position: position.id, Quantity: model.Quantity, ExpectedPositionName: &expectedName}
	if e = ValidateNeededPositionCreate(input); e != nil {
		return input, e
	}
	return input, nil
}
func (l *Library) recheckCopySource(ctx context.Context, input planCopyInput) error {
	fresh, e := l.api.get(ctx, planPath(input.scope), "Plan", input.source.id)
	if e != nil {
		return e
	}
	if !sameCopyRecord(input.source, fresh) {
		return changed("source plan")
	}
	for _, collection := range input.collections {
		fresh, e := l.api.collect(ctx, collection.path, collection.kind, nil)
		if e != nil {
			return e
		}
		if len(fresh) != len(collection.records) {
			return changed("source " + collection.kind + " collection")
		}
		for i, row := range fresh {
			if row.id != collection.records[i].id {
				return changed("source " + collection.kind + " order")
			}
			if !sameCopyRecord(collection.records[i], row) {
				return changed("source " + collection.kind)
			}
		}
	}
	return nil
}
func copyErrorClassification(err error, writeStage bool) (string, string) {
	var transport *planningcenter.TransportError
	if errors.As(err, &transport) && transport.UnknownOutcome {
		return "outcome_unknown", "unknown"
	}
	var service *Error
	if errors.As(err, &service) {
		if service.Code == "outcome_unknown" {
			return service.Code, "unknown"
		}
		if service.Code == "verification_failed" {
			return service.Code, "verification_failed"
		}
		return service.Code, "rejected"
	}
	var api planningcenter.APIError
	if errors.As(err, &api) {
		if writeStage && api.StatusCode >= 500 {
			return "outcome_unknown", "unknown"
		}
		if writeStage && api.StatusCode >= 400 && api.StatusCode < 500 {
			return "api_error", "rejected"
		}
	}
	if !writeStage {
		return "verification_failed", "verification_failed"
	}
	return "outcome_unknown", "unknown"
}
func (l *Library) CopyPlan(ctx context.Context, s PlanningScope, target string, o PlanCopyOptions) (PlanCopyResult, error) {
	result := PlanCopyResult{Created: []PlanningReceipt{}, UnverifiedFields: []CopyUnverifiedField{}}
	input, e := l.preflightPlanCopy(ctx, s, target, o)
	if e != nil {
		return result, e
	}
	if e = l.recheckCopyReferences(ctx, input); e != nil {
		return result, e
	}
	if e = l.recheckCopySource(ctx, input); e != nil {
		return result, e
	}
	result.SourcePlanID = input.source.id
	result.Components = input.components
	result.Outcome = "verified"
	recovery := &PlanningRecovery{Operation: "copy", Resource: "Plan", ServiceTypeID: input.scope.ServiceType, SourcePlanID: input.source.id, Created: []PlanningReceipt{}}
	seen := map[string]map[string]bool{}
	collision := func(kind, id string) bool {
		canonical := copyCanonicalID(id)
		if kind == "Plan" && input.existing[canonical] {
			return true
		}
		if input.sourceIDs[kind][canonical] {
			return true
		}
		return seen[kind][canonical]
	}
	observe := func(receipt PlanningReceipt) { recovery.Observed = append(recovery.Observed, receipt) }
	recordCreated := func(receipt PlanningReceipt) bool {
		if collision(receipt.Resource, receipt.ID) {
			observe(receipt)
			return false
		}
		if seen[receipt.Resource] == nil {
			seen[receipt.Resource] = map[string]bool{}
		}
		seen[receipt.Resource][copyCanonicalID(receipt.ID)] = true
		recovery.Created = append(recovery.Created, receipt)
		result.Created = append(result.Created, receipt)
		return true
	}
	stopped := func(stage string, cause error, writeStage bool) (PlanCopyResult, error) {
		recovery.Stage = stage
		code, last := copyErrorClassification(cause, writeStage)
		recovery.LastOutcome = last
		var inner *Error
		if errors.As(cause, &inner) && inner.PlanningRecovery != nil {
			for _, receipt := range inner.PlanningRecovery.Created {
				if !recordCreated(receipt) {
					code = "outcome_unknown"
					recovery.LastOutcome = "unknown"
				}
			}
			if inner.PlanningRecovery.PlanID != "" && recovery.PlanID == "" && !input.existing[copyCanonicalID(inner.PlanningRecovery.PlanID)] {
				recovery.PlanID = inner.PlanningRecovery.PlanID
			}
		}
		recovery.LastErrorCode = code
		if recovery.LastOutcome == "rejected" && len(recovery.Created) > 0 {
			code = "partial_success"
		}
		return PlanCopyResult{}, &Error{Code: code, Message: "plan copy stopped during " + stage + "; inspect recovery records before retrying: " + cause.Error(), PlanningRecovery: recovery, Cause: cause}
	}
	collisionError := func(stage, kind, id string) (PlanCopyResult, error) {
		recovery.Stage = stage
		recovery.LastOutcome = "unknown"
		return PlanCopyResult{}, &Error{Code: "outcome_unknown", Message: "plan copy response identified a pre-existing or previously created " + kind + "; no further steps were attempted", PlanningRecovery: recovery}
	}
	created, e := l.CreatePlan(ctx, PlanningScope{ServiceType: input.scope.ServiceType, IncludeArchived: s.IncludeArchived}, input.plan)
	if e != nil {
		return stopped("create-plan", e, true)
	}
	receipt := PlanningReceipt{Resource: "Plan", ID: created.ID, ServiceTypeID: input.scope.ServiceType, PlanID: created.ID}
	if !recordCreated(receipt) {
		return collisionError("create-plan", "Plan", created.ID)
	}
	recovery.PlanID = created.ID
	recovery.TargetID = created.ID
	destination := PlanningScope{ServiceType: input.scope.ServiceType, Plan: created.ID, IncludeArchived: s.IncludeArchived}
	if input.settings != nil {
		if _, e = l.UpdatePlan(ctx, destination, created.ID, *input.settings); e != nil {
			return stopped("copy-settings", e, true)
		}
	}
	for _, prepared := range input.times {
		copied, e := l.CreatePlanTime(ctx, destination, prepared.input)
		if e != nil {
			return stopped("copy-times", e, true)
		}
		if !recordCreated(PlanningReceipt{Resource: "PlanTime", ID: copied.ID, ServiceTypeID: destination.ServiceType, PlanID: destination.Plan}) {
			return collisionError("copy-times", "PlanTime", copied.ID)
		}
	}
	itemIDs := []string{}
	for _, prepared := range input.items {
		copied, e := l.CreateItem(ctx, destination, prepared.input)
		if e != nil {
			return stopped("copy-items", e, true)
		}
		if !recordCreated(PlanningReceipt{Resource: "Item", ID: copied.ID, ServiceTypeID: destination.ServiceType, PlanID: destination.Plan}) {
			return collisionError("copy-items", "Item", copied.ID)
		}
		itemIDs = append(itemIDs, copied.ID)
	}
	if len(itemIDs) > 0 {
		if _, e = l.ReorderItems(ctx, destination, itemIDs); e != nil {
			return stopped("reorder-items", e, true)
		}
	}
	for _, prepared := range input.people {
		copied, e := l.CreateScheduledPerson(ctx, destination, prepared.input)
		if e != nil {
			return stopped("copy-people", e, true)
		}
		if !recordCreated(PlanningReceipt{Resource: "PlanPerson", ID: copied.ID, ServiceTypeID: destination.ServiceType, PlanID: destination.Plan}) {
			return collisionError("copy-people", "PlanPerson", copied.ID)
		}
	}
	for _, prepared := range input.needs {
		copied, e := l.CreateNeededPosition(ctx, destination, prepared.input)
		if e != nil {
			return stopped("copy-needed-positions", e, true)
		}
		if !recordCreated(PlanningReceipt{Resource: "NeededPosition", ID: copied.NeededPosition.ID, ServiceTypeID: destination.ServiceType, PlanID: destination.Plan}) {
			return collisionError("copy-needed-positions", "NeededPosition", copied.NeededPosition.ID)
		}
		result.Outcome = "accepted"
		for _, field := range copied.UnverifiedFields {
			requested := ""
			if copied.RequestedPositionID != nil {
				requested = *copied.RequestedPositionID
			}
			entry := CopyUnverifiedField{Resource: "NeededPosition", ID: copied.NeededPosition.ID, Field: field, RequestedID: requested}
			result.UnverifiedFields = append(result.UnverifiedFields, entry)
			recovery.UnverifiedFields = append(recovery.UnverifiedFields, entry)
		}
	}
	final, e := l.ShowPlan(ctx, destination, created.ID)
	if e != nil {
		return stopped("final-readback", e, false)
	}
	result.Plan = final
	return result, nil
}

func sameCopyRecord(before, after record) bool {
	if before.id != after.id || before.kind != after.kind || !samePlanningAttributes(before, after) {
		return false
	}
	relationships := func(raw json.RawMessage) any {
		if len(raw) == 0 {
			return map[string]any{}
		}
		var value any
		if json.Unmarshal(raw, &value) != nil {
			return nil
		}
		return value
	}
	return reflect.DeepEqual(relationships(before.rel), relationships(after.rel))
}

func (l *Library) recheckCopyReferences(ctx context.Context, input planCopyInput) error {
	for _, item := range input.items {
		if _, _, e := l.itemLink(ctx, item.input.Song); e != nil {
			return e
		}
	}
	for _, person := range input.people {
		scope, team, e := l.teamContext(ctx, TeamScope{ServiceType: input.scope.ServiceType, Team: person.input.Team, IncludeArchived: input.scope.IncludeArchived})
		if e != nil {
			return e
		}
		if e = wholePlanTeam(team); e != nil {
			return e
		}
		if _, e = l.member(ctx, scope.Team, person.input.Person); e != nil {
			return e
		}
		position, e := l.schedulingPosition(ctx, scope.Team, person.input.Position, true)
		if e != nil {
			return e
		}
		var name string
		_ = json.Unmarshal(position.attrs["name"], &name)
		if person.input.ExpectedPositionName != nil && *person.input.ExpectedPositionName != name {
			return changed("source team position name")
		}
	}
	for _, need := range input.needs {
		scope, team, e := l.teamContext(ctx, TeamScope{ServiceType: input.scope.ServiceType, Team: need.input.Team, IncludeArchived: input.scope.IncludeArchived})
		if e != nil {
			return e
		}
		if e = wholePlanTeam(team); e != nil {
			return e
		}
		position, e := l.schedulingPosition(ctx, scope.Team, need.input.Position, true)
		if e != nil {
			return e
		}
		var name string
		_ = json.Unmarshal(position.attrs["name"], &name)
		if need.input.ExpectedPositionName != nil && *need.input.ExpectedPositionName != name {
			return changed("source needed position name")
		}
	}
	return nil
}
