package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/scott9/clikeep/internal/cli"
	"github.com/scott9/clikeep/internal/paths"
)

func main() {
	p := paths.Default()
	code := cli.Run(context.Background(), os.Args[1:], cli.Deps{
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		ConfigHome: filepath.Dir(p.ConfigFile),
		StateHome:  p.StateDir,
	})
	os.Exit(code)
}
