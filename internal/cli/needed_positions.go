package cli

import (
	"context"

	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func newNeededPositionsCommand(j *bool) *cobra.Command {
	g := group("needed-positions", "Manage whole-plan unfilled team positions")
	g.AddCommand(planningListCommand(j, "List needed positions within a plan", true, true,
		func(ctx context.Context, l *services.Library, s services.PlanningScope, o services.ListOptions) (services.ListResult[services.NeededPosition], error) {
			return l.ListNeededPositions(ctx, s, o)
		},
		func(ctx context.Context, l *services.Library, s services.PlanningScope, emit func(services.NeededPosition) error) (services.ListMeta, error) {
			return l.WalkNeededPositions(ctx, s, emit)
		}))
	var show services.PlanningScope
	c := dataCommand("show NEED", "Show one needed-position record", 1, func(c *cobra.Command, args []string) error {
		if e := checkPlanningScope(show, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowNeededPosition(c.Context(), show, args[0])
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	planningSelection(c, &show, true)
	g.AddCommand(c)
	var s services.PlanningScope
	var p services.NeededPositionCreate
	b := newBindings()
	c = dataCommand("create", "Add a whole-plan needed position with an explicit quantity", 0, func(c *cobra.Command, _ []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := services.ValidateNeededPositionCreate(p); e != nil {
			return e
		}
		if e := checkPlanningScope(s, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.CreateNeededPosition(c.Context(), s, p)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, p)
	})
	planningSelection(c, &s, true)
	c.Flags().StringVar(&p.Team, "team", "", "whole-plan team ID or exact name (required)")
	c.Flags().StringVar(&p.Position, "position", "", "team position ID or exact name (required)")
	b.integer(c, "quantity", "number of needed people, nonnegative (required)", &p.Quantity)
	g.AddCommand(c)
	var us services.PlanningScope
	var patch services.NeededPositionPatch
	ub := newBindings()
	c = dataCommand("update NEED", "Replace a whole-plan needed quantity", 1, func(c *cobra.Command, args []string) error {
		if e := ub.apply(c); e != nil {
			return e
		}
		if e := services.ValidateNeededPositionPatch(patch); e != nil {
			return e
		}
		if e := checkPlanningScope(us, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.UpdateNeededPosition(c.Context(), us, args[0], patch)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, patch)
	})
	planningSelection(c, &us, true)
	ub.integer(c, "quantity", "replacement quantity, nonnegative (required)", &patch.Quantity)
	g.AddCommand(c)
	var ds services.PlanningScope
	c = dataCommand("delete NEED", "Delete one whole-plan needed-position record", 1, func(c *cobra.Command, args []string) error {
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
		if e = a.library.DeleteNeededPosition(c.Context(), ds, args[0]); e != nil {
			return e
		}
		return writeMutationData(c, *j, a, receipt("NeededPosition", args[0]), nil)
	})
	planningSelection(c, &ds, true)
	deleteFlag(c)
	g.AddCommand(c)
	return g
}
