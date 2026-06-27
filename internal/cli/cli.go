package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/catwithtudou/clikeep/internal/executor"
	"github.com/catwithtudou/clikeep/internal/output"
	"github.com/catwithtudou/clikeep/internal/paths"
	"github.com/catwithtudou/clikeep/internal/planner"
	"github.com/catwithtudou/clikeep/internal/profile"
	"github.com/catwithtudou/clikeep/internal/runlog"
)

type Deps struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// ConfigHome is the config file path used by the CLI.
	ConfigHome       string
	StateHome        string
	StdoutIsTTY      bool
	StdinIsTTY       bool
	NoColor          bool
	Version          string
	SelfUpdateRunner func(context.Context, []string, io.Writer, io.Writer) error
}

func Run(ctx context.Context, args []string, deps Deps) int {
	if len(args) == 0 {
		printHelp(deps.Stdout)
		return 0
	}
	if args[0] == "--help" || args[0] == "-h" {
		printHelp(deps.Stdout)
		return 0
	}
	if args[0] == "--version" {
		return runVersion(deps)
	}
	if args[0] == "--no-color" {
		deps.NoColor = true
		args = args[1:]
		if len(args) == 0 {
			printHelp(deps.Stdout)
			return 0
		}
	}

	switch args[0] {
	case "help":
		return runHelp(args[1:], deps)
	case "version":
		return runVersion(deps)
	case "init":
		return runInit(deps)
	case "add":
		return runAdd(args[1:], deps)
	case "list":
		return runList(args[1:], deps)
	case "doctor":
		return runDoctor(args[1:], deps)
	case "up", "update":
		return runUp(ctx, args[1:], deps)
	case "self-update", "self-upgrade":
		return runSelfUpdate(ctx, args[1:], deps)
	case "status":
		return runStatus(args[1:], deps)
	default:
		fmt.Fprintf(deps.Stderr, "unknown command: %s\n", args[0])
		fmt.Fprintln(deps.Stderr, "run `clikeep help` for usage")
		return 2
	}
}

func runHelp(args []string, deps Deps) int {
	if len(args) > 1 {
		fmt.Fprintln(deps.Stderr, "usage: clikeep help [command]")
		return 2
	}
	if len(args) == 1 {
		printCommandHelp(deps.Stdout, args[0])
		return 0
	}
	printHelp(deps.Stdout)
	return 0
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "clikeep - local-first update manager for CLI tools you already trust.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  clikeep <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init          initialize local config and state directories")
	fmt.Fprintln(w, "  add           add a confirmed CLI update profile")
	fmt.Fprintln(w, "  list          list configured profiles")
	fmt.Fprintln(w, "  update        update configured profiles")
	fmt.Fprintln(w, "  up            short alias for update")
	fmt.Fprintln(w, "  status        show profile state and latest run result")
	fmt.Fprintln(w, "  doctor        check config, command paths, and latest run state")
	fmt.Fprintln(w, "  self-update   upgrade clikeep itself via go install")
	fmt.Fprintln(w, "  version       print the clikeep version")
	fmt.Fprintln(w, "  help          show help")
}

func printCommandHelp(w io.Writer, command string) {
	switch command {
	case "add":
		fmt.Fprintln(w, "usage: clikeep add <name> --update <command> [--version <command>] [--yes]")
	case "update", "up":
		fmt.Fprintln(w, "usage: clikeep update [profile...] [--dry-run] [--yes] [--fail-fast] [--json] [--no-color]")
	case "status":
		fmt.Fprintln(w, "usage: clikeep status [name] [--no-color]")
	case "self-update", "self-upgrade":
		fmt.Fprintln(w, "usage: clikeep self-update [--version <version>] [--dry-run] [--yes]")
		fmt.Fprintln(w, "updates clikeep itself with: go install github.com/catwithtudou/clikeep/cmd/clikeep@<version>")
	default:
		printHelp(w)
	}
}

func runVersion(deps Deps) int {
	version := deps.Version
	if version == "" {
		version = "dev"
	}
	fmt.Fprintf(deps.Stdout, "clikeep %s\n", version)
	return 0
}

const selfUpdateModule = "github.com/catwithtudou/clikeep/cmd/clikeep"

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

func runList(args []string, deps Deps) int {
	for _, arg := range args {
		switch arg {
		case "--no-color":
			deps.NoColor = true
		default:
			fmt.Fprintf(deps.Stderr, "unknown list option: %s\n", arg)
			return 2
		}
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
	if len(cfg.Tools) == 0 {
		fmt.Fprintln(deps.Stdout, "no profiles configured")
		return 0
	}
	if err := writeListText(deps.Stdout, cfg, styleFor(deps, false)); err != nil {
		fmt.Fprintf(deps.Stderr, "write list: %v\n", err)
		return 1
	}
	return 0
}

func runDoctor(args []string, deps Deps) int {
	for _, arg := range args {
		switch arg {
		case "--no-color":
			deps.NoColor = true
		default:
			fmt.Fprintf(deps.Stderr, "unknown doctor option: %s\n", arg)
			return 2
		}
	}

	cfg, err := profile.Load(deps.ConfigHome)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "load config: %v\n", err)
		return 1
	}
	problems := planner.Doctor(cfg, cliPaths(deps))
	if _, _, err := runlog.ReadLatest(deps.StateHome); err != nil {
		problems = append(problems, profile.Problem{Severity: "error", Message: "latest run summary unreadable: " + err.Error()})
	}
	if err := writeDoctorText(deps.Stdout, problems, styleFor(deps, false)); err != nil {
		fmt.Fprintf(deps.Stderr, "write doctor: %v\n", err)
		return 1
	}
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
	style := styleFor(deps, opts.noColor)

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
		if err := output.WritePlanText(deps.Stdout, plan, style.Enabled); err != nil {
			fmt.Fprintf(deps.Stderr, "write plan: %v\n", err)
			return 1
		}
		return 0
	}

	if !opts.json {
		if err := output.WritePlanText(deps.Stdout, plan, style.Enabled); err != nil {
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
		fmt.Fprintln(deps.Stdout, style.Heading("Run"))
	}

	runOpts := executor.Options{
		StateDir: deps.StateHome,
		FailFast: opts.failFast,
	}
	if !opts.json {
		runOpts.Stdout = deps.Stdout
		runOpts.Stderr = deps.Stderr
		runOpts.Progress = deps.Stdout
		runOpts.ProgressFormat = func(current, total int, name, status string) string {
			return output.ProgressLine(style, current, total, name, status)
		}
	}
	summary, err := executor.Run(ctx, plan, runOpts)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "run updates: %v\n", err)
		return 1
	}
	if opts.json {
		if err := output.WriteJSON(deps.Stdout, summary); err != nil {
			fmt.Fprintf(deps.Stderr, "write json: %v\n", err)
			return 1
		}
	} else if err := writeRunSummaryText(deps.Stdout, summary, style); err != nil {
		fmt.Fprintf(deps.Stderr, "write summary: %v\n", err)
		return 1
	}
	if summaryHasFailure(summary) {
		return 1
	}
	return 0
}

type selfUpdateOptions struct {
	version string
	dryRun  bool
	yes     bool
}

func runSelfUpdate(ctx context.Context, args []string, deps Deps) int {
	opts, ok := parseSelfUpdateArgs(args, deps.Stderr)
	if !ok {
		return 2
	}
	target := selfUpdateModule + "@" + opts.version
	commandArgs := []string{"install", target}

	if opts.dryRun {
		fmt.Fprintln(deps.Stdout, "Self-update")
		fmt.Fprintf(deps.Stdout, "  command: go %s\n", strings.Join(commandArgs, " "))
		return 0
	}

	if !opts.yes {
		ok, err := confirmSelfUpdate(target, deps)
		if err != nil {
			fmt.Fprintln(deps.Stderr, err)
			return 2
		}
		if !ok {
			fmt.Fprintln(deps.Stderr, "self-update not confirmed")
			return 1
		}
	}

	fmt.Fprintln(deps.Stdout, "Self-update")
	fmt.Fprintf(deps.Stdout, "  command: go %s\n", strings.Join(commandArgs, " "))
	if err := runSelfUpdateCommand(ctx, commandArgs, deps); err != nil {
		fmt.Fprintf(deps.Stderr, "self-update failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(deps.Stdout, "updated clikeep")
	return 0
}

func parseSelfUpdateArgs(args []string, stderr io.Writer) (selfUpdateOptions, bool) {
	opts := selfUpdateOptions{version: "latest"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version":
			i++
			if i >= len(args) || args[i] == "" {
				fmt.Fprintln(stderr, "--version requires a version")
				return selfUpdateOptions{}, false
			}
			if strings.Contains(args[i], "@") {
				fmt.Fprintln(stderr, "--version should not include @")
				return selfUpdateOptions{}, false
			}
			opts.version = args[i]
		case "--dry-run":
			opts.dryRun = true
		case "--yes":
			opts.yes = true
		default:
			fmt.Fprintf(stderr, "unknown self-update option: %s\n", args[i])
			return selfUpdateOptions{}, false
		}
	}
	return opts, true
}

func confirmSelfUpdate(target string, deps Deps) (bool, error) {
	if !deps.StdinIsTTY {
		return false, errors.New("self-update requires --yes when stdin is non-interactive")
	}
	if deps.Stdin == nil {
		return false, errors.New("stdin is not available for confirmation")
	}
	fmt.Fprintf(deps.Stdout, "Self-update clikeep with `go install %s`? [y/N] ", target)
	answer, err := bufio.NewReader(deps.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func runSelfUpdateCommand(ctx context.Context, args []string, deps Deps) error {
	if deps.SelfUpdateRunner != nil {
		return deps.SelfUpdateRunner(ctx, args, deps.Stdout, deps.Stderr)
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = deps.Stdout
	cmd.Stderr = deps.Stderr
	return cmd.Run()
}

func runStatus(args []string, deps Deps) int {
	var name string
	for _, arg := range args {
		switch {
		case arg == "--no-color":
			deps.NoColor = true
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(deps.Stderr, "unknown status option: %s\n", arg)
			return 2
		case name == "":
			name = arg
		default:
			fmt.Fprintln(deps.Stderr, "usage: clikeep status [name] [--no-color]")
			return 2
		}
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
	var rows [][]string
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
		rows = append(rows, []string{tool.Name, boolText(tool.Enabled), boolText(tool.Confirmed), latestStatus, logPath})
	}
	if name != "" && !found {
		fmt.Fprintf(deps.Stderr, "profile not found: %s\n", name)
		return 2
	}
	if err := writeStatusText(deps.Stdout, rows, styleFor(deps, false)); err != nil {
		fmt.Fprintf(deps.Stderr, "write status: %v\n", err)
		return 1
	}
	return 0
}

func writeListText(w io.Writer, cfg profile.Config, style output.Style) error {
	rows := make([][]string, 0, len(cfg.Tools))
	for _, tool := range cfg.Tools {
		rows = append(rows, []string{
			tool.Name,
			boolText(tool.Enabled),
			boolText(tool.Confirmed),
			profile.RenderCommand(tool.Update),
		})
	}
	return writeTable(w, "Profiles", []string{"NAME", "ENABLED", "CONFIRMED", "UPDATE"}, rows, style)
}

func writeStatusText(w io.Writer, rows [][]string, style output.Style) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no profiles configured")
		return err
	}
	return writeTable(w, "Status", []string{"NAME", "ENABLED", "CONFIRMED", "LATEST", "LOG"}, rows, style)
}

func writeDoctorText(w io.Writer, problems []profile.Problem, style output.Style) error {
	if _, err := fmt.Fprintln(w, style.Heading("Doctor")); err != nil {
		return err
	}
	groups := []struct {
		title    string
		problems []profile.Problem
	}{
		{title: "Config", problems: filterDoctorProblems(problems, "config")},
		{title: "Commands", problems: filterDoctorProblems(problems, "commands")},
		{title: "Latest Run", problems: filterDoctorProblems(problems, "latest-run")},
	}
	for _, group := range groups {
		if _, err := fmt.Fprintf(w, "  %s\n", group.title); err != nil {
			return err
		}
		if len(group.problems) == 0 {
			if _, err := fmt.Fprintln(w, "    ok"); err != nil {
				return err
			}
			continue
		}
		for _, problem := range group.problems {
			subject := problem.Tool
			if subject == "" {
				subject = "-"
			}
			severity := problem.Severity
			switch problem.Severity {
			case "error":
				severity = style.Error(problem.Severity)
			case "warning":
				severity = style.Warning(problem.Severity)
			}
			if _, err := fmt.Fprintf(w, "    %-7s  %-18s  %s\n", severity, subject, problem.Message); err != nil {
				return err
			}
		}
	}
	return nil
}

func filterDoctorProblems(problems []profile.Problem, category string) []profile.Problem {
	var filtered []profile.Problem
	for _, problem := range problems {
		if doctorProblemCategory(problem) == category {
			filtered = append(filtered, problem)
		}
	}
	return filtered
}

func doctorProblemCategory(problem profile.Problem) string {
	message := strings.ToLower(problem.Message)
	switch {
	case strings.Contains(message, "latest run"):
		return "latest-run"
	case strings.Contains(message, "not found"):
		return "commands"
	default:
		return "config"
	}
}

func writeTable(w io.Writer, title string, headers []string, rows [][]string, style output.Style) error {
	if _, err := fmt.Fprintln(w, style.Heading(title)); err != nil {
		return err
	}
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	if err := writeTableRow(w, headers, widths); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writeTableRow(w, row, widths); err != nil {
			return err
		}
	}
	return nil
}

func writeTableRow(w io.Writer, row []string, widths []int) error {
	if _, err := fmt.Fprint(w, "  "); err != nil {
		return err
	}
	for i, width := range widths {
		cell := ""
		if i < len(row) {
			cell = row[i]
		}
		if i > 0 {
			if _, err := fmt.Fprint(w, "  "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "%-*s", width, cell); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func styleFor(deps Deps, noColor bool) output.Style {
	return output.NewStyle(colorEnabled(deps, noColor))
}

func colorEnabled(deps Deps, noColor bool) bool {
	if noColor || deps.NoColor || !deps.StdoutIsTTY {
		return false
	}
	_, disabled := os.LookupEnv("NO_COLOR")
	return !disabled
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
	noColor  bool
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
			opts.noColor = true
		case "--yes":
			opts.yes = true
		case "--fail-fast":
			opts.failFast = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown update option: %s\n", arg)
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
		return false, errors.New("update requires --yes when stdin is non-interactive")
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

func writeRunSummaryText(w io.Writer, summary runlog.Summary, style output.Style) error {
	if _, err := fmt.Fprintln(w, style.Heading("Summary")); err != nil {
		return err
	}
	if len(summary.Results) == 0 {
		_, err := fmt.Fprintln(w, "  no profiles selected")
		return err
	}
	successes, failures, skipped := resultCounts(summary)
	if _, err := fmt.Fprintf(w, "  %s: %d  %s: %d  %s: %d\n",
		style.Success("success"), successes,
		style.Error("failed"), failures,
		style.Warning("skipped"), skipped); err != nil {
		return err
	}
	for _, result := range summary.Results {
		line := fmt.Sprintf("  %s  %s", statusCell(style, result.Status), result.Name)
		if result.Status == "failed" && result.LogPath != "" {
			line += "  log: " + result.LogPath
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

func statusCell(style output.Style, status string) string {
	const width = 7
	if len(status) >= width {
		return style.Status(status)
	}
	return style.Status(status) + strings.Repeat(" ", width-len(status))
}

func resultCounts(summary runlog.Summary) (successes, failures, skipped int) {
	for _, result := range summary.Results {
		switch result.Status {
		case "success":
			successes++
		case "failed":
			failures++
		case "skipped":
			skipped++
		}
	}
	return successes, failures, skipped
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
