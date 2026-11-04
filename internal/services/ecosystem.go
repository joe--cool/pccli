package services

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/joe--cool/pccli/internal/planningcenter"
	"reflect"
	"sort"
	"strings"
)

type TagGroup struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	TagsFor                 string `json:"tags_for"`
	AllowMultipleSelections bool   `json:"allow_multiple_selections"`
	Required                bool   `json:"required"`
	ServiceTypeFolderName   string `json:"service_type_folder_name"`
}
type Tag struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TagGroupID   string `json:"tag_group_id"`
	TagGroupName string `json:"tag_group_name"`
	TagsFor      string `json:"tags_for"`
}
type AttachmentType struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Aliases           string `json:"aliases"`
	BuiltIn           bool   `json:"built_in"`
	CapoedChordCharts bool   `json:"capoed_chord_charts"`
	ChordCharts       bool   `json:"chord_charts"`
	Lyrics            bool   `json:"lyrics"`
	NumberCharts      bool   `json:"number_charts"`
	NumeralCharts     bool   `json:"numeral_charts"`
	Exclusions        string `json:"exclusions"`
}
type AncillaryRecord struct {
	ID         string                     `json:"id"`
	Attributes map[string]json.RawMessage `json:"attributes"`
}
type AttachmentPatch struct {
	Filename            *string `json:"filename,omitempty"`
	AttachmentTypeIDs   *string `json:"attachment_type_ids,omitempty"`
	PageOrder           *string `json:"page_order,omitempty"`
	Content             *string `json:"content,omitempty"`
	RemoteLink          *string `json:"remote_link,omitempty"`
	FileUploadID        *string `json:"file_upload_identifier,omitempty"`
	SongPart            *string `json:"song_part,omitempty"`
	ImportToItemDetails *bool   `json:"import_to_item_details,omitempty"`
}

func (l *Library) tagGroups(ctx context.Context) ([]TagGroup, error) {
	rows, err := l.api.collect(ctx, "/services/v2/tag_groups", "TagGroup", nil)
	if err != nil {
		return nil, err
	}
	groups := []TagGroup{}
	for _, r := range rows {
		var group TagGroup
		if err = r.model(&group); err != nil {
			return nil, err
		}
		if group.TagsFor == "song" || group.TagsFor == "arrangement" {
			groups = append(groups, group)
		}
	}
	return groups, nil
}
func (l *Library) ListTagGroups(ctx context.Context, o ListOptions) (ListResult[TagGroup], error) {
	if _, err := bound([]TagGroup{}, o); err != nil {
		return ListResult[TagGroup]{}, err
	}
	groups, err := l.tagGroups(ctx)
	if err != nil {
		return ListResult[TagGroup]{}, err
	}
	return bound(groups, o)
}
func (l *Library) tagCatalog(ctx context.Context) ([]Tag, []TagGroup, error) {
	groups, err := l.tagGroups(ctx)
	if err != nil {
		return nil, nil, err
	}
	tags := []Tag{}
	for _, group := range groups {
		rows, err := l.api.collect(ctx, "/services/v2/tag_groups/"+group.ID+"/tags", "Tag", nil)
		if err != nil {
			return nil, nil, err
		}
		for _, r := range rows {
			if err = r.scope("tag_group", "TagGroup", group.ID); err != nil {
				return nil, nil, err
			}
			var tag Tag
			if err = r.model(&tag); err != nil {
				return nil, nil, err
			}
			tag.TagGroupID = group.ID
			tag.TagGroupName = group.Name
			tag.TagsFor = group.TagsFor
			tags = append(tags, tag)
		}
	}
	return tags, groups, nil
}
func (l *Library) tagDestination(ctx context.Context, s Scope, write bool) (string, string, error) {
	if s.Key != "" {
		return "", "", fail("invalid_input", "tags belong to songs or arrangements; key is not supported")
	}
	p, err := l.parents(ctx, s, s.Arrangement != "", write)
	if err != nil {
		return "", "", err
	}
	if p.arr.id != "" {
		return arrangementPath(p.song.id, p.arr.id), "arrangement", nil
	}
	return songPath(p.song.id), "song", nil
}
func (l *Library) ListTags(ctx context.Context, s Scope, o ListOptions) (ListResult[Tag], error) {
	if s.Key != "" {
		return ListResult[Tag]{}, fail("invalid_input", "tags belong to songs or arrangements; key is not supported")
	}
	if _, err := bound([]Tag{}, o); err != nil {
		return ListResult[Tag]{}, err
	}
	if s.Song == "" && (s.Arrangement != "" || s.Key != "") {
		return ListResult[Tag]{}, fail("invalid_input", "song is required for scoped tags")
	}
	catalog, _, err := l.tagCatalog(ctx)
	if err != nil {
		return ListResult[Tag]{}, err
	}
	if s.Song == "" {
		return bound(catalog, o)
	}
	path, kind, err := l.tagDestination(ctx, s, false)
	if err != nil {
		return ListResult[Tag]{}, err
	}
	return l.tagsAt(ctx, path, kind, catalog, o)
}
func (l *Library) tagsAt(ctx context.Context, path, kind string, catalog []Tag, o ListOptions) (ListResult[Tag], error) {
	rows, err := l.api.collect(ctx, path+"/tags", "Tag", nil)
	if err != nil {
		return ListResult[Tag]{}, err
	}
	byID := map[string]Tag{}
	for _, tag := range catalog {
		if tag.TagsFor == kind {
			byID[tag.ID] = tag
		}
	}
	out := []Tag{}
	for _, r := range rows {
		tag, ok := byID[r.id]
		if !ok {
			return ListResult[Tag]{}, fail("invalid_response", "assigned tag is absent from the matching catalog")
		}
		var name string
		if json.Unmarshal(r.attrs["name"], &name) != nil || name != tag.Name {
			return ListResult[Tag]{}, changed("Tag")
		}
		out = append(out, tag)
	}
	return bound(out, o)
}

// ReplaceTags transmits the complete desired assignment. An empty selector slice
// explicitly clears tags. It never infers append semantics from existing values.
func (l *Library) ReplaceTags(ctx context.Context, s Scope, selectors []string) (ListResult[Tag], error) {
	if s.Song == "" {
		return ListResult[Tag]{}, fail("invalid_input", "song is required")
	}
	for _, selector := range selectors {
		if _, _, _, err := ParseSelector(selector); err != nil {
			return ListResult[Tag]{}, err
		}
	}
	path, kind, err := l.tagDestination(ctx, s, true)
	if err != nil {
		return ListResult[Tag]{}, err
	}
	catalog, groups, err := l.tagCatalog(ctx)
	if err != nil {
		return ListResult[Tag]{}, incomplete(err)
	}
	chosen := []Tag{}
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, selector := range selectors {
		id, name, _, _ := ParseSelector(selector)
		matches := []Tag{}
		for _, tag := range catalog {
			if tag.TagsFor == kind && ((id != "" && id == tag.ID) || (id == "" && strings.EqualFold(name, tag.Name))) {
				matches = append(matches, tag)
			}
		}
		if len(matches) == 0 {
			return ListResult[Tag]{}, fail("not_found", "tag not found in "+kind+" tag groups: "+selector)
		}
		if len(matches) > 1 {
			return ListResult[Tag]{}, &Error{Code: "ambiguous_target", Message: "tag name matches multiple groups; choose a tag ID", Candidates: matches}
		}
		tag := matches[0]
		if !seen[tag.ID] {
			seen[tag.ID] = true
			chosen = append(chosen, tag)
			counts[tag.TagGroupID]++
		}
	}
	for _, group := range groups {
		if group.TagsFor != kind {
			continue
		}
		if group.Required && counts[group.ID] == 0 {
			return ListResult[Tag]{}, fail("invalid_input", "required tag group needs a selection: "+group.Name)
		}
		if !group.AllowMultipleSelections && counts[group.ID] > 1 {
			return ListResult[Tag]{}, fail("invalid_input", "tag group allows only one selection: "+group.Name)
		}
	}
	refs := []map[string]string{}
	for _, tag := range chosen {
		refs = append(refs, map[string]string{"type": "Tag", "id": tag.ID})
	}
	body := map[string]any{"data": map[string]any{"type": "TagAssignment", "attributes": map[string]any{}, "relationships": map[string]any{"tags": map[string]any{"data": refs}}}}
	if err = l.api.http.Post(ctx, path+"/assign_tags", nil, body, nil); err != nil {
		return ListResult[Tag]{}, err
	}
	actual, err := l.tagsAt(ctx, path, kind, catalog, ListOptions{All: true})
	if err != nil {
		return ListResult[Tag]{}, &Error{Code: "verification_failed", Message: "tag replacement submitted but readback failed", Cause: err}
	}
	wantIDs, gotIDs := []string{}, []string{}
	for _, tag := range chosen {
		wantIDs = append(wantIDs, tag.ID)
	}
	for _, tag := range actual.Data {
		gotIDs = append(gotIDs, tag.ID)
	}
	sort.Strings(wantIDs)
	sort.Strings(gotIDs)
	if !reflect.DeepEqual(wantIDs, gotIDs) {
		return ListResult[Tag]{}, fail("verification_failed", "tag replacement submitted but assignment differs on readback")
	}
	return actual, nil
}
func (l *Library) ListAttachmentTypes(ctx context.Context, o ListOptions) (ListResult[AttachmentType], error) {
	if _, err := bound([]AttachmentType{}, o); err != nil {
		return ListResult[AttachmentType]{}, err
	}
	rows, err := l.api.collectLimited(ctx, "/services/v2/attachment_types", "AttachmentType", nil, displayLimit(o), nil)
	if err != nil {
		return ListResult[AttachmentType]{}, err
	}
	out := []AttachmentType{}
	for _, r := range rows {
		var x AttachmentType
		if err = r.model(&x); err != nil {
			return ListResult[AttachmentType]{}, err
		}
		out = append(out, x)
	}
	return bound(out, o)
}
func (l *Library) songAncillary(ctx context.Context, s Scope, o ListOptions, suffix, kind string, query url.Values) (ListResult[AncillaryRecord], error) {
	if _, err := bound([]AncillaryRecord{}, o); err != nil {
		return ListResult[AncillaryRecord]{}, err
	}
	if s.Arrangement != "" || s.Key != "" {
		return ListResult[AncillaryRecord]{}, fail("invalid_input", "this workflow selects a song only")
	}
	p, err := l.parents(ctx, s, false, false)
	if err != nil {
		return ListResult[AncillaryRecord]{}, err
	}
	rows, err := l.api.collectLimited(ctx, songPath(p.song.id)+"/"+suffix, kind, query, displayLimit(o), nil)
	if err != nil {
		return ListResult[AncillaryRecord]{}, err
	}
	out := []AncillaryRecord{}
	for _, r := range rows {
		out = append(out, AncillaryRecord{r.id, r.attrs})
	}
	return bound(out, o)
}
func (l *Library) ListSongSchedules(ctx context.Context, s Scope, o ListOptions) (ListResult[AncillaryRecord], error) {
	return l.songAncillary(ctx, s, o, "song_schedules", "SongSchedule", nil)
}
func (l *Library) LastScheduledItems(ctx context.Context, s Scope, o ListOptions) (ListResult[AncillaryRecord], error) {
	if !validID(s.ServiceTypeID) {
		return ListResult[AncillaryRecord]{}, fail("invalid_input", "last scheduled item requires a positive service type ID")
	}
	return l.songAncillary(ctx, s, o, "last_scheduled_item", "Item", url.Values{"service_type": {strings.TrimLeft(s.ServiceTypeID, "0")}})
}
func (l *Library) ArrangementSections(ctx context.Context, s Scope) (AncillaryRecord, error) {
	if s.Key != "" {
		return AncillaryRecord{}, fail("invalid_input", "sections belong to an arrangement")
	}
	p, err := l.parents(ctx, s, true, false)
	if err != nil {
		return AncillaryRecord{}, err
	}
	r, err := l.api.get(ctx, arrangementPath(p.song.id, p.arr.id)+"/sections", "ArrangementSections", "")
	return AncillaryRecord{r.id, r.attrs}, err
}
func (l *Library) archiveArrangement(ctx context.Context, s Scope, target string, restore bool) (Arrangement, error) {
	s.Arrangement = target
	if restore && !s.IncludeArchived {
		return Arrangement{}, fail("invalid_input", "unarchive requires include-archived selection")
	}
	p, err := l.parents(ctx, s, true, true)
	if err != nil {
		return Arrangement{}, err
	}
	path := arrangementPath(p.song.id, p.arr.id)
	if s.IncludeArchived {
		path = songPath(p.song.id) + "/unscoped_arrangements/" + p.arr.id
	}
	action := "archive"
	if restore {
		action = "unarchive"
	}
	if err = l.api.http.Post(ctx, path+"/"+action, nil, nil, nil); err != nil {
		return Arrangement{}, err
	}
	// Archived arrangements leave the ordinary association; read back through the
	// documented unscoped association for both operations.
	r, err := l.api.get(ctx, songPath(p.song.id)+"/unscoped_arrangements/"+p.arr.id, "Arrangement", p.arr.id)
	if err != nil {
		return Arrangement{}, &Error{Code: "verification_failed", Message: action + " submitted but readback failed", Cause: err}
	}
	var out Arrangement
	if err = r.model(&out); err != nil {
		return out, err
	}
	if (out.ArchivedAt == "") != restore {
		return Arrangement{}, fail("verification_failed", action+" submitted but archive state differs on readback")
	}
	return out, nil
}
func (l *Library) ArchiveArrangement(ctx context.Context, s Scope, target string) (Arrangement, error) {
	return l.archiveArrangement(ctx, s, target, false)
}
func (l *Library) UnarchiveArrangement(ctx context.Context, s Scope, target string) (Arrangement, error) {
	return l.archiveArrangement(ctx, s, target, true)
}
func (l *Library) attachment(ctx context.Context, s Scope, target string, write bool) (record, string, error) {
	if _, _, _, err := ParseSelector(target); err != nil {
		return record{}, "", err
	}
	_, path, err := l.attachmentScope(ctx, s, write)
	if err != nil {
		return record{}, "", err
	}
	rows, err := l.api.collect(ctx, path, "Attachment", nil)
	if err != nil {
		return record{}, "", incomplete(err)
	}
	r, err := pick(rows, target, write, "filename")
	if err != nil {
		return record{}, "", err
	}
	actual, err := l.api.get(ctx, path+"/"+r.id, "Attachment", r.id)
	if err != nil {
		return record{}, "", err
	}
	if !reflect.DeepEqual(r.attrs, actual.attrs) {
		return record{}, "", changed("Attachment")
	}
	return actual, path + "/" + r.id, nil
}
func (l *Library) ShowAttachment(ctx context.Context, s Scope, target string) (Attachment, error) {
	r, _, err := l.attachment(ctx, s, target, false)
	var out Attachment
	if err == nil {
		err = r.model(&out)
	}
	return out, err
}

// AttachmentUpdateResult distinguishes a confirmed attribute update from an
// accepted write containing fields that the API does not expose for readback.
type AttachmentUpdateResult struct {
	Attachment       Attachment `json:"attachment"`
	Outcome          string     `json:"outcome"`
	UnverifiedFields []string   `json:"unverified_fields"`
}

// ValidateAttachmentPatch preserves explicit empty/false values and checks all
// local constraints before selection or configuration/network initialization.
func ValidateAttachmentPatch(m AttachmentPatch) error {
	attrs := fields(m)
	if m.RemoteLink != nil && *m.RemoteLink != "" {
		u, err := url.Parse(*m.RemoteLink)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return fail("invalid_input", "remote link must be an absolute HTTP or HTTPS URL without credentials")
		}
	}
	sources := 0
	for _, value := range []*string{m.RemoteLink, m.Content, m.FileUploadID} {
		if value != nil && *value != "" {
			sources++
		}
	}
	if sources > 1 {
		return fail("invalid_input", "attachment update accepts at most one replacement source")
	}
	if len(attrs) == 0 {
		return fail("invalid_input", "at least one changed field is required")
	}
	if m.Filename != nil && strings.TrimSpace(*m.Filename) == "" {
		return fail("invalid_input", "filename cannot be empty")
	}

	if m.FileUploadID != nil && *m.FileUploadID != "" && !validReceiptID(*m.FileUploadID) {
		return fail("invalid_input", "upload ID cannot contain whitespace or control characters")
	}
	return nil
}
func (l *Library) UpdateAttachment(ctx context.Context, s Scope, target string, m AttachmentPatch) (AttachmentUpdateResult, error) {
	if err := ValidateAttachmentPatch(m); err != nil {
		return AttachmentUpdateResult{}, err
	}
	attrs := fields(m)
	_, path, err := l.attachment(ctx, s, target, true)
	if err != nil {
		return AttachmentUpdateResult{}, err
	}

	r, err := l.api.mutate(ctx, "PATCH", path, "Attachment", attrs)
	if err != nil {
		return AttachmentUpdateResult{}, err
	}
	expectedID := path[strings.LastIndex(path, "/")+1:]
	if r.id != expectedID {
		return AttachmentUpdateResult{}, fail("outcome_unknown", "attachment update submitted but response identified another resource")
	}
	readable := map[string]any{}
	unverified := []string{}
	for name, value := range attrs {
		switch name {
		case "attachment_type_ids", "song_part":
			unverified = append(unverified, name)
		default:
			readable[name] = value
		}
	}
	if len(readable) > 0 {
		// UUID is deliberately omitted from ordinary responses. Request the sparse
		// fields explicitly, alongside the useful detail fields, before verification.
		selected := []string{"filename", "display_name", "content_type", "file_size", "filetype", "pco_type", "remote_link", "url", "thumbnail_url", "downloadable", "streamable", "transposable", "licenses_purchased", "licenses_remaining", "attachable_type"}
		fieldSet := map[string]bool{}
		for _, name := range selected {
			fieldSet[name] = true
		}
		for name := range readable {
			fieldSet[name] = true
		}
		selected = selected[:0]
		for name := range fieldSet {
			selected = append(selected, name)
		}
		sort.Strings(selected)
		var response planningcenter.Single[json.RawMessage]
		if err = l.api.http.Get(ctx, path, url.Values{"fields[Attachment]": {strings.Join(selected, ",")}}, &response); err != nil {
			return AttachmentUpdateResult{}, &Error{Code: "verification_failed", Message: "attachment update accepted but readback failed", Cause: err}
		}
		r, err = decodeRecord(response.Data, "Attachment")
		r = r.inPath(path)
		if err == nil {
			err = r.validateScope()
		}
		if err != nil || r.id != expectedID {
			return AttachmentUpdateResult{}, fail("verification_failed", "attachment update accepted but readback identified an invalid resource")
		}
		if err = verify(r, readable); err != nil {
			return AttachmentUpdateResult{}, err
		}
	}
	var out Attachment
	if err = r.model(&out); err != nil {
		return AttachmentUpdateResult{}, err
	}
	sort.Strings(unverified)
	outcome := "verified"
	if len(unverified) > 0 {
		outcome = "accepted"
	}
	return AttachmentUpdateResult{Attachment: out, Outcome: outcome, UnverifiedFields: unverified}, nil
}
func (l *Library) DeleteAttachment(ctx context.Context, s Scope, target string) error {
	_, path, err := l.attachment(ctx, s, target, true)
	if err != nil {
		return err
	}
	return l.api.http.Delete(ctx, path, nil)
}
