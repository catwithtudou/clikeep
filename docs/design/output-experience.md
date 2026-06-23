# Output Experience

## Goal

clikeep's terminal output should make update execution easy to follow. The user should be able to see what clikeep is about to do, what is running now, what finished, what failed, and where to inspect details.

## Current Design Intent

The output should be:

- clear about process and state
- pleasant to read in a terminal
- structured enough to scan during multi-tool updates
- conservative about visual decoration
- useful in both interactive and scripted usage

## Default Output Shape

V0.1 default output should be staged and scannable:

```text
Plan -> Confirm -> Run -> Summary
```

The default terminal output should show:

- selected tools
- resolved command and path
- raw version output summary when available
- current running tool
- per-tool result
- elapsed time
- log path
- final success, failed, and skipped counts

The default terminal output should not stream complete raw stdout/stderr for every command. Complete stdout/stderr belongs in per-tool logs. On failure, clikeep may show a short tail of the failed command output plus the log path.

Reason: multi-tool updates become hard to follow when every command writes raw output into the same terminal stream. clikeep should keep the main output readable while preserving full evidence in logs.

## Color and Symbols

V0.1 may use a small amount of color and symbolic markers to improve scanning. These markers can highlight:

- running
- success
- failed
- skipped
- warning

Color and symbols are presentation hints. They must not carry the only copy of any meaning.

V0.1 must fall back to plain text when:

- stdout is not a TTY
- `NO_COLOR` is set
- `--no-color` is passed
- the selected output format is machine-readable

Reason: terminal output should be pleasant in interactive use without becoming fragile in scripts, CI logs, copied text, or accessibility contexts.

## Failure Output

When a tool update fails, V0.1 default output should show:

- tool name
- result status
- failed phase when known
- exit code or timeout
- short tail of command output
- complete log path

The short output tail should be enough to recognize common failures without flooding the terminal. A default around the last 20 lines is appropriate.

V0.1 should not print the complete stdout/stderr stream in the default terminal output. Complete stdout/stderr belongs in the per-tool log file.

Reason: failure output should support immediate triage while keeping the main update summary readable.

## Machine-Readable Output

V0.1 should support `--json` for automation.

Supported JSON outputs:

- dry-run Update Plan
- final run result

Not supported in V0.1:

- realtime JSON event stream
- raw stdout/stderr as inline JSON payloads
- long-lived machine protocol

The JSON output should include enough information for scripts to understand selected tools, skipped tools, per-tool status, exit code, timeout or start failure, elapsed time, and log path.

When `--json` is selected, output must be plain machine-readable data without color, symbols, progress UI, or confirmation prompts. Commands that would otherwise require confirmation must also require `--yes` or remain non-executing.

Reason: scripts need stable final data, but V0.1 should not take on the complexity of a streaming event protocol.

## Non-Interactive Output

V0.1 non-interactive plain-text output should favor stability over presentation.

When stdout is not an interactive TTY, clikeep should not render dynamic progress UI. It should print a final summary after the run completes.

The final summary should include:

- run id or timestamp
- selected tool count
- success, failed, and skipped counts
- per-tool result status
- exit code or timeout for failures
- log path for each tool, or at least for failures

Non-interactive output should not include:

- spinners
- cursor movement
- live progress rewrites
- color-only or symbol-only meaning
- confirmation prompts

Reason: CI, scripts, and captured logs need stable output that can be copied, searched, and parsed by humans. Interactive terminals can use staged progress; non-interactive output should remain predictable.

## Verbose Output

V0.1 should not include a `--verbose` mode that streams complete stdout/stderr to the terminal in real time.

Complete stdout/stderr should still be written to per-tool logs. Default terminal output should show a short failure tail and the complete log path when a command fails.

Reason: realtime verbose streaming adds output interleaving, TTY behavior, and double-write complexity before the default output model has been validated.
