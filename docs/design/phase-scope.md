# Phase Scope

## V0.1

V0.1 validates confirmed CLI Update Profiles, update execution, status, logs, and failure diagnosis. It does not include automatic discovery.

Included:

- `clikeep init`
- `clikeep add`
- `clikeep list`
- `clikeep up`
- `clikeep status`
- `clikeep doctor`

Excluded:

- `clikeep scan`
- PATH-only discovery
- shell history discovery
- source-hint ranking
- package-manager adapters
- dedicated `enable` / `disable` commands

Reason: V0.1 should prove whether confirmed profiles and diagnosable update runs are valuable beyond Topgrade custom commands. Discovery belongs to a later phase once the execution model is worth building.

### Init Boundary

`clikeep init` creates the config file and state directories needed by V0.1.

V0.1 init may write:

- an empty config file
- explanatory comments
- state and run-log directories

V0.1 init must not write:

- example Profiles
- unconfirmed Profiles
- preconfigured internal CLI entries

Reason: example Profiles can look executable or authoritative. Profiles should enter config through explicit user confirmation.

### Add Confirmation

`clikeep add <tool> --update ...` creates a confirmed Profile only after showing the parsed command invocation and receiving confirmation.

V0.1 add behavior:

- interactive TTY asks for confirmation before writing `confirmed = true`
- non-interactive add requires `--yes` to write `confirmed = true`
- command parsing and safety validation happen before confirmation
- rejected commands do not create confirmed Profiles

Reason: Profile Confirmation should mean the user accepted the parsed self-update command, not merely that a command string appeared in a config file.

### Doctor Boundary

`clikeep doctor` diagnoses whether configured profiles and local state are usable. It does not check whether a newer version is available.

V0.1 checks:

- configured commands exist or can be resolved
- configured version commands can run
- update commands do not violate obvious safety rules, including the `sudo` ban
- configuration can be parsed
- unsupported Profile fields such as profile-level `env` are reported
- state and log directories are writable

V0.1 does not check:

- remote latest versions
- package-manager outdated state
- update availability for self-updating CLIs
- health of unconfigured CLIs

### Dry Run Boundary

`clikeep up --dry-run` previews the update plan without executing any profile's update command.

V0.1 dry-run may run:

- command resolution
- version commands
- configuration parsing
- safety validation
- log path calculation

V0.1 dry-run must not run:

- `update.command`
- package-manager upgrade commands
- inferred update commands
- any command whose primary purpose is to mutate the CLI installation

### Execution Confirmation

`clikeep up` shows the Update Plan and asks for confirmation before executing selected update commands.

V0.1 defaults:

- interactive runs require confirmation
- `--yes` is the explicit non-interactive override
- `--dry-run` never executes update commands
- Profile existence is not enough to imply execution consent

V0.1 selection:

- `clikeep up` selects all enabled and confirmed profiles
- `clikeep up <tool>...` narrows selection to named tools
- unconfirmed profiles are skipped or blocked
- disabled profiles are skipped by default
- explicitly named disabled profiles are still skipped or blocked
- complex TUI multi-select is not included
- `-i` / `--interactive` is not included

`--yes` may skip:

- the final interactive confirmation prompt

`--yes` must not skip:

- the requirement that each selected tool has a confirmed Profile
- configuration parsing
- command resolution
- safety validation
- doctor-level blocking failures

Reason: update commands mutate local CLI installations. The default should preserve user control while still allowing scripts to opt into non-interactive execution deliberately.

Machine-readable output does not change confirmation requirements. A mutating command such as `clikeep up --json` still needs `--yes`; otherwise JSON output is suitable for non-mutating commands such as dry-run plans.

### Failure Isolation

`clikeep up` continues with later selected tools when one tool update fails. It records the failure and shows a final summary after all selected tools have either run or been skipped by policy.

V0.1 defaults:

- continue after per-tool update failure
- preserve each tool's exit code and log path
- return an overall non-zero exit code when any selected tool fails
- support `--fail-fast` to stop after the first failed update

Reason: batch updates should not let one transient internal CLI failure block unrelated tools. This is not a transaction; clikeep does not roll back earlier successful updates.

### Run Result Model

V0.1 records a narrow per-tool result status:

- `success`: the update command ran and exited with code 0
- `failed`: the update command ran and exited non-zero, timed out, or failed to start
- `skipped`: clikeep did not run the update command because of user selection, policy, validation, or fail-fast interruption

V0.1 may record:

- raw version output before the update
- raw version output after the update
- update command
- exit code
- timeout or start failure
- log path

V0.1 does not classify:

- `updated`
- `already up to date`
- semantic version increase or decrease
- remote latest version

Reason: self-updating CLIs often return unstructured output and inconsistent version strings. V0.1 should preserve evidence without pretending to understand every tool's update semantics.
