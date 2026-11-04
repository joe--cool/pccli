package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

type adapter struct{ http HTTPAPI }
type record struct {
	id, kind string
	attrs    map[string]json.RawMessage
	rel      json.RawMessage
	parents  Scope
}

func decodeRecord(r planningcenter.Resource[json.RawMessage], kind string) (record, error) {
	if r.Type != kind || !validID(r.ID) {
		return record{}, fail("invalid_response", "unexpected resource type or ID")
	}
	var a map[string]json.RawMessage
	if err := json.Unmarshal(r.Attributes, &a); err != nil || a == nil {
		return record{}, fail("invalid_response", "resource attributes must be an object")
	}
	nameField := ""
	switch kind {
	case "Song":
		nameField = "title"
	case "Arrangement", "Key":
		nameField = "name"
	}
	if nameField != "" {
		var name string
		raw, ok := a[nameField]
		if !ok || string(raw) == "null" || json.Unmarshal(raw, &name) != nil || strings.TrimSpace(name) == "" {
			return record{}, fail("invalid_response", "resource name missing or invalid")
		}
	}
	var model any
	switch kind {
	case "Song":
		model = &Song{}
	case "Arrangement":
		model = &Arrangement{}
	case "Key":
		model = &Key{}
	case "Attachment":
		model = &Attachment{}
	}
	if model != nil {
		if err := json.Unmarshal(r.Attributes, model); err != nil {
			return record{}, fail("invalid_response", "invalid typed resource attributes")
		}
	}
	return record{id: r.ID, kind: kind, attrs: a, rel: r.Relationships}, nil
}
func (r record) model(dst any) error {
	b, _ := json.Marshal(r.attrs)
	if err := json.Unmarshal(b, dst); err != nil {
		return fail("invalid_response", "invalid resource attributes")
	}
	reflect.ValueOf(dst).Elem().FieldByName("ID").SetString(r.id)
	switch model := dst.(type) {
	case *Arrangement:
		model.SongID = r.parents.Song
	case *Key:
		model.SongID = r.parents.Song
		model.ArrangementID = r.parents.Arrangement
	case *Attachment:
		model.SongID = r.parents.Song
		model.ArrangementID = r.parents.Arrangement
		model.KeyID = r.parents.Key
	}
	return nil
}
func (r record) scope(name, kind, id string) error {
	if len(r.rel) == 0 {
		return nil
	}
	var relationships map[string]json.RawMessage
	if json.Unmarshal(r.rel, &relationships) != nil || relationships == nil {
		return fail("invalid_response", "invalid resource relationships")
	}
	raw, provided := relationships[name]
	if !provided {
		return nil
	}
	var relationship struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &relationship) != nil || len(relationship.Data) == 0 || relationship.Data[0] != '{' {
		return fail("invalid_response", "invalid requested parent relationship")
	}
	var data struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if json.Unmarshal(relationship.Data, &data) != nil || data.ID != id || data.Type != kind {
		return fail("invalid_response", "resource relationship does not match requested parent")
	}
	return nil
}
func (r record) validateScope() error {
	switch r.kind {
	case "Arrangement":
		if r.parents.Song != "" {
			return r.scope("song", "Song", r.parents.Song)
		}
	case "Key":
		if r.parents.Arrangement != "" {
			return r.scope("arrangement", "Arrangement", r.parents.Arrangement)
		}
	case "Attachment":
		if r.parents.Key != "" {
			return r.scope("attachable", "Key", r.parents.Key)
		}
		if r.parents.Arrangement != "" {
			return r.scope("attachable", "Arrangement", r.parents.Arrangement)
		}
		if r.parents.Song != "" {
			return r.scope("attachable", "Song", r.parents.Song)
		}
	}
	return nil
}
func (a *adapter) get(ctx context.Context, path, kind, id string) (record, error) {
	var s planningcenter.Single[json.RawMessage]
	if err := a.http.Get(ctx, path, nil, &s); err != nil {
		return record{}, err
	}
	r, e := decodeRecord(s.Data, kind)
	r = r.inPath(path)
	if e == nil {
		e = r.validateScope()
	}
	if e == nil && id != "" && r.id != id {
		return record{}, fail("invalid_response", "response ID does not match requested ID")
	}
	return r, e
}
func (a *adapter) collect(ctx context.Context, path, kind string, q url.Values) ([]record, error) {
	return a.collectLimited(ctx, path, kind, q, -1, nil)
}

// collectLimited validates each visited page and reads one extra accepted identity
// to establish truncation. A negative limit exhausts the collection for resolution.
func (a *adapter) collectLimited(ctx context.Context, path, kind string, q url.Values, limit int, accept func(record) (bool, error)) ([]record, error) {
	cursor := a.cursor(path, kind, q)
	out := []record{}
	for {
		r, ok, err := cursor.next(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return out, nil
		}
		include := true
		if accept != nil {
			include, err = accept(r)
			if err != nil {
				return nil, err
			}
		}
		if include {
			out = append(out, r)
			if limit >= 0 && len(out) > limit {
				return out, nil
			}
		}
	}
}

// recordCursor owns one validated page plus identity hashes. All collection
// consumers use this cursor, so exports cannot bypass response or cursor policy.
type recordCursor struct {
	api             *adapter
	path, kind      string
	query, original url.Values
	seenPage        map[string]bool
	seen            map[string][32]byte
	page            []record
	index, offset   int
	done            bool
}

func (a *adapter) cursor(path, kind string, q url.Values) *recordCursor {
	q = cloneQuery(q)
	q.Set("per_page", "100")
	return &recordCursor{api: a, path: path, kind: kind, query: q, original: cloneQuery(q), seenPage: map[string]bool{}, seen: map[string][32]byte{}}
}
func (c *recordCursor) next(ctx context.Context) (record, bool, error) {
	for {
		if err := ctx.Err(); err != nil {
			return record{}, false, err
		}
		if c.index < len(c.page) {
			r := c.page[c.index]
			c.page[c.index] = record{}
			c.index++
			return r, true, nil
		}
		if c.done {
			return record{}, false, nil
		}
		if err := c.fetch(ctx); err != nil {
			return record{}, false, err
		}
	}
}
func (c *recordCursor) fetch(ctx context.Context) error {
	signature := c.query.Encode()
	if c.seenPage[signature] {
		return fail("invalid_response", "pagination cycle")
	}
	c.seenPage[signature] = true
	var page planningcenter.Collection[json.RawMessage]
	if err := c.api.http.Get(ctx, c.path, c.query, &page); err != nil {
		return err
	}
	if page.Data == nil {
		return fail("invalid_response", "collection data is missing")
	}
	if len(page.Data) > 100 {
		return fail("invalid_response", "collection exceeded requested page size")
	}
	rows := make([]record, 0, len(page.Data))
	for _, x := range page.Data {
		r, err := decodeRecord(x, c.kind)
		if err != nil {
			return err
		}
		r = r.inPath(c.path)
		if err = r.validateScope(); err != nil {
			return err
		}
		fingerprint, err := recordHash(r)
		if err != nil {
			return err
		}
		if old, ok := c.seen[r.id]; ok {
			if old != fingerprint {
				return changed(c.kind)
			}
			continue
		}
		c.seen[r.id] = fingerprint
		rows = append(rows, r)
	}
	if page.Meta.Count > len(page.Data) || page.Meta.TotalCount < 0 || page.Meta.Count < 0 {
		return fail("invalid_response", "collection metadata contradicts returned records")
	}
	if page.Links.Next == "" {
		if page.Meta.TotalCount > len(c.seen) {
			return fail("invalid_response", "collection pagination ended before reported total")
		}
		c.done = true
	} else {
		if len(page.Data) == 0 {
			return fail("invalid_response", "empty page cannot establish a next page")
		}
		u, err := url.Parse(page.Links.Next)
		trustedOrigin := !uIsAbsolute(u)
		if resolver, ok := c.api.http.(interface {
			ResolveEndpoint(string) (*url.URL, error)
		}); ok {
			_, validationErr := resolver.ResolveEndpoint(page.Links.Next)
			trustedOrigin = validationErr == nil
		} else if u != nil && u.IsAbs() {
			trustedOrigin = u.Scheme == "https" && u.Host == "api.planningcenteronline.com"
		}
		if err != nil || u == nil || !trustedOrigin || u.User != nil || u.Fragment != "" || u.RawPath != "" || u.Path != c.path || (!u.IsAbs() && u.Host != "") {
			return fail("invalid_response", "untrusted pagination URL")
		}
		next, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return fail("invalid_response", "invalid pagination query")
		}
		n, err := strconv.Atoi(next.Get("offset"))
		if err != nil || n <= c.offset || n < 0 || len(next["offset"]) != 1 {
			return fail("invalid_response", "pagination offset must advance")
		}
		next.Del("offset")
		if next.Encode() != c.original.Encode() {
			return fail("invalid_response", "pagination changed collection filters")
		}
		c.offset = n
		next.Set("offset", strconv.Itoa(n))
		c.query = next
	}
	c.page = rows
	c.index = 0
	return nil
}
func recordHash(r record) ([32]byte, error) {
	var relationships any
	if len(r.rel) > 0 {
		if err := json.Unmarshal(r.rel, &relationships); err != nil {
			return [32]byte{}, fail("invalid_response", "invalid resource relationships")
		}
	}
	b, err := json.Marshal(struct {
		ID, Kind      string
		Attrs         map[string]json.RawMessage
		Relationships any
		Parents       Scope
	}{r.id, r.kind, r.attrs, relationships, r.parents})
	if err != nil {
		return [32]byte{}, fail("invalid_response", "invalid resource snapshot")
	}
	return sha256.Sum256(b), nil
}
func (a *adapter) walk(ctx context.Context, path, kind string, q url.Values, emit func(record) error) error {
	cursor := a.cursor(path, kind, q)
	for {
		r, ok, err := cursor.next(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if err = emit(r); err != nil {
			return err
		}
	}
}

func (a *adapter) mutate(ctx context.Context, method, path, kind string, attrs map[string]any) (record, error) {
	var s planningcenter.Single[json.RawMessage]
	body := map[string]any{"data": map[string]any{"type": kind, "attributes": attrs}}
	var err error
	if method == "POST" {
		err = a.http.Post(ctx, path, nil, body, &s)
	} else {
		err = a.http.Patch(ctx, path, nil, body, &s)
	}
	if err != nil {
		return record{}, err
	}
	r, err := decodeRecord(s.Data, kind)
	r = r.inPath(path)
	if err != nil {
		return record{}, &Error{Code: "outcome_unknown", Message: "write submitted but response was invalid", Cause: err}
	}
	if err = r.validateScope(); err != nil {
		return record{}, &Error{Code: "outcome_unknown", Message: "write submitted but response parent was inconsistent", Cause: err}
	}
	return r, nil
}
func verify(r record, attrs map[string]any) error {
	for k, want := range attrs {
		raw, ok := r.attrs[k]
		if !ok && k == "mtid" {
			raw, ok = r.attrs["rehearsal_mix_id"]
		}
		if !ok && k == "rehearsal_mix_id" {
			raw, ok = r.attrs["mtid"]
		}
		if !ok {
			return fail("verification_failed", "write submitted, but response did not confirm field "+k)
		}
		var got any
		if json.Unmarshal(raw, &got) != nil {
			return fail("verification_failed", "write submitted; invalid field "+k)
		}
		b, _ := json.Marshal(want)
		var expected any
		_ = json.Unmarshal(b, &expected)
		if !reflect.DeepEqual(expected, got) {
			return fail("verification_failed", "write submitted, but returned value differs for "+k)
		}
	}
	return nil
}
func songPath(id string) string               { return "/services/v2/songs/" + id }
func arrangementPath(song, arr string) string { return songPath(song) + "/arrangements/" + arr }
func keyPath(song, arr, key string) string    { return arrangementPath(song, arr) + "/keys/" + key }
func validID(s string) bool {
	if s == "" {
		return false
	}
	positive := false
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
		positive = positive || c != '0'
	}
	return positive
}
func ParseSelector(s string) (id, name string, exact bool, err error) {
	if strings.HasPrefix(s, "name:") {
		name = strings.TrimPrefix(s, "name:")
		if strings.TrimSpace(name) == "" {
			err = fail("invalid_input", "name selector cannot be empty")
		}
		return "", name, true, err
	}
	forced := strings.HasPrefix(s, "id:")
	if forced {
		s = strings.TrimPrefix(s, "id:")
	}
	if validID(s) {
		return strings.TrimLeft(s, "0"), "", true, nil
	}
	numeric := s != ""
	for _, c := range s {
		if c < '0' || c > '9' {
			numeric = false
		}
	}
	signedNumeric := false
	if len(s) > 1 && (s[0] == '-' || s[0] == '+') {
		signedNumeric = true
		for _, r := range s[1:] {
			if r < '0' || r > '9' {
				signedNumeric = false
			}
		}
	}
	if forced || numeric || signedNumeric {
		return "", "", false, fail("invalid_input", "ID must be a positive decimal integer")
	}
	if strings.TrimSpace(s) == "" {
		return "", "", false, fail("invalid_input", "selector is required")
	}
	return "", s, false, nil
}
func pick(records []record, target string, write bool, nameField string) (record, error) {
	selected, err := newRecordSelection(target, write, nameField)
	if err != nil {
		return record{}, err
	}
	for _, r := range records {
		if err = selected.add(r); err != nil {
			return record{}, err
		}
	}
	return selected.result()
}

func cloneQuery(q url.Values) url.Values {
	out := url.Values{}
	for k, v := range q {
		out[k] = append([]string(nil), v...)
	}
	return out
}

func uIsAbsolute(u *url.URL) bool { return u != nil && u.IsAbs() }

func foldContains(value, fragment string) bool {
	if fragment == "" {
		return true
	}
	runes := []rune(value)
	n := len([]rune(fragment))
	for i := 0; i+n <= len(runes); i++ {
		if strings.EqualFold(string(runes[i:i+n]), fragment) {
			return true
		}
	}
	return false
}

// inPath carries canonical parent IDs from the scoped endpoint into product
// models even when the server omits optional relationship objects.
func (r record) inPath(path string) record {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if i+1 >= len(parts) {
			continue
		}
		id := parts[i+1]
		if !validID(id) {
			continue
		}
		switch p {
		case "songs":
			r.parents.Song = id
		case "arrangements", "unscoped_arrangements":
			r.parents.Arrangement = id
		case "keys":
			r.parents.Key = id
		}
	}
	return r
}
