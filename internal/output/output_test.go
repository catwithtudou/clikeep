package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/scott9/clikeep/internal/planner"
	"github.com/scott9/clikeep/internal/profile"
)

func TestWritePlanTextUsesStagedShape(t *testing.T) {
	var buf bytes.Buffer
	plan := planner.Plan{Items: []planner.Item{{
		Name:         "lark-cli",
		Update:       profile.Command{Command: "lark-cli", Args: []string{"update"}},
		ResolvedPath: "/opt/homebrew/bin/lark-cli",
	}}}
	if err := WritePlanText(&buf, plan, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Plan", "lark-cli", "/opt/homebrew/bin/lark-cli"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output = %q, missing %q", out, want)
		}
	}
}
