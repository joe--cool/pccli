package services

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestExtendedArrangementFieldsPreserveExplicitValues(t *testing.T) {
	m := ArrangementPatch{
		ChordChart: ptr(""), ChordChartChordColor: ptr(0), ChordChartColumns: ptr(2),
		ChordChartFont: ptr("Arial"), ChordChartFontSize: ptr(12), ISRC: ptr(123),
		MTID: ptr("mix"), NumberChartEnabled: ptr(false), NumeralChartEnabled: ptr(false),
		PrintMargin: ptr("0.0in"), PrintOrientation: ptr("Landscape"), PrintPageSize: ptr("A4"),
		RehearsalMixID: ptr("mix"), RehearsalMixThumbnail: ptr(""),
		ImportChartPro: ptr("chart"), ImportChartProOptions: ptr("{}"),
		ImportRehearsalMix: ptr("mix"), ImportRehearsalMixOptions: ptr("{}"),
		ImportRehearsalPack: ptr("pack"), ImportRehearsalPackOptions: ptr("{}"),
	}
	if err := ValidateMutation(m, "Arrangement"); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"chord_chart": "", "chord_chart_chord_color": 0, "chord_chart_columns": 2,
		"chord_chart_font": "Arial", "chord_chart_font_size": 12, "isrc": 123,
		"mtid": "mix", "number_chart_enabled": false, "numeral_chart_enabled": false,
		"print_margin": "0.0in", "print_orientation": "Landscape", "print_page_size": "A4",
		"rehearsal_mix_id": "mix", "rehearsal_mix_thumbnail": "",
		"import_chart_pro": "chart", "import_chart_pro_options": "{}",
		"import_rehearsal_mix": "mix", "import_rehearsal_mix_options": "{}",
		"import_rehearsal_pack": "pack", "import_rehearsal_pack_options": "{}",
	}
	if got := fields(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("fields = %#v, want %#v", got, want)
	}
}

func TestCreateOnlyImportFields(t *testing.T) {
	for _, m := range []any{
		SongCreate{Title: "Song", ImportCCLIChordChart: ptr("chart"), ImportCCLIChordPro: ptr("pro"), ImportCCLILyrics: ptr("lyrics")},
		ArrangementCreate{Name: "Arrangement", ImportCCLIChordChart: ptr("chart"), ImportCCLIChordPro: ptr("pro"), ImportCCLILyrics: ptr("lyrics")},
	} {
		got := fields(m)
		for _, key := range []string{"import_ccli_chord_chart", "import_ccli_chord_pro", "import_ccli_lyrics"} {
			if _, ok := got[key]; !ok {
				t.Fatalf("%T missing %s", m, key)
			}
		}
	}
}

func TestExtendedKeyFieldsUseStringContract(t *testing.T) {
	alternates := `[{"name":"Female lead","key":"Eb"}]`
	m := KeyPatch{AlternateKeys: &alternates, ImportCCLIChordChart: ptr("chart"), ImportCCLIChordChartOptions: ptr("{}"), ImportCCLILyrics: ptr("lyrics"), ImportChartPro: ptr("pro"), ImportRehearsalMix: ptr("mix"), ImportRehearsalPack: ptr("pack")}
	if err := ValidateMutation(m, "Key"); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(fields(m))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["alternate_keys"] != alternates || len(got) != 7 {
		t.Fatalf("wrong wire attributes: %s", data)
	}
}

func TestMutationEncodingFailuresDoNotReachNetwork(t *testing.T) {
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		f := &fixture{t: t, handle: func(request) (string, error) { t.Fatal("invalid input reached network"); return "", nil }}
		l := NewLibrary(f)
		_, err := l.CreateArrangement(context.Background(), Scope{Song: "1"}, ArrangementCreate{Name: "Mixed fields", BPM: &bad, Notes: ptr("valid")})
		assertCode(t, err, "invalid_input")
		_, err = l.UpdateArrangement(context.Background(), Scope{Song: "1"}, "2", ArrangementPatch{BPM: &bad, Notes: ptr("valid")})
		assertCode(t, err, "invalid_input")
		if len(f.requests) != 0 {
			t.Fatal("expected zero requests")
		}
	}
}

func TestExtendedFieldValidation(t *testing.T) {
	for _, m := range []ArrangementPatch{
		{ChordChartFontSize: ptr(17)}, {ChordChartColumns: ptr(0)}, {ISRC: ptr(-1)},
		{PrintMargin: ptr("2in")}, {PrintOrientation: ptr("Sideways")}, {PrintPageSize: ptr("Unknown")},
	} {
		assertCode(t, ValidateMutation(m, "Arrangement"), "invalid_input")
	}
	for _, v := range []string{"", "null", `{}`, `[{"name":"Lead","key":"H"}]`} {
		assertCode(t, ValidateMutation(KeyPatch{AlternateKeys: &v}, "Key"), "invalid_input")
	}
	assertCode(t, ValidateMutation(SongPatch{Author: ptr("valid")}, "Key"), "invalid_input")
}

func TestReadableArrangementMetadata(t *testing.T) {
	var got Arrangement
	err := json.Unmarshal([]byte(`{"chord_chart":"[C]Words","chord_chart_font_size":12,"number_chart_enabled":true,"print_page_size":"Letter","isrc":123,"rehearsal_mix_id":"mix","sequence_full":["Verse 1"],"created_at":"2026-01-01"}`), &got)
	if err != nil {
		t.Fatal(err)
	}
	if got.ChordChart != "[C]Words" || got.ChordChartFontSize != 12 || !got.NumberChartEnabled || got.ISRC != 123 || got.RehearsalMixID != "mix" || len(got.SequenceFull) != 1 || !strings.HasPrefix(got.CreatedAt, "2026") {
		t.Fatalf("missing metadata: %#v", got)
	}
}

func TestCCLIMetadataImportAcceptsResolvedTitle(t *testing.T) {
	number := 22025
	f := &fixture{t: t, handle: func(r request) (string, error) {
		if r.method != "POST" || r.path != "/services/v2/songs" {
			t.Fatalf("unexpected request %#v", r)
		}
		return `{"data":{"id":"1","type":"Song","attributes":{"title":"Amazing Grace","ccli_number":22025,"hidden":false}}}`, nil
	}}
	song, err := NewLibrary(f).CreateSong(context.Background(), SongCreate{CCLINumber: &number})
	if err != nil || song.Title != "Amazing Grace" {
		t.Fatalf("metadata import: %#v %v", song, err)
	}
	assertCode(t, ValidateMutation(SongCreate{Title: "22025"}, "Song"), "invalid_input")
}
