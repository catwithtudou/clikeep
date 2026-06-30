package executor

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/catwithtudou/clikeep/internal/planner"
	"github.com/catwithtudou/clikeep/internal/profile"
)

func TestRunContinuesAfterFailure(t *testing.T) {
	plan := planner.Plan{Items: []planner.Item{
		{Name: "bad", Update: profile.Command{Command: "bad"}},
		{Name: "good", Update: profile.Command{Command: "good"}},
	}}
	var mu sync.Mutex
	calls := []string{}

	summary, err := Run(context.Background(), plan, Options{
		StateDir: t.TempDir(),
		RunCommand: func(ctx context.Context, cmd profile.Command) CommandResult {
			mu.Lock()
			calls = append(calls, cmd.Command)
			mu.Unlock()
			if cmd.Command == "bad" {
				return CommandResult{Stderr: []byte("boom"), ExitCode: 1, Err: errors.New("failed")}
			}
			return CommandResult{Stdout: []byte("ok"), ExitCode: 0}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	gotCalls := len(calls)
	mu.Unlock()
	if gotCalls != 2 {
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

func TestRunWritesProgressForRunningSuccessAndSkipped(t *testing.T) {
	plan := planner.Plan{Items: []planner.Item{
		{Name: "bad", Update: profile.Command{Command: "bad"}},
		{Name: "skipped", Update: profile.Command{Command: "skipped"}},
	}}
	var progress bytes.Buffer

	_, err := Run(context.Background(), plan, Options{
		StateDir: t.TempDir(),
		FailFast: true,
		Progress: &progress,
		RunCommand: func(ctx context.Context, cmd profile.Command) CommandResult {
			return CommandResult{ExitCode: 1, Err: errors.New("failed")}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := progress.String()
	for _, want := range []string{"bad", "running", "failed", "skipped"} {
		if !strings.Contains(out, want) {
			t.Fatalf("progress = %q, missing %q", out, want)
		}
	}
}

func TestRunRunsItemsConcurrentlyByDefault(t *testing.T) {
	plan := planner.Plan{Items: []planner.Item{
		{Name: "first", Update: profile.Command{Command: "first"}},
		{Name: "second", Update: profile.Command{Command: "second"}},
	}}
	started := make(chan string, 2)
	release := make(chan struct{})
	done := make(chan struct{})
	var summaryErr error
	stateDir := t.TempDir()

	go func() {
		_, summaryErr = Run(context.Background(), plan, Options{
			StateDir: stateDir,
			RunCommand: func(ctx context.Context, cmd profile.Command) CommandResult {
				started <- cmd.Command
				<-release
				return CommandResult{}
			},
		})
		close(done)
	}()

	got := []string{waitForStarted(t, started), waitForStarted(t, started)}
	close(release)
	waitForDone(t, done)
	if summaryErr != nil {
		t.Fatal(summaryErr)
	}
	if strings.Join(got, ",") != "first,second" && strings.Join(got, ",") != "second,first" {
		t.Fatalf("started = %#v, want first and second", got)
	}
}

func TestRunConcurrentResultsKeepPlanOrder(t *testing.T) {
	plan := planner.Plan{Items: []planner.Item{
		{Name: "slow", Update: profile.Command{Command: "slow"}},
		{Name: "fast", Update: profile.Command{Command: "fast"}},
	}}
	releaseSlow := make(chan struct{})
	fastDone := make(chan struct{})

	done := make(chan struct{})
	var summaryErr error
	var summaryNames []string
	stateDir := t.TempDir()
	go func() {
		summary, err := Run(context.Background(), plan, Options{
			StateDir: stateDir,
			Jobs:     2,
			RunCommand: func(ctx context.Context, cmd profile.Command) CommandResult {
				if cmd.Command == "slow" {
					<-releaseSlow
					return CommandResult{}
				}
				close(fastDone)
				return CommandResult{}
			},
		})
		summaryErr = err
		for _, result := range summary.Results {
			summaryNames = append(summaryNames, result.Name)
		}
		close(done)
	}()
	waitForClosed(t, fastDone)
	close(releaseSlow)
	waitForDone(t, done)
	if summaryErr != nil {
		t.Fatal(summaryErr)
	}
	if strings.Join(summaryNames, ",") != "slow,fast" {
		t.Fatalf("result names = %#v, want plan order", summaryNames)
	}
}

func TestRunConcurrentFailFastSkipsUnstartedItems(t *testing.T) {
	plan := planner.Plan{Items: []planner.Item{
		{Name: "bad", Update: profile.Command{Command: "bad"}},
		{Name: "slow", Update: profile.Command{Command: "slow"}},
		{Name: "skipped", Update: profile.Command{Command: "skipped"}},
	}}
	slowStarted := make(chan struct{})
	releaseSlow := make(chan struct{})
	var mu sync.Mutex
	var calls []string
	progress := &signalWriter{pattern: "bad", status: "failed", signal: make(chan struct{})}

	done := make(chan struct{})
	var summaryErr error
	var statuses []string
	stateDir := t.TempDir()
	go func() {
		summary, err := Run(context.Background(), plan, Options{
			StateDir: stateDir,
			FailFast: true,
			Jobs:     2,
			RunCommand: func(ctx context.Context, cmd profile.Command) CommandResult {
				mu.Lock()
				calls = append(calls, cmd.Command)
				mu.Unlock()
				switch cmd.Command {
				case "bad":
					<-slowStarted
					return CommandResult{ExitCode: 1, Err: errors.New("failed")}
				case "slow":
					close(slowStarted)
					<-releaseSlow
					return CommandResult{}
				default:
					return CommandResult{}
				}
			},
			Progress: progress,
		})
		summaryErr = err
		for _, result := range summary.Results {
			statuses = append(statuses, result.Status)
		}
		close(done)
	}()

	waitForClosed(t, progress.signal)
	close(releaseSlow)
	waitForDone(t, done)
	if summaryErr != nil {
		t.Fatal(summaryErr)
	}
	mu.Lock()
	gotCalls := strings.Join(calls, ",")
	mu.Unlock()
	if strings.Contains(gotCalls, "skipped") {
		t.Fatalf("calls = %q, skipped item should not run", gotCalls)
	}
	if strings.Join(statuses, ",") != "failed,success,skipped" {
		t.Fatalf("statuses = %#v, want failed, success, skipped", statuses)
	}
}

type signalWriter struct {
	pattern string
	status  string
	signal  chan struct{}
	once    sync.Once
}

func (w *signalWriter) Write(data []byte) (int, error) {
	text := string(data)
	if strings.Contains(text, w.pattern) && strings.Contains(text, w.status) {
		w.once.Do(func() {
			close(w.signal)
		})
	}
	return len(data), nil
}

func waitForStarted(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for command to start")
		return ""
	}
}

func waitForClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for signal")
	}
}

func waitForDone(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for run to finish")
	}
}
