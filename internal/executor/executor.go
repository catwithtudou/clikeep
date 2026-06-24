package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/scott9/clikeep/internal/planner"
	"github.com/scott9/clikeep/internal/profile"
	"github.com/scott9/clikeep/internal/runlog"
)

const defaultTimeout = 10 * time.Minute

type Options struct {
	StateDir   string
	FailFast   bool
	Timeout    time.Duration
	Now        func() time.Time
	RunCommand func(context.Context, profile.Command) CommandResult
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
	for _, item := range plan.Items {
		logPath := filepath.Join(runDir, logFileName(item.Name))
		if opts.FailFast && failed {
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

		commandResult := runCommand(ctx, item.Update, opts)
		status := "success"
		if commandResult.Err != nil || commandResult.ExitCode != 0 {
			status = "failed"
			failed = true
		}
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
	command.Stderr = &stderr
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

func logFileName(name string) string {
	if name == "" {
		return "tool.log"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	return replacer.Replace(name) + ".log"
}
