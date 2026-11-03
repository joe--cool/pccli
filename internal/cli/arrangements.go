package cli

import (
	"fmt"

	"github.com/joe--cool/pccli/internal/services"
	"github.com/spf13/cobra"
)

func newArrangementsCommand(j *bool) *cobra.Command {
	g := group("arrangements", "Maintain song versions and chord charts")
	var s services.Scope
	var o services.ListOptions
	c := dataCommand("list", "List arrangements within a song", 0, func(c *cobra.Command, _ []string) error {
		if e := checkScope(s, false); e != nil {
			return e
		}
		var e error
		o, e = listOptions(c, o)
		if e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if *j && o.All {
			return writeStreamList(c, func(yield func(services.Arrangement) error) (services.ListMeta, error) {
				return a.library.WalkArrangements(c.Context(), s, yield)
			})
		}
		r, e := a.library.ListArrangements(c.Context(), s, o)
		if e != nil {
			return e
		}
		return writeList(c, *j, a, r)
	})
	selection(c, &s, false, false)
	listFlags(c, &o)
	g.AddCommand(c)
	var sh services.Scope
	c = dataCommand("show ARRANGEMENT", "Show one arrangement and its keys", 1, func(c *cobra.Command, x []string) error {
		if e := checkScope(sh, false); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowArrangement(c.Context(), sh, x[0])
		if e != nil {
			return e
		}
		return writeArrangementDetail(c, *j, a, r)
	})
	selection(c, &sh, false, false)
	g.AddCommand(c)
	var cs services.Scope
	var m services.ArrangementCreate
	b := newBindings()
	c = dataCommand("create", "Create an explicitly scoped arrangement", 0, func(c *cobra.Command, _ []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := checkScope(cs, false); e != nil {
			return e
		}
		if e := services.ValidateMutation(m, "Arrangement"); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.CreateArrangement(c.Context(), cs, m)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, m)
	})
	selection(c, &cs, false, false)
	c.Flags().StringVar(&m.Name, "name", "", "arrangement name (required)")
	bindArrangement(c, b, &m.BPM, &m.Meter, &m.ChordChartKey, &m.Notes, &m.Length, &m.LyricsEnabled, &m.Sequence)
	bindExtraFields(c, b, &m)
	g.AddCommand(c)
	var us services.Scope
	var p services.ArrangementPatch
	b2 := newBindings()
	c = dataCommand("update ARRANGEMENT", "Update arrangement metadata or chart settings", 1, func(c *cobra.Command, x []string) error {
		if e := b2.apply(c); e != nil {
			return e
		}
		if e := checkScope(us, false); e != nil {
			return e
		}
		if e := services.ValidateMutation(p, "Arrangement"); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.UpdateArrangement(c.Context(), us, x[0], p)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, p)
	})
	selection(c, &us, false, false)
	b2.str(c, "name", "arrangement name", &p.Name)
	bindArrangement(c, b2, &p.BPM, &p.Meter, &p.ChordChartKey, &p.Notes, &p.Length, &p.LyricsEnabled, &p.Sequence)
	bindExtraFields(c, b2, &p)
	g.AddCommand(c)
	var ds services.Scope
	c = dataCommand("delete ARRANGEMENT", "Permanently delete an arrangement", 1, func(c *cobra.Command, x []string) error {
		if e := confirm(c); e != nil {
			return e
		}
		if e := checkScope(ds, false); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if e = a.library.DeleteArrangement(c.Context(), ds, x[0]); e != nil {
			return e
		}
		return writeMutationData(c, *j, a, receipt("arrangement", x[0]), nil)
	})
	selection(c, &ds, false, false)
	deleteFlag(c)
	g.AddCommand(c)
	for _, act := range []string{"archive", "unarchive"} {
		var scope services.Scope
		action := act
		c = dataCommand(action+" ARRANGEMENT", action+" a song arrangement", 1, func(c *cobra.Command, x []string) error {
			if action == "unarchive" && !scope.IncludeArchived {
				return usageError{fmt.Errorf("unarchive requires --include-archived")}
			}
			if e := checkScope(scope, false); e != nil {
				return e
			}
			a, e := loadApp(c)
			if e != nil {
				return e
			}
			var r services.Arrangement
			if action == "archive" {
				r, e = a.library.ArchiveArrangement(c.Context(), scope, x[0])
			} else {
				r, e = a.library.UnarchiveArrangement(c.Context(), scope, x[0])
			}
			if e != nil {
				return e
			}
			return writeMutationData(c, *j, a, r, nil)
		})
		selection(c, &scope, false, false)
		g.AddCommand(c)
	}
	var sections services.Scope
	c = dataCommand("sections ARRANGEMENT", "Inspect available arrangement sections", 1, func(c *cobra.Command, x []string) error {
		if e := checkScope(sections, false); e != nil {
			return e
		}
		sections.Arrangement = x[0]
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ArrangementSections(c.Context(), sections)
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	selection(c, &sections, false, false)
	g.AddCommand(c)
	return g
}
