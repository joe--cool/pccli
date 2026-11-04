package cli

import (
	"fmt"
	"strings"

	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func newSongsCommand(j *bool) *cobra.Command {
	g := group("songs", "Find, inspect, and maintain the song library")
	var opts services.SongListOptions
	c := dataCommand("list", "Find songs by title or metadata", 0, func(c *cobra.Command, _ []string) error {
		var e error
		opts.ListOptions, e = listOptions(c, opts.ListOptions)
		if e != nil {
			return e
		}
		if c.Flags().Changed("ccli") && opts.CCLINumber <= 0 {
			return usageError{fmt.Errorf("--ccli must be positive")}
		}
		if c.Flags().Changed("search") && c.Flags().Changed("title") {
			return usageError{fmt.Errorf("--search and --title are mutually exclusive")}
		}
		if !validOrder(opts.Order) {
			return usageError{fmt.Errorf("invalid --order; use title, created_at, updated_at, or last_scheduled_at")}
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if *j && opts.All {
			return writeStreamList(c, func(yield func(services.Song) error) (services.ListMeta, error) {
				return a.library.WalkSongs(c.Context(), opts, yield)
			})
		}
		r, e := a.library.ListSongs(c.Context(), opts)
		if e != nil {
			return e
		}
		return writeList(c, *j, a, r)
	})
	listFlags(c, &opts.ListOptions)
	visibility(c, &opts.Visibility)
	c.Flags().StringVarP(&opts.Search, "search", "q", "", "literal title contains search")
	c.Flags().StringVar(&opts.Title, "title", "", "literal exact title")
	c.Flags().StringVar(&opts.Author, "author", "", "exact author")
	c.Flags().IntVar(&opts.CCLINumber, "ccli", 0, "CCLI number")
	c.Flags().StringVar(&opts.Themes, "themes", "", "themes text")
	c.Flags().StringVar(&opts.Order, "order", "", "sort by title, created_at, updated_at or last_scheduled_at; prefix - to reverse")
	g.AddCommand(c)
	var show services.SongShowOptions
	c = dataCommand("show SONG", "Show a song and its arrangements", 1, func(c *cobra.Command, x []string) error {
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowSong(c.Context(), x[0], show)
		if e != nil {
			return e
		}
		return writeSongDetail(c, *j, a, r)
	})
	visibility(c, &show.Visibility)
	c.Flags().BoolVar(&show.IncludeArchived, "include-archived", false, "include archived arrangements explicitly")
	c.Flags().StringVar(&show.Arrangement, "arrangement", "", "explicit arrangement to expand with its keys")
	g.AddCommand(c)
	var create services.SongCreate
	b := newBindings()
	c = dataCommand("create", "Create a song or import metadata by CCLI number", 0, func(c *cobra.Command, _ []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := services.ValidateMutation(create, "Song"); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.CreateSong(c.Context(), create)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, create)
	})
	c.Flags().StringVar(&create.Title, "title", "", "song title; use --ccli alone for metadata import")
	bindSong(c, b, &create.Admin, &create.Author, &create.Copyright, &create.Themes, &create.CCLINumber, &create.Hidden)
	bindExtraFields(c, b, &create)
	g.AddCommand(c)
	var patch services.SongPatch
	var v services.Visibility
	b2 := newBindings()
	c = dataCommand("update SONG", "Change only explicitly supplied metadata", 1, func(c *cobra.Command, x []string) error {
		if e := b2.apply(c); e != nil {
			return e
		}
		if e := services.ValidateMutation(patch, "Song"); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.UpdateSong(c.Context(), x[0], v, patch)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, patch)
	})
	b2.str(c, "title", "song title", &patch.Title)
	bindSong(c, b2, &patch.Admin, &patch.Author, &patch.Copyright, &patch.Themes, &patch.CCLINumber, &patch.Hidden)
	visibility(c, &v)
	g.AddCommand(c)
	var dv services.Visibility
	c = dataCommand("delete SONG", "Permanently delete a song", 1, func(c *cobra.Command, x []string) error {
		if e := confirm(c); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if e = a.library.DeleteSong(c.Context(), x[0], dv); e != nil {
			return e
		}
		return writeMutationData(c, *j, a, receipt("song", x[0]), nil)
	})
	visibility(c, &dv)
	deleteFlag(c)
	g.AddCommand(c)
	var scope services.Scope
	var lo services.ListOptions
	c = dataCommand("schedules SONG", "List uses of a song in plans", 1, func(c *cobra.Command, x []string) error {
		var e error
		lo, e = listOptions(c, lo)
		if e != nil {
			return e
		}
		scope.Song = x[0]
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if *j && lo.All {
			return writeStreamList(c, func(yield func(services.AncillaryRecord) error) (services.ListMeta, error) {
				return a.library.WalkSongSchedules(c.Context(), scope, yield)
			})
		}
		r, e := a.library.ListSongSchedules(c.Context(), scope, lo)
		if e != nil {
			return e
		}
		return writeList(c, *j, a, r)
	})
	visibility(c, &scope.Visibility)
	listFlags(c, &lo)
	g.AddCommand(c)
	var last services.Scope
	var lastOpts services.ListOptions
	c = dataCommand("last-scheduled SONG", "Inspect the most recently scheduled plan item", 1, func(c *cobra.Command, x []string) error {
		var e error
		if id, _, _, err := services.ParseSelector("id:" + last.ServiceTypeID); err != nil || id == "" {
			return usageError{fmt.Errorf("--service-type requires a positive decimal ID")}
		}
		lastOpts, e = listOptions(c, lastOpts)
		if e != nil {
			return e
		}
		last.Song = x[0]
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if *j && lastOpts.All {
			return writeStreamList(c, func(yield func(services.AncillaryRecord) error) (services.ListMeta, error) {
				return a.library.WalkLastScheduledItems(c.Context(), last, yield)
			})
		}
		r, e := a.library.LastScheduledItems(c.Context(), last, lastOpts)
		if e != nil {
			return e
		}
		return writeList(c, *j, a, r)
	})
	visibility(c, &last.Visibility)
	c.Flags().StringVar(&last.ServiceTypeID, "service-type", "", "service type ID (required)")
	listFlags(c, &lastOpts)
	g.AddCommand(c)
	return g
}
func validOrder(s string) bool {
	if s == "" {
		return true
	}
	s = strings.TrimPrefix(s, "-")
	switch s {
	case "title", "created_at", "updated_at", "last_scheduled_at":
		return true
	}
	return false
}
