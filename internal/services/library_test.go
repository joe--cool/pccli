package services

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

type fakeAPI struct {
	path  string
	query url.Values
	body  string
}

func (api *fakeAPI) Get(ctx context.Context, path string, query url.Values, dst any) error {
	api.path = path
	api.query = query
	return json.Unmarshal([]byte(api.body), dst)
}

func TestListSongsBuildsPlanningCenterQuery(t *testing.T) {
	api := &fakeAPI{body: `{"data":[{"id":"1001","type":"Song","attributes":{"title":"Amazing Grace","author":"John Newton","ccli_number":22025,"primary_key":"G"}}]}`}
	library := NewLibrary(api)

	songs, err := library.ListSongs(context.Background(), ListSongsOptions{
		Title:      "Amazing%",
		Author:     "Newton",
		CCLINumber: 22025,
		PerPage:    10,
	})
	if err != nil {
		t.Fatalf("ListSongs returned error: %v", err)
	}
	if api.path != "/services/v2/songs" {
		t.Fatalf("unexpected path: %s", api.path)
	}
	if got := api.query.Get("where[title]"); got != "Amazing%" {
		t.Fatalf("unexpected title query: %q", got)
	}
	if got := api.query.Get("where[author]"); got != "Newton" {
		t.Fatalf("unexpected author query: %q", got)
	}
	if got := api.query.Get("where[ccli_number]"); got != "22025" {
		t.Fatalf("unexpected ccli query: %q", got)
	}
	if got := api.query.Get("per_page"); got != "10" {
		t.Fatalf("unexpected per_page query: %q", got)
	}
	if len(songs) != 1 || songs[0].Title != "Amazing Grace" {
		t.Fatalf("unexpected songs: %#v", songs)
	}
}

func TestListArrangementsMapsAttributes(t *testing.T) {
	api := &fakeAPI{body: `{"data":[{"id":"2001","type":"Arrangement","attributes":{"name":"Full Band","bpm":72,"meter":"3/4","length":255,"chord_chart_key":"G","has_chord_chart":true,"has_chords":true,"lyrics_enabled":true,"sequence_short":["V1","V2"]}}]}`}
	library := NewLibrary(api)

	arrangements, err := library.ListArrangements(context.Background(), "1001", ListArrangementsOptions{PerPage: 5})
	if err != nil {
		t.Fatalf("ListArrangements returned error: %v", err)
	}
	if api.path != "/services/v2/songs/1001/arrangements" {
		t.Fatalf("unexpected path: %s", api.path)
	}
	if got := api.query.Get("per_page"); got != "5" {
		t.Fatalf("unexpected per_page query: %q", got)
	}
	if len(arrangements) != 1 || arrangements[0].ChordChartKey != "G" || arrangements[0].SequenceShort[1] != "V2" {
		t.Fatalf("unexpected arrangements: %#v", arrangements)
	}
}
