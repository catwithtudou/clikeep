package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/catwithtudou/clikeep/internal/planner"
	"github.com/catwithtudou/clikeep/internal/profile"
)

func TestWritePlanTextUsesStagedShape(t *testing.T) {
	var buf bytes.Buffer
	plan := planner.Plan{Items: []planner.Item{{
		Name:         "demo-cli",
		Update:       profile.Command{Command: "demo-cli", Args: []string{"update"}},
		ResolvedPath: "/usr/local/bin/demo-cli",
	}}}
	if err := WritePlanText(&buf, plan, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"Plan",
		"  1. demo-cli",
		"     command: demo-cli update",
		"     path:    /usr/local/bin/demo-cli",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output = %q, missing %q", out, want)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("plain output contains ANSI: %q", out)
	}
}

func TestWritePlanTextUsesANSIWhenColorEnabled(t *testing.T) {
	var buf bytes.Buffer
	plan := planner.Plan{Items: []planner.Item{{
		Name:         "demo-cli",
		Update:       profile.Command{Command: "demo-cli", Args: []string{"update"}},
		ResolvedPath: "/usr/local/bin/demo-cli",
	}}}
	if err := WritePlanText(&buf, plan, true); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("color output = %q, want ANSI", out)
	}
	if !strings.Contains(out, "demo-cli") || !strings.Contains(out, "/usr/local/bin/demo-cli") {
		t.Fatalf("color output = %q, missing plan text", out)
	}
}

func TestProgressLineColorsStatus(t *testing.T) {
	line := ProgressLine(NewStyle(true), 1, 2, "demo-cli", "success")
	for _, want := range []string{"[1/2]", "demo-cli", "\x1b[32msuccess\x1b[0m"} {
		if !strings.Contains(line, want) {
			t.Fatalf("line = %q, missing %q", line, want)
		}
	}
	if strings.Contains(line, "#####") {
		t.Fatalf("line = %q, want status-first progress without a sequential bar", line)
	}
}
