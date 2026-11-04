package cli

import (
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func addPlanLifecycleCommands(g *cobra.Command, j *bool) {
	var s services.PlanningScope
	var p services.PlanCreate
	b := newBindings()
	c := dataCommand("create", "Create an undated plan; optional labels and public access", 0, func(c *cobra.Command, _ []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := services.ValidatePlanCreate(p); e != nil {
			return e
		}
		if e := checkPlanningScope(s, false); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.CreatePlan(c.Context(), s, p)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, p)
	})
	planningSelection(c, &s, false)
	b.str(c, "title", "plan title; empty creates an untitled plan", &p.Title)
	b.str(c, "series-title", "series label; conflicts with --series-id", &p.SeriesTitle)
	b.str(c, "series-id", "existing series ID; conflicts with --series-title", &p.SeriesID)
	b.boolean(c, "public", "enable or disable public access", &p.Public)
	g.AddCommand(c)
	var d services.PlanningScope
	c = dataCommand("delete PLAN", "Delete a plan and verify that it is no longer readable", 1, func(c *cobra.Command, args []string) error {
		if e := confirm(c); e != nil {
			return e
		}
		if e := checkPlanningScope(d, false); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if e = a.library.DeletePlan(c.Context(), d, args[0]); e != nil {
			return e
		}
		return writeMutationData(c, *j, a, receipt("Plan", args[0]), nil)
	})
	planningSelection(c, &d, false)
	deleteFlag(c)
	g.AddCommand(c)
}
