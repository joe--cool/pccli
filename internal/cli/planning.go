package cli

import (
	"context"
	"fmt"

	"github.com/joe--cool/pccli/v2/internal/output"
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func planningSelection(c *cobra.Command, s *services.PlanningScope, plan bool) {
	c.Flags().BoolVar(&s.IncludeArchived, "include-archived", false, "include returned archived service types and teams")
	c.Flags().StringVar(&s.ServiceType, "service-type", "", "service type ID or exact name (required)")
	if plan {
		c.Flags().StringVar(&s.Plan, "plan", "", "plan ID or exact title (required; IDs recommended for untitled plans)")
	}
}
func checkPlanningScope(s services.PlanningScope, plan bool) error {
	if s.ServiceType == "" {
		return usageError{fmt.Errorf("--service-type is required")}
	}
	if _, _, _, e := services.ParseSelector(s.ServiceType); e != nil {
		return e
	}
	if plan {
		if s.Plan == "" {
			return usageError{fmt.Errorf("--plan is required")}
		}
		if _, _, _, e := services.ParseSelector(s.Plan); e != nil {
			return e
		}
	}
	return nil
}
func planningListCommand[T any](j *bool, short string, scope bool, plan bool, list func(context.Context, *services.Library, services.PlanningScope, services.ListOptions) (services.ListResult[T], error), walk func(context.Context, *services.Library, services.PlanningScope, func(T) error) (services.ListMeta, error)) *cobra.Command {
	var s services.PlanningScope
	var o services.ListOptions
	c := dataCommand("list", short, 0, func(c *cobra.Command, _ []string) error {
		if scope {
			if e := checkPlanningScope(s, plan); e != nil {
				return e
			}
		}
		options, e := listOptions(c, o)
		if e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if *j && options.All {
			return writeStreamList(c, func(emit func(T) error) (services.ListMeta, error) { return walk(c.Context(), a.library, s, emit) })
		}
		result, e := list(c.Context(), a.library, s, options)
		if e != nil {
			return e
		}
		return writeList(c, *j, a, result)
	})
	if scope {
		planningSelection(c, &s, plan)
	} else {
		c.Flags().BoolVar(&s.IncludeArchived, "include-archived", false, "include returned archived service types")
	}
	listFlags(c, &o)
	return c
}
func newServiceTypesCommand(j *bool) *cobra.Command {
	g := group("service-types", "Discover service types for planning")
	g.AddCommand(planningListCommand(j, "List available service types", false, false,
		func(ctx context.Context, l *services.Library, s services.PlanningScope, o services.ListOptions) (services.ListResult[services.ServiceType], error) {
			return l.ListServiceTypes(ctx, services.ServiceTypeListOptions{ListOptions: o, IncludeArchived: s.IncludeArchived})
		},
		func(ctx context.Context, l *services.Library, s services.PlanningScope, emit func(services.ServiceType) error) (services.ListMeta, error) {
			return l.WalkServiceTypes(ctx, s.IncludeArchived, emit)
		}))
	var includeArchived bool
	show := dataCommand("show SERVICE_TYPE", "Show a service type", 1, func(c *cobra.Command, args []string) error {
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowServiceType(c.Context(), args[0], includeArchived)
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	show.Flags().BoolVar(&includeArchived, "include-archived", false, "include returned archived service types")
	g.AddCommand(show)
	return g
}
func newPlansCommand(j *bool) *cobra.Command {
	g := group("plans", "Create, inspect, update, and delete service plans")
	var s services.PlanningScope
	var o services.ListOptions
	var filter string
	c := dataCommand("list", "List plans within an explicit service type", 0, func(c *cobra.Command, _ []string) error {
		if e := checkPlanningScope(s, false); e != nil {
			return e
		}
		if filter != "" && filter != "future" && filter != "past" && filter != "no_dates" {
			return usageError{fmt.Errorf("--filter must be future, past, or no_dates")}
		}
		options, e := listOptions(c, o)
		if e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if *j && options.All {
			return writeStreamList(c, func(emit func(services.Plan) error) (services.ListMeta, error) {
				return a.library.WalkPlans(c.Context(), s, filter, emit)
			})
		}
		r, e := a.library.ListPlans(c.Context(), s, services.PlanListOptions{ListOptions: options, Filter: filter})
		if e != nil {
			return e
		}
		return writeList(c, *j, a, r)
	})
	planningSelection(c, &s, false)
	listFlags(c, &o)
	c.Flags().StringVar(&filter, "filter", "", "Planning Center plan filter: future, past, or no_dates")
	g.AddCommand(c)
	var show services.PlanningScope
	c = dataCommand("show PLAN", "Show plan dates, titles, and staffing summary", 1, func(c *cobra.Command, args []string) error {
		if e := checkPlanningScope(show, false); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowPlan(c.Context(), show, args[0])
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	planningSelection(c, &show, false)
	g.AddCommand(c)
	var update services.PlanningScope
	var patch services.PlanPatch
	b := newBindings()
	c = dataCommand("update PLAN", "Change plan labels, visibility, series, or reminders", 1, func(c *cobra.Command, args []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := checkPlanningScope(update, false); e != nil {
			return e
		}
		if e := services.ValidatePlanPatch(patch); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.UpdatePlan(c.Context(), update, args[0], patch)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, patch)
	})
	planningSelection(c, &update, false)
	b.str(c, "title", "plan title; empty explicitly clears", &patch.Title)
	b.str(c, "series-title", "series title; empty explicitly clears", &patch.SeriesTitle)
	b.str(c, "series-id", "existing series ID; conflicts with --series-title", &patch.SeriesID)
	b.boolean(c, "public", "enable or disable public access", &patch.Public)
	b.boolean(c, "reminders-disabled", "disable or enable plan reminders", &patch.RemindersDisabled)
	g.AddCommand(c)
	addPlanLifecycleCommands(g, j)
	return g
}
func newItemsCommand(j *bool) *cobra.Command {
	g := group("items", "Inspect a plan's order of service")
	g.AddCommand(planningListCommand(j, "List plan items in the order returned by Planning Center", true, true,
		func(ctx context.Context, l *services.Library, s services.PlanningScope, o services.ListOptions) (services.ListResult[services.Item], error) {
			return l.ListItems(ctx, s, o)
		},
		func(ctx context.Context, l *services.Library, s services.PlanningScope, emit func(services.Item) error) (services.ListMeta, error) {
			return l.WalkItems(ctx, s, emit)
		}))
	var s services.PlanningScope
	c := dataCommand("show ITEM", "Show an item within an explicit plan", 1, func(c *cobra.Command, args []string) error {
		if e := checkPlanningScope(s, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowItem(c.Context(), s, args[0])
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	planningSelection(c, &s, true)
	g.AddCommand(c)
	return g
}
func newTeamsCommand(j *bool) *cobra.Command {
	g := group("teams", "Inspect teams available to a service type")
	g.AddCommand(planningListCommand(j, "List teams associated with a service type", true, false,
		func(ctx context.Context, l *services.Library, s services.PlanningScope, o services.ListOptions) (services.ListResult[services.Team], error) {
			return l.ListTeams(ctx, s, o)
		},
		func(ctx context.Context, l *services.Library, s services.PlanningScope, emit func(services.Team) error) (services.ListMeta, error) {
			return l.WalkTeams(ctx, s, emit)
		}))
	var s services.PlanningScope
	c := dataCommand("show TEAM", "Show a team associated with a service type", 1, func(c *cobra.Command, args []string) error {
		if e := checkPlanningScope(s, false); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowTeam(c.Context(), s, args[0])
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	planningSelection(c, &s, false)
	g.AddCommand(c)
	return g
}
func newPlanTimesCommand(j *bool) *cobra.Command {
	g := group("plan-times", "Inspect service, rehearsal, and other plan times")
	g.AddCommand(planningListCommand(j, "List a plan's times", true, true,
		func(ctx context.Context, l *services.Library, s services.PlanningScope, o services.ListOptions) (services.ListResult[services.PlanTime], error) {
			return l.ListPlanTimes(ctx, s, o)
		},
		func(ctx context.Context, l *services.Library, s services.PlanningScope, emit func(services.PlanTime) error) (services.ListMeta, error) {
			return l.WalkPlanTimes(ctx, s, emit)
		}))
	return g
}
func newScheduledPeopleCommand(j *bool) *cobra.Command {
	g := group("scheduled-people", "Inspect plan assignments and response status")
	g.AddCommand(planningListCommand(j, "List people scheduled in a plan", true, true,
		func(ctx context.Context, l *services.Library, s services.PlanningScope, o services.ListOptions) (services.ListResult[services.ScheduledPerson], error) {
			return l.ListScheduledPeople(ctx, s, o)
		},
		func(ctx context.Context, l *services.Library, s services.PlanningScope, emit func(services.ScheduledPerson) error) (services.ListMeta, error) {
			return l.WalkScheduledPeople(ctx, s, emit)
		}))
	return g
}

func planningTable[T any](r output.Renderer, items []T) (bool, error) {
	var columns []string
	switch any(items[0]).(type) {
	case services.ServiceType:
		columns = []string{"id", "name", "frequency", "archived_at"}
	case services.Plan:
		columns = []string{"id", "dates", "title", "series_title", "items_count", "plan_people_count", "needed_positions_count"}
	case services.Item:
		columns = []string{"id", "sequence", "service_position", "item_type", "title", "song_id", "length"}
	case services.Team:
		columns = []string{"id", "name", "schedule_to", "archived_at"}
	case services.PlanTime:
		columns = []string{"id", "time_type", "name", "starts_at", "ends_at"}
	case services.ScheduledPerson:
		columns = []string{"id", "person_id", "team_id", "name", "team_position_name", "status"}
	default:
		return false, nil
	}
	headers := make([]string, 0, len(columns))
	for _, name := range columns {
		header := label(name)
		if concise, ok := map[string]string{"items_count": "Items", "plan_people_count": "Scheduled", "needed_positions_count": "Needed", "service_position": "Position", "item_type": "Type", "time_type": "Type", "length": "Seconds", "starts_at": "Starts", "ends_at": "Ends", "team_position_name": "Position"}[name]; ok {
			header = concise
		}
		headers = append(headers, header)
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		m := object(item)
		row := make([]string, 0, len(columns))
		for _, name := range columns {
			value := text(m[name])
			if name == "status" {
				switch value {
				case "C":
					value = "Confirmed"
				case "U":
					value = "Unconfirmed"
				case "D":
					value = "Declined"
				}
			}
			if n, ok := m[name].(float64); ok {
				value = fmt.Sprintf("%v", n)
			}
			if name == "title" && value == "" {
				value = "(untitled)"
			}
			row = append(row, value)
		}
		rows = append(rows, row)
	}
	return true, r.Table(headers, rows)
}
