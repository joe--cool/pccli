package services

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

type ItemPatch struct {
	Title           *string   `json:"title,omitempty"`
	Description     *string   `json:"description,omitempty"`
	HTMLDetails     *string   `json:"html_details,omitempty"`
	Length          *int      `json:"length,omitempty"`
	ServicePosition *string   `json:"service_position,omitempty"`
	ChartSequence   *[]string `json:"custom_arrangement_sequence,omitempty"`
	Song            *Scope    `json:"-"`
}
type ItemCreate struct {
	ItemPatch
	Type     *string `json:"item_type,omitempty"`
	Position *int    `json:"sequence,omitempty"`
}

func validateItemFields(p ItemPatch) error {
	if p.Length != nil && *p.Length < 0 {
		return fail("invalid_input", "length must be nonnegative")
	}
	if p.ServicePosition != nil && !oneOf(*p.ServicePosition, "pre", "during", "post") {
		return fail("invalid_input", "service-position must be pre, during, or post")
	}
	if p.ChartSequence != nil {
		for _, section := range *p.ChartSequence {
			if strings.TrimSpace(section) == "" {
				return fail("invalid_input", "chart-sequence cannot contain empty labels")
			}
		}
	}
	if p.Song != nil {
		if p.Song.Song == "" || p.Song.Arrangement == "" {
			return fail("invalid_input", "song linkage requires --song and --arrangement")
		}
		for _, selector := range []string{p.Song.Song, p.Song.Arrangement, p.Song.Key} {
			if selector != "" {
				if _, _, _, e := ParseSelector(selector); e != nil {
					return e
				}
			}
		}
		if _, e := visible(p.Song.Visibility); e != nil {
			return e
		}
	}
	return nil
}
func ValidateItemCreate(p ItemCreate) error {
	if e := validateItemFields(p.ItemPatch); e != nil {
		return e
	}
	if p.Type != nil && !oneOf(*p.Type, "item", "header") {
		return fail("invalid_input", "type must be item or header; use --song for a song item")
	}
	if p.Type != nil && p.Song != nil {
		return fail("invalid_input", "explicit type and song linkage conflict")
	}
	if p.Position != nil && *p.Position < 0 {
		return fail("invalid_input", "position must be nonnegative")
	}
	return nil
}
func ValidateItemPatch(p ItemPatch) error {
	if e := validateItemFields(p); e != nil {
		return e
	}
	if len(fields(p)) == 0 && p.Song == nil {
		return fail("invalid_input", "at least one item field or song linkage is required")
	}
	return nil
}
func (l *Library) itemLink(ctx context.Context, p *Scope) (map[string]any, map[string]string, error) {
	attrs := map[string]any{}
	rel := map[string]string{}
	if p == nil {
		return attrs, rel, nil
	}
	parents, e := l.parents(ctx, *p, true, true)
	if e != nil {
		return nil, nil, e
	}
	attrs["song_id"] = parents.song.id
	rel["song"] = parents.song.id
	attrs["arrangement_id"] = parents.arr.id
	rel["arrangement"] = parents.arr.id
	if p.Key != "" {
		key, e := l.key(ctx, parents, p.Key, true)
		if e != nil {
			return nil, nil, e
		}
		attrs["key_id"] = key.id
		rel["key"] = key.id
	}
	return attrs, rel, nil
}
func itemPath(s PlanningScope) string { return planPath(s) + "/items" }
func (l *Library) CreateItem(ctx context.Context, s PlanningScope, p ItemCreate) (Item, error) {
	var out Item
	if e := ValidateItemCreate(p); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	attrs, rel, e := l.itemLink(ctx, p.Song)
	if e != nil {
		return out, e
	}
	for k, v := range fields(p.ItemPatch) {
		attrs[k] = v
	}
	if p.Type != nil && *p.Type == "header" {
		attrs["item_type"] = "header"
	}
	if p.Position != nil {
		attrs["sequence"] = *p.Position
	}
	if p.Song != nil {
		rel["item_type"] = "song"
	} else if p.Type != nil {
		rel["item_type"] = *p.Type
	}
	return writePlanningChild[Item](ctx, l, "POST", itemPath(s), "Item", s, "", attrs, rel)
}
func (l *Library) UpdateItem(ctx context.Context, s PlanningScope, target string, p ItemPatch) (Item, error) {
	var out Item
	if e := ValidateItemPatch(p); e != nil {
		return out, e
	}
	if _, _, _, e := ParseSelector(target); e != nil {
		return out, e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	row, e := l.selectPlanning(ctx, itemPath(s), "Item", target, true, "title", s.IncludeArchived)
	if e != nil {
		return out, e
	}
	attrs, rel, e := l.itemLink(ctx, p.Song)
	if e != nil {
		return out, e
	}
	for k, v := range fields(p) {
		attrs[k] = v
	}
	if p.Song != nil {
		var typ string
		_ = json.Unmarshal(row.attrs["item_type"], &typ)
		if typ == "header" {
			return out, fail("invalid_input", "cannot attach a song to a header item")
		}
		rel["item_type"] = "song"
	}
	path := itemPath(s) + "/" + row.id
	if e = l.recheckPlanningRecord(ctx, path, row); e != nil {
		return out, e
	}
	return writePlanningChild[Item](ctx, l, "PATCH", path, "Item", s, row.id, attrs, rel)
}
func (l *Library) recheckPlanningRecord(ctx context.Context, path string, row record) error {
	current, e := l.api.get(ctx, path, row.kind, row.id)
	if e != nil {
		return e
	}
	if row.kind == "NeededPosition" {
		for name, kind := range map[string]string{"time": "PlanTime", "time_preference_option": "TimePreferenceOption"} {
			id, e := current.relatedID(name, kind)
			if e != nil {
				return e
			}
			if id != nil {
				return fail("unsupported_capability", "time-specific needed positions cannot be changed by whole-plan commands")
			}
		}
	}
	if !samePlanningAttributes(row, current) {
		return changed(row.kind)
	}
	// Collection responses may omit relationships expanded on detail. Compare
	// known supplied linkage semantically, including an explicit empty linkage.
	var previous, next map[string]json.RawMessage
	if len(row.rel) > 0 {
		_ = json.Unmarshal(row.rel, &previous)
	}
	if len(current.rel) > 0 {
		_ = json.Unmarshal(current.rel, &next)
	}
	links := map[string]string{"song": "Song", "arrangement": "Arrangement", "key": "Key"}
	if row.kind == "NeededPosition" {
		links = map[string]string{"team": "Team", "time": "PlanTime", "time_preference_option": "TimePreferenceOption"}
	}
	if row.kind == "PlanPerson" {
		links = map[string]string{"person": "Person", "team": "Team"}
	}
	for name, kind := range links {
		if _, known := previous[name]; !known {
			continue
		}
		if _, provided := next[name]; !provided {
			return changed(row.kind)
		}
		before, e := row.relatedID(name, kind)
		if e != nil {
			return e
		}
		after, e := current.relatedID(name, kind)
		if e != nil {
			return e
		}
		if !reflect.DeepEqual(before, after) {
			return changed(row.kind)
		}
	}

	return nil
}
func (l *Library) DeleteItem(ctx context.Context, s PlanningScope, target string) error {
	if _, _, _, e := ParseSelector(target); e != nil {
		return e
	}
	s, e := l.planningParents(ctx, s, true)
	if e != nil {
		return e
	}
	row, e := l.selectPlanning(ctx, itemPath(s), "Item", target, true, "title", s.IncludeArchived)
	if e != nil {
		return e
	}
	path := itemPath(s) + "/" + row.id
	if e = l.recheckPlanningRecord(ctx, path, row); e != nil {
		return e
	}
	return l.deletePlanningChild(ctx, path, "Item", s, row.id)
}
func canonicalItemSequence(sequence []string) ([]string, error) {
	if len(sequence) == 0 {
		return nil, fail("invalid_input", "sequence must contain every item ID")
	}
	out := make([]string, 0, len(sequence))
	seen := map[string]bool{}
	for _, id := range sequence {
		if !validID(id) {
			return nil, fail("invalid_input", "sequence must contain positive decimal IDs")
		}
		canonical := strings.TrimLeft(id, "0")
		if seen[canonical] {
			return nil, fail("invalid_input", "sequence contains duplicate item IDs")
		}
		seen[canonical] = true
		out = append(out, canonical)
	}
	return out, nil
}
func ValidateItemSequence(sequence []string) error { _, e := canonicalItemSequence(sequence); return e }
func (l *Library) ReorderItems(ctx context.Context, s PlanningScope, sequence []string) (ListResult[Item], error) {
	out := ListResult[Item]{}
	sequence, e := canonicalItemSequence(sequence)
	if e != nil {
		return out, e
	}
	s, e = l.planningParents(ctx, s, true)
	if e != nil {
		return out, e
	}
	before, e := l.api.collect(ctx, itemPath(s), "Item", nil)
	if e != nil {
		return out, e
	}
	if len(before) != len(sequence) {
		return out, fail("invalid_input", "sequence must contain every scoped item exactly once")
	}
	known := map[string]record{}
	for _, r := range before {
		known[r.id] = r
	}
	for _, id := range sequence {
		if _, ok := known[id]; !ok {
			return out, fail("invalid_input", "sequence contains an item outside the selected plan")
		}
	}
	checked, e := l.api.collect(ctx, itemPath(s), "Item", nil)
	if e != nil {
		return out, e
	}
	if len(checked) != len(before) {
		return out, changed("Item collection")
	}
	for _, r := range checked {
		old, ok := known[r.id]
		if !ok {
			return out, changed("Item collection")
		}
		a, e := recordHash(old)
		if e != nil {
			return out, e
		}
		b, e := recordHash(r)
		if e != nil {
			return out, e
		}
		if a != b {
			return out, changed("Item collection")
		}
	}
	body := map[string]any{"data": map[string]any{"type": "PlanItemReorder", "attributes": map[string]any{"sequence": sequence}}}
	if e = l.api.http.Post(ctx, planPath(s)+"/item_reorder", nil, body, nil); e != nil {
		return out, e
	}
	recovery := childRecovery("reorder", "Item", s, "")
	failed := func(message string, cause error) (ListResult[Item], error) {
		return out, &Error{Code: "verification_failed", Message: message, PlanningRecovery: recovery, Cause: cause}
	}
	after, e := l.api.collect(ctx, itemPath(s), "Item", nil)
	if e != nil {
		return failed("item reorder accepted but full readback failed", e)
	}
	if len(after) != len(sequence) {
		return failed("item reorder accepted but returned membership changed", nil)
	}
	lastSequence := map[string]int{}
	for i, r := range after {
		if r.id != sequence[i] {
			return failed("item reorder accepted but returned order differs", nil)
		}
		var item Item
		if e = r.model(&item); e != nil {
			return failed("item reorder accepted but returned item was invalid", e)
		}
		if item.Sequence != nil {
			group := ""
			if item.ServicePosition != nil {
				group = *item.ServicePosition
			}
			if prior, ok := lastSequence[group]; ok && *item.Sequence <= prior {
				return failed("item reorder accepted but sequence values contradict returned order", nil)
			}
			lastSequence[group] = *item.Sequence
		}
		out.Data = append(out.Data, item)
	}
	out.Meta = ListMeta{Returned: len(out.Data), Complete: true}
	return out, nil
}

func samePlanningAttributes(a, b record) bool {
	rawA, _ := json.Marshal(a.attrs)
	rawB, _ := json.Marshal(b.attrs)
	var attrsA, attrsB any
	if json.Unmarshal(rawA, &attrsA) != nil || json.Unmarshal(rawB, &attrsB) != nil {
		return false
	}
	if a.kind == "PlanPerson" {
		for _, attrs := range []any{attrsA, attrsB} {
			m, ok := attrs.(map[string]any)
			if !ok {
				return false
			}
			if status, ok := m["status"].(string); ok {
				if canonical, e := canonicalScheduleStatus(status); e == nil {
					m["status"] = canonical
				}
			}
		}
	}
	if a.kind == "PlanTime" {
		for _, attrs := range []any{attrsA, attrsB} {
			m, ok := attrs.(map[string]any)
			if !ok {
				return false
			}
			for _, name := range []string{"starts_at", "ends_at"} {
				if value, ok := m[name].(string); ok {
					parsed, e := parsePlanningTime(value)
					if e != nil {
						return false
					}
					m[name] = parsed.UTC().Format(time.RFC3339Nano)
				}
			}
		}
	}
	return reflect.DeepEqual(attrsA, attrsB)
}
