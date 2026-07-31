package services

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

type fakeAPI struct {
	path        string
	query       url.Values
	body        string
	requestBody any
	uploadPath  string
}

func (api *fakeAPI) Get(ctx context.Context, path string, query url.Values, dst any) error {
	api.path = path
	api.query = query
	return json.Unmarshal([]byte(api.body), dst)
}

func (api *fakeAPI) Post(ctx context.Context, path string, query url.Values, body any, dst any) error {
	api.path = path
	api.query = query
	api.requestBody = body
	return json.Unmarshal([]byte(api.body), dst)
}

func (api *fakeAPI) Patch(ctx context.Context, path string, query url.Values, body any, dst any) error {
	api.path = path
	api.query = query
	api.requestBody = body
	return json.Unmarshal([]byte(api.body), dst)
}

func (api *fakeAPI) Delete(ctx context.Context, path string, query url.Values) error {
	api.path = path
	api.query = query
	return nil
}

func (api *fakeAPI) UploadFile(ctx context.Context, path string, dst any) error {
	api.uploadPath = path
	return json.Unmarshal([]byte(`{"data":[{"id":"us1-file","type":"File","attributes":{"name":"chart.pdf","content_type":"application/pdf","file_size":1234}}]}`), dst)
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

func TestListSongsSearchUsesContainsWildcard(t *testing.T) {
	api := &fakeAPI{body: `{"data":[]}`}
	library := NewLibrary(api)

	_, err := library.ListSongs(context.Background(), ListSongsOptions{Search: "Grace"})
	if err != nil {
		t.Fatalf("ListSongs returned error: %v", err)
	}
	if got := api.query.Get("where[title]"); got != "%Grace%" {
		t.Fatalf("unexpected search query: %q", got)
	}
}

func TestListSongsCanFilterVisibleSongs(t *testing.T) {
	api := &fakeAPI{body: `{"data":[]}`}
	library := NewLibrary(api)
	visible := false

	_, err := library.ListSongs(context.Background(), ListSongsOptions{Hidden: &visible})
	if err != nil {
		t.Fatalf("ListSongs returned error: %v", err)
	}
	if got := api.query.Get("where[hidden]"); got != "false" {
		t.Fatalf("unexpected hidden query: %q", got)
	}
}

func TestCreateSongBuildsJSONAPIPayload(t *testing.T) {
	api := &fakeAPI{body: `{"data":{"id":"1003","type":"Song","attributes":{"title":"New Song","author":"Writer","ccli_number":12345}}}`}
	library := NewLibrary(api)

	song, err := library.CreateSong(context.Background(), SongMutation{
		Title:      "New Song",
		Author:     "Writer",
		CCLINumber: 12345,
	})
	if err != nil {
		t.Fatalf("CreateSong returned error: %v", err)
	}
	if api.path != "/services/v2/songs" {
		t.Fatalf("unexpected path: %s", api.path)
	}
	payload, ok := api.requestBody.(mutationPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", api.requestBody)
	}
	if payload.Data.Type != "Song" || payload.Data.Attributes["title"] != "New Song" || payload.Data.Attributes["ccli_number"] != 12345 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if song.ID != "1003" || song.Title != "New Song" {
		t.Fatalf("unexpected song: %#v", song)
	}
}

func TestDeleteSongUsesSongPath(t *testing.T) {
	api := &fakeAPI{}
	library := NewLibrary(api)

	if err := library.DeleteSong(context.Background(), "1001"); err != nil {
		t.Fatalf("DeleteSong returned error: %v", err)
	}
	if api.path != "/services/v2/songs/1001" {
		t.Fatalf("unexpected path: %s", api.path)
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

func TestCreateArrangementBuildsJSONAPIPayload(t *testing.T) {
	api := &fakeAPI{body: `{"data":{"id":"2003","type":"Arrangement","attributes":{"name":"Choir","bpm":72,"meter":"3/4","length":255,"chord_chart_key":"G","lyrics_enabled":true,"sequence_short":["V1","V2"]}}}`}
	library := NewLibrary(api)
	lyricsEnabled := true

	arrangement, err := library.CreateArrangement(context.Background(), "1001", ArrangementMutation{
		Name:          "Choir",
		BPM:           72,
		Meter:         "3/4",
		Length:        255,
		ChordChartKey: "G",
		LyricsEnabled: &lyricsEnabled,
		Sequence:      []string{"V1", "V2"},
	})
	if err != nil {
		t.Fatalf("CreateArrangement returned error: %v", err)
	}
	if api.path != "/services/v2/songs/1001/arrangements" {
		t.Fatalf("unexpected path: %s", api.path)
	}
	payload, ok := api.requestBody.(mutationPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", api.requestBody)
	}
	if payload.Data.Type != "Arrangement" || payload.Data.Attributes["name"] != "Choir" || payload.Data.Attributes["length"] != 255 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if sequence, ok := payload.Data.Attributes["sequence"].([]string); !ok || sequence[1] != "V2" {
		t.Fatalf("unexpected sequence payload: %#v", payload.Data.Attributes["sequence"])
	}
	if arrangement.ID != "2003" || arrangement.Name != "Choir" {
		t.Fatalf("unexpected arrangement: %#v", arrangement)
	}
}

func TestListKeysMapsAttributes(t *testing.T) {
	api := &fakeAPI{body: `{"data":[{"id":"3001","type":"Key","attributes":{"name":"Default","starting_key":"G","ending_key":"G"}}]}`}
	library := NewLibrary(api)

	keys, err := library.ListKeys(context.Background(), "1001", "2001", ListKeysOptions{PerPage: 5})
	if err != nil {
		t.Fatalf("ListKeys returned error: %v", err)
	}
	if api.path != "/services/v2/songs/1001/arrangements/2001/keys" {
		t.Fatalf("unexpected path: %s", api.path)
	}
	if got := api.query.Get("per_page"); got != "5" {
		t.Fatalf("unexpected per_page query: %q", got)
	}
	if len(keys) != 1 || keys[0].StartingKey != "G" {
		t.Fatalf("unexpected keys: %#v", keys)
	}
}

func TestCreateKeyBuildsJSONAPIPayload(t *testing.T) {
	api := &fakeAPI{body: `{"data":{"id":"3003","type":"Key","attributes":{"name":"Tenor Lead","starting_key":"A","ending_key":"A"}}}`}
	library := NewLibrary(api)

	key, err := library.CreateKey(context.Background(), "1001", "2001", KeyMutation{
		Name:        "Tenor Lead",
		StartingKey: "A",
		EndingKey:   "A",
	})
	if err != nil {
		t.Fatalf("CreateKey returned error: %v", err)
	}
	if api.path != "/services/v2/songs/1001/arrangements/2001/keys" {
		t.Fatalf("unexpected path: %s", api.path)
	}
	payload, ok := api.requestBody.(mutationPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", api.requestBody)
	}
	if payload.Data.Type != "Key" || payload.Data.Attributes["name"] != "Tenor Lead" || payload.Data.Attributes["starting_key"] != "A" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if key.ID != "3003" || key.StartingKey != "A" {
		t.Fatalf("unexpected key: %#v", key)
	}
}

func TestListAttachmentsUsesKeyScopedPath(t *testing.T) {
	api := &fakeAPI{body: `{"data":[{"id":"4001","type":"Attachment","attributes":{"display_name":"Chart","filename":"chart.pdf","content_type":"application/pdf","file_size":1234,"downloadable":true}}]}`}
	library := NewLibrary(api)

	attachments, err := library.ListAttachments(context.Background(), AttachmentScope{
		SongID:        "1001",
		ArrangementID: "2001",
		KeyID:         "3001",
	}, ListAttachmentsOptions{Filename: "chart", PerPage: 5})
	if err != nil {
		t.Fatalf("ListAttachments returned error: %v", err)
	}
	if api.path != "/services/v2/songs/1001/arrangements/2001/keys/3001/attachments" {
		t.Fatalf("unexpected path: %s", api.path)
	}
	if got := api.query.Get("where[filename_like]"); got != "chart" {
		t.Fatalf("unexpected filename query: %q", got)
	}
	if len(attachments) != 1 || attachments[0].Filename != "chart.pdf" {
		t.Fatalf("unexpected attachments: %#v", attachments)
	}
}

func TestAddAttachmentUploadsLocalFileFirst(t *testing.T) {
	api := &fakeAPI{body: `{"data":{"id":"4001","type":"Attachment","attributes":{"display_name":"Chart","filename":"chart.pdf","file_upload_identifier":"us1-file"}}}`}
	library := NewLibrary(api)

	attachment, err := library.AddAttachment(context.Background(), AttachmentScope{SongID: "1001"}, AttachmentMutation{FilePath: "chart.pdf"})
	if err != nil {
		t.Fatalf("AddAttachment returned error: %v", err)
	}
	if api.uploadPath != "chart.pdf" {
		t.Fatalf("unexpected upload path: %q", api.uploadPath)
	}
	if api.path != "/services/v2/songs/1001/attachments" {
		t.Fatalf("unexpected path: %s", api.path)
	}
	payload, ok := api.requestBody.(mutationPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", api.requestBody)
	}
	if payload.Data.Attributes["file_upload_identifier"] != "us1-file" || payload.Data.Attributes["filename"] != "chart.pdf" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if attachment.FileUploadID != "us1-file" {
		t.Fatalf("unexpected attachment: %#v", attachment)
	}
}
