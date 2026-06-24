package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/joe--cool/pccli/internal/output"
	"github.com/joe--cool/pccli/internal/services"
	"github.com/spf13/cobra"
)

func newServicesCommand(jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "services",
		Short:   "Work with Planning Center Services",
		GroupID: "products",
	}
	silenceCobra(cmd)
	cmd.AddGroup(&cobra.Group{ID: "music-library", Title: "Music Library"})
	songs := newSongsCommand(jsonOutput)
	songs.GroupID = "music-library"
	cmd.AddCommand(songs)
	return cmd
}

func newSongsCommand(jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "songs",
		Short: "Find songs and arrangements",
	}
	silenceCobra(cmd)
	cmd.AddCommand(newSongsListCommand(jsonOutput))
	cmd.AddCommand(newSongsSearchCommand(jsonOutput))
	cmd.AddCommand(newSongsShowCommand(jsonOutput))
	cmd.AddCommand(newSongsCreateCommand(jsonOutput))
	cmd.AddCommand(newSongsUpdateCommand(jsonOutput))
	cmd.AddCommand(newSongsDeleteCommand(jsonOutput))
	cmd.AddCommand(newArrangementsCommand(jsonOutput))
	cmd.AddCommand(newKeysCommand(jsonOutput))
	cmd.AddCommand(newAttachmentsCommand(jsonOutput))
	cmd.AddCommand(newAttachCommand(jsonOutput))
	return cmd
}

func newSongsListCommand(jsonOutput *bool) *cobra.Command {
	var opts services.ListSongsOptions
	var hidden bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Search the Services song library",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			visible := false
			opts.Hidden = &visible
			if cmd.Flags().Changed("hidden") {
				opts.Hidden = &hidden
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)

			songs, err := app.library.ListSongs(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(songs)
			}
			return renderer.Table(
				[]string{"ID", "Title", "Author", "CCLI"},
				songRows(songs),
			)
		},
	}
	silenceCobra(cmd)
	cmd.Flags().StringVarP(&opts.Search, "search", "q", "", "search song titles with a contains match")
	cmd.Flags().StringVar(&opts.Title, "title", "", "filter by exact or wildcard title, e.g. 'Amazing%'")
	cmd.Flags().StringVar(&opts.Author, "author", "", "filter by exact or wildcard author")
	cmd.Flags().IntVar(&opts.CCLINumber, "ccli", 0, "filter by CCLI number")
	cmd.Flags().BoolVar(&hidden, "hidden", false, "show only hidden songs")
	cmd.Flags().StringVar(&opts.Key, "key", "", "filter by song key, e.g. G or Cm")
	cmd.Flags().StringVar(&opts.Meter, "meter", "", "filter by meter, e.g. 4/4")
	cmd.Flags().StringVar(&opts.Themes, "themes", "", "filter by themes")
	cmd.Flags().StringVar(&opts.Order, "order", "", "Planning Center order value, e.g. title or -updated_at")
	cmd.Flags().IntVar(&opts.PerPage, "limit", 25, "number of songs to fetch")
	return cmd
}

func newSongsSearchCommand(jsonOutput *bool) *cobra.Command {
	var opts services.ListSongsOptions
	var hidden bool
	cmd := &cobra.Command{
		Use:   "search QUERY",
		Short: "Search songs by title",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageError{err: fmt.Errorf("expected exactly one QUERY")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			opts.Search = args[0]
			visible := false
			opts.Hidden = &visible
			if cmd.Flags().Changed("hidden") {
				opts.Hidden = &hidden
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)

			songs, err := app.library.ListSongs(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(songs)
			}
			return renderer.Table(
				[]string{"ID", "Title", "Author", "CCLI"},
				songRows(songs),
			)
		},
	}
	silenceCobra(cmd)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "search hidden songs")
	cmd.Flags().IntVar(&opts.PerPage, "limit", 10, "number of songs to fetch")
	return cmd
}

func newSongsShowCommand(jsonOutput *bool) *cobra.Command {
	var hidden bool
	cmd := &cobra.Command{
		Use:   "show SONG [ARRANGEMENT]",
		Short: "Show one song with arrangement and key context",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return usageError{err: fmt.Errorf("expected SONG and optional ARRANGEMENT")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)

			songID, err := resolveSongID(cmd, app, args[0], allowFuzzySongMatch, hidden)
			if err != nil {
				return err
			}
			song, err := app.library.GetSong(cmd.Context(), songID)
			if err != nil {
				return err
			}
			if song.Hidden && !hidden {
				return fmt.Errorf("song %s is hidden; pass --hidden to show hidden songs", songID)
			}
			arrangements, err := app.library.ListArrangements(cmd.Context(), songID, services.ListArrangementsOptions{PerPage: 25})
			if err != nil {
				return err
			}
			var selected *services.Arrangement
			if len(arrangements) > 0 {
				target := ""
				if len(args) == 2 {
					target = args[1]
				}
				arrangement, err := selectArrangement(cmd, app, songID, target, arrangements, allowFuzzyArrangementMatch)
				if err != nil {
					return err
				}
				selected = &arrangement
			}
			keys := []services.Key{}
			if selected != nil {
				keys, err = app.library.ListKeys(cmd.Context(), songID, selected.ID, services.ListKeysOptions{PerPage: 25})
				if err != nil {
					return err
				}
			}
			if renderer.JSON {
				return renderer.WriteJSON(songDetail{
					Song:         song,
					Arrangements: arrangements,
					Arrangement:  selected,
					Keys:         keys,
				})
			}
			return renderSongShow(renderer, song, arrangements, selected, keys, len(args) == 2, hidden)
		},
	}
	silenceCobra(cmd)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "show a hidden song and include hidden status")
	return cmd
}

func newSongsDeleteCommand(jsonOutput *bool) *cobra.Command {
	var yes bool
	var hidden bool
	cmd := &cobra.Command{
		Use:   "delete SONG",
		Short: "Delete a song",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageError{err: fmt.Errorf("expected exactly one SONG")}
			}
			if !yes {
				return usageError{err: fmt.Errorf("pass --yes to confirm deleting the song")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			songID, err := resolveSongID(cmd, app, args[0], requireExactSongMatch, hidden)
			if err != nil {
				return err
			}
			if err := app.library.DeleteSong(cmd.Context(), songID); err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(deleteResult{Deleted: true, SongID: songID})
			}
			_, err = fmt.Fprintf(renderer.Out, "Deleted song %s.\n", songID)
			return err
		},
	}
	silenceCobra(cmd)
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	cmd.Flags().BoolVar(&hidden, "hidden", false, "delete a hidden song")
	return cmd
}

type songDetail struct {
	Song         services.Song          `json:"song"`
	Arrangements []services.Arrangement `json:"arrangements,omitempty"`
	Arrangement  *services.Arrangement  `json:"arrangement,omitempty"`
	Keys         []services.Key         `json:"keys,omitempty"`
}

type arrangementDetail struct {
	Arrangement services.Arrangement `json:"arrangement"`
	Keys        []services.Key       `json:"keys,omitempty"`
}

type deleteResult struct {
	Deleted       bool   `json:"deleted"`
	SongID        string `json:"song_id,omitempty"`
	ArrangementID string `json:"arrangement_id,omitempty"`
	KeyID         string `json:"key_id,omitempty"`
}

func newSongsCreateCommand(jsonOutput *bool) *cobra.Command {
	var mutation services.SongMutation
	var hidden bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a song in the Services song library",
		RunE: func(cmd *cobra.Command, args []string) error {
			if mutation.Title == "" && mutation.CCLINumber == 0 {
				return usageError{err: fmt.Errorf("set --title or --ccli")}
			}
			if mutation.Title == "" && mutation.CCLINumber != 0 {
				mutation.Title = strconv.Itoa(mutation.CCLINumber)
			}
			if cmd.Flags().Changed("hidden") {
				mutation.Hidden = &hidden
			}

			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			song, err := app.library.CreateSong(cmd.Context(), mutation)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(song)
			}
			return renderSongSummary(renderer, song, cmd.Flags().Changed("hidden"))
		},
	}
	silenceCobra(cmd)
	addSongMutationFlags(cmd, &mutation, &hidden)
	return cmd
}

func newSongsUpdateCommand(jsonOutput *bool) *cobra.Command {
	var mutation services.SongMutation
	var hidden bool
	cmd := &cobra.Command{
		Use:   "update SONG",
		Short: "Update song metadata",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageError{err: fmt.Errorf("expected exactly one SONG")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("hidden") {
				mutation.Hidden = &hidden
			}
			if !songMutationChanged(cmd) {
				return usageError{err: fmt.Errorf("set at least one song metadata flag")}
			}

			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			songID, err := resolveSongID(cmd, app, args[0], requireExactSongMatch, cmd.Flags().Changed("hidden"))
			if err != nil {
				return err
			}
			song, err := app.library.UpdateSong(cmd.Context(), songID, mutation)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(song)
			}
			return renderSongSummary(renderer, song, cmd.Flags().Changed("hidden"))
		},
	}
	silenceCobra(cmd)
	addSongMutationFlags(cmd, &mutation, &hidden)
	return cmd
}

func newArrangementsCommand(jsonOutput *bool) *cobra.Command {
	var opts services.ListArrangementsOptions
	var hidden bool
	cmd := &cobra.Command{
		Use:   "arrangements SONG [ARRANGEMENT]",
		Short: "List or show arrangements for a song",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return usageError{err: fmt.Errorf("expected SONG and optional ARRANGEMENT")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)

			songID, err := resolveSongID(cmd, app, args[0], allowFuzzySongMatch, hidden)
			if err != nil {
				return err
			}
			arrangements, err := app.library.ListArrangements(cmd.Context(), songID, opts)
			if err != nil {
				return err
			}
			if len(args) == 2 {
				arrangement, err := selectArrangement(cmd, app, songID, args[1], arrangements, allowFuzzyArrangementMatch)
				if err != nil {
					return err
				}
				keys, err := app.library.ListKeys(cmd.Context(), songID, arrangement.ID, services.ListKeysOptions{PerPage: 25})
				if err != nil {
					return err
				}
				if renderer.JSON {
					return renderer.WriteJSON(arrangementDetail{Arrangement: arrangement, Keys: keys})
				}
				if err := renderArrangementSummary(renderer, arrangement); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(renderer.Out, "\nKeys"); err != nil {
					return err
				}
				return renderer.Table(
					[]string{"ID", "Name", "Start", "End"},
					keyRows(keys),
				)
			}
			if renderer.JSON {
				return renderer.WriteJSON(arrangements)
			}
			return renderer.Table(
				[]string{"ID", "Name", "Key", "BPM", "Meter", "Length", "Sequence"},
				arrangementRows(arrangements),
			)
		},
	}
	silenceCobra(cmd)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "look up arrangements for a hidden song")
	cmd.Flags().IntVar(&opts.PerPage, "limit", 25, "number of arrangements to fetch")
	cmd.AddCommand(newArrangementCreateCommand(jsonOutput))
	cmd.AddCommand(newArrangementUpdateCommand(jsonOutput))
	cmd.AddCommand(newArrangementDeleteCommand(jsonOutput))
	return cmd
}

func newArrangementCreateCommand(jsonOutput *bool) *cobra.Command {
	var mutation services.ArrangementMutation
	var length string
	var lyricsEnabled bool
	var hidden bool
	cmd := &cobra.Command{
		Use:   "create SONG",
		Short: "Create an arrangement for a song",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageError{err: fmt.Errorf("expected exactly one SONG")}
			}
			if mutation.Name == "" {
				return usageError{err: fmt.Errorf("set --name")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := applyArrangementFlags(cmd, &mutation, length, lyricsEnabled); err != nil {
				return err
			}
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			songID, err := resolveSongID(cmd, app, args[0], requireExactSongMatch, hidden)
			if err != nil {
				return err
			}
			arrangement, err := app.library.CreateArrangement(cmd.Context(), songID, mutation)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(arrangement)
			}
			return renderArrangementSummary(renderer, arrangement)
		},
	}
	silenceCobra(cmd)
	addArrangementMutationFlags(cmd, &mutation, &length, &lyricsEnabled)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "create the arrangement on a hidden song")
	return cmd
}

func newArrangementUpdateCommand(jsonOutput *bool) *cobra.Command {
	var mutation services.ArrangementMutation
	var length string
	var lyricsEnabled bool
	var hidden bool
	cmd := &cobra.Command{
		Use:   "update SONG ARRANGEMENT",
		Short: "Update arrangement metadata",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return usageError{err: fmt.Errorf("expected SONG and ARRANGEMENT")}
			}
			if !arrangementMutationChanged(cmd) {
				return usageError{err: fmt.Errorf("set at least one arrangement metadata flag")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := applyArrangementFlags(cmd, &mutation, length, lyricsEnabled); err != nil {
				return err
			}
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			songID, err := resolveSongID(cmd, app, args[0], requireExactSongMatch, hidden)
			if err != nil {
				return err
			}
			arrangement, err := resolveArrangement(cmd, app, songID, args[1], requireExactArrangementMatch)
			if err != nil {
				return err
			}
			updated, err := app.library.UpdateArrangement(cmd.Context(), songID, arrangement.ID, mutation)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(updated)
			}
			return renderArrangementSummary(renderer, updated)
		},
	}
	silenceCobra(cmd)
	addArrangementMutationFlags(cmd, &mutation, &length, &lyricsEnabled)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "update an arrangement on a hidden song")
	return cmd
}

func newArrangementDeleteCommand(jsonOutput *bool) *cobra.Command {
	var yes bool
	var hidden bool
	cmd := &cobra.Command{
		Use:   "delete SONG ARRANGEMENT",
		Short: "Delete an arrangement",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return usageError{err: fmt.Errorf("expected SONG and ARRANGEMENT")}
			}
			if !yes {
				return usageError{err: fmt.Errorf("pass --yes to confirm deleting the arrangement")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			songID, err := resolveSongID(cmd, app, args[0], requireExactSongMatch, hidden)
			if err != nil {
				return err
			}
			arrangement, err := resolveArrangement(cmd, app, songID, args[1], requireExactArrangementMatch)
			if err != nil {
				return err
			}
			if err := app.library.DeleteArrangement(cmd.Context(), songID, arrangement.ID); err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(deleteResult{Deleted: true, SongID: songID, ArrangementID: arrangement.ID})
			}
			_, err = fmt.Fprintf(renderer.Out, "Deleted arrangement %s from song %s.\n", arrangement.ID, songID)
			return err
		},
	}
	silenceCobra(cmd)
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	cmd.Flags().BoolVar(&hidden, "hidden", false, "delete an arrangement from a hidden song")
	return cmd
}

func newKeysCommand(jsonOutput *bool) *cobra.Command {
	var opts services.ListKeysOptions
	var hidden bool
	cmd := &cobra.Command{
		Use:   "keys SONG [ARRANGEMENT]",
		Short: "List keys for an arrangement",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return usageError{err: fmt.Errorf("expected SONG and optional ARRANGEMENT")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)

			songID, err := resolveSongID(cmd, app, args[0], allowFuzzySongMatch, hidden)
			if err != nil {
				return err
			}
			target := ""
			if len(args) == 2 {
				target = args[1]
			}
			arrangement, err := resolveArrangement(cmd, app, songID, target, allowFuzzyArrangementMatch)
			if err != nil {
				return err
			}
			keys, err := app.library.ListKeys(cmd.Context(), songID, arrangement.ID, opts)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(keys)
			}
			return renderer.Table(
				[]string{"ID", "Name", "Start", "End"},
				keyRows(keys),
			)
		},
	}
	silenceCobra(cmd)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "look up keys for a hidden song")
	cmd.Flags().IntVar(&opts.PerPage, "limit", 25, "number of keys to fetch")
	cmd.AddCommand(newKeyCreateCommand(jsonOutput))
	cmd.AddCommand(newKeyUpdateCommand(jsonOutput))
	cmd.AddCommand(newKeyDeleteCommand(jsonOutput))
	return cmd
}

func newKeyCreateCommand(jsonOutput *bool) *cobra.Command {
	var mutation services.KeyMutation
	var hidden bool
	cmd := &cobra.Command{
		Use:   "create SONG [ARRANGEMENT]",
		Short: "Create a key for an arrangement",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return usageError{err: fmt.Errorf("expected SONG and optional ARRANGEMENT")}
			}
			if mutation.Name == "" {
				return usageError{err: fmt.Errorf("set --name")}
			}
			if mutation.StartingKey == "" {
				return usageError{err: fmt.Errorf("set --start")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if mutation.EndingKey == "" {
				mutation.EndingKey = mutation.StartingKey
			}
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			songID, err := resolveSongID(cmd, app, args[0], requireExactSongMatch, hidden)
			if err != nil {
				return err
			}
			target := ""
			if len(args) == 2 {
				target = args[1]
			}
			arrangement, err := resolveArrangement(cmd, app, songID, target, requireExactArrangementMatch)
			if err != nil {
				return err
			}
			key, err := app.library.CreateKey(cmd.Context(), songID, arrangement.ID, mutation)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(key)
			}
			return renderKeySummary(renderer, key)
		},
	}
	silenceCobra(cmd)
	addKeyMutationFlags(cmd, &mutation)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "create the key on a hidden song")
	return cmd
}

func newKeyUpdateCommand(jsonOutput *bool) *cobra.Command {
	var mutation services.KeyMutation
	var hidden bool
	cmd := &cobra.Command{
		Use:   "update SONG ARRANGEMENT KEY",
		Short: "Update key metadata",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				return usageError{err: fmt.Errorf("expected SONG, ARRANGEMENT, and KEY")}
			}
			if !keyMutationChanged(cmd) {
				return usageError{err: fmt.Errorf("set at least one key metadata flag")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			songID, err := resolveSongID(cmd, app, args[0], requireExactSongMatch, hidden)
			if err != nil {
				return err
			}
			arrangement, err := resolveArrangement(cmd, app, songID, args[1], requireExactArrangementMatch)
			if err != nil {
				return err
			}
			key, err := resolveKey(cmd, app, songID, arrangement.ID, args[2], requireExactKeyMatch)
			if err != nil {
				return err
			}
			updated, err := app.library.UpdateKey(cmd.Context(), songID, arrangement.ID, key.ID, mutation)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(updated)
			}
			return renderKeySummary(renderer, updated)
		},
	}
	silenceCobra(cmd)
	addKeyMutationFlags(cmd, &mutation)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "update a key on a hidden song")
	return cmd
}

func newKeyDeleteCommand(jsonOutput *bool) *cobra.Command {
	var yes bool
	var hidden bool
	cmd := &cobra.Command{
		Use:   "delete SONG ARRANGEMENT KEY",
		Short: "Delete a key",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				return usageError{err: fmt.Errorf("expected SONG, ARRANGEMENT, and KEY")}
			}
			if !yes {
				return usageError{err: fmt.Errorf("pass --yes to confirm deleting the key")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			songID, err := resolveSongID(cmd, app, args[0], requireExactSongMatch, hidden)
			if err != nil {
				return err
			}
			arrangement, err := resolveArrangement(cmd, app, songID, args[1], requireExactArrangementMatch)
			if err != nil {
				return err
			}
			key, err := resolveKey(cmd, app, songID, arrangement.ID, args[2], requireExactKeyMatch)
			if err != nil {
				return err
			}
			if err := app.library.DeleteKey(cmd.Context(), songID, arrangement.ID, key.ID); err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(deleteResult{Deleted: true, SongID: songID, ArrangementID: arrangement.ID, KeyID: key.ID})
			}
			_, err = fmt.Fprintf(renderer.Out, "Deleted key %s from arrangement %s.\n", key.ID, arrangement.ID)
			return err
		},
	}
	silenceCobra(cmd)
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	cmd.Flags().BoolVar(&hidden, "hidden", false, "delete a key from a hidden song")
	return cmd
}

func newAttachmentsCommand(jsonOutput *bool) *cobra.Command {
	var opts services.ListAttachmentsOptions
	var scope services.AttachmentScope
	var hidden bool
	cmd := &cobra.Command{
		Use:   "attachments SONG",
		Short: "List song, arrangement, or key attachments",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageError{err: fmt.Errorf("expected exactly one SONG")}
			}
			if scope.KeyID != "" && scope.ArrangementID == "" {
				return usageError{err: fmt.Errorf("--key requires --arrangement")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			scope.SongID, err = resolveSongID(cmd, app, args[0], allowFuzzySongMatch, hidden)
			if err != nil {
				return err
			}
			if err := resolveAttachmentScope(cmd, app, &scope, allowFuzzyArrangementMatch, allowFuzzyKeyMatch); err != nil {
				return err
			}

			attachments, err := app.library.ListAttachments(cmd.Context(), scope, opts)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(attachments)
			}
			return renderer.Table(
				[]string{"ID", "Name", "File", "Type", "Size", "Link"},
				attachmentRows(attachments),
			)
		},
	}
	silenceCobra(cmd)
	addAttachmentScopeFlags(cmd, &scope)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "look up attachments for a hidden song")
	cmd.Flags().StringVar(&opts.Filename, "filename", "", "filter by filename")
	cmd.Flags().StringVar(&opts.Type, "type", "", "filter by Planning Center attachment type")
	cmd.Flags().IntVar(&opts.PerPage, "limit", 25, "number of attachments to fetch")
	return cmd
}

func newAttachCommand(jsonOutput *bool) *cobra.Command {
	var mutation services.AttachmentMutation
	var scope services.AttachmentScope
	var importToItemDetails bool
	var hidden bool
	cmd := &cobra.Command{
		Use:   "attach SONG",
		Short: "Attach a file or link to a song, arrangement, or key",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageError{err: fmt.Errorf("expected exactly one SONG")}
			}
			if scope.KeyID != "" && scope.ArrangementID == "" {
				return usageError{err: fmt.Errorf("--key requires --arrangement")}
			}
			sources := 0
			for _, value := range []string{mutation.FilePath, mutation.FileUploadID, mutation.RemoteLink, mutation.Content} {
				if value != "" {
					sources++
				}
			}
			if sources != 1 {
				return usageError{err: fmt.Errorf("set exactly one of --file, --upload-id, --url, or --content")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("item-details") {
				mutation.ImportToItemDetails = &importToItemDetails
			}
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)
			scope.SongID, err = resolveSongID(cmd, app, args[0], requireExactSongMatch, hidden)
			if err != nil {
				return err
			}
			if err := resolveAttachmentScope(cmd, app, &scope, requireExactArrangementMatch, requireExactKeyMatch); err != nil {
				return err
			}

			attachment, err := app.library.AddAttachment(cmd.Context(), scope, mutation)
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(attachment)
			}
			return renderAttachmentSummary(renderer, attachment)
		},
	}
	silenceCobra(cmd)
	addAttachmentScopeFlags(cmd, &scope)
	cmd.Flags().BoolVar(&hidden, "hidden", false, "attach to a hidden song")
	cmd.Flags().StringVar(&mutation.FilePath, "file", "", "upload and attach a local file such as a PDF")
	cmd.Flags().StringVar(&mutation.FileUploadID, "upload-id", "", "attach an existing Planning Center upload UUID")
	cmd.Flags().StringVar(&mutation.RemoteLink, "url", "", "attach a remote link")
	cmd.Flags().StringVar(&mutation.Content, "content", "", "attach inline attachment content")
	cmd.Flags().StringVar(&mutation.Filename, "filename", "", "filename to show in Planning Center")
	cmd.Flags().StringVar(&mutation.AttachmentTypeIDs, "attachment-type-ids", "", "comma-separated Planning Center attachment type IDs")
	cmd.Flags().BoolVar(&importToItemDetails, "item-details", false, "import this attachment to item details")
	cmd.Flags().StringVar(&mutation.PageOrder, "page-order", "", "Planning Center page order value")
	cmd.Flags().StringVar(&mutation.SongPart, "song-part", "", "song part label for generated charts")
	return cmd
}

func songRows(songs []services.Song) [][]string {
	rows := make([][]string, 0, len(songs))
	for _, song := range songs {
		rows = append(rows, []string{
			song.ID,
			song.Title,
			song.Author,
			output.Int(song.CCLINumber),
		})
	}
	return rows
}

func keyRows(keys []services.Key) [][]string {
	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{
			key.ID,
			key.Name,
			key.StartingKey,
			key.EndingKey,
		})
	}
	return rows
}

func attachmentRows(attachments []services.Attachment) [][]string {
	rows := make([][]string, 0, len(attachments))
	for _, attachment := range attachments {
		rows = append(rows, []string{
			attachment.ID,
			firstNonEmpty(attachment.DisplayName, attachment.Filename),
			attachment.Filename,
			firstNonEmpty(attachment.PCOType, attachment.FileType, attachment.ContentType),
			output.Bytes(attachment.FileSize),
			firstNonEmpty(attachment.RemoteLink, attachment.URL),
		})
	}
	return rows
}

func arrangementRows(arrangements []services.Arrangement) [][]string {
	rows := make([][]string, 0, len(arrangements))
	for _, arrangement := range arrangements {
		rows = append(rows, []string{
			arrangement.ID,
			arrangement.Name,
			arrangement.ChordChartKey,
			output.Float(arrangement.BPM),
			arrangement.Meter,
			output.DurationSeconds(arrangement.Length),
			strings.Join(arrangement.SequenceShort, " "),
		})
	}
	return rows
}

func addSongMutationFlags(cmd *cobra.Command, mutation *services.SongMutation, hidden *bool) {
	cmd.Flags().StringVar(&mutation.Title, "title", "", "song title")
	cmd.Flags().StringVar(&mutation.Admin, "admin", "", "song administrator")
	cmd.Flags().StringVar(&mutation.Author, "author", "", "song author")
	cmd.Flags().IntVar(&mutation.CCLINumber, "ccli", 0, "CCLI number")
	cmd.Flags().StringVar(&mutation.Copyright, "copyright", "", "copyright text")
	cmd.Flags().BoolVar(hidden, "hidden", false, "mark the song hidden")
	cmd.Flags().StringVar(&mutation.Themes, "themes", "", "themes or tags text")
}

func addAttachmentScopeFlags(cmd *cobra.Command, scope *services.AttachmentScope) {
	cmd.Flags().StringVar(&scope.ArrangementID, "arrangement", "", "scope to an arrangement ID or exact name")
	cmd.Flags().StringVar(&scope.KeyID, "key", "", "scope to a key ID or exact name; requires --arrangement")
}

func addArrangementMutationFlags(cmd *cobra.Command, mutation *services.ArrangementMutation, length *string, lyricsEnabled *bool) {
	cmd.Flags().StringVar(&mutation.Name, "name", "", "arrangement name")
	cmd.Flags().Float64Var(&mutation.BPM, "bpm", 0, "beats per minute")
	cmd.Flags().StringVar(&mutation.Meter, "meter", "", "meter, e.g. 4/4")
	cmd.Flags().StringVar(length, "length", "", "arrangement length as seconds or m:ss")
	cmd.Flags().StringVar(&mutation.ChordChartKey, "key", "", "chord chart key, e.g. G or Cm")
	cmd.Flags().StringVar(&mutation.Notes, "notes", "", "arrangement notes")
	cmd.Flags().BoolVar(lyricsEnabled, "lyrics-enabled", false, "enable lyrics for the arrangement")
	cmd.Flags().StringSliceVar(&mutation.Sequence, "sequence", nil, "song sequence, e.g. V1,V2,C")
}

func applyArrangementFlags(cmd *cobra.Command, mutation *services.ArrangementMutation, length string, lyricsEnabled bool) error {
	if cmd.Flags().Changed("length") {
		seconds, err := parseLengthSeconds(length)
		if err != nil {
			return usageError{err: err}
		}
		mutation.Length = seconds
	}
	if cmd.Flags().Changed("lyrics-enabled") {
		mutation.LyricsEnabled = &lyricsEnabled
	}
	return nil
}

func addKeyMutationFlags(cmd *cobra.Command, mutation *services.KeyMutation) {
	cmd.Flags().StringVar(&mutation.Name, "name", "", "key name")
	cmd.Flags().StringVar(&mutation.StartingKey, "start", "", "starting key, e.g. G or Cm")
	cmd.Flags().StringVar(&mutation.EndingKey, "end", "", "ending key, e.g. G or Cm")
}

type songResolutionMode int

const (
	allowFuzzySongMatch songResolutionMode = iota
	requireExactSongMatch
)

type arrangementResolutionMode int

const (
	allowFuzzyArrangementMatch arrangementResolutionMode = iota
	requireExactArrangementMatch
)

type keyResolutionMode int

const (
	allowFuzzyKeyMatch keyResolutionMode = iota
	requireExactKeyMatch
)

func resolveSongID(cmd *cobra.Command, app *app, value string, mode songResolutionMode, includeHidden bool) (string, error) {
	if isNumericID(value) {
		if includeHidden {
			return value, nil
		}
		song, err := app.library.GetSong(cmd.Context(), value)
		if err != nil {
			return "", err
		}
		if song.Hidden {
			return "", fmt.Errorf("song %s is hidden; pass --hidden to include hidden songs", value)
		}
		return value, nil
	}

	hidden := includeHidden
	songs, err := app.library.ListSongs(cmd.Context(), services.ListSongsOptions{
		Search:  value,
		Hidden:  &hidden,
		PerPage: 10,
	})
	if err != nil {
		return "", err
	}

	exactMatches := make([]services.Song, 0, len(songs))
	for _, song := range songs {
		if strings.EqualFold(song.Title, value) {
			exactMatches = append(exactMatches, song)
		}
	}
	if len(exactMatches) == 1 {
		return exactMatches[0].ID, nil
	}
	if len(exactMatches) > 1 {
		return "", fmt.Errorf("multiple songs exactly match %q; use the song ID: %s", value, songChoices(exactMatches))
	}

	if mode == requireExactSongMatch {
		return "", fmt.Errorf("no exact song title matches %q; run `pccli services songs search %q` and use the song ID or exact title", value, value)
	}
	fuzzyMatches := make([]services.Song, 0, len(songs))
	lowerValue := strings.ToLower(value)
	for _, song := range songs {
		if strings.Contains(strings.ToLower(song.Title), lowerValue) {
			fuzzyMatches = append(fuzzyMatches, song)
		}
	}
	if len(fuzzyMatches) == 1 {
		return fuzzyMatches[0].ID, nil
	}
	if len(fuzzyMatches) > 1 {
		return "", fmt.Errorf("multiple songs match %q; use the song ID or a more specific title: %s", value, songChoices(fuzzyMatches))
	}
	if len(songs) == 1 {
		return songs[0].ID, nil
	}
	if len(songs) == 0 {
		return "", fmt.Errorf("no songs match %q", value)
	}
	return "", fmt.Errorf("multiple songs match %q; use the song ID or a more specific title: %s", value, songChoices(songs))
}

func resolveArrangement(cmd *cobra.Command, app *app, songID, value string, mode arrangementResolutionMode) (services.Arrangement, error) {
	arrangements, err := app.library.ListArrangements(cmd.Context(), songID, services.ListArrangementsOptions{PerPage: 100})
	if err != nil {
		return services.Arrangement{}, err
	}
	return selectArrangement(cmd, app, songID, value, arrangements, mode)
}

func selectArrangement(cmd *cobra.Command, app *app, songID, value string, arrangements []services.Arrangement, mode arrangementResolutionMode) (services.Arrangement, error) {
	if value == "" {
		arrangement, ok := defaultArrangement(arrangements)
		if !ok {
			return services.Arrangement{}, fmt.Errorf("song %s has no arrangements", songID)
		}
		return arrangement, nil
	}

	if isNumericID(value) {
		for _, arrangement := range arrangements {
			if arrangement.ID == value {
				return arrangement, nil
			}
		}
		return app.library.GetArrangement(cmd.Context(), songID, value)
	}

	exactMatches := make([]services.Arrangement, 0, len(arrangements))
	for _, arrangement := range arrangements {
		if strings.EqualFold(arrangement.Name, value) {
			exactMatches = append(exactMatches, arrangement)
		}
	}
	if len(exactMatches) == 1 {
		return exactMatches[0], nil
	}
	if len(exactMatches) > 1 {
		return services.Arrangement{}, fmt.Errorf("multiple arrangements exactly match %q; use the arrangement ID: %s", value, arrangementChoices(exactMatches))
	}
	if mode == requireExactArrangementMatch {
		return services.Arrangement{}, fmt.Errorf("no exact arrangement name matches %q; run `pccli services songs arrangements %s` and use the arrangement ID or exact name", value, songID)
	}

	fuzzyMatches := make([]services.Arrangement, 0, len(arrangements))
	lowerValue := strings.ToLower(value)
	for _, arrangement := range arrangements {
		if strings.Contains(strings.ToLower(arrangement.Name), lowerValue) {
			fuzzyMatches = append(fuzzyMatches, arrangement)
		}
	}
	if len(fuzzyMatches) == 1 {
		return fuzzyMatches[0], nil
	}
	if len(fuzzyMatches) == 0 {
		return services.Arrangement{}, fmt.Errorf("no arrangements match %q", value)
	}
	return services.Arrangement{}, fmt.Errorf("multiple arrangements match %q; use the arrangement ID or a more specific name: %s", value, arrangementChoices(fuzzyMatches))
}

func resolveKey(cmd *cobra.Command, app *app, songID, arrangementID, value string, mode keyResolutionMode) (services.Key, error) {
	if isNumericID(value) {
		return services.Key{ID: value}, nil
	}
	keys, err := app.library.ListKeys(cmd.Context(), songID, arrangementID, services.ListKeysOptions{PerPage: 100})
	if err != nil {
		return services.Key{}, err
	}
	exactMatches := make([]services.Key, 0, len(keys))
	for _, key := range keys {
		if strings.EqualFold(key.Name, value) {
			exactMatches = append(exactMatches, key)
		}
	}
	if len(exactMatches) == 1 {
		return exactMatches[0], nil
	}
	if len(exactMatches) > 1 {
		return services.Key{}, fmt.Errorf("multiple keys exactly match %q; use the key ID: %s", value, keyChoices(exactMatches))
	}
	if mode == requireExactKeyMatch {
		return services.Key{}, fmt.Errorf("no exact key name matches %q; run `pccli services songs keys %s %s` and use the key ID or exact name", value, songID, arrangementID)
	}

	fuzzyMatches := make([]services.Key, 0, len(keys))
	lowerValue := strings.ToLower(value)
	for _, key := range keys {
		if strings.Contains(strings.ToLower(key.Name), lowerValue) {
			fuzzyMatches = append(fuzzyMatches, key)
		}
	}
	if len(fuzzyMatches) == 1 {
		return fuzzyMatches[0], nil
	}
	if len(fuzzyMatches) == 0 {
		return services.Key{}, fmt.Errorf("no keys match %q", value)
	}
	return services.Key{}, fmt.Errorf("multiple keys match %q; use the key ID or a more specific name: %s", value, keyChoices(fuzzyMatches))
}

func resolveAttachmentScope(cmd *cobra.Command, app *app, scope *services.AttachmentScope, arrangementMode arrangementResolutionMode, keyMode keyResolutionMode) error {
	if scope.ArrangementID == "" {
		return nil
	}
	arrangement, err := resolveArrangement(cmd, app, scope.SongID, scope.ArrangementID, arrangementMode)
	if err != nil {
		return err
	}
	scope.ArrangementID = arrangement.ID
	if scope.KeyID == "" {
		return nil
	}
	key, err := resolveKey(cmd, app, scope.SongID, scope.ArrangementID, scope.KeyID, keyMode)
	if err != nil {
		return err
	}
	scope.KeyID = key.ID
	return nil
}

func defaultArrangement(arrangements []services.Arrangement) (services.Arrangement, bool) {
	if len(arrangements) == 0 {
		return services.Arrangement{}, false
	}
	for _, arrangement := range arrangements {
		if strings.EqualFold(arrangement.Name, "default") {
			return arrangement, true
		}
	}
	return arrangements[0], true
}

func isNumericID(value string) bool {
	if value == "" {
		return false
	}
	_, err := strconv.ParseInt(value, 10, 64)
	return err == nil
}

func songChoices(songs []services.Song) string {
	choices := make([]string, 0, len(songs))
	for _, song := range songs {
		details := []string{fmt.Sprintf("%s %q", song.ID, song.Title)}
		if song.Author != "" {
			details = append(details, "by "+song.Author)
		}
		if song.CCLINumber != 0 {
			details = append(details, "CCLI "+output.Int(song.CCLINumber))
		}
		choices = append(choices, strings.Join(details, " "))
	}
	return strings.Join(choices, ", ")
}

func arrangementChoices(arrangements []services.Arrangement) string {
	choices := make([]string, 0, len(arrangements))
	for _, arrangement := range arrangements {
		details := []string{fmt.Sprintf("%s %q", arrangement.ID, arrangement.Name)}
		if arrangement.ChordChartKey != "" {
			details = append(details, "key "+arrangement.ChordChartKey)
		}
		choices = append(choices, strings.Join(details, " "))
	}
	return strings.Join(choices, ", ")
}

func keyChoices(keys []services.Key) string {
	choices := make([]string, 0, len(keys))
	for _, key := range keys {
		details := []string{fmt.Sprintf("%s %q", key.ID, key.Name)}
		if key.StartingKey != "" {
			details = append(details, "start "+key.StartingKey)
		}
		choices = append(choices, strings.Join(details, " "))
	}
	return strings.Join(choices, ", ")
}

func songMutationChanged(cmd *cobra.Command) bool {
	for _, name := range []string{"title", "admin", "author", "ccli", "copyright", "hidden", "themes"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func arrangementMutationChanged(cmd *cobra.Command) bool {
	for _, name := range []string{"name", "bpm", "meter", "length", "key", "notes", "lyrics-enabled", "sequence"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func keyMutationChanged(cmd *cobra.Command) bool {
	for _, name := range []string{"name", "start", "end"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func renderSongShow(renderer output.Renderer, song services.Song, arrangements []services.Arrangement, selected *services.Arrangement, keys []services.Key, explicitArrangement, includeHidden bool) error {
	if _, err := fmt.Fprintf(renderer.Out, "%s %s\n", renderer.Title(song.Title), renderer.Muted("(song #"+song.ID+")")); err != nil {
		return err
	}
	rows := [][2]string{
		{"Author", song.Author},
		{"Admin", song.Admin},
		{"CCLI", output.Int(song.CCLINumber)},
		{"Primary Key", song.PrimaryKey},
		{"Copyright", song.Copyright},
		{"Themes", song.Themes},
		{"Last Scheduled", firstNonEmpty(song.LastScheduled, song.LastScheduledAt)},
		{"Notes", song.Notes},
	}
	if includeHidden {
		rows = append(rows, [2]string{"Hidden", output.Bool(song.Hidden)})
	}
	if err := renderFieldRows(renderer, rows); err != nil {
		return err
	}

	if selected == nil {
		if len(arrangements) == 0 {
			_, err := fmt.Fprintf(renderer.Out, "\n%s\n  No arrangements found.\n", renderer.Section("Arrangements"))
			return err
		}
		return nil
	}

	title := "Default arrangement"
	if explicitArrangement {
		title = "Arrangement"
	}
	if _, err := fmt.Fprintf(renderer.Out, "\n%s\n", renderer.Section(title)); err != nil {
		return err
	}
	if err := renderArrangementDetails(renderer, *selected); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(renderer.Out, "\n%s\n", renderer.Section("Keys")); err != nil {
		return err
	}
	if len(keys) == 0 {
		if _, err := fmt.Fprintln(renderer.Out, "  No keys found."); err != nil {
			return err
		}
	} else {
		if err := renderer.Table(
			[]string{"ID", "Name", "Start", "End"},
			keyRows(keys),
		); err != nil {
			return err
		}
	}

	if len(arrangements) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(renderer.Out, "\n%s\n", renderer.Section("Arrangements")); err != nil {
		return err
	}
	return renderer.Table(
		[]string{"Use", "ID", "Name", "Key", "BPM", "Meter", "Length", "Lyrics", "Chart", "Chords", "Sequence"},
		arrangementDetailRows(arrangements, selected.ID),
	)
}

func renderSongSummary(renderer output.Renderer, song services.Song, includeHidden bool) error {
	rows := [][2]string{
		{"ID", song.ID},
		{"Title", song.Title},
		{"Author", song.Author},
		{"CCLI", output.Int(song.CCLINumber)},
	}
	if includeHidden {
		rows = append(rows, [2]string{"Hidden", output.Bool(song.Hidden)})
	}
	return renderer.KeyValues(rows)
}

func renderArrangementSummary(renderer output.Renderer, arrangement services.Arrangement) error {
	return renderer.KeyValues([][2]string{
		{"ID", arrangement.ID},
		{"Name", arrangement.Name},
		{"Key", arrangement.ChordChartKey},
		{"BPM", output.Float(arrangement.BPM)},
		{"Meter", arrangement.Meter},
		{"Length", output.DurationSeconds(arrangement.Length)},
		{"Lyrics", output.Bool(arrangement.LyricsEnabled)},
		{"Chord Chart", output.Bool(arrangement.HasChordChart)},
		{"Sequence", strings.Join(arrangement.SequenceShort, " ")},
	})
}

func renderKeySummary(renderer output.Renderer, key services.Key) error {
	return renderer.KeyValues([][2]string{
		{"ID", key.ID},
		{"Name", key.Name},
		{"Start", key.StartingKey},
		{"End", key.EndingKey},
	})
}

func renderAttachmentSummary(renderer output.Renderer, attachment services.Attachment) error {
	return renderer.KeyValues([][2]string{
		{"ID", attachment.ID},
		{"Name", firstNonEmpty(attachment.DisplayName, attachment.Filename)},
		{"File", attachment.Filename},
		{"Type", firstNonEmpty(attachment.PCOType, attachment.FileType, attachment.ContentType)},
		{"Link", firstNonEmpty(attachment.RemoteLink, attachment.URL)},
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func renderFieldRows(renderer output.Renderer, rows [][2]string) error {
	width := 0
	for _, row := range rows {
		if row[1] != "" && len(row[0]) > width {
			width = len(row[0])
		}
	}
	for _, row := range rows {
		if row[1] == "" {
			continue
		}
		label := fmt.Sprintf("  %-*s", width, row[0])
		if _, err := fmt.Fprintf(renderer.Out, "%s  %s\n", renderer.Muted(label), row[1]); err != nil {
			return err
		}
	}
	return nil
}

func renderArrangementDetails(renderer output.Renderer, arrangement services.Arrangement) error {
	return renderFieldRows(renderer, [][2]string{
		{"ID", arrangement.ID},
		{"Name", arrangement.Name},
		{"Key", arrangement.ChordChartKey},
		{"BPM", output.Float(arrangement.BPM)},
		{"Meter", arrangement.Meter},
		{"Length", output.DurationSeconds(arrangement.Length)},
		{"Lyrics", output.Bool(arrangement.LyricsEnabled)},
		{"Chord Chart", output.Bool(arrangement.HasChordChart)},
		{"Chords", output.Bool(arrangement.HasChords)},
		{"Sequence", strings.Join(arrangement.SequenceShort, " ")},
	})
}

func arrangementDetailRows(arrangements []services.Arrangement, selectedID string) [][]string {
	rows := make([][]string, 0, len(arrangements))
	for _, arrangement := range arrangements {
		selected := ""
		if arrangement.ID == selectedID {
			selected = "*"
		}
		rows = append(rows, []string{
			selected,
			arrangement.ID,
			arrangement.Name,
			arrangement.ChordChartKey,
			output.Float(arrangement.BPM),
			arrangement.Meter,
			output.DurationSeconds(arrangement.Length),
			output.Bool(arrangement.LyricsEnabled),
			output.Bool(arrangement.HasChordChart),
			output.Bool(arrangement.HasChords),
			strings.Join(arrangement.SequenceShort, " "),
		})
	}
	return rows
}

func parseLengthSeconds(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("length cannot be empty")
	}
	parts := strings.Split(value, ":")
	switch len(parts) {
	case 1:
		seconds, err := strconv.Atoi(parts[0])
		if err != nil || seconds < 0 {
			return 0, fmt.Errorf("invalid --length %q: use seconds or m:ss", value)
		}
		return seconds, nil
	case 2:
		minutes, err := strconv.Atoi(parts[0])
		if err != nil || minutes < 0 {
			return 0, fmt.Errorf("invalid --length %q: use seconds or m:ss", value)
		}
		seconds, err := strconv.Atoi(parts[1])
		if err != nil || seconds < 0 || seconds > 59 {
			return 0, fmt.Errorf("invalid --length %q: use seconds or m:ss", value)
		}
		return minutes*60 + seconds, nil
	default:
		return 0, fmt.Errorf("invalid --length %q: use seconds or m:ss", value)
	}
}
