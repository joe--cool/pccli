package services

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
)

type SongCreate struct {
	Title                string  `json:"title"`
	Admin                *string `json:"admin,omitempty"`
	Author               *string `json:"author,omitempty"`
	Copyright            *string `json:"copyright,omitempty"`
	Themes               *string `json:"themes,omitempty"`
	CCLINumber           *int    `json:"ccli_number,omitempty"`
	Hidden               *bool   `json:"hidden,omitempty"`
	ImportCCLIChordChart *string `json:"import_ccli_chord_chart,omitempty"`
	ImportCCLIChordPro   *string `json:"import_ccli_chord_pro,omitempty"`
	ImportCCLILyrics     *string `json:"import_ccli_lyrics,omitempty"`
}
type SongPatch struct {
	Title      *string `json:"title,omitempty"`
	Admin      *string `json:"admin,omitempty"`
	Author     *string `json:"author,omitempty"`
	Copyright  *string `json:"copyright,omitempty"`
	Themes     *string `json:"themes,omitempty"`
	CCLINumber *int    `json:"ccli_number,omitempty"`
	Hidden     *bool   `json:"hidden,omitempty"`
}
type ArrangementCreate struct {
	Name                       string    `json:"name"`
	BPM                        *float64  `json:"bpm,omitempty"`
	Meter                      *string   `json:"meter,omitempty"`
	ChordChartKey              *string   `json:"chord_chart_key,omitempty"`
	Notes                      *string   `json:"notes,omitempty"`
	Length                     *int      `json:"length,omitempty"`
	LyricsEnabled              *bool     `json:"lyrics_enabled,omitempty"`
	Sequence                   *[]string `json:"sequence,omitempty"`
	ChordChart                 *string   `json:"chord_chart,omitempty"`
	ChordChartChordColor       *int      `json:"chord_chart_chord_color,omitempty"`
	ChordChartColumns          *int      `json:"chord_chart_columns,omitempty"`
	ChordChartFont             *string   `json:"chord_chart_font,omitempty"`
	ChordChartFontSize         *int      `json:"chord_chart_font_size,omitempty"`
	ImportChartPro             *string   `json:"import_chart_pro,omitempty"`
	ImportChartProOptions      *string   `json:"import_chart_pro_options,omitempty"`
	ImportRehearsalMix         *string   `json:"import_rehearsal_mix,omitempty"`
	ImportRehearsalMixOptions  *string   `json:"import_rehearsal_mix_options,omitempty"`
	ImportRehearsalPack        *string   `json:"import_rehearsal_pack,omitempty"`
	ImportRehearsalPackOptions *string   `json:"import_rehearsal_pack_options,omitempty"`
	ISRC                       *int      `json:"isrc,omitempty"`
	MTID                       *string   `json:"mtid,omitempty"`
	NumberChartEnabled         *bool     `json:"number_chart_enabled,omitempty"`
	NumeralChartEnabled        *bool     `json:"numeral_chart_enabled,omitempty"`
	PrintMargin                *string   `json:"print_margin,omitempty"`
	PrintOrientation           *string   `json:"print_orientation,omitempty"`
	PrintPageSize              *string   `json:"print_page_size,omitempty"`
	RehearsalMixID             *string   `json:"rehearsal_mix_id,omitempty"`
	RehearsalMixThumbnail      *string   `json:"rehearsal_mix_thumbnail,omitempty"`
	ImportCCLIChordChart       *string   `json:"import_ccli_chord_chart,omitempty"`
	ImportCCLIChordPro         *string   `json:"import_ccli_chord_pro,omitempty"`
	ImportCCLILyrics           *string   `json:"import_ccli_lyrics,omitempty"`
}
type ArrangementPatch struct {
	Name                       *string   `json:"name,omitempty"`
	BPM                        *float64  `json:"bpm,omitempty"`
	Meter                      *string   `json:"meter,omitempty"`
	ChordChartKey              *string   `json:"chord_chart_key,omitempty"`
	Notes                      *string   `json:"notes,omitempty"`
	Length                     *int      `json:"length,omitempty"`
	LyricsEnabled              *bool     `json:"lyrics_enabled,omitempty"`
	Sequence                   *[]string `json:"sequence,omitempty"`
	ChordChart                 *string   `json:"chord_chart,omitempty"`
	ChordChartChordColor       *int      `json:"chord_chart_chord_color,omitempty"`
	ChordChartColumns          *int      `json:"chord_chart_columns,omitempty"`
	ChordChartFont             *string   `json:"chord_chart_font,omitempty"`
	ChordChartFontSize         *int      `json:"chord_chart_font_size,omitempty"`
	ImportChartPro             *string   `json:"import_chart_pro,omitempty"`
	ImportChartProOptions      *string   `json:"import_chart_pro_options,omitempty"`
	ImportRehearsalMix         *string   `json:"import_rehearsal_mix,omitempty"`
	ImportRehearsalMixOptions  *string   `json:"import_rehearsal_mix_options,omitempty"`
	ImportRehearsalPack        *string   `json:"import_rehearsal_pack,omitempty"`
	ImportRehearsalPackOptions *string   `json:"import_rehearsal_pack_options,omitempty"`
	ISRC                       *int      `json:"isrc,omitempty"`
	MTID                       *string   `json:"mtid,omitempty"`
	NumberChartEnabled         *bool     `json:"number_chart_enabled,omitempty"`
	NumeralChartEnabled        *bool     `json:"numeral_chart_enabled,omitempty"`
	PrintMargin                *string   `json:"print_margin,omitempty"`
	PrintOrientation           *string   `json:"print_orientation,omitempty"`
	PrintPageSize              *string   `json:"print_page_size,omitempty"`
	RehearsalMixID             *string   `json:"rehearsal_mix_id,omitempty"`
	RehearsalMixThumbnail      *string   `json:"rehearsal_mix_thumbnail,omitempty"`
}
type KeyCreate struct {
	Name                        string  `json:"name"`
	StartingKey                 *string `json:"starting_key,omitempty"`
	EndingKey                   *string `json:"ending_key,omitempty"`
	AlternateKeys               *string `json:"alternate_keys,omitempty"`
	ImportCCLIChordChart        *string `json:"import_ccli_chord_chart,omitempty"`
	ImportCCLIChordChartOptions *string `json:"import_ccli_chord_chart_options,omitempty"`
	ImportCCLILyrics            *string `json:"import_ccli_lyrics,omitempty"`
	ImportChartPro              *string `json:"import_chart_pro,omitempty"`
	ImportRehearsalMix          *string `json:"import_rehearsal_mix,omitempty"`
	ImportRehearsalPack         *string `json:"import_rehearsal_pack,omitempty"`
}
type KeyPatch struct {
	Name                        *string `json:"name,omitempty"`
	StartingKey                 *string `json:"starting_key,omitempty"`
	EndingKey                   *string `json:"ending_key,omitempty"`
	AlternateKeys               *string `json:"alternate_keys,omitempty"`
	ImportCCLIChordChart        *string `json:"import_ccli_chord_chart,omitempty"`
	ImportCCLIChordChartOptions *string `json:"import_ccli_chord_chart_options,omitempty"`
	ImportCCLILyrics            *string `json:"import_ccli_lyrics,omitempty"`
	ImportChartPro              *string `json:"import_chart_pro,omitempty"`
	ImportRehearsalMix          *string `json:"import_rehearsal_mix,omitempty"`
	ImportRehearsalPack         *string `json:"import_rehearsal_pack,omitempty"`
}
type AttachmentCreate struct {
	FilePath, FileUploadID, Filename, RemoteLink, Content, AttachmentTypeIDs, PageOrder, SongPart string
	ImportToItemDetails                                                                           *bool
}
type UploadedFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	FileSize    int    `json:"file_size"`
	ExpiresAt   string `json:"expires_at"`
}
type AttachmentResult struct {
	Attachment *Attachment   `json:"attachment"`
	Upload     *UploadedFile `json:"upload"`
	Outcome    string        `json:"outcome"`
	Scope      Scope         `json:"scope"`
}

func fields(v any) map[string]any {
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}
	out := make(map[string]any)
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		name := strings.Split(value.Type().Field(i).Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				continue
			}
			field = field.Elem()
		}
		if field.Kind() == reflect.Slice && field.IsNil() {
			out[name] = []string{}
		} else {
			out[name] = field.Interface()
		}
	}
	return out
}

// ValidateMutation validates a complete typed mutation before any HTTP request.
// In particular, encoding failures must never silently discard supplied fields.
func ValidateMutation(v any, kind string) error {
	validType := false
	switch kind {
	case "Song":
		switch v.(type) {
		case SongCreate, SongPatch, *SongCreate, *SongPatch:
			validType = true
		}
	case "Arrangement":
		switch v.(type) {
		case ArrangementCreate, ArrangementPatch, *ArrangementCreate, *ArrangementPatch:
			validType = true
		}
	case "Key":
		switch v.(type) {
		case KeyCreate, KeyPatch, *KeyCreate, *KeyPatch:
			validType = true
		}
	}
	if !validType {
		return fail("invalid_input", "mutation type does not match resource")
	}
	if _, err := json.Marshal(v); err != nil {
		return fail("invalid_input", "mutation contains a value that cannot be encoded as JSON")
	}
	var creation *SongCreate
	switch value := v.(type) {
	case SongCreate:
		creation = &value
	case *SongCreate:
		creation = value
	}
	if creation != nil {
		normalized := normalizeSongCreate(*creation)
		if normalized.Title != "" && strings.Trim(normalized.Title, "0123456789") == "" && (normalized.CCLINumber == nil || strconv.Itoa(*normalized.CCLINumber) != normalized.Title) {
			return fail("invalid_input", "a numeric create title triggers CCLI import; use --ccli explicitly")
		}
		v = normalized
	}
	return validate(fields(v), kind)
}

func validate(attrs map[string]any, kind string) error {
	if len(attrs) == 0 {
		return fail("invalid_input", "at least one changed field is required")
	}
	for k, v := range attrs {
		switch k {
		case "title", "name":
			if strings.TrimSpace(v.(string)) == "" {
				return fail("invalid_input", k+" cannot be empty")
			}
		case "ccli_number":
			if v.(int) <= 0 {
				return fail("invalid_input", "CCLI number must be positive")
			}
		case "bpm":
			n := v.(float64)
			if math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 {
				return fail("invalid_input", "BPM must be finite and positive")
			}
		case "isrc", "chord_chart_chord_color":
			if v.(int) < 0 {
				return fail("invalid_input", k+" cannot be negative")
			}
		case "chord_chart_columns":
			if v.(int) <= 0 {
				return fail("invalid_input", "chord chart columns must be positive")
			}
		case "chord_chart_font_size":
			valid := false
			for _, n := range []int{10, 11, 12, 13, 14, 15, 16, 18, 20, 22, 24, 26, 28, 32, 36, 42, 48} {
				if n == v.(int) {
					valid = true
				}
			}
			if !valid {
				return fail("invalid_input", "unsupported chord chart font size")
			}
		case "print_margin":
			if !oneOf(v.(string), "0.0in", "0.25in", "0.5in", "0.75in", "1.0in") {
				return fail("invalid_input", "unsupported print margin")
			}
		case "print_orientation":
			if !oneOf(v.(string), "Portrait", "Landscape") {
				return fail("invalid_input", "unsupported print orientation")
			}
		case "print_page_size":
			if !oneOf(v.(string), "Widescreen (16x9)", "Fullscreen (4x3)", "A4", "Letter", "Legal", "11x17") {
				return fail("invalid_input", "unsupported print page size")
			}
		case "alternate_keys":
			var keys []struct {
				Name string `json:"name"`
				Key  string `json:"key"`
			}
			if err := json.Unmarshal([]byte(v.(string)), &keys); err != nil || keys == nil {
				return fail("invalid_input", "alternate keys must be a JSON array of name/key objects")
			}
			for _, key := range keys {
				if strings.TrimSpace(key.Name) == "" || !ValidPitch(key.Key) {
					return fail("invalid_input", "alternate keys require a name and valid musical pitch")
				}
			}
		case "length":
			if v.(int) < 0 || v.(int) > 2147483647 {
				return fail("invalid_input", "length must be between 0 and 2147483647 seconds")
			}
		case "starting_key", "ending_key", "chord_chart_key":
			if !ValidPitch(v.(string)) {
				return fail("invalid_input", "invalid musical pitch for "+k)
			}
		case "meter":
			if !strings.Contains("|2/2|2/4|3/2|3/4|4/2|4/4|5/4|6/4|3/8|6/8|7/4|7/8|9/8|12/4|12/8|", "|"+v.(string)+"|") {
				return fail("invalid_input", "invalid meter")
			}
		case "sequence":
			for _, s := range v.([]string) {
				if strings.TrimSpace(s) == "" {
					return fail("invalid_input", "sequence cannot contain empty elements")
				}
			}
		}
	}
	return nil
}
func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func ValidPitch(s string) bool {
	base := strings.TrimSuffix(s, "m")
	switch base {
	case "Ab", "A", "A#", "Bb", "B", "C", "C#", "Db", "D", "D#", "Eb", "E", "F", "F#", "Gb", "G", "G#":
		return true
	}
	return false
}
func (l *Library) mutate(ctx context.Context, method, path, kind string, attrs map[string]any, skipVerify ...string) (record, error) {
	if e := validate(attrs, kind); e != nil {
		return record{}, e
	}
	r, e := l.api.mutate(ctx, method, path, kind, attrs)
	if e != nil {
		return record{}, e
	}
	if method == "PATCH" && r.id != path[strings.LastIndex(path, "/")+1:] {
		return record{}, fail("outcome_unknown", "write submitted but response identified a different resource")
	}
	readbackAttrs := make(map[string]any)
	for k, value := range attrs {
		skip := strings.HasPrefix(k, "import_")
		for _, field := range skipVerify {
			if k == field {
				skip = true
			}
		}
		if !skip {
			readbackAttrs[k] = value
		}
	}
	missing := false
	for k := range readbackAttrs {
		if _, ok := r.attrs[k]; !ok {
			missing = true
		}
	}
	if missing {
		r, e = l.api.get(ctx, pathForCreated(path, method, r.id), kind, r.id)
		if e != nil {
			return record{}, &Error{Code: "verification_failed", Message: "write submitted but readback failed", Cause: e}
		}
	}
	if e = verify(r, readbackAttrs); e != nil {
		return record{}, e
	}
	return r, nil
}
func pathForCreated(path, method, id string) string {
	if method == "POST" {
		return path + "/" + id
	}
	return path
}
func normalizeSongCreate(m SongCreate) SongCreate {
	if m.Title == "" && m.CCLINumber != nil {
		m.Title = strconv.Itoa(*m.CCLINumber)
	}
	return m
}
func (l *Library) CreateSong(ctx context.Context, m SongCreate) (Song, error) {
	m = normalizeSongCreate(m)
	if err := ValidateMutation(m, "Song"); err != nil {
		return Song{}, err
	}
	// Numeric title is the documented CCLI metadata request, not the expected
	// resulting song title. The resolved title must remain visible to the caller.
	skip := []string{}
	if m.CCLINumber != nil && m.Title == strconv.Itoa(*m.CCLINumber) {
		skip = append(skip, "title")
	}
	r, e := l.mutate(ctx, "POST", "/services/v2/songs", "Song", fields(m), skip...)
	var out Song
	if e == nil {
		e = r.model(&out)
	}
	return out, e
}
func (l *Library) UpdateSong(ctx context.Context, target string, v Visibility, m SongPatch) (Song, error) {
	if err := ValidateMutation(m, "Song"); err != nil {
		return Song{}, err
	}
	a := fields(m)
	if e := validate(a, "Song"); e != nil {
		return Song{}, e
	}
	r, e := l.song(ctx, target, v, true)
	if e != nil {
		return Song{}, e
	}
	r, e = l.mutate(ctx, "PATCH", songPath(r.id), "Song", a)
	var out Song
	if e == nil {
		e = r.model(&out)
	}
	return out, e
}
func (l *Library) DeleteSong(ctx context.Context, target string, v Visibility) error {
	r, e := l.song(ctx, target, v, true)
	if e != nil {
		return e
	}
	return l.api.http.Delete(ctx, songPath(r.id), nil)
}
func (l *Library) CreateArrangement(ctx context.Context, s Scope, m ArrangementCreate) (Arrangement, error) {
	if err := ValidateMutation(m, "Arrangement"); err != nil {
		return Arrangement{}, err
	}
	a := fields(m)
	if e := validate(a, "Arrangement"); e != nil {
		return Arrangement{}, e
	}
	p, e := l.parents(ctx, s, false, true)
	if e != nil {
		return Arrangement{}, e
	}
	r, e := l.mutate(ctx, "POST", songPath(p.song.id)+"/arrangements", "Arrangement", a)
	var out Arrangement
	if e == nil {
		e = r.model(&out)
	}
	return out, e
}
func (l *Library) UpdateArrangement(ctx context.Context, s Scope, target string, m ArrangementPatch) (Arrangement, error) {
	if err := ValidateMutation(m, "Arrangement"); err != nil {
		return Arrangement{}, err
	}
	a := fields(m)
	if e := validate(a, "Arrangement"); e != nil {
		return Arrangement{}, e
	}
	s.Arrangement = target
	p, e := l.parents(ctx, s, true, true)
	if e != nil {
		return Arrangement{}, e
	}
	r, e := l.mutate(ctx, "PATCH", arrangementsPath(p.song.id, s.IncludeArchived)+"/"+p.arr.id, "Arrangement", a)
	var out Arrangement
	if e == nil {
		e = r.model(&out)
	}
	return out, e
}
func (l *Library) DeleteArrangement(ctx context.Context, s Scope, target string) error {
	s.Arrangement = target
	p, e := l.parents(ctx, s, true, true)
	if e != nil {
		return e
	}
	return l.api.http.Delete(ctx, arrangementsPath(p.song.id, s.IncludeArchived)+"/"+p.arr.id, nil)
}
func (l *Library) CreateKey(ctx context.Context, s Scope, m KeyCreate) (Key, error) {
	if err := ValidateMutation(m, "Key"); err != nil {
		return Key{}, err
	}
	a := fields(m)
	if e := validate(a, "Key"); e != nil {
		return Key{}, e
	}
	p, e := l.parents(ctx, s, true, true)
	if e != nil {
		return Key{}, e
	}
	r, e := l.mutate(ctx, "POST", arrangementPath(p.song.id, p.arr.id)+"/keys", "Key", a)
	var out Key
	if e == nil {
		e = r.model(&out)
	}
	return out, e
}
func (l *Library) UpdateKey(ctx context.Context, s Scope, target string, m KeyPatch) (Key, error) {
	if err := ValidateMutation(m, "Key"); err != nil {
		return Key{}, err
	}
	a := fields(m)
	if e := validate(a, "Key"); e != nil {
		return Key{}, e
	}
	p, e := l.parents(ctx, s, true, true)
	if e != nil {
		return Key{}, e
	}
	r, e := l.key(ctx, p, target, true)
	if e != nil {
		return Key{}, e
	}
	r, e = l.mutate(ctx, "PATCH", keyPath(p.song.id, p.arr.id, r.id), "Key", a)
	var out Key
	if e == nil {
		e = r.model(&out)
	}
	return out, e
}
func (l *Library) DeleteKey(ctx context.Context, s Scope, target string) error {
	p, e := l.parents(ctx, s, true, true)
	if e != nil {
		return e
	}
	r, e := l.key(ctx, p, target, true)
	if e != nil {
		return e
	}
	return l.api.http.Delete(ctx, keyPath(p.song.id, p.arr.id, r.id), nil)
}
