---
name: clikeep
description: "Use when the user wants an agent to manage local CLI updates through clikeep: bootstrap the clikeep CLI with one-time user confirmation, initialize config, inspect profiles, add trusted profiles, preview update plans, run updates after explicit authorization, read status or doctor output, and triage failed logs. Do not use for general package-manager advice or to guess unsupported update commands."
---

# Clikeep

Use this skill to hide clikeep internals behind a safe local CLI update workflow. Prefer the smallest action that answers the user's request.

## Core Boundary

- Treat clikeep as a local-first update manager for CLI tools the user already trusts.
- Manage explicit profiles; do not auto-discover tools or invent update commands.
- Do not replace package managers, mutate package sources, use `sudo`, or run shell snippets.
- Verify facts from the local command surface before writing docs or profiles.
- Do not silently install clikeep. If the CLI is missing, ask once before installing it with Go.

## Bootstrap First

Start every clikeep task by running:

```bash
bash scripts/bootstrap_clikeep.sh --check --json
```

Interpret the result before taking the next action:

- `ready`: continue with the requested clikeep operation.
- `cli_missing`: tell the user clikeep is not installed and ask for permission to install it with `go install github.com/catwithtudou/clikeep/cmd/clikeep@latest`.
- `go_missing`: explain that Go is required before clikeep can be installed.
- `config_missing`: run `bash scripts/bootstrap_clikeep.sh --init --json` when the user wants to use clikeep locally.
- `path_missing`: explain the reported PATH hint; do not edit shell startup files.

When the user authorizes installation, run:

```bash
bash scripts/bootstrap_clikeep.sh --install --init --json
```

The bootstrap script verifies `clikeep version` after install and initializes config/state when requested.

## Standard Workflow

1. Inspect current state:
   - `clikeep list`
   - `clikeep status`
   - `clikeep doctor`
2. Add a profile only after the update command is known:
   - Verify with `<tool> --help`, official docs, or a user-provided command.
   - Show the profile name and update command when they were not already explicit.
   - Write with `clikeep add <name> --update "<command>" [--version "<command>"] --yes`.
3. Preview before mutation:
   - `clikeep update --dry-run`
   - `clikeep update <profile> --dry-run`
4. Run updates only after clear authorization:
   - `clikeep update`
   - `clikeep update <profile>`
   - Use `--yes` only when the user explicitly authorized non-interactive execution.
   - For multiple profiles, expect default concurrent execution. Use `--sequential` when the user wants serialized updates, and use `--jobs <n>` only when the user asks for a specific concurrency cap.
   - Use `--fail-fast` when the user wants to stop starting later profiles after the first failure.
5. Triage results:
   - Summarize successes and failures from terminal output.
   - Include the reported run mode when it matters, especially `parallel`, `sequential`, `jobs`, `failed`, or `skipped`.
   - On failure, run `clikeep status <profile>` or `clikeep status`, read the log path, and inspect only the relevant tail.

## Safety Rules

- Never guess update commands for a tool. Ask or verify.
- Do not save profiles for commands containing `sudo`, pipes, redirects, command substitution, or multi-command shell composition.
- Do not run `clikeep update`, `clikeep update --yes`, or `clikeep self-update` unless user intent to mutate local installs is explicit.
- Use `clikeep self-update --dry-run` before self-updating clikeep itself.
- `--dry-run`, `list`, `status`, and `doctor` are safe first moves.
- If output will be parsed by a script, prefer `--json` where supported; if copied into docs, prefer `--no-color` where supported.

## Response Style

- Keep user-facing summaries short: action taken, profiles touched, result, and next command if relevant.
- Do not explain config, state, or log internals unless they help resolve a failure.
- When blocked, state the missing fact, such as unknown update command, clikeep not installed, Go missing, or profile not found.
