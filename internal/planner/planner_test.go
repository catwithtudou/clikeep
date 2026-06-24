package planner

import (
	"errors"
	"testing"

	"github.com/scott9/clikeep/internal/profile"
)

func TestBuildSelectsEnabledConfirmedProfiles(t *testing.T) {
	cfg := profile.Config{Tools: []profile.Profile{
		{Name: "enabled", Enabled: true, Confirmed: true, Update: profile.Command{Command: "echo", Args: []string{"ok"}}},
		{Name: "disabled", Enabled: false, Confirmed: true, Update: profile.Command{Command: "echo", Args: []string{"skip"}}},
		{Name: "unconfirmed", Enabled: true, Confirmed: false, Update: profile.Command{Command: "echo", Args: []string{"skip"}}},
	}}

	plan, problems := Build(cfg, nil, Options{LookupPath: func(string) (string, error) { return "/bin/echo", nil }})
	if len(problems) != 0 {
		t.Fatalf("problems = %#v, want none", problems)
	}
	if len(plan.Items) != 1 || plan.Items[0].Name != "enabled" {
		t.Fatalf("plan items = %#v, want only enabled", plan.Items)
	}
}

func TestBuildDoesNotOverrideDisabledByExplicitName(t *testing.T) {
	cfg := profile.Config{Tools: []profile.Profile{
		{Name: "disabled", Enabled: false, Confirmed: true, Update: profile.Command{Command: "echo", Args: []string{"skip"}}},
	}}

	plan, _ := Build(cfg, []string{"disabled"}, Options{LookupPath: func(string) (string, error) { return "/bin/echo", nil }})
	if len(plan.Items) != 0 {
		t.Fatalf("plan items = %#v, want none", plan.Items)
	}
}

func TestBuildReportsMissingUpdateCommandPath(t *testing.T) {
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "missing",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "missing-cli", Args: []string{"update"}},
	}}}

	plan, problems := Build(cfg, nil, Options{LookupPath: func(string) (string, error) {
		return "", errors.New("not found")
	}})
	if len(plan.Items) != 0 {
		t.Fatalf("plan items = %#v, want none", plan.Items)
	}
	if len(problems) != 1 || problems[0].Tool != "missing" {
		t.Fatalf("problems = %#v, want missing command problem", problems)
	}
}
