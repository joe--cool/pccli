package services

type Song struct {
	ID                      string `json:"id"`
	Title                   string `json:"title"`
	Admin                   string `json:"admin"`
	Author                  string `json:"author"`
	CCLINumber              int    `json:"ccli_number"`
	Copyright               string `json:"copyright"`
	Hidden                  bool   `json:"hidden"`
	LastScheduledAt         string `json:"last_scheduled_at"`
	LastScheduled           string `json:"last_scheduled"`
	Notes                   string `json:"notes"`
	PrimaryKey              string `json:"primary_key"`
	Themes                  string `json:"themes"`
	ArrangementCount        int    `json:"arrangement_count"`
	CreatedAt               string `json:"created_at"`
	LastScheduledShortDates string `json:"last_scheduled_short_dates"`
	UpdatedAt               string `json:"updated_at"`
}

type Arrangement struct {
	SongID                string   `json:"song_id"`
	Notes                 string   `json:"notes"`
	Sequence              []string `json:"sequence"`
	ArchivedAt            string   `json:"archived_at"`
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	BPM                   float64  `json:"bpm"`
	Meter                 string   `json:"meter"`
	Length                int      `json:"length"`
	ChordChartKey         string   `json:"chord_chart_key"`
	HasChordChart         bool     `json:"has_chord_chart"`
	HasChords             bool     `json:"has_chords"`
	LyricsEnabled         bool     `json:"lyrics_enabled"`
	SequenceShort         []string `json:"sequence_short"`
	ChordChart            string   `json:"chord_chart"`
	ChordChartChordColor  int      `json:"chord_chart_chord_color"`
	ChordChartColumns     int      `json:"chord_chart_columns"`
	ChordChartFont        string   `json:"chord_chart_font"`
	ChordChartFontSize    int      `json:"chord_chart_font_size"`
	CreatedAt             string   `json:"created_at"`
	ISRC                  int      `json:"isrc"`
	Lyrics                string   `json:"lyrics"`
	MTID                  string   `json:"mtid"`
	NumberChartEnabled    bool     `json:"number_chart_enabled"`
	NumeralChartEnabled   bool     `json:"numeral_chart_enabled"`
	PrintMargin           string   `json:"print_margin"`
	PrintOrientation      string   `json:"print_orientation"`
	PrintPageSize         string   `json:"print_page_size"`
	RehearsalMixID        string   `json:"rehearsal_mix_id"`
	RehearsalMixThumbnail string   `json:"rehearsal_mix_thumbnail"`
	SequenceFull          []string `json:"sequence_full"`
	UpdatedAt             string   `json:"updated_at"`
}

type Key struct {
	SongID        string `json:"song_id"`
	ArrangementID string `json:"arrangement_id"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	StartingKey   string `json:"starting_key"`
	EndingKey     string `json:"ending_key"`
	AlternateKeys string `json:"alternate_keys"`
	CreatedAt     string `json:"created_at"`
	EndingMinor   bool   `json:"ending_minor"`
	StartingMinor bool   `json:"starting_minor"`
	UpdatedAt     string `json:"updated_at"`
}

type Attachment struct {
	SongID            string `json:"song_id"`
	ArrangementID     string `json:"arrangement_id"`
	KeyID             string `json:"key_id"`
	ID                string `json:"id"`
	DisplayName       string `json:"display_name"`
	Filename          string `json:"filename"`
	ContentType       string `json:"content_type"`
	FileSize          int    `json:"file_size"`
	FileType          string `json:"filetype"`
	PCOType           string `json:"pco_type"`
	RemoteLink        string `json:"remote_link"`
	URL               string `json:"url"`
	ThumbnailURL      string `json:"thumbnail_url"`
	Downloadable      bool   `json:"downloadable"`
	Streamable        bool   `json:"streamable"`
	Transposable      bool   `json:"transposable"`
	LicensesPurchased int    `json:"licenses_purchased"`
	LicensesRemaining int    `json:"licenses_remaining"`
	FileUploadID      string `json:"file_upload_identifier"`
	AttachableType    string `json:"attachable_type"`
}
