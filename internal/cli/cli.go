package cli

import (
	"context"
	"fmt"
	"io"
)

type Deps struct {
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	ConfigHome  string
	StateHome   string
	StdoutIsTTY bool
	StdinIsTTY  bool
}

func Run(ctx context.Context, args []string, deps Deps) int {
	_ = ctx

	if len(args) == 0 {
		fmt.Fprintln(deps.Stderr, "usage: clikeep <command>")
		return 2
	}

	switch args[0] {
	case "init", "add", "list", "up", "status", "doctor":
		fmt.Fprintf(deps.Stderr, "%s is not implemented yet\n", args[0])
		return 2
	default:
		fmt.Fprintf(deps.Stderr, "unknown command: %s\n", args[0])
		return 2
	}
}
