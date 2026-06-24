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

	"github.com/scott9/clikeep/internal/executor"
	"github.com/scott9/clikeep/internal/output"
	"github.com/scott9/clikeep/internal/paths"
	"github.com/scott9/clikeep/internal/planner"
	"github.com/scott9/clikeep/internal/profile"
	"github.com/scott9/clikeep/internal/runlog"
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
	if len(args) == 0 {
		printUsage(deps.Stderr)
		return 2
	}

	switch args[0] {
	case "init":
		return runInit(deps)
	case "add":
		return runAdd(args[1:], deps)
	case "list":
		return runList(deps)
	case "doctor":
		return runDoctor(deps)
	case "up":
		return runUp(ctx, args[1:], deps)
	case "status":
		return runStatus(args[1:], deps)
	default:
		fmt.Fprintf(deps.Stderr, "unknown command: %s\n", args[0])
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: clikeep <command>")
	fmt.Fprintln(w, "commands: init, add, list, up, status, doctor")
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

func runDoctor(deps Deps) int {
	cfg, err := profile.Load(deps.ConfigHome)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "load config: %v\n", err)
		return 1
	}
	problems := planner.Doctor(cfg, cliPaths(deps))
	if len(problems) == 0 {
		fmt.Fprintln(deps.Stdout, "doctor ok")
		return 0
	}
	printProblems(problems, deps.Stderr)
	if hasError(problems) {
		return 1
	}
	return 0
}

func runUp(ctx context.Context, args []string, deps Deps) int {
	opts, ok := parseUpArgs(args, deps.Stderr)
	if !ok {
		return 2
	}

	plan, problems, err := buildPlan(deps, opts.names)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "load config: %v\n", err)
		return 1
	}
	if len(problems) > 0 {
		printProblems(problems, deps.Stderr)
		return 2
	}

	if opts.dryRun {
		if opts.json {
			if err := output.WriteJSON(deps.Stdout, plan); err != nil {
				fmt.Fprintf(deps.Stderr, "write json: %v\n", err)
				return 1
			}
			return 0
		}
		if err := output.WritePlanText(deps.Stdout, plan, deps.StdoutIsTTY); err != nil {
			fmt.Fprintf(deps.Stderr, "write plan: %v\n", err)
			return 1
		}
		return 0
	}

	if !opts.json {
		if err := output.WritePlanText(deps.Stdout, plan, deps.StdoutIsTTY); err != nil {
			fmt.Fprintf(deps.Stderr, "write plan: %v\n", err)
			return 1
		}
	}
	if !opts.yes {
		ok, err := confirmRun(deps)
		if err != nil {
			fmt.Fprintln(deps.Stderr, err)
			return 2
		}
		if !ok {
			fmt.Fprintln(deps.Stderr, "update run not confirmed")
			return 1
		}
	}
	if !opts.json {
		fmt.Fprintln(deps.Stdout, "Run")
	}

	summary, err := executor.Run(ctx, plan, executor.Options{
		StateDir: deps.StateHome,
		FailFast: opts.failFast,
	})
	if err != nil {
		fmt.Fprintf(deps.Stderr, "run updates: %v\n", err)
		return 1
	}
	if opts.json {
		if err := output.WriteJSON(deps.Stdout, summary); err != nil {
			fmt.Fprintf(deps.Stderr, "write json: %v\n", err)
			return 1
		}
	} else if err := writeRunSummaryText(deps.Stdout, summary); err != nil {
		fmt.Fprintf(deps.Stderr, "write summary: %v\n", err)
		return 1
	}
	if summaryHasFailure(summary) {
		return 1
	}
	return 0
}

func runStatus(args []string, deps Deps) int {
	if len(args) > 1 {
		fmt.Fprintln(deps.Stderr, "usage: clikeep status [name]")
		return 2
	}
	var name string
	if len(args) == 1 {
		name = args[0]
	}

	cfg, err := profile.Load(deps.ConfigHome)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "load config: %v\n", err)
		return 1
	}
	if problems := profile.ValidateConfig(cfg); len(problems) > 0 {
		printProblems(problems, deps.Stderr)
		return 2
	}
	latest, hasLatest, err := runlog.ReadLatest(deps.StateHome)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "read latest run: %v\n", err)
		return 1
	}
	results := make(map[string]runlog.Result, len(latest.Results))
	if hasLatest {
		for _, result := range latest.Results {
			results[result.Name] = result
		}
	}

	found := false
	for _, tool := range cfg.Tools {
		if name != "" && tool.Name != name {
			continue
		}
		found = true
		result, ok := results[tool.Name]
		latestStatus := "none"
		logPath := ""
		if ok {
			latestStatus = result.Status
			logPath = result.LogPath
		}
		fmt.Fprintf(deps.Stdout, "%s\tenabled=%t\tconfirmed=%t\tlatest=%s",
			tool.Name, tool.Enabled, tool.Confirmed, latestStatus)
		if logPath != "" {
			fmt.Fprintf(deps.Stdout, "\tlog=%s", logPath)
		}
		fmt.Fprintln(deps.Stdout)
	}
	if name != "" && !found {
		fmt.Fprintf(deps.Stderr, "profile not found: %s\n", name)
		return 2
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

type upOptions struct {
	dryRun   bool
	json     bool
	yes      bool
	failFast bool
	names    []string
}

func parseUpArgs(args []string, stderr io.Writer) (upOptions, bool) {
	var opts upOptions
	for _, arg := range args {
		switch arg {
		case "--dry-run":
			opts.dryRun = true
		case "--json":
			opts.json = true
		case "--no-color":
			// Output is currently plain text, but accepting the flag keeps scripts stable.
		case "--yes":
			opts.yes = true
		case "--fail-fast":
			opts.failFast = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown up option: %s\n", arg)
				return upOptions{}, false
			}
			opts.names = append(opts.names, arg)
		}
	}
	return opts, true
}

func buildPlan(deps Deps, names []string) (planner.Plan, []profile.Problem, error) {
	cfg, err := profile.Load(deps.ConfigHome)
	if err != nil {
		return planner.Plan{}, nil, err
	}
	if problems := profile.ValidateConfig(cfg); len(problems) > 0 {
		return planner.Plan{}, problems, nil
	}
	plan, problems := planner.Build(cfg, names, planner.Options{})
	return plan, problems, nil
}

func confirmRun(deps Deps) (bool, error) {
	if !deps.StdinIsTTY {
		return false, errors.New("up requires --yes when stdin is non-interactive")
	}
	if deps.Stdin == nil {
		return false, errors.New("stdin is not available for confirmation")
	}
	fmt.Fprint(deps.Stdout, "Run update plan? [y/N] ")
	answer, err := bufio.NewReader(deps.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func writeRunSummaryText(w io.Writer, summary runlog.Summary) error {
	if _, err := fmt.Fprintln(w, "Summary"); err != nil {
		return err
	}
	if len(summary.Results) == 0 {
		_, err := fmt.Fprintln(w, "- no profiles selected")
		return err
	}
	for _, result := range summary.Results {
		line := fmt.Sprintf("- %s: %s", result.Name, result.Status)
		if result.LogPath != "" {
			line += " (" + result.LogPath + ")"
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
		if result.Status == "failed" {
			if result.Error != "" {
				if _, err := fmt.Fprintf(w, "  error: %s\n", result.Error); err != nil {
					return err
				}
			}
			tail, err := readTail(result.LogPath, 20)
			if err == nil && len(tail) > 0 {
				if _, err := fmt.Fprintln(w, "  tail:"); err != nil {
					return err
				}
				for _, tailLine := range tail {
					if _, err := fmt.Fprintf(w, "    %s\n", tailLine); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func readTail(path string, maxLines int) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return nil, nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return lines, nil
	}
	return lines[len(lines)-maxLines:], nil
}

func summaryHasFailure(summary runlog.Summary) bool {
	for _, result := range summary.Results {
		if result.Status == "failed" {
			return true
		}
	}
	return false
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

func hasError(problems []profile.Problem) bool {
	for _, problem := range problems {
		if problem.Severity == "error" {
			return true
		}
	}
	return false
}

func cliPaths(deps Deps) paths.Paths {
	return paths.Paths{
		ConfigFile: deps.ConfigHome,
		StateDir:   deps.StateHome,
		RunsDir:    filepath.Join(deps.StateHome, "runs"),
		LatestFile: filepath.Join(deps.StateHome, "latest-run"),
	}
}
