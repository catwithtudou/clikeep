package executor

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/scott9/clikeep/internal/planner"
	"github.com/scott9/clikeep/internal/profile"
)

func TestRunContinuesAfterFailure(t *testing.T) {
	plan := planner.Plan{Items: []planner.Item{
		{Name: "bad", Update: profile.Command{Command: "bad"}},
		{Name: "good", Update: profile.Command{Command: "good"}},
	}}
	calls := []string{}

	summary, err := Run(context.Background(), plan, Options{
		StateDir: t.TempDir(),
		RunCommand: func(ctx context.Context, cmd profile.Command) CommandResult {
			calls = append(calls, cmd.Command)
			if cmd.Command == "bad" {
				return CommandResult{Stderr: []byte("boom"), ExitCode: 1, Err: errors.New("failed")}
			}
			return CommandResult{Stdout: []byte("ok"), ExitCode: 0}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %#v, want both commands", calls)
	}
	if summary.Results[0].Status != "failed" || summary.Results[1].Status != "success" {
		t.Fatalf("results = %#v", summary.Results)
	}
	if _, err := os.Stat(summary.Results[0].LogPath); err != nil {
		t.Fatalf("log path missing: %v", err)
	}
}

func TestRunFailFastSkipsRemainingItems(t *testing.T) {
	plan := planner.Plan{Items: []planner.Item{
		{Name: "bad", Update: profile.Command{Command: "bad"}},
		{Name: "skipped", Update: profile.Command{Command: "skipped"}},
	}}
	calls := []string{}

	summary, err := Run(context.Background(), plan, Options{
		StateDir: t.TempDir(),
		FailFast: true,
		RunCommand: func(ctx context.Context, cmd profile.Command) CommandResult {
			calls = append(calls, cmd.Command)
			return CommandResult{ExitCode: 1, Err: errors.New("failed")}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %#v, want fail-fast to stop after first command", calls)
	}
	if summary.Results[0].Status != "failed" || summary.Results[1].Status != "skipped" {
		t.Fatalf("results = %#v", summary.Results)
	}
}
