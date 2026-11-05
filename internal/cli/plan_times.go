package cli

import (
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func bindPlanTimeFields(c *cobra.Command, b *bindings, p *services.PlanTimePatch) {
	b.str(c, "name", "time label; empty clears", &p.Name)
	b.str(c, "starts-at", "RFC3339 start timestamp with offset", &p.StartsAt)
	b.str(c, "ends-at", "RFC3339 end timestamp with offset", &p.EndsAt)
	b.str(c, "time-type", "service, rehearsal, or other", &p.TimeType)
}
func addPlanTimeCommands(g *cobra.Command, j *bool) {
	var show services.PlanningScope
	c := dataCommand("show TIME", "Show a time within an explicit plan", 1, func(c *cobra.Command, args []string) error {
		if e := checkPlanningScope(show, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowPlanTime(c.Context(), show, args[0])
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	planningSelection(c, &show, true)
	g.AddCommand(c)
	var s services.PlanningScope
	var p services.PlanTimePatch
	b := newBindings()
	c = dataCommand("create", "Add an explicitly dated service, rehearsal, or other time", 0, func(c *cobra.Command, _ []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := services.ValidatePlanTimeCreate(services.PlanTimeCreate(p)); e != nil {
			return e
		}
		if e := checkPlanningScope(s, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.CreatePlanTime(c.Context(), s, services.PlanTimeCreate(p))
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, p)
	})
	planningSelection(c, &s, true)
	bindPlanTimeFields(c, b, &p)
	g.AddCommand(c)
	var us services.PlanningScope
	var patch services.PlanTimePatch
	ub := newBindings()
	c = dataCommand("update TIME", "Change one plan time without editing later plans", 1, func(c *cobra.Command, args []string) error {
		if e := ub.apply(c); e != nil {
			return e
		}
		if e := services.ValidatePlanTimePatch(patch); e != nil {
			return e
		}
		if e := checkPlanningScope(us, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.UpdatePlanTime(c.Context(), us, args[0], patch)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, patch)
	})
	planningSelection(c, &us, true)
	bindPlanTimeFields(c, ub, &patch)
	g.AddCommand(c)
	var ds services.PlanningScope
	c = dataCommand("delete TIME", "Delete one plan time and verify absence", 1, func(c *cobra.Command, args []string) error {
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
		if e = a.library.DeletePlanTime(c.Context(), ds, args[0]); e != nil {
			return e
		}
		return writeMutationData(c, *j, a, receipt("PlanTime", args[0]), nil)
	})
	planningSelection(c, &ds, true)
	deleteFlag(c)
	g.AddCommand(c)
}
