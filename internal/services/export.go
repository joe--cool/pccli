package services

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Walk methods visit the complete selected collection. Callbacks may run before a
// later page fails; callers must stage output privately until the method succeeds.
// The caller owns any model it elects to retain. The walkers retain one API page
// and fixed-size identity hashes, rather than the complete resource collection.
func (l *Library) WalkSongs(ctx context.Context, o SongListOptions, emit func(Song) error) (ListMeta, error) {
	if emit == nil {
		return ListMeta{}, fail("invalid_input", "an export callback is required")
	}
	if o.Limit != 0 {
		return ListMeta{}, fail("invalid_input", "complete exports do not accept a limit")
	}
	if o.CCLINumber < 0 {
		return ListMeta{}, fail("invalid_input", "CCLI number must be positive")
	}
	if o.Title != "" && o.Search != "" {
		return ListMeta{}, fail("invalid_input", "title and search cannot be combined")
	}
	order := strings.TrimPrefix(o.Order, "-")
	if order != "" && order != "title" && order != "created_at" && order != "updated_at" && order != "last_scheduled_at" {
		return ListMeta{}, fail("invalid_input", "unsupported song order")
	}
	meta := ListMeta{}
	visit := func(song Song) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := emit(song); err != nil {
			return err
		}
		meta.Returned++
		return nil
	}
	walk := func(consume func(Song) error) error {
		return l.walkSongRecords(ctx, o.Visibility, func(r record) error {
			var song Song
			if err := r.model(&song); err != nil {
				return err
			}
			if o.Title != "" && !strings.EqualFold(song.Title, o.Title) || o.Search != "" && !foldContains(song.Title, o.Search) || o.Author != "" && !strings.EqualFold(song.Author, o.Author) || o.CCLINumber != 0 && song.CCLINumber != o.CCLINumber || o.Themes != "" && !strings.EqualFold(song.Themes, o.Themes) {
				return nil
			}
			return consume(song)
		})
	}
	var err error
	if order == "" {
		err = walk(visit)
	} else {
		err = sortSongExport(ctx, o.Order, walk, visit)
	}
	if err != nil {
		return meta, err
	}
	meta.Complete = true
	return meta, nil
}
func (l *Library) walkSongRecords(ctx context.Context, v Visibility, emit func(record) error) error {
	v, err := visible(v)
	if err != nil {
		return err
	}
	states := []bool{v == Hidden}
	if v == All {
		states = []bool{false, true}
	}
	seen := map[string][32]byte{}
	for _, hidden := range states {
		q := url.Values{"where[hidden]": {map[bool]string{true: "true", false: "false"}[hidden]}}
		if err = l.api.walk(ctx, "/services/v2/songs", "Song", q, func(r record) error {
			var got bool
			raw, ok := r.attrs["hidden"]
			if !ok || string(raw) == "null" || json.Unmarshal(raw, &got) != nil {
				return fail("invalid_response", "song hidden state is missing or invalid")
			}
			if got != hidden {
				return fail("invalid_response", "song visibility contradicts requested filter")
			}
			hash, err := recordHash(r)
			if err != nil {
				return err
			}
			if old, ok := seen[r.id]; ok {
				if old != hash {
					return changed("Song")
				}
				return nil
			}
			seen[r.id] = hash
			return emit(r)
		}); err != nil {
			return err
		}
	}
	return nil
}
func walkModels[T any](ctx context.Context, l *Library, path, kind string, q url.Values, accept func(record) (bool, error), emit func(T) error) (ListMeta, error) {
	meta := ListMeta{}
	if emit == nil {
		return meta, fail("invalid_input", "an export callback is required")
	}
	err := l.api.walk(ctx, path, kind, q, func(r record) error {
		if accept != nil {
			include, err := accept(r)
			if err != nil {
				return err
			}
			if !include {
				return nil
			}
		}
		var model T
		if err := r.model(&model); err != nil {
			return err
		}
		if err := emit(model); err != nil {
			return err
		}
		meta.Returned++
		return nil
	})
	if err != nil {
		return meta, err
	}
	meta.Complete = true
	return meta, nil
}
func (l *Library) WalkArrangements(ctx context.Context, s Scope, emit func(Arrangement) error) (ListMeta, error) {
	p, err := l.parents(ctx, s, false, false)
	if err != nil {
		return ListMeta{}, err
	}
	return walkModels(ctx, l, arrangementsPath(p.song.id, s.IncludeArchived), "Arrangement", nil, func(r record) (bool, error) {
		if err := r.scope("song", "Song", p.song.id); err != nil {
			return false, err
		}
		return !isArchived(r) || s.IncludeArchived, nil
	}, emit)
}
func (l *Library) WalkKeys(ctx context.Context, s Scope, emit func(Key) error) (ListMeta, error) {
	p, err := l.parents(ctx, s, true, false)
	if err != nil {
		return ListMeta{}, err
	}
	return walkModels(ctx, l, arrangementPath(p.song.id, p.arr.id)+"/keys", "Key", nil, func(r record) (bool, error) { return true, r.scope("arrangement", "Arrangement", p.arr.id) }, emit)
}
func (l *Library) WalkAttachments(ctx context.Context, s Scope, emit func(Attachment) error) (ListMeta, error) {
	_, path, err := l.attachmentScope(ctx, s, false)
	if err != nil {
		return ListMeta{}, err
	}
	return walkModels(ctx, l, path, "Attachment", nil, nil, emit)
}
func (l *Library) WalkTagGroups(ctx context.Context, emit func(TagGroup) error) (ListMeta, error) {
	return walkModels(ctx, l, "/services/v2/tag_groups", "TagGroup", nil, func(r record) (bool, error) {
		var group TagGroup
		if err := r.model(&group); err != nil {
			return false, err
		}
		return group.TagsFor == "song" || group.TagsFor == "arrangement", nil
	}, emit)
}
func (l *Library) WalkAttachmentTypes(ctx context.Context, emit func(AttachmentType) error) (ListMeta, error) {
	return walkModels(ctx, l, "/services/v2/attachment_types", "AttachmentType", nil, nil, emit)
}
func (l *Library) WalkTags(ctx context.Context, s Scope, emit func(Tag) error) (ListMeta, error) {
	meta := ListMeta{}
	if emit == nil {
		return meta, fail("invalid_input", "an export callback is required")
	}
	if s.Key != "" || s.Song == "" && s.Arrangement != "" {
		return meta, fail("invalid_input", "tags select a song or arrangement, or omit both for the catalog")
	}
	// For scoped exports retain only the assigned IDs and name hashes, then enrich
	// them while streaming their authoritative catalog one group at a time.
	assigned := map[string][32]byte{}
	kind := ""
	if s.Song != "" {
		path, k, err := l.tagDestination(ctx, s, false)
		if err != nil {
			return meta, err
		}
		kind = k
		if err = l.api.walk(ctx, path+"/tags", "Tag", nil, func(r record) error {
			var name string
			if err := json.Unmarshal(r.attrs["name"], &name); err != nil {
				return fail("invalid_response", "assigned tag name missing or invalid")
			}
			assigned[r.id] = tagNameHash(name)
			return nil
		}); err != nil {
			return meta, err
		}
	}
	seen := map[string][32]byte{}
	_, err := l.WalkTagGroups(ctx, func(group TagGroup) error {
		if kind != "" && group.TagsFor != kind {
			return nil
		}
		return l.api.walk(ctx, "/services/v2/tag_groups/"+group.ID+"/tags", "Tag", nil, func(r record) error {
			if err := r.scope("tag_group", "TagGroup", group.ID); err != nil {
				return err
			}
			var tag Tag
			if err := r.model(&tag); err != nil {
				return err
			}
			tag.TagGroupID = group.ID
			tag.TagGroupName = group.Name
			tag.TagsFor = group.TagsFor
			hash, err := recordHash(r)
			if err != nil {
				return err
			}
			// Group membership is part of the tag identity snapshot, including
			// when the API omits the optional tag_group relationship object.
			hash = sha256.Sum256(append(hash[:], []byte(group.ID)...))
			if old, ok := seen[tag.ID]; ok {
				if old != hash {
					return changed("Tag")
				}
				return nil
			}
			seen[tag.ID] = hash
			if kind != "" {
				want, ok := assigned[tag.ID]
				if !ok {
					return nil
				}
				if want != tagNameHash(tag.Name) {
					return changed("Tag")
				}
				delete(assigned, tag.ID)
			}
			if err := emit(tag); err != nil {
				return err
			}
			meta.Returned++
			return nil
		})
	})
	if err != nil {
		return meta, err
	}
	if len(assigned) > 0 {
		return meta, fail("invalid_response", "assigned tag is absent from the matching catalog")
	}
	meta.Complete = true
	return meta, nil
}
func (l *Library) walkSongAncillary(ctx context.Context, s Scope, suffix, kind string, q url.Values, emit func(AncillaryRecord) error) (ListMeta, error) {
	meta := ListMeta{}
	if emit == nil {
		return meta, fail("invalid_input", "an export callback is required")
	}
	if s.Arrangement != "" || s.Key != "" {
		return meta, fail("invalid_input", "this workflow selects a song only")
	}
	p, err := l.parents(ctx, s, false, false)
	if err != nil {
		return meta, err
	}
	err = l.api.walk(ctx, songPath(p.song.id)+"/"+suffix, kind, q, func(r record) error {
		if err := emit(AncillaryRecord{ID: r.id, Attributes: r.attrs}); err != nil {
			return err
		}
		meta.Returned++
		return nil
	})
	if err != nil {
		return meta, err
	}
	meta.Complete = true
	return meta, nil
}
func (l *Library) WalkSongSchedules(ctx context.Context, s Scope, emit func(AncillaryRecord) error) (ListMeta, error) {
	return l.walkSongAncillary(ctx, s, "song_schedules", "SongSchedule", nil, emit)
}
func (l *Library) WalkLastScheduledItems(ctx context.Context, s Scope, emit func(AncillaryRecord) error) (ListMeta, error) {
	if !validID(s.ServiceTypeID) {
		return ListMeta{}, fail("invalid_input", "last scheduled item requires a positive service type ID")
	}
	return l.walkSongAncillary(ctx, s, "last_scheduled_item", "Item", url.Values{"service_type": {strings.TrimLeft(s.ServiceTypeID, "0")}}, emit)
}

// Song sorting uses bounded private runs because the API does not document a
// collation suitable for safely merging visible and hidden ordered collections.
const exportSortRunSize = 100
const exportSortFanIn = 16

func tagNameHash(name string) [32]byte { return sha256.Sum256([]byte(name)) }
func songLess(order string) func(Song, Song) bool {
	reverse := strings.HasPrefix(order, "-")
	field := strings.TrimPrefix(order, "-")
	value := func(song Song) string {
		switch field {
		case "title":
			return strings.ToLower(song.Title)
		case "created_at":
			return song.CreatedAt
		case "updated_at":
			return song.UpdatedAt
		case "last_scheduled_at":
			return song.LastScheduledAt
		}
		return ""
	}
	return func(a, b Song) bool {
		av, bv := value(a), value(b)
		if reverse {
			return av > bv
		}
		return av < bv
	}
}
func sortSongExport(ctx context.Context, order string, walk func(func(Song) error) error, emit func(Song) error) (err error) {
	directory, err := os.MkdirTemp("", "pccli-export-sort-")
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(directory); err == nil && cleanupErr != nil {
			err = cleanupErr
		}
	}()
	less := songLess(order)
	runs := []string{}
	chunk := make([]Song, 0, exportSortRunSize)
	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		sort.SliceStable(chunk, func(i, j int) bool { return less(chunk[i], chunk[j]) })
		name, err := writeSongRun(directory, func(encode func(Song) error) error {
			for _, song := range chunk {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := encode(song); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		runs = append(runs, name)
		clear(chunk)
		chunk = chunk[:0]
		return nil
	}
	if err = walk(func(song Song) error {
		chunk = append(chunk, song)
		if len(chunk) == exportSortRunSize {
			return flush()
		}
		return nil
	}); err != nil {
		return err
	}
	if err = flush(); err != nil {
		return err
	}
	for len(runs) > exportSortFanIn {
		next := []string{}
		for start := 0; start < len(runs); start += exportSortFanIn {
			end := min(start+exportSortFanIn, len(runs))
			group := runs[start:end]
			name, writeErr := writeSongRun(directory, func(encode func(Song) error) error { return mergeSongRuns(ctx, group, less, encode) })
			if writeErr != nil {
				return writeErr
			}
			next = append(next, name)
			for _, old := range group {
				if err = os.Remove(old); err != nil {
					return err
				}
			}
		}
		runs = next
	}
	return mergeSongRuns(ctx, runs, less, emit)
}
func writeSongRun(directory string, produce func(func(Song) error) error) (name string, err error) {
	file, err := os.CreateTemp(directory, "songs-*.jsonl")
	if err != nil {
		return "", err
	}
	name = file.Name()
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	encoder := json.NewEncoder(file)
	err = produce(func(song Song) error { return encoder.Encode(song) })
	return name, err
}
func mergeSongRuns(ctx context.Context, paths []string, less func(Song, Song) bool, emit func(Song) error) (err error) {
	files := make([]*os.File, 0, len(paths))
	decoders := make([]*json.Decoder, 0, len(paths))
	current := make([]Song, len(paths))
	active := make([]bool, len(paths))
	defer func() {
		for _, file := range files {
			if closeErr := file.Close(); err == nil && closeErr != nil {
				err = closeErr
			}
		}
	}()
	read := func(i int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		var song Song
		if err := decoders[i].Decode(&song); err != nil {
			if errors.Is(err, io.EOF) {
				active[i] = false
				current[i] = Song{}
				return nil
			}
			return err
		}
		current[i] = song
		active[i] = true
		return nil
	}
	for _, path := range paths {
		if err = ctx.Err(); err != nil {
			return err
		}
		file, openErr := os.Open(filepath.Clean(path))
		if openErr != nil {
			return openErr
		}
		files = append(files, file)
		decoders = append(decoders, json.NewDecoder(bufio.NewReader(file)))
		if err = read(len(files) - 1); err != nil {
			return err
		}
	}
	for {
		if err = ctx.Err(); err != nil {
			return err
		}
		selected := -1
		for i := range current {
			if active[i] && (selected < 0 || less(current[i], current[selected])) {
				selected = i
			}
		}
		if selected < 0 {
			return nil
		}
		if err = emit(current[selected]); err != nil {
			return err
		}
		if err = read(selected); err != nil {
			return err
		}
	}
}
