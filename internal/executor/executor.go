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
	"sync"
	"time"

	"github.com/catwithtudou/clikeep/internal/planner"
	"github.com/catwithtudou/clikeep/internal/profile"
	"github.com/catwithtudou/clikeep/internal/runlog"
)

const defaultTimeout = 10 * time.Minute

type Options struct {
	StateDir       string
	FailFast       bool
	Jobs           int
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

	jobs := opts.Jobs
	if len(plan.Items) == 0 {
		jobs = 0
	} else if jobs <= 0 && opts.FailFast {
		jobs = 1
	} else if jobs <= 0 || jobs > len(plan.Items) {
		jobs = len(plan.Items)
	}
	var outputMu sync.Mutex
	runOpts := opts
	runOpts.Stdout = lockWriter(&outputMu, opts.Stdout)
	runOpts.Stderr = lockWriter(&outputMu, opts.Stderr)
	progress := lockWriter(&outputMu, opts.Progress)

	if jobs > 1 {
		concurrentSummary, err := runConcurrent(ctx, plan, runDir, summary, runOpts, progress, jobs)
		if err != nil {
			return concurrentSummary, err
		}
		summary = concurrentSummary
		summary.EndedAt = now().UTC()
		if err := runlog.WriteSummary(opts.StateDir, summary); err != nil {
			return summary, err
		}
		return summary, nil
	}

	failed := false
	for i, item := range plan.Items {
		logPath := filepath.Join(runDir, logFileName(item.Name))
		if opts.FailFast && failed {
			writeProgress(progress, opts.ProgressFormat, i+1, len(plan.Items), item.Name, "skipped")
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

		writeProgress(progress, opts.ProgressFormat, i+1, len(plan.Items), item.Name, "running")
		commandResult := runCommand(ctx, item.Update, runOpts)
		status := "success"
		if commandResult.Err != nil || commandResult.ExitCode != 0 {
			status = "failed"
			failed = true
		}
		writeProgress(progress, opts.ProgressFormat, i+1, len(plan.Items), item.Name, status)
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

func runConcurrent(ctx context.Context, plan planner.Plan, runDir string, summary runlog.Summary, opts Options, progress io.Writer, jobs int) (runlog.Summary, error) {
	results := make([]runlog.Result, len(plan.Items))
	started := make([]bool, len(plan.Items))

	var mu sync.Mutex
	next := 0
	stop := false
	var firstErr error

	nextItem := func() (int, planner.Item, bool) {
		mu.Lock()
		defer mu.Unlock()
		if stop || next >= len(plan.Items) {
			return 0, planner.Item{}, false
		}
		index := next
		next++
		started[index] = true
		return index, plan.Items[index], true
	}

	recordError := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if firstErr == nil {
			firstErr = err
		}
	}

	markStop := func(status string) {
		if !opts.FailFast || status != "failed" {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		stop = true
	}

	recordResult := func(index int, result runlog.Result) {
		mu.Lock()
		defer mu.Unlock()
		results[index] = result
	}

	var wg sync.WaitGroup
	wg.Add(jobs)
	for worker := 0; worker < jobs; worker++ {
		go func() {
			defer wg.Done()
			for {
				index, item, ok := nextItem()
				if !ok {
					return
				}
				result, err := runItem(ctx, item, index, len(plan.Items), runDir, opts, progress, markStop)
				if err != nil {
					recordError(err)
					return
				}
				recordResult(index, result)
			}
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return summary, firstErr
	}
	for i, item := range plan.Items {
		if started[i] {
			continue
		}
		result, err := skipItem(item, i, len(plan.Items), runDir, opts, progress)
		if err != nil {
			return summary, err
		}
		results[i] = result
	}

	summary.Results = results
	return summary, nil
}

func runItem(ctx context.Context, item planner.Item, index, total int, runDir string, opts Options, progress io.Writer, statusHook func(string)) (runlog.Result, error) {
	logPath := filepath.Join(runDir, logFileName(item.Name))
	writeProgress(progress, opts.ProgressFormat, index+1, total, item.Name, "running")
	commandResult := runCommand(ctx, item.Update, opts)
	status := "success"
	if commandResult.Err != nil || commandResult.ExitCode != 0 {
		status = "failed"
	}
	if statusHook != nil {
		statusHook(status)
	}
	writeProgress(progress, opts.ProgressFormat, index+1, total, item.Name, status)
	if err := writeLog(logPath, item, commandResult, status); err != nil {
		return runlog.Result{}, err
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
	return result, nil
}

func skipItem(item planner.Item, index, total int, runDir string, opts Options, progress io.Writer) (runlog.Result, error) {
	logPath := filepath.Join(runDir, logFileName(item.Name))
	writeProgress(progress, opts.ProgressFormat, index+1, total, item.Name, "skipped")
	result := runlog.Result{
		Name:    item.Name,
		Status:  "skipped",
		Error:   "skipped after fail-fast",
		LogPath: logPath,
	}
	if err := writeLog(logPath, item, CommandResult{Err: errors.New(result.Error)}, result.Status); err != nil {
		return runlog.Result{}, err
	}
	return result, nil
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

type lockedWriter struct {
	mu     *sync.Mutex
	writer io.Writer
}

func lockWriter(mu *sync.Mutex, writer io.Writer) io.Writer {
	if writer == nil {
		return nil
	}
	return lockedWriter{mu: mu, writer: writer}
}

func (w lockedWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Write(data)
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
