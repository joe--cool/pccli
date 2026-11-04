package cli

import (
	"fmt"

	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func newKeysCommand(j *bool) *cobra.Command {
	g := group("keys", "Maintain named musical keys within an arrangement")
	var s services.Scope
	var o services.ListOptions
	c := dataCommand("list", "List keys in an explicit arrangement", 0, func(c *cobra.Command, _ []string) error {
		if e := checkScope(s, true); e != nil {
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
			return writeStreamList(c, func(yield func(services.Key) error) (services.ListMeta, error) {
				return a.library.WalkKeys(c.Context(), s, yield)
			})
		}
		r, e := a.library.ListKeys(c.Context(), s, o)
		if e != nil {
			return e
		}
		return writeList(c, *j, a, r)
	})
	selection(c, &s, true, false)
	listFlags(c, &o)
	g.AddCommand(c)
	var sh services.Scope
	c = dataCommand("show KEY", "Show one key record", 1, func(c *cobra.Command, x []string) error {
		if e := checkScope(sh, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowKey(c.Context(), sh, x[0])
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	selection(c, &sh, true, false)
	g.AddCommand(c)
	var cs services.Scope
	var m services.KeyCreate
	b := newBindings()
	c = dataCommand("create", "Create a named key for an arrangement", 0, func(c *cobra.Command, _ []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := checkScope(cs, true); e != nil {
			return e
		}
		if m.StartingKey == nil {
			return usageError{fmt.Errorf("--start is required")}
		}
		if m.EndingKey == nil {
			m.EndingKey = m.StartingKey
		}
		if e := services.ValidateMutation(m, "Key"); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.CreateKey(c.Context(), cs, m)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, m)
	})
	selection(c, &cs, true, false)
	c.Flags().StringVar(&m.Name, "name", "", "key record name (required)")
	b.str(c, "start", "starting pitch, e.g. Bb or Cm", &m.StartingKey)
	b.str(c, "end", "ending pitch; defaults to start on create", &m.EndingKey)
	bindExtraFields(c, b, &m)
	g.AddCommand(c)
	var us services.Scope
	var p services.KeyPatch
	b2 := newBindings()
	c = dataCommand("update KEY", "Update only supplied key metadata", 1, func(c *cobra.Command, x []string) error {
		if e := b2.apply(c); e != nil {
			return e
		}
		if e := checkScope(us, true); e != nil {
			return e
		}
		if e := services.ValidateMutation(p, "Key"); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.UpdateKey(c.Context(), us, x[0], p)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, p)
	})
	selection(c, &us, true, false)
	b2.str(c, "name", "key record name", &p.Name)
	b2.str(c, "start", "starting musical pitch", &p.StartingKey)
	b2.str(c, "end", "ending musical pitch", &p.EndingKey)
	bindExtraFields(c, b2, &p)
	g.AddCommand(c)
	var ds services.Scope
	c = dataCommand("delete KEY", "Permanently delete a key record", 1, func(c *cobra.Command, x []string) error {
		if e := confirm(c); e != nil {
			return e
		}
		if e := checkScope(ds, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if e = a.library.DeleteKey(c.Context(), ds, x[0]); e != nil {
			return e
		}
		return writeMutationData(c, *j, a, receipt("key", x[0]), nil)
	})
	selection(c, &ds, true, false)
	deleteFlag(c)
	g.AddCommand(c)
	return g
}
