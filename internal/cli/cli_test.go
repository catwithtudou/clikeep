package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"unknown"}, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q, want unknown command", stderr.String())
	}
}
