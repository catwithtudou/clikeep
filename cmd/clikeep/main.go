package main

import (
	"context"
	"os"
	"runtime/debug"

	"github.com/catwithtudou/clikeep/internal/cli"
	"github.com/catwithtudou/clikeep/internal/paths"
)

var version = "dev"

func main() {
	p := paths.Default()
	code := cli.Run(context.Background(), os.Args[1:], cli.Deps{
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		ConfigHome:  p.ConfigFile,
		StateHome:   p.StateDir,
		StdoutIsTTY: isTerminal(os.Stdout),
		StdinIsTTY:  isTerminal(os.Stdin),
		Version:     currentVersion(),
	})
	os.Exit(code)
}

func currentVersion() string {
	buildVersion := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		buildVersion = info.Main.Version
	}
	return resolveVersion(version, buildVersion)
}

func resolveVersion(injected, buildVersion string) string {
	if injected != "" && injected != "dev" {
		return injected
	}
	if buildVersion != "" && buildVersion != "(devel)" {
		return buildVersion
	}
	if injected != "" {
		return injected
	}
	return "dev"
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
