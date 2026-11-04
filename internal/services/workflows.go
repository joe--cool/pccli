package services

import (
	"context"
	"encoding/json"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

func (l *Library) songs(ctx context.Context, v Visibility) ([]record, error) {
	return l.songsLimited(ctx, v, -1)
}
func (l *Library) songsLimited(ctx context.Context, v Visibility, limit int) ([]record, error) {
	v, e := visible(v)
	if e != nil {
		return nil, e
	}
	states := []bool{v == Hidden}
	if v == All {
		states = []bool{false, true}
	}
	out := []record{}
	seen := map[string]record{}
	for _, hidden := range states {
		q := url.Values{"where[hidden]": {map[bool]string{true: "true", false: "false"}[hidden]}}
		remaining := limit
		if limit >= 0 {
			remaining = limit - len(out)
		}
		r, e := l.api.collectLimited(ctx, "/services/v2/songs", "Song", q, remaining, func(x record) (bool, error) {
			var got bool
			raw, ok := x.attrs["hidden"]
			if !ok || string(raw) == "null" || json.Unmarshal(raw, &got) != nil {
				return false, fail("invalid_response", "song hidden state is missing or invalid")
			}
			if got != hidden {
				return false, fail("invalid_response", "song visibility contradicts requested filter")
			}
			return true, nil
		})
		if e != nil {
			return nil, e
		}
		for _, x := range r {
			var got bool
			raw, ok := x.attrs["hidden"]
			if !ok || string(raw) == "null" || json.Unmarshal(raw, &got) != nil {
				return nil, fail("invalid_response", "song hidden state is missing or invalid")
			}
			if got != hidden {
				return nil, fail("invalid_response", "song visibility contradicts requested filter")
			}
			if old, ok := seen[x.id]; ok {
				if !reflect.DeepEqual(old, x) {
					return nil, changed("Song")
				}
				continue
			}
			seen[x.id] = x
			out = append(out, x)
		}
		if limit >= 0 && len(out) > limit {
			return out[:limit+1], nil
		}
	}
	return out, nil
}
func (l *Library) song(ctx context.Context, target string, v Visibility, write bool) (record, error) {
	v, e := visible(v)
	if e != nil {
		return record{}, e
	}
	id, _, _, e := ParseSelector(target)
	if e != nil {
		return record{}, e
	}
	var r record
	if id != "" {
		r, e = l.api.get(ctx, songPath(id), "Song", id)
	} else {
		selected, err := newRecordSelection(target, write, "title")
		if err != nil {
			return record{}, err
		}
		if err = l.walkSongRecords(ctx, v, selected.add); err != nil {
			return record{}, incomplete(err)
		}
		r, e = selected.result()
	}
	if e != nil {
		return record{}, e
	}
	var hidden bool
	raw, ok := r.attrs["hidden"]
	if !ok || string(raw) == "null" || json.Unmarshal(raw, &hidden) != nil {
		return record{}, fail("invalid_response", "song hidden state is missing or invalid")
	}
	if v != All && hidden != (v == Hidden) {
		return record{}, fail("not_found", "song is outside selected visibility")
	}
	if write {
		again, e := l.api.get(ctx, songPath(r.id), "Song", r.id)
		if e != nil {
			return record{}, e
		}
		if !reflect.DeepEqual(r.attrs, again.attrs) {
			return record{}, changed("Song")
		}
		r = again
	}
	return r, nil
}

type resolvedParents struct{ song, arr record }

func (l *Library) parents(ctx context.Context, s Scope, arrRequired, write bool) (resolvedParents, error) {
	if e := required(s.Song, "song"); e != nil {
		return resolvedParents{}, e
	}
	if arrRequired {
		if e := required(s.Arrangement, "arrangement"); e != nil {
			return resolvedParents{}, e
		}
	}
	song, e := l.song(ctx, s.Song, s.Visibility, true)
	if e != nil {
		return resolvedParents{}, e
	}
	p := resolvedParents{song: song}
	if s.Arrangement == "" {
		return p, nil
	}
	id, _, _, e := ParseSelector(s.Arrangement)
	if e != nil {
		return p, e
	}
	path := arrangementsPath(song.id, s.IncludeArchived)
	var r record
	if id != "" {
		r, e = l.api.get(ctx, path+"/"+id, "Arrangement", id)
	} else {
		r, e = l.resolve(ctx, path, "Arrangement", nil, s.Arrangement, true, "name", func(row record) (bool, error) {
			if err := row.scope("song", "Song", song.id); err != nil {
				return false, err
			}
			return !isArchived(row) || s.IncludeArchived, nil
		})
	}
	if e != nil {
		return p, e
	}
	if e = r.scope("song", "Song", song.id); e != nil {
		return p, e
	}
	if isArchived(r) && !s.IncludeArchived {
		return p, fail("not_found", "arrangement is archived; include archived arrangements explicitly")
	}
	if write {
		again, e := l.api.get(ctx, path+"/"+r.id, "Arrangement", r.id)
		if e != nil {
			return p, e
		}
		if e = again.scope("song", "Song", song.id); e != nil {
			return p, e
		}
		if !reflect.DeepEqual(r.attrs, again.attrs) {
			return p, changed("Arrangement")
		}
		r = again
	}
	p.arr = r
	return p, nil
}
func (l *Library) key(ctx context.Context, p resolvedParents, target string, write bool) (record, error) {
	id, _, _, e := ParseSelector(target)
	if e != nil {
		return record{}, e
	}
	path := arrangementPath(p.song.id, p.arr.id) + "/keys"
	var r record
	if id != "" {
		r, e = l.api.get(ctx, path+"/"+id, "Key", id)
	} else {
		r, e = l.resolve(ctx, path, "Key", nil, target, write, "name", func(row record) (bool, error) { return true, row.scope("arrangement", "Arrangement", p.arr.id) })
	}
	if e != nil {
		return record{}, e
	}
	if e = r.scope("arrangement", "Arrangement", p.arr.id); e != nil {
		return record{}, e
	}
	if write {
		again, e := l.api.get(ctx, path+"/"+r.id, "Key", r.id)
		if e != nil {
			return record{}, e
		}
		if e = again.scope("arrangement", "Arrangement", p.arr.id); e != nil {
			return record{}, e
		}
		if !reflect.DeepEqual(r.attrs, again.attrs) {
			return record{}, changed("Key")
		}
		r = again
	}
	return r, nil
}
func (l *Library) ListSongs(ctx context.Context, o SongListOptions) (ListResult[Song], error) {
	if o.CCLINumber < 0 {
		return ListResult[Song]{}, fail("invalid_input", "CCLI number must be positive")
	}
	order := strings.TrimPrefix(o.Order, "-")
	if order != "" && order != "title" && order != "created_at" && order != "updated_at" && order != "last_scheduled_at" {
		return ListResult[Song]{}, fail("invalid_input", "unsupported song order")
	}

	if _, e := bound([]Song{}, o.ListOptions); e != nil {
		return ListResult[Song]{}, e
	}
	if o.Search != "" && o.Title != "" {
		return ListResult[Song]{}, fail("invalid_input", "title and search cannot be combined")
	}
	limit := -1
	if order == "" && o.Title == "" && o.Search == "" && o.Author == "" && o.CCLINumber == 0 && o.Themes == "" {
		limit = displayLimit(o.ListOptions)
	}
	rows, e := l.songsLimited(ctx, o.Visibility, limit)
	if e != nil {
		return ListResult[Song]{}, e
	}
	if order != "" {
		sort.SliceStable(rows, func(i, j int) bool {
			var a, b string
			_ = json.Unmarshal(rows[i].attrs[order], &a)
			_ = json.Unmarshal(rows[j].attrs[order], &b)
			if order == "title" {
				a = strings.ToLower(a)
				b = strings.ToLower(b)
			}
			if strings.HasPrefix(o.Order, "-") {
				return a > b
			}
			return a < b
		})
	}
	out := []Song{}
	for _, r := range rows {
		var x Song
		if e = r.model(&x); e != nil {
			return ListResult[Song]{}, e
		}
		if o.Title != "" && !strings.EqualFold(x.Title, o.Title) {
			continue
		}
		if o.Search != "" && !foldContains(x.Title, o.Search) {
			continue
		}
		if o.Author != "" && !strings.EqualFold(x.Author, o.Author) {
			continue
		}
		if o.CCLINumber != 0 && x.CCLINumber != o.CCLINumber {
			continue
		}
		if o.Themes != "" && !strings.EqualFold(x.Themes, o.Themes) {
			continue
		}
		out = append(out, x)
	}
	return bound(out, o.ListOptions)
}
func (l *Library) ListArrangements(ctx context.Context, s Scope, o ListOptions) (ListResult[Arrangement], error) {
	if _, e := bound([]Arrangement{}, o); e != nil {
		return ListResult[Arrangement]{}, e
	}
	p, e := l.parents(ctx, s, false, false)
	if e != nil {
		return ListResult[Arrangement]{}, e
	}
	rows, e := l.api.collectLimited(ctx, arrangementsPath(p.song.id, s.IncludeArchived), "Arrangement", nil, displayLimit(o), func(r record) (bool, error) {
		if err := r.scope("song", "Song", p.song.id); err != nil {
			return false, err
		}
		return !isArchived(r) || s.IncludeArchived, nil
	})
	if e != nil {
		return ListResult[Arrangement]{}, e
	}
	out := []Arrangement{}
	for _, r := range rows {
		if e = r.scope("song", "Song", p.song.id); e != nil {
			return ListResult[Arrangement]{}, e
		}
		if isArchived(r) && !s.IncludeArchived {
			continue
		}
		var x Arrangement
		if e = r.model(&x); e != nil {
			return ListResult[Arrangement]{}, e
		}
		out = append(out, x)
	}
	return bound(out, o)
}
func (l *Library) ListKeys(ctx context.Context, s Scope, o ListOptions) (ListResult[Key], error) {
	if _, e := bound([]Key{}, o); e != nil {
		return ListResult[Key]{}, e
	}
	p, e := l.parents(ctx, s, true, false)
	if e != nil {
		return ListResult[Key]{}, e
	}
	rows, e := l.api.collectLimited(ctx, arrangementPath(p.song.id, p.arr.id)+"/keys", "Key", nil, displayLimit(o), func(r record) (bool, error) {
		if err := r.scope("arrangement", "Arrangement", p.arr.id); err != nil {
			return false, err
		}
		return true, nil
	})
	if e != nil {
		return ListResult[Key]{}, e
	}
	out := []Key{}
	for _, r := range rows {
		if e = r.scope("arrangement", "Arrangement", p.arr.id); e != nil {
			return ListResult[Key]{}, e
		}
		var x Key
		if e = r.model(&x); e != nil {
			return ListResult[Key]{}, e
		}
		out = append(out, x)
	}
	return bound(out, o)
}
func (l *Library) ShowSong(ctx context.Context, target string, o SongShowOptions) (SongDetail, error) {
	r, e := l.song(ctx, target, o.Visibility, false)
	if e != nil {
		return SongDetail{}, e
	}
	var d SongDetail
	if e = r.model(&d.Song); e != nil {
		return d, e
	}
	s := Scope{Song: r.id, Visibility: o.Visibility, IncludeArchived: o.IncludeArchived}
	d.Arrangements, e = l.ListArrangements(ctx, s, ListOptions{All: true})
	if e != nil {
		return d, e
	}
	d.Song.ArrangementCount = len(d.Arrangements.Data)
	if o.Arrangement != "" {
		var a ArrangementDetail
		a, e = l.ShowArrangement(ctx, s, o.Arrangement)
		if e != nil {
			return d, e
		}
		d.Arrangement = &a
	}
	return d, nil
}
func (l *Library) ShowArrangement(ctx context.Context, s Scope, target string) (ArrangementDetail, error) {

	s.Arrangement = ""
	p, e := l.parents(ctx, s, false, false)
	if e != nil {
		return ArrangementDetail{}, e
	}
	id, _, _, e := ParseSelector(target)
	if e != nil {
		return ArrangementDetail{}, e
	}
	path := arrangementsPath(p.song.id, s.IncludeArchived)
	var r record
	if id != "" {
		r, e = l.api.get(ctx, path+"/"+id, "Arrangement", id)
	} else {
		r, e = l.resolve(ctx, path, "Arrangement", nil, target, false, "name", func(row record) (bool, error) {
			if err := row.scope("song", "Song", p.song.id); err != nil {
				return false, err
			}
			return !isArchived(row) || s.IncludeArchived, nil
		})
	}
	if e != nil {
		return ArrangementDetail{}, e
	}
	if e = r.scope("song", "Song", p.song.id); e != nil {
		return ArrangementDetail{}, e
	}
	if isArchived(r) && !s.IncludeArchived {
		return ArrangementDetail{}, fail("not_found", "arrangement is archived; include archived arrangements explicitly")
	}
	p.arr = r
	var d ArrangementDetail
	if e = p.arr.model(&d.Arrangement); e != nil {
		return d, e
	}
	s.Song = p.song.id
	s.Arrangement = p.arr.id
	d.Keys, e = l.ListKeys(ctx, s, ListOptions{All: true})
	return d, e
}
func (l *Library) ShowKey(ctx context.Context, s Scope, target string) (Key, error) {
	p, e := l.parents(ctx, s, true, false)
	if e != nil {
		return Key{}, e
	}
	r, e := l.key(ctx, p, target, false)
	var out Key
	if e == nil {
		e = r.model(&out)
	}
	return out, e
}

func arrangementsPath(song string, includeArchived bool) string {
	if includeArchived {
		return songPath(song) + "/unscoped_arrangements"
	}
	return songPath(song) + "/arrangements"
}
func isArchived(r record) bool {
	raw := r.attrs["archived_at"]
	return len(raw) > 0 && string(raw) != "null" && string(raw) != "\"\""
}

// A bounded accumulator establishes identity after complete traversal without
// retaining all candidates. Candidate summaries are capped for error output.
type recordSelection struct {
	target, nameField, id, name        string
	write, exact                       bool
	exactCount, partialCount           int
	firstExact, firstPartial           record
	exactCandidates, partialCandidates []map[string]string
}

func newRecordSelection(target string, write bool, nameField string) (*recordSelection, error) {
	id, name, exact, err := ParseSelector(target)
	if err != nil {
		return nil, err
	}
	return &recordSelection{target: target, nameField: nameField, id: id, name: name, write: write, exact: exact}, nil
}
func (s *recordSelection) add(r record) error {
	var name string
	if err := json.Unmarshal(r.attrs[s.nameField], &name); err != nil {
		return fail("invalid_response", "resource name missing or invalid")
	}
	exact := s.id != "" && r.id == s.id || s.id == "" && strings.EqualFold(name, s.name)
	partial := s.id == "" && !s.exact && !s.write && foldContains(name, s.name)
	if exact {
		s.exactCount++
		if s.exactCount == 1 {
			s.firstExact = r
		}
		if len(s.exactCandidates) < 20 {
			s.exactCandidates = append(s.exactCandidates, map[string]string{"id": r.id, "name": name})
		}
	} else if partial {
		s.partialCount++
		if s.partialCount == 1 {
			s.firstPartial = r
		}
		if len(s.partialCandidates) < 20 {
			s.partialCandidates = append(s.partialCandidates, map[string]string{"id": r.id, "name": name})
		}
	}
	return nil
}
func (s *recordSelection) result() (record, error) {
	count, first, candidates := s.exactCount, s.firstExact, s.exactCandidates
	if count == 0 && !s.write && !s.exact && s.id == "" {
		count, first, candidates = s.partialCount, s.firstPartial, s.partialCandidates
	}
	if count == 0 {
		return record{}, fail("not_found", "no matching "+s.target+" in selected scope")
	}
	if count > 1 {
		return record{}, &Error{Code: "ambiguous_target", Message: "selector matches multiple resources; choose an ID", Candidates: candidates}
	}
	return first, nil
}
func (l *Library) resolve(ctx context.Context, path, kind string, q url.Values, target string, write bool, nameField string, accept func(record) (bool, error)) (record, error) {
	selected, err := newRecordSelection(target, write, nameField)
	if err != nil {
		return record{}, err
	}
	err = l.api.walk(ctx, path, kind, q, func(r record) error {
		if accept != nil {
			include, err := accept(r)
			if err != nil {
				return err
			}
			if !include {
				return nil
			}
		}
		return selected.add(r)
	})
	if err != nil {
		return record{}, incomplete(err)
	}
	return selected.result()
}
