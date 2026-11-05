package cli

import (
	"context"
	"fmt"

	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func teamSelection(c *cobra.Command, s *services.TeamScope) {
	c.Flags().StringVar(&s.ServiceType, "service-type", "", "service type ID or exact name (required)")
	c.Flags().StringVar(&s.Team, "team", "", "team ID or exact name (required)")
	c.Flags().BoolVar(&s.IncludeArchived, "include-archived", false, "include returned archived service types, teams, and candidate people")
}
func checkTeamSelection(s services.TeamScope) error {
	if s.ServiceType == "" || s.Team == "" {
		return usageError{fmt.Errorf("--service-type and --team are required")}
	}
	for _, value := range []string{s.ServiceType, s.Team} {
		if _, _, _, e := services.ParseSelector(value); e != nil {
			return e
		}
	}
	return nil
}
func teamCatalogCommand[T any](j *bool, use, short string, list func(context.Context, *services.Library, services.TeamScope, services.ListOptions) (services.ListResult[T], error), walk func(context.Context, *services.Library, services.TeamScope, func(T) error) (services.ListMeta, error), show func(context.Context, *services.Library, services.TeamScope, string) (T, error)) *cobra.Command {
	g := group(use, short)
	var s services.TeamScope
	var o services.ListOptions
	c := dataCommand("list", short, 0, func(c *cobra.Command, _ []string) error {
		if e := checkTeamSelection(s); e != nil {
			return e
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
	teamSelection(c, &s)
	listFlags(c, &o)
	g.AddCommand(c)
	var ss services.TeamScope
	c = dataCommand("show TARGET", "Show one record within an explicitly selected team", 1, func(c *cobra.Command, args []string) error {
		if e := checkTeamSelection(ss); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		result, e := show(c.Context(), a.library, ss, args[0])
		if e != nil {
			return e
		}
		return writeData(c, *j, a, result, nil)
	})
	teamSelection(c, &ss)
	g.AddCommand(c)
	return g
}
func newTeamPeopleCommand(j *bool) *cobra.Command {
	return teamCatalogCommand(j, "team-people", "Discover team members as scheduling candidates",
		func(ctx context.Context, l *services.Library, s services.TeamScope, o services.ListOptions) (services.ListResult[services.TeamPerson], error) {
			return l.ListTeamPeople(ctx, s, o)
		},
		func(ctx context.Context, l *services.Library, s services.TeamScope, emit func(services.TeamPerson) error) (services.ListMeta, error) {
			return l.WalkTeamPeople(ctx, s, emit)
		},
		func(ctx context.Context, l *services.Library, s services.TeamScope, target string) (services.TeamPerson, error) {
			return l.ShowTeamPerson(ctx, s, target)
		})
}
func newTeamPositionsCommand(j *bool) *cobra.Command {
	return teamCatalogCommand(j, "team-positions", "Discover positions within a team",
		func(ctx context.Context, l *services.Library, s services.TeamScope, o services.ListOptions) (services.ListResult[services.TeamPosition], error) {
			return l.ListTeamPositions(ctx, s, o)
		},
		func(ctx context.Context, l *services.Library, s services.TeamScope, emit func(services.TeamPosition) error) (services.ListMeta, error) {
			return l.WalkTeamPositions(ctx, s, emit)
		},
		func(ctx context.Context, l *services.Library, s services.TeamScope, target string) (services.TeamPosition, error) {
			return l.ShowTeamPosition(ctx, s, target)
		})
}
func bindAssignmentFields(c *cobra.Command, b *bindings, status, notes, reason **string) {
	b.str(c, "status", "unconfirmed, confirmed, declined, U, C, or D", status)
	b.str(c, "notes", "assignment notes; empty clears", notes)
	b.str(c, "decline-reason", "decline reason; empty clears", reason)
}
func addScheduledPeopleCommands(g *cobra.Command, j *bool) {
	var show services.PlanningScope
	c := dataCommand("show ASSIGNMENT", "Show one person's assignment within a plan", 1, func(c *cobra.Command, args []string) error {
		if e := checkPlanningScope(show, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.ShowScheduledPerson(c.Context(), show, args[0])
		if e != nil {
			return e
		}
		return writeData(c, *j, a, r, nil)
	})
	planningSelection(c, &show, true)
	g.AddCommand(c)
	var s services.PlanningScope
	var p services.ScheduledPersonCreate
	b := newBindings()
	c = dataCommand("create", "Schedule a whole-plan team member without preparing a notification", 0, func(c *cobra.Command, _ []string) error {
		if e := b.apply(c); e != nil {
			return e
		}
		if e := services.ValidateScheduledPersonCreate(p); e != nil {
			return e
		}
		if e := checkPlanningScope(s, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.CreateScheduledPerson(c.Context(), s, p)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, p)
	})
	planningSelection(c, &s, true)
	c.Flags().StringVar(&p.Team, "team", "", "whole-plan team ID or exact name (required)")
	c.Flags().StringVar(&p.Person, "person", "", "member ID or exact full name (required)")
	c.Flags().StringVar(&p.Position, "position", "", "team position ID or exact name (required)")
	bindAssignmentFields(c, b, &p.Status, &p.Notes, &p.DeclineReason)
	g.AddCommand(c)
	var us services.PlanningScope
	var patch services.ScheduledPersonPatch
	ub := newBindings()
	c = dataCommand("update ASSIGNMENT", "Update a whole-plan assignment's position, status, or notes", 1, func(c *cobra.Command, args []string) error {
		if e := ub.apply(c); e != nil {
			return e
		}
		if c.Flags().Changed("team") && patch.Team == "" {
			return usageError{fmt.Errorf("--team cannot be empty")}
		}
		if e := services.ValidateScheduledPersonPatch(patch); e != nil {
			return e
		}
		if e := checkPlanningScope(us, true); e != nil {
			return e
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		r, e := a.library.UpdateScheduledPerson(c.Context(), us, args[0], patch)
		if e != nil {
			return e
		}
		return writeMutationData(c, *j, a, r, patch)
	})
	planningSelection(c, &us, true)
	c.Flags().StringVar(&patch.Team, "team", "", "optional exact team guard; does not move the assignment")
	ub.str(c, "position", "replacement team position ID or exact name", &patch.Position)
	bindAssignmentFields(c, ub, &patch.Status, &patch.Notes, &patch.DeclineReason)
	g.AddCommand(c)
	var ds services.PlanningScope
	var team string
	c = dataCommand("delete ASSIGNMENT", "Remove a whole-plan assignment and verify absence", 1, func(c *cobra.Command, args []string) error {
		if e := confirm(c); e != nil {
			return e
		}
		if e := checkPlanningScope(ds, true); e != nil {
			return e
		}
		if c.Flags().Changed("team") {
			if _, _, _, e := services.ParseSelector(team); e != nil {
				return e
			}
		}
		a, e := loadApp(c)
		if e != nil {
			return e
		}
		if e = a.library.DeleteScheduledPerson(c.Context(), ds, args[0], team); e != nil {
			return e
		}
		return writeMutationData(c, *j, a, receipt("PlanPerson", args[0]), nil)
	})
	planningSelection(c, &ds, true)
	c.Flags().StringVar(&team, "team", "", "optional exact team guard")
	deleteFlag(c)
	g.AddCommand(c)
}
