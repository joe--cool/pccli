package services

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
)

type API interface {
	Get(ctx context.Context, path string, query url.Values, dst any) error
	Post(ctx context.Context, path string, query url.Values, body any, dst any) error
	Patch(ctx context.Context, path string, query url.Values, body any, dst any) error
	Delete(ctx context.Context, path string, query url.Values) error
	UploadFile(ctx context.Context, path string, dst any) error
}

type Library struct {
	api API
}

func NewLibrary(api API) *Library {
	return &Library{api: api}
}

type Song struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Admin            string `json:"admin,omitempty"`
	Author           string `json:"author,omitempty"`
	CCLINumber       int    `json:"ccli_number,omitempty"`
	Copyright        string `json:"copyright,omitempty"`
	Hidden           bool   `json:"hidden,omitempty"`
	LastScheduledAt  string `json:"last_scheduled_at,omitempty"`
	LastScheduled    string `json:"last_scheduled,omitempty"`
	Notes            string `json:"notes,omitempty"`
	PrimaryKey       string `json:"primary_key,omitempty"`
	Themes           string `json:"themes,omitempty"`
	ArrangementCount int    `json:"arrangement_count,omitempty"`
}

type Arrangement struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	BPM           float64  `json:"bpm,omitempty"`
	Meter         string   `json:"meter,omitempty"`
	Length        int      `json:"length,omitempty"`
	ChordChartKey string   `json:"chord_chart_key,omitempty"`
	HasChordChart bool     `json:"has_chord_chart,omitempty"`
	HasChords     bool     `json:"has_chords,omitempty"`
	LyricsEnabled bool     `json:"lyrics_enabled,omitempty"`
	SequenceShort []string `json:"sequence_short,omitempty"`
}

type Key struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	StartingKey string `json:"starting_key,omitempty"`
	EndingKey   string `json:"ending_key,omitempty"`
}

type Attachment struct {
	ID                string `json:"id"`
	DisplayName       string `json:"display_name,omitempty"`
	Filename          string `json:"filename,omitempty"`
	ContentType       string `json:"content_type,omitempty"`
	FileSize          int    `json:"file_size,omitempty"`
	FileType          string `json:"filetype,omitempty"`
	PCOType           string `json:"pco_type,omitempty"`
	RemoteLink        string `json:"remote_link,omitempty"`
	URL               string `json:"url,omitempty"`
	ThumbnailURL      string `json:"thumbnail_url,omitempty"`
	Downloadable      bool   `json:"downloadable,omitempty"`
	Streamable        bool   `json:"streamable,omitempty"`
	Transposable      bool   `json:"transposable,omitempty"`
	LicensesPurchased int    `json:"licenses_purchased,omitempty"`
	LicensesRemaining int    `json:"licenses_remaining,omitempty"`
	FileUploadID      string `json:"file_upload_identifier,omitempty"`
	AttachableType    string `json:"attachable_type,omitempty"`
}

type SongAttributes struct {
	Title           string `json:"title"`
	Admin           string `json:"admin"`
	Author          string `json:"author"`
	CCLINumber      int    `json:"ccli_number"`
	Copyright       string `json:"copyright"`
	Hidden          bool   `json:"hidden"`
	LastScheduledAt string `json:"last_scheduled_at"`
	LastScheduled   string `json:"last_scheduled_short_dates"`
	Notes           string `json:"notes"`
	PrimaryKey      string `json:"primary_key"`
	Themes          string `json:"themes"`
}

type ArrangementAttributes struct {
	Name          string   `json:"name"`
	BPM           float64  `json:"bpm"`
	Meter         string   `json:"meter"`
	Length        int      `json:"length"`
	ChordChartKey string   `json:"chord_chart_key"`
	HasChordChart bool     `json:"has_chord_chart"`
	HasChords     bool     `json:"has_chords"`
	LyricsEnabled bool     `json:"lyrics_enabled"`
	SequenceShort []string `json:"sequence_short"`
}

type KeyAttributes struct {
	Name        string `json:"name"`
	StartingKey string `json:"starting_key"`
	EndingKey   string `json:"ending_key"`
}

type AttachmentAttributes struct {
	AttachableType       string `json:"attachable_type"`
	ContentType          string `json:"content_type"`
	DisplayName          string `json:"display_name"`
	Downloadable         bool   `json:"downloadable"`
	FileSize             int    `json:"file_size"`
	FileUploadIdentifier string `json:"file_upload_identifier"`
	Filename             string `json:"filename"`
	FileType             string `json:"filetype"`
	LicensesPurchased    int    `json:"licenses_purchased"`
	LicensesRemaining    int    `json:"licenses_remaining"`
	PCOType              string `json:"pco_type"`
	RemoteLink           string `json:"remote_link"`
	Streamable           bool   `json:"streamable"`
	ThumbnailURL         string `json:"thumbnail_url"`
	Transposable         bool   `json:"transposable"`
	URL                  string `json:"url"`
}

type ListSongsOptions struct {
	Title      string
	Search     string
	Author     string
	CCLINumber int
	Hidden     *bool
	Key        string
	Meter      string
	Themes     string
	Order      string
	PerPage    int
}

type ListArrangementsOptions struct {
	PerPage int
}

type ListKeysOptions struct {
	PerPage int
}

type AttachmentScope struct {
	SongID        string
	ArrangementID string
	KeyID         string
}

type ListAttachmentsOptions struct {
	Filename string
	Type     string
	PerPage  int
}

type SongMutation struct {
	Title              string
	Admin              string
	Author             string
	Copyright          string
	CCLINumber         int
	Hidden             *bool
	Themes             string
	ImportCCLIChart    string
	ImportCCLIChordPro string
	ImportCCLILyrics   string
}

type ArrangementMutation struct {
	Name          string
	BPM           float64
	Meter         string
	Length        int
	ChordChartKey string
	Notes         string
	LyricsEnabled *bool
	Sequence      []string
}

type KeyMutation struct {
	Name        string
	StartingKey string
	EndingKey   string
}

type AttachmentMutation struct {
	FilePath            string
	FileUploadID        string
	Filename            string
	RemoteLink          string
	Content             string
	AttachmentTypeIDs   string
	ImportToItemDetails *bool
	PageOrder           string
	SongPart            string
}

func (l *Library) ListSongs(ctx context.Context, opts ListSongsOptions) ([]Song, error) {
	query := url.Values{}
	switch {
	case opts.Search != "":
		query.Set("where[title]", wildcard(opts.Search))
	case opts.Title != "":
		query.Set("where[title]", opts.Title)
	}
	if opts.Author != "" {
		query.Set("where[author]", opts.Author)
	}
	if opts.CCLINumber != 0 {
		query.Set("where[ccli_number]", fmt.Sprintf("%d", opts.CCLINumber))
	}
	if opts.Hidden != nil {
		query.Set("where[hidden]", fmt.Sprintf("%t", *opts.Hidden))
	}
	if opts.Key != "" {
		query.Set("where[key_name]", opts.Key)
	}
	if opts.Meter != "" {
		query.Set("where[meter]", opts.Meter)
	}
	if opts.Themes != "" {
		query.Set("where[themes]", opts.Themes)
	}
	if opts.Order != "" {
		query.Set("order", opts.Order)
	}
	setPerPage(query, opts.PerPage)

	var response collection[SongAttributes]
	if err := l.api.Get(ctx, "/services/v2/songs", query, &response); err != nil {
		return nil, err
	}

	songs := make([]Song, 0, len(response.Data))
	for _, item := range response.Data {
		songs = append(songs, songFromResource(item))
	}
	return songs, nil
}

func (l *Library) GetSong(ctx context.Context, songID string) (Song, error) {
	var response single[SongAttributes]
	if err := l.api.Get(ctx, "/services/v2/songs/"+url.PathEscape(songID), nil, &response); err != nil {
		return Song{}, err
	}
	return songFromResource(response.Data), nil
}

func (l *Library) CreateSong(ctx context.Context, mutation SongMutation) (Song, error) {
	var response single[SongAttributes]
	if err := l.api.Post(ctx, "/services/v2/songs", nil, resourcePayload("Song", mutation.songAttributes()), &response); err != nil {
		return Song{}, err
	}
	return songFromResource(response.Data), nil
}

func (l *Library) UpdateSong(ctx context.Context, songID string, mutation SongMutation) (Song, error) {
	var response single[SongAttributes]
	if err := l.api.Patch(ctx, "/services/v2/songs/"+url.PathEscape(songID), nil, resourcePayload("Song", mutation.songAttributes()), &response); err != nil {
		return Song{}, err
	}
	return songFromResource(response.Data), nil
}

func (l *Library) DeleteSong(ctx context.Context, songID string) error {
	return l.api.Delete(ctx, "/services/v2/songs/"+url.PathEscape(songID), nil)
}

func (l *Library) ListArrangements(ctx context.Context, songID string, opts ListArrangementsOptions) ([]Arrangement, error) {
	query := url.Values{}
	setPerPage(query, opts.PerPage)

	path := fmt.Sprintf("/services/v2/songs/%s/arrangements", url.PathEscape(songID))
	var response collection[ArrangementAttributes]
	if err := l.api.Get(ctx, path, query, &response); err != nil {
		return nil, err
	}

	arrangements := make([]Arrangement, 0, len(response.Data))
	for _, item := range response.Data {
		arrangements = append(arrangements, arrangementFromResource(item))
	}
	return arrangements, nil
}

func (l *Library) GetArrangement(ctx context.Context, songID, arrangementID string) (Arrangement, error) {
	var response single[ArrangementAttributes]
	path := fmt.Sprintf("/services/v2/songs/%s/arrangements/%s", url.PathEscape(songID), url.PathEscape(arrangementID))
	if err := l.api.Get(ctx, path, nil, &response); err != nil {
		return Arrangement{}, err
	}
	return arrangementFromResource(response.Data), nil
}

func (l *Library) CreateArrangement(ctx context.Context, songID string, mutation ArrangementMutation) (Arrangement, error) {
	var response single[ArrangementAttributes]
	path := fmt.Sprintf("/services/v2/songs/%s/arrangements", url.PathEscape(songID))
	if err := l.api.Post(ctx, path, nil, resourcePayload("Arrangement", mutation.arrangementAttributes()), &response); err != nil {
		return Arrangement{}, err
	}
	return arrangementFromResource(response.Data), nil
}

func (l *Library) UpdateArrangement(ctx context.Context, songID, arrangementID string, mutation ArrangementMutation) (Arrangement, error) {
	var response single[ArrangementAttributes]
	path := fmt.Sprintf("/services/v2/songs/%s/arrangements/%s", url.PathEscape(songID), url.PathEscape(arrangementID))
	if err := l.api.Patch(ctx, path, nil, resourcePayload("Arrangement", mutation.arrangementAttributes()), &response); err != nil {
		return Arrangement{}, err
	}
	return arrangementFromResource(response.Data), nil
}

func (l *Library) DeleteArrangement(ctx context.Context, songID, arrangementID string) error {
	path := fmt.Sprintf("/services/v2/songs/%s/arrangements/%s", url.PathEscape(songID), url.PathEscape(arrangementID))
	return l.api.Delete(ctx, path, nil)
}

func (l *Library) ListKeys(ctx context.Context, songID, arrangementID string, opts ListKeysOptions) ([]Key, error) {
	query := url.Values{}
	setPerPage(query, opts.PerPage)

	path := fmt.Sprintf("/services/v2/songs/%s/arrangements/%s/keys", url.PathEscape(songID), url.PathEscape(arrangementID))
	var response collection[KeyAttributes]
	if err := l.api.Get(ctx, path, query, &response); err != nil {
		return nil, err
	}

	keys := make([]Key, 0, len(response.Data))
	for _, item := range response.Data {
		keys = append(keys, keyFromResource(item))
	}
	return keys, nil
}

func (l *Library) GetKey(ctx context.Context, songID, arrangementID, keyID string) (Key, error) {
	var response single[KeyAttributes]
	path := fmt.Sprintf("/services/v2/songs/%s/arrangements/%s/keys/%s", url.PathEscape(songID), url.PathEscape(arrangementID), url.PathEscape(keyID))
	if err := l.api.Get(ctx, path, nil, &response); err != nil {
		return Key{}, err
	}
	return keyFromResource(response.Data), nil
}

func (l *Library) CreateKey(ctx context.Context, songID, arrangementID string, mutation KeyMutation) (Key, error) {
	var response single[KeyAttributes]
	path := fmt.Sprintf("/services/v2/songs/%s/arrangements/%s/keys", url.PathEscape(songID), url.PathEscape(arrangementID))
	if err := l.api.Post(ctx, path, nil, resourcePayload("Key", mutation.keyAttributes()), &response); err != nil {
		return Key{}, err
	}
	return keyFromResource(response.Data), nil
}

func (l *Library) UpdateKey(ctx context.Context, songID, arrangementID, keyID string, mutation KeyMutation) (Key, error) {
	var response single[KeyAttributes]
	path := fmt.Sprintf("/services/v2/songs/%s/arrangements/%s/keys/%s", url.PathEscape(songID), url.PathEscape(arrangementID), url.PathEscape(keyID))
	if err := l.api.Patch(ctx, path, nil, resourcePayload("Key", mutation.keyAttributes()), &response); err != nil {
		return Key{}, err
	}
	return keyFromResource(response.Data), nil
}

func (l *Library) DeleteKey(ctx context.Context, songID, arrangementID, keyID string) error {
	path := fmt.Sprintf("/services/v2/songs/%s/arrangements/%s/keys/%s", url.PathEscape(songID), url.PathEscape(arrangementID), url.PathEscape(keyID))
	return l.api.Delete(ctx, path, nil)
}

func (l *Library) ListAttachments(ctx context.Context, scope AttachmentScope, opts ListAttachmentsOptions) ([]Attachment, error) {
	query := url.Values{}
	if opts.Filename != "" {
		query.Set("where[filename_like]", opts.Filename)
	}
	if opts.Type != "" {
		query.Set("where[type]", opts.Type)
	}
	setPerPage(query, opts.PerPage)

	var response collection[AttachmentAttributes]
	if err := l.api.Get(ctx, attachmentCollectionPath(scope), query, &response); err != nil {
		return nil, err
	}

	attachments := make([]Attachment, 0, len(response.Data))
	for _, item := range response.Data {
		attachments = append(attachments, attachmentFromResource(item))
	}
	return attachments, nil
}

func (l *Library) AddAttachment(ctx context.Context, scope AttachmentScope, mutation AttachmentMutation) (Attachment, error) {
	if mutation.FilePath != "" {
		upload, err := l.uploadFile(ctx, mutation.FilePath)
		if err != nil {
			return Attachment{}, err
		}
		mutation.FileUploadID = upload.ID
		if mutation.Filename == "" {
			mutation.Filename = upload.Name
		}
	}

	var response single[AttachmentAttributes]
	if err := l.api.Post(ctx, attachmentCollectionPath(scope), nil, resourcePayload("Attachment", mutation.attachmentAttributes()), &response); err != nil {
		return Attachment{}, err
	}
	return attachmentFromResource(response.Data), nil
}

func (l *Library) uploadFile(ctx context.Context, path string) (UploadedFile, error) {
	var response uploadResponse
	if err := l.api.UploadFile(ctx, path, &response); err != nil {
		return UploadedFile{}, err
	}
	if len(response.Data) == 0 {
		return UploadedFile{}, fmt.Errorf("Planning Center upload response did not include a file id")
	}
	file := response.Data[0]
	return UploadedFile{
		ID:          file.ID,
		Name:        file.Attributes.Name,
		ContentType: file.Attributes.ContentType,
		FileSize:    file.Attributes.FileSize,
	}, nil
}

type UploadedFile struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	FileSize    int    `json:"file_size,omitempty"`
}

type collection[T any] struct {
	Data []resource[T] `json:"data"`
}

type single[T any] struct {
	Data resource[T] `json:"data"`
}

type resource[T any] struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes T      `json:"attributes"`
}

type mutationPayload struct {
	Data mutationResource `json:"data"`
}

type mutationResource struct {
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes"`
}

type uploadResponse struct {
	Data []uploadResource `json:"data"`
}

type uploadResource struct {
	ID         string               `json:"id"`
	Type       string               `json:"type"`
	Attributes uploadFileAttributes `json:"attributes"`
}

type uploadFileAttributes struct {
	ContentType string `json:"content_type"`
	FileSize    int    `json:"file_size"`
	Name        string `json:"name"`
}

func resourcePayload(resourceType string, attrs map[string]any) mutationPayload {
	return mutationPayload{Data: mutationResource{Type: resourceType, Attributes: attrs}}
}

func (m SongMutation) songAttributes() map[string]any {
	attrs := map[string]any{}
	setString(attrs, "title", m.Title)
	setString(attrs, "admin", m.Admin)
	setString(attrs, "author", m.Author)
	setString(attrs, "copyright", m.Copyright)
	setInt(attrs, "ccli_number", m.CCLINumber)
	setBool(attrs, "hidden", m.Hidden)
	setString(attrs, "themes", m.Themes)
	setString(attrs, "import_ccli_chord_chart", m.ImportCCLIChart)
	setString(attrs, "import_ccli_chord_pro", m.ImportCCLIChordPro)
	setString(attrs, "import_ccli_lyrics", m.ImportCCLILyrics)
	return attrs
}

func (m ArrangementMutation) arrangementAttributes() map[string]any {
	attrs := map[string]any{}
	setString(attrs, "name", m.Name)
	setFloat(attrs, "bpm", m.BPM)
	setString(attrs, "meter", m.Meter)
	setInt(attrs, "length", m.Length)
	setString(attrs, "chord_chart_key", m.ChordChartKey)
	setString(attrs, "notes", m.Notes)
	setBool(attrs, "lyrics_enabled", m.LyricsEnabled)
	setStringSlice(attrs, "sequence", m.Sequence)
	return attrs
}

func (m KeyMutation) keyAttributes() map[string]any {
	attrs := map[string]any{}
	setString(attrs, "name", m.Name)
	setString(attrs, "starting_key", m.StartingKey)
	setString(attrs, "ending_key", m.EndingKey)
	return attrs
}

func (m AttachmentMutation) attachmentAttributes() map[string]any {
	attrs := map[string]any{}
	setString(attrs, "attachment_type_ids", m.AttachmentTypeIDs)
	setString(attrs, "content", m.Content)
	setString(attrs, "file_upload_identifier", m.FileUploadID)
	setString(attrs, "filename", m.Filename)
	setBool(attrs, "import_to_item_details", m.ImportToItemDetails)
	setString(attrs, "remote_link", m.RemoteLink)
	setString(attrs, "song_part", m.SongPart)
	setString(attrs, "page_order", m.PageOrder)
	return attrs
}

func songFromResource(item resource[SongAttributes]) Song {
	return Song{
		ID:              item.ID,
		Title:           item.Attributes.Title,
		Admin:           item.Attributes.Admin,
		Author:          item.Attributes.Author,
		CCLINumber:      item.Attributes.CCLINumber,
		Copyright:       item.Attributes.Copyright,
		Hidden:          item.Attributes.Hidden,
		LastScheduledAt: item.Attributes.LastScheduledAt,
		LastScheduled:   item.Attributes.LastScheduled,
		Notes:           item.Attributes.Notes,
		PrimaryKey:      item.Attributes.PrimaryKey,
		Themes:          item.Attributes.Themes,
	}
}

func arrangementFromResource(item resource[ArrangementAttributes]) Arrangement {
	return Arrangement{
		ID:            item.ID,
		Name:          item.Attributes.Name,
		BPM:           item.Attributes.BPM,
		Meter:         item.Attributes.Meter,
		Length:        item.Attributes.Length,
		ChordChartKey: item.Attributes.ChordChartKey,
		HasChordChart: item.Attributes.HasChordChart,
		HasChords:     item.Attributes.HasChords,
		LyricsEnabled: item.Attributes.LyricsEnabled,
		SequenceShort: item.Attributes.SequenceShort,
	}
}

func keyFromResource(item resource[KeyAttributes]) Key {
	return Key{
		ID:          item.ID,
		Name:        item.Attributes.Name,
		StartingKey: item.Attributes.StartingKey,
		EndingKey:   item.Attributes.EndingKey,
	}
}

func attachmentFromResource(item resource[AttachmentAttributes]) Attachment {
	return Attachment{
		ID:                item.ID,
		DisplayName:       item.Attributes.DisplayName,
		Filename:          item.Attributes.Filename,
		ContentType:       item.Attributes.ContentType,
		FileSize:          item.Attributes.FileSize,
		FileType:          item.Attributes.FileType,
		PCOType:           item.Attributes.PCOType,
		RemoteLink:        item.Attributes.RemoteLink,
		URL:               item.Attributes.URL,
		ThumbnailURL:      item.Attributes.ThumbnailURL,
		Downloadable:      item.Attributes.Downloadable,
		Streamable:        item.Attributes.Streamable,
		Transposable:      item.Attributes.Transposable,
		LicensesPurchased: item.Attributes.LicensesPurchased,
		LicensesRemaining: item.Attributes.LicensesRemaining,
		FileUploadID:      item.Attributes.FileUploadIdentifier,
		AttachableType:    item.Attributes.AttachableType,
	}
}

func attachmentCollectionPath(scope AttachmentScope) string {
	songID := url.PathEscape(scope.SongID)
	if scope.ArrangementID == "" {
		return fmt.Sprintf("/services/v2/songs/%s/attachments", songID)
	}
	arrangementID := url.PathEscape(scope.ArrangementID)
	if scope.KeyID == "" {
		return fmt.Sprintf("/services/v2/songs/%s/arrangements/%s/attachments", songID, arrangementID)
	}
	return fmt.Sprintf("/services/v2/songs/%s/arrangements/%s/keys/%s/attachments", songID, arrangementID, url.PathEscape(scope.KeyID))
}

func wildcard(value string) string {
	if value == "" || value[0] == '%' || value[len(value)-1] == '%' {
		return value
	}
	return "%" + value + "%"
}

func setString(attrs map[string]any, key, value string) {
	if value != "" {
		attrs[key] = value
	}
}

func setInt(attrs map[string]any, key string, value int) {
	if value != 0 {
		attrs[key] = value
	}
}

func setFloat(attrs map[string]any, key string, value float64) {
	if value != 0 {
		attrs[key] = value
	}
}

func setBool(attrs map[string]any, key string, value *bool) {
	if value != nil {
		attrs[key] = *value
	}
}

func setStringSlice(attrs map[string]any, key string, value []string) {
	if len(value) != 0 {
		attrs[key] = value
	}
}

func setPerPage(query url.Values, perPage int) {
	if perPage > 0 {
		query.Set("per_page", fmt.Sprintf("%d", perPage))
	}
}

func AttachmentNameFromPath(path string) string {
	return filepath.Base(path)
}
