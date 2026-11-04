package cli

import (
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func addPlanCopyCommand(g *cobra.Command, j *bool) {
	var scope services.PlanningScope
	var options services.PlanCopyOptions
	b := newBindings()
	c := dataCommand("copy SOURCE", "Copy selected plan components into a new plan in the same service type", 1, func(c *cobra.Command, args []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := services.ValidatePlanCopyOptions(options); e != nil {
			return e
		}
		if e := checkPlanningScope(scope, false); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		result, e := a.library.CopyPlan(c.Context(), scope, args[0], options)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, result, nil)
	})
	planningSelection(c, &scope, false)
	visibility(c, &options.Visibility)
	b.str(c, "title", "new plan title; omitted copies the available source title", &options.Title)
	c.Flags().BoolVar(&options.CopyItems, "copy-items", false, "copy supported items and their order; excludes files/media/layouts")
	c.Flags().BoolVar(&options.CopyTimes, "copy-times", false, "copy times; requires time-offset or preserve-dates")
	c.Flags().BoolVar(&options.CopyPeople, "copy-people", false, "copy whole-plan members as unconfirmed without preparing notifications")
	c.Flags().BoolVar(&options.CopyNeededPositions, "copy-needed-positions", false, "copy whole-plan needed quantities; position IDs remain unverified")
	c.Flags().BoolVar(&options.CopySettings, "copy-settings", false, "copy known public access and reminder settings; otherwise public is set false")
	b.str(c, "time-offset", "elapsed Go duration such as 168h, -24h, or 0; requires copy-times", &options.TimeOffset)
	c.Flags().BoolVar(&options.PreserveDates, "preserve-dates", false, "retain source dates when copying times; conflicts with time-offset")
	g.AddCommand(c)
}
