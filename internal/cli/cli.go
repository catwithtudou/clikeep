package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/scott9/clikeep/internal/profile"
)

type Deps struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// ConfigHome is the config file path used by the CLI.
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
	case "init":
		return runInit(deps)
	case "add":
		return runAdd(args[1:], deps)
	case "list":
		return runList(deps)
	case "up", "status", "doctor":
		fmt.Fprintf(deps.Stderr, "%s is not implemented yet\n", args[0])
		return 2
	default:
		fmt.Fprintf(deps.Stderr, "unknown command: %s\n", args[0])
		return 2
	}
}

func runInit(deps Deps) int {
	if err := os.MkdirAll(filepath.Dir(deps.ConfigHome), 0o755); err != nil {
		fmt.Fprintf(deps.Stderr, "create config dir: %v\n", err)
		return 1
	}
	file, err := os.OpenFile(deps.ConfigHome, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil && !errors.Is(err, os.ErrExist) {
		fmt.Fprintf(deps.Stderr, "create config file: %v\n", err)
		return 1
	}
	if file != nil {
		if err := file.Close(); err != nil {
			fmt.Fprintf(deps.Stderr, "close config file: %v\n", err)
			return 1
		}
	}
	if err := os.MkdirAll(filepath.Join(deps.StateHome, "runs"), 0o755); err != nil {
		fmt.Fprintf(deps.Stderr, "create state dir: %v\n", err)
		return 1
	}
	fmt.Fprintln(deps.Stdout, "initialized clikeep")
	return 0
}

func runAdd(args []string, deps Deps) int {
	if len(args) == 0 || args[0] == "" {
		fmt.Fprintln(deps.Stderr, "usage: clikeep add <name> --update <command> [--version <command>] [--yes]")
		return 2
	}

	name := args[0]
	var updateInput, versionInput string
	yes := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--update":
			i++
			if i >= len(args) || args[i] == "" {
				fmt.Fprintln(deps.Stderr, "--update requires a command")
				return 2
			}
			updateInput = args[i]
		case "--version":
			i++
			if i >= len(args) || args[i] == "" {
				fmt.Fprintln(deps.Stderr, "--version requires a command")
				return 2
			}
			versionInput = args[i]
		case "--yes":
			yes = true
		default:
			fmt.Fprintf(deps.Stderr, "unknown add option: %s\n", args[i])
			return 2
		}
	}
	if updateInput == "" {
		fmt.Fprintln(deps.Stderr, "--update is required")
		return 2
	}

	updateCmd, err := profile.ParseCommand(updateInput)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "invalid update command: %v\n", err)
		return 2
	}
	var versionCmd *profile.Command
	if versionInput != "" {
		cmd, err := profile.ParseCommand(versionInput)
		if err != nil {
			fmt.Fprintf(deps.Stderr, "invalid version command: %v\n", err)
			return 2
		}
		versionCmd = &cmd
	}

	cfg, err := profile.Load(deps.ConfigHome)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "load config: %v\n", err)
		return 1
	}
	for _, tool := range cfg.Tools {
		if tool.Name == name {
			fmt.Fprintf(deps.Stderr, "profile already exists: %s\n", name)
			return 2
		}
	}

	if !yes {
		ok, err := confirmAdd(name, updateCmd, deps)
		if err != nil {
			fmt.Fprintln(deps.Stderr, err)
			return 2
		}
		if !ok {
			fmt.Fprintln(deps.Stderr, "profile not confirmed")
			return 1
		}
	}

	cfg.Tools = append(cfg.Tools, profile.Profile{
		Name:      name,
		Enabled:   true,
		Confirmed: true,
		Update:    updateCmd,
		Version:   versionCmd,
	})
	if problems := profile.ValidateConfig(cfg); len(problems) > 0 {
		printProblems(problems, deps.Stderr)
		return 2
	}
	if err := profile.Save(deps.ConfigHome, cfg); err != nil {
		fmt.Fprintf(deps.Stderr, "save config: %v\n", err)
		return 1
	}

	fmt.Fprintf(deps.Stdout, "added %s\n", name)
	return 0
}

func runList(deps Deps) int {
	cfg, err := profile.Load(deps.ConfigHome)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "load config: %v\n", err)
		return 1
	}
	if problems := profile.ValidateConfig(cfg); len(problems) > 0 {
		printProblems(problems, deps.Stderr)
		return 2
	}
	if len(cfg.Tools) == 0 {
		fmt.Fprintln(deps.Stdout, "no profiles configured")
		return 0
	}
	for _, tool := range cfg.Tools {
		fmt.Fprintf(deps.Stdout, "%s\tenabled=%t\tconfirmed=%t\tupdate=%s\n",
			tool.Name, tool.Enabled, tool.Confirmed, profile.RenderCommand(tool.Update))
	}
	return 0
}

func confirmAdd(name string, update profile.Command, deps Deps) (bool, error) {
	if !deps.StdinIsTTY {
		return false, errors.New("add requires --yes when stdin is non-interactive")
	}
	if deps.Stdin == nil {
		return false, errors.New("stdin is not available for confirmation")
	}
	fmt.Fprintf(deps.Stdout, "Add %s with update `%s`? [y/N] ", name, profile.RenderCommand(update))
	answer, err := bufio.NewReader(deps.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func printProblems(problems []profile.Problem, w io.Writer) {
	for _, problem := range problems {
		if problem.Tool == "" {
			fmt.Fprintf(w, "%s: %s\n", problem.Severity, problem.Message)
			continue
		}
		fmt.Fprintf(w, "%s: %s: %s\n", problem.Severity, problem.Tool, problem.Message)
	}
}
