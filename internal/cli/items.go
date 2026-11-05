package cli

import (
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func bindItemFields(c *cobra.Command, b *bindings, p *services.ItemPatch, scope *services.PlanningScope) {
	b.str(c, "title", "item title; empty clears", &p.Title)
	b.str(c, "description", "item description; empty clears", &p.Description)
	b.str(c, "html-details", "item HTML details; empty clears", &p.HTMLDetails)
	b.length(c, &p.Length)
	b.str(c, "service-position", "pre, during, or post", &p.ServicePosition)
	labels := c.Flags().StringSlice("chart-sequence", nil, "arrangement section labels; empty clears")
	var song services.Scope
	c.Flags().StringVar(&song.Song, "song", "", "song ID or exact name; requires --arrangement")
	c.Flags().StringVar(&song.Arrangement, "arrangement", "", "arrangement ID or exact name; requires --song")
	c.Flags().StringVar(&song.Key, "key", "", "key record ID or exact name; requires --song and --arrangement")
	visibility(c, &song.Visibility)
	b.steps = append(b.steps, func(c *cobra.Command) error {
		if c.Flags().Changed("chart-sequence") {
			copy := append([]string{}, (*labels)...)
			p.ChartSequence = &copy
		}
		if c.Flags().Changed("song") || c.Flags().Changed("arrangement") || c.Flags().Changed("key") || c.Flags().Changed("visibility") {
			song.IncludeArchived = scope.IncludeArchived
			p.Song = &song
		}
		return nil
	})
}
func addItemMutationCommands(g *cobra.Command, j *bool) {
	var s services.PlanningScope
	var p services.ItemCreate
	b := newBindings()
	c := dataCommand("create", "Add an item, header, or explicitly selected song to a plan", 0, func(c *cobra.Command, _ []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := services.ValidateItemCreate(p); e != nil {
			return e
		}
		if e := checkPlanningScope(s, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.CreateItem(c.Context(), s, p)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, p)
	})
	planningSelection(c, &s, true)
	bindItemFields(c, b, &p.ItemPatch, &s)
	b.str(c, "type", "item or header; songs use --song", &p.Type)
	b.integer(c, "position", "initial sequence position, nonnegative; use reorder for existing items", &p.Position)
	g.AddCommand(c)
	var us services.PlanningScope
	var patch services.ItemPatch
	ub := newBindings()
	c = dataCommand("update ITEM", "Update item fields or its explicitly selected song", 1, func(c *cobra.Command, args []string) error {
		if e := ub.apply(c); e != nil {
			return e
		}
		if e := services.ValidateItemPatch(patch); e != nil {
			return e
		}
		if e := checkPlanningScope(us, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.UpdateItem(c.Context(), us, args[0], patch)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, patch)
	})
	planningSelection(c, &us, true)
	bindItemFields(c, ub, &patch, &us)
	g.AddCommand(c)
	var ds services.PlanningScope
	c = dataCommand("delete ITEM", "Delete an item and verify absence", 1, func(c *cobra.Command, args []string) error {
		if e := confirm(c); e != nil {
			return e
		}
		if e := checkPlanningScope(ds, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if e = a.library.DeleteItem(c.Context(), ds, args[0]); e != nil {
			return e
		}
		return writeMutationData(c, *j, a, receipt("Item", args[0]), nil)
	})
	planningSelection(c, &ds, true)
	deleteFlag(c)
	g.AddCommand(c)
	var rs services.PlanningScope
	var sequence []string
	c = dataCommand("reorder", "Replace a plan's complete item order with explicit IDs", 0, func(c *cobra.Command, _ []string) error {
		if e := services.ValidateItemSequence(sequence); e != nil {
			return e
		}
		if e := checkPlanningScope(rs, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ReorderItems(c.Context(), rs, sequence)
		if e != nil {
			return e
		}
		if e = writeList(c, *j, a, r); e != nil {
			return &committedOutputError{Result: r, Cause: e}
		}
		return nil
	})
	planningSelection(c, &rs, true)
	c.Flags().StringSliceVar(&sequence, "sequence", nil, "every item ID in desired order, comma separated (required)")
	g.AddCommand(c)
}
