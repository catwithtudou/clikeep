package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/catwithtudou/clikeep/internal/planner"
	"github.com/catwithtudou/clikeep/internal/profile"
	"github.com/catwithtudou/clikeep/internal/runlog"
)

const defaultTimeout = 10 * time.Minute

type Options struct {
	StateDir       string
	FailFast       bool
	Timeout        time.Duration
	Now            func() time.Time
	Stdout         io.Writer
	Stderr         io.Writer
	Progress       io.Writer
	ProgressFormat func(current, total int, name, status string) string
	RunCommand     func(context.Context, profile.Command) CommandResult
}

type CommandResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
}

func Run(ctx context.Context, plan planner.Plan, opts Options) (runlog.Summary, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	started := now().UTC()
	summary := runlog.Summary{
		RunID:     started.Format("20060102T150405.000000000Z"),
		StartedAt: started,
	}
	runDir := filepath.Join(opts.StateDir, "runs", summary.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return summary, err
	}

	failed := false
	for i, item := range plan.Items {
		logPath := filepath.Join(runDir, logFileName(item.Name))
		if opts.FailFast && failed {
			writeProgress(opts.Progress, opts.ProgressFormat, i+1, len(plan.Items), item.Name, "skipped")
			result := runlog.Result{
				Name:    item.Name,
				Status:  "skipped",
				Error:   "skipped after fail-fast",
				LogPath: logPath,
			}
			if err := writeLog(logPath, item, CommandResult{Err: errors.New(result.Error)}, result.Status); err != nil {
				return summary, err
			}
			summary.Results = append(summary.Results, result)
			continue
		}

		writeProgress(opts.Progress, opts.ProgressFormat, i+1, len(plan.Items), item.Name, "running")
		commandResult := runCommand(ctx, item.Update, opts)
		status := "success"
		if commandResult.Err != nil || commandResult.ExitCode != 0 {
			status = "failed"
			failed = true
		}
		writeProgress(opts.Progress, opts.ProgressFormat, i+1, len(plan.Items), item.Name, status)
		if err := writeLog(logPath, item, commandResult, status); err != nil {
			return summary, err
		}
		result := runlog.Result{
			Name:     item.Name,
			Status:   status,
			ExitCode: commandResult.ExitCode,
			LogPath:  logPath,
		}
		if commandResult.Err != nil {
			result.Error = commandResult.Err.Error()
		}
		summary.Results = append(summary.Results, result)
	}

	summary.EndedAt = now().UTC()
	if err := runlog.WriteSummary(opts.StateDir, summary); err != nil {
		return summary, err
	}
	return summary, nil
}

func runCommand(ctx context.Context, cmd profile.Command, opts Options) CommandResult {
	if opts.RunCommand != nil {
		return opts.RunCommand(ctx, cmd)
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	command := exec.CommandContext(cmdCtx, cmd.Command, cmd.Args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	if opts.Stdout != nil {
		command.Stdout = io.MultiWriter(opts.Stdout, &stdout)
	}
	command.Stderr = &stderr
	if opts.Stderr != nil {
		command.Stderr = io.MultiWriter(opts.Stderr, &stderr)
	}
	err := command.Run()

	exitCode := 0
	if err != nil {
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	if cmdCtx.Err() != nil && err == nil {
		err = cmdCtx.Err()
		exitCode = 1
	}
	return CommandResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: exitCode,
		Err:      err,
	}
}

func writeLog(path string, item planner.Item, result CommandResult, status string) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "tool: %s\n", item.Name)
	fmt.Fprintf(&buf, "command: %s\n", profile.RenderCommand(item.Update))
	fmt.Fprintf(&buf, "status: %s\n", status)
	fmt.Fprintf(&buf, "exit_code: %d\n", result.ExitCode)
	if result.Err != nil {
		fmt.Fprintf(&buf, "error: %s\n", result.Err)
	}
	if len(result.Stdout) > 0 {
		buf.WriteString("\n[stdout]\n")
		buf.Write(result.Stdout)
		if !bytes.HasSuffix(result.Stdout, []byte("\n")) {
			buf.WriteByte('\n')
		}
	}
	if len(result.Stderr) > 0 {
		buf.WriteString("\n[stderr]\n")
		buf.Write(result.Stderr)
		if !bytes.HasSuffix(result.Stderr, []byte("\n")) {
			buf.WriteByte('\n')
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func writeProgress(w io.Writer, format func(current, total int, name, status string) string, current, total int, name, status string) {
	if w == nil {
		return
	}
	line := ""
	if format != nil {
		line = format(current, total, name, status)
	} else {
		line = fmt.Sprintf("  [%d/%d] %s  [%s] %s", current, total, name, progressBar(current, total), status)
	}
	fmt.Fprintln(w, line)
}

func progressBar(current, total int) string {
	const width = 10
	if total <= 0 {
		return strings.Repeat(".", width)
	}
	done := current * width / total
	if done < 1 {
		done = 1
	}
	if done > width {
		done = width
	}
	return strings.Repeat("#", done) + strings.Repeat(".", width-done)
}

func logFileName(name string) string {
	if name == "" {
		return "tool.log"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	return replacer.Replace(name) + ".log"
}
