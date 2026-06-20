package services

import (
	"context"
	"fmt"
	"net/url"
)

type API interface {
	Get(ctx context.Context, path string, query url.Values, dst any) error
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
	Author           string `json:"author,omitempty"`
	CCLINumber       int    `json:"ccli_number,omitempty"`
	PrimaryKey       string `json:"primary_key,omitempty"`
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

type SongAttributes struct {
	Title      string `json:"title"`
	Author     string `json:"author"`
	CCLINumber int    `json:"ccli_number"`
	PrimaryKey string `json:"primary_key"`
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

type ListSongsOptions struct {
	Title      string
	Author     string
	CCLINumber int
	PerPage    int
}

type ListArrangementsOptions struct {
	PerPage int
}

func (l *Library) ListSongs(ctx context.Context, opts ListSongsOptions) ([]Song, error) {
	query := url.Values{}
	if opts.Title != "" {
		query.Set("where[title]", opts.Title)
	}
	if opts.Author != "" {
		query.Set("where[author]", opts.Author)
	}
	if opts.CCLINumber != 0 {
		query.Set("where[ccli_number]", fmt.Sprintf("%d", opts.CCLINumber))
	}
	setPerPage(query, opts.PerPage)

	var response collection[SongAttributes]
	if err := l.api.Get(ctx, "/services/v2/songs", query, &response); err != nil {
		return nil, err
	}

	songs := make([]Song, 0, len(response.Data))
	for _, item := range response.Data {
		songs = append(songs, Song{
			ID:         item.ID,
			Title:      item.Attributes.Title,
			Author:     item.Attributes.Author,
			CCLINumber: item.Attributes.CCLINumber,
			PrimaryKey: item.Attributes.PrimaryKey,
		})
	}
	return songs, nil
}

func (l *Library) GetSong(ctx context.Context, songID string) (Song, error) {
	var response single[SongAttributes]
	if err := l.api.Get(ctx, "/services/v2/songs/"+url.PathEscape(songID), nil, &response); err != nil {
		return Song{}, err
	}
	return Song{
		ID:         response.Data.ID,
		Title:      response.Data.Attributes.Title,
		Author:     response.Data.Attributes.Author,
		CCLINumber: response.Data.Attributes.CCLINumber,
		PrimaryKey: response.Data.Attributes.PrimaryKey,
	}, nil
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
		arrangements = append(arrangements, Arrangement{
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
		})
	}
	return arrangements, nil
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

func setPerPage(query url.Values, perPage int) {
	if perPage > 0 {
		query.Set("per_page", fmt.Sprintf("%d", perPage))
	}
}
