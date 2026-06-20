package cli

import (
	"fmt"
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
	cmd.AddCommand(newSongsShowCommand(jsonOutput))
	cmd.AddCommand(newArrangementsCommand(jsonOutput))
	return cmd
}

func newSongsListCommand(jsonOutput *bool) *cobra.Command {
	var opts services.ListSongsOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List songs",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
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
	cmd.Flags().StringVar(&opts.Title, "title", "", "filter by exact or wildcard title, e.g. 'Amazing%'")
	cmd.Flags().StringVar(&opts.Author, "author", "", "filter by exact or wildcard author")
	cmd.Flags().IntVar(&opts.CCLINumber, "ccli", 0, "filter by CCLI number")
	cmd.Flags().IntVar(&opts.PerPage, "limit", 25, "number of songs to fetch")
	return cmd
}

func newSongsShowCommand(jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show SONG_ID",
		Short: "Show one song",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageError{err: fmt.Errorf("expected exactly one SONG_ID")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)

			song, err := app.library.GetSong(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if renderer.JSON {
				return renderer.WriteJSON(song)
			}
			return renderer.KeyValues([][2]string{
				{"ID", song.ID},
				{"Title", song.Title},
				{"Author", song.Author},
				{"CCLI", output.Int(song.CCLINumber)},
				{"Primary Key", song.PrimaryKey},
			})
		},
	}
	silenceCobra(cmd)
	return cmd
}

func newArrangementsCommand(jsonOutput *bool) *cobra.Command {
	var opts services.ListArrangementsOptions
	cmd := &cobra.Command{
		Use:   "arrangements SONG_ID",
		Short: "List arrangements for a song",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageError{err: fmt.Errorf("expected exactly one SONG_ID")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := loadApp(cmd)
			if err != nil {
				return err
			}
			renderer := output.NewRenderer(cmd.OutOrStdout(), *jsonOutput, app.cfg.Color)

			arrangements, err := app.library.ListArrangements(cmd.Context(), args[0], opts)
			if err != nil {
				return err
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
	cmd.Flags().IntVar(&opts.PerPage, "limit", 25, "number of arrangements to fetch")
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
