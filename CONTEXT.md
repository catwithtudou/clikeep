# clikeep

clikeep is a local-first context for keeping frequently used CLI tools discoverable, updatable, and under user control. The language here describes the product domain, not the implementation.

## Language

**clikeep**:
A local-first update manager for frequently used CLI tools. It coordinates known update entries for CLIs the user cares about, but it is not a package manager or version manager.
_Avoid_: Package manager, binary manager, Topgrade replacement

**Frequent CLI**:
A command-line tool that the user uses often enough to explicitly track and update. Frequency is a signal for relevance, not automatic permission to manage the tool.
_Avoid_: Installed package, PATH binary, system command

**CLI Update Profile**:
A user-confirmed update entry for one CLI. It records the intended way to detect, inspect, update, and verify that CLI.
_Avoid_: Package, binary, dependency

**Profile Confirmation**:
The explicit state that turns a CLI Update Profile into an executable update entry. It confirms the user has reviewed the self-update command for that CLI.
_Avoid_: Command presence, config existence

**Enabled Profile**:
A confirmed CLI Update Profile that is eligible for default update selection. Disabled profiles remain configured but are skipped by default.
_Avoid_: Deleted profile, selected profile

**Profile Name**:
The unique identifier for a CLI Update Profile. In V0.1 command selection uses Profile names, not aliases.
_Avoid_: Alias, display label

**Minimum Profile**:
The smallest CLI Update Profile that clikeep can execute in V0.1. It includes a CLI name, a confirmed self-update command, and the confirmation state.
_Avoid_: Complete profile, discovered profile

**Runtime Resolution**:
The act of resolving a CLI command to its current local path when clikeep builds or runs an Update Plan. It is runtime evidence, not the Profile's source of truth.
_Avoid_: Stored executable truth, install ownership

**Config-State Separation**:
The boundary between user-authored update intent and runtime evidence. Profiles belong in config; run results, logs, and status evidence belong in state.
_Avoid_: Runtime config, self-updating profile

**Self-update Command**:
The confirmed command invocation exposed by a CLI itself, such as an `update`, `upgrade`, or `self-update` subcommand. It is represented as a command plus arguments, not as arbitrary shell script.
_Avoid_: Package upgrade, automatic update, shell script

**Mode A**:
The self-update flow for frequent CLIs managed through CLI Update Profiles. This is the primary product mode for clikeep.
_Avoid_: Package manager upgrade

**Mode B**:
The package-manager-backed flow that delegates to existing ecosystem upgrade commands. This is an adapter mode, not the main clikeep model.
_Avoid_: Unified package management, system upgrade

**Source Hint**:
A non-authoritative clue about where a CLI may have come from. It can help explain an update option, but it is not treated as proof.
_Avoid_: Source of truth, ownership record

**Update Plan**:
A preview of selected CLI Update Profiles and the commands clikeep is preparing to run. It exists so the user can inspect the action before execution.
_Avoid_: Run result, lockfile

**Dry Run**:
A non-executing preview of an Update Plan. In V0.1 it must not invoke a profile's self-update command.
_Avoid_: Trial update, check-only update

**Execution Confirmation**:
The user's explicit approval to run the selected self-update commands after inspecting an Update Plan. In V0.1 normal interactive runs require it; the non-interactive override skips only the prompt, not profile or safety requirements.
_Avoid_: Implicit consent, profile existence, force mode

**Selection Scope**:
The set of confirmed profiles chosen for an update run. In V0.1 it is determined by default-all selection or explicit tool arguments, not by a multi-select TUI.
_Avoid_: Discovery result, interactive checklist

**Failure Isolation**:
The execution rule that one tool's update failure is recorded without automatically stopping later selected tools. In V0.1 interruption requires an explicit fail-fast mode.
_Avoid_: All-or-nothing update, transaction

**Run Result**:
The per-tool outcome recorded after an update attempt or policy skip. In V0.1 the canonical statuses are success, failed, and skipped.
_Avoid_: Update classification, version comparison

**Run Log**:
The durable per-run evidence written under clikeep state. It includes a run summary and per-tool command logs.
_Avoid_: Terminal output, profile config

**Status View**:
The user's view of configured profiles and their latest known run results. In V0.1 it is based on current configuration and the latest run summary.
_Avoid_: Historical report, audit log

**Run Output**:
The terminal-facing presentation of an update run. It summarizes plan, confirmation, progress, and results while keeping complete command output in logs.
_Avoid_: Raw stdout dump, log file

**Machine-Readable Output**:
A structured output form for scripts and automation. In V0.1 it represents dry-run plans or final run results, not a realtime event stream.
_Avoid_: Interactive output, log stream

**Non-Interactive Output**:
The terminal output used when clikeep is running without an interactive TTY or progress UI. It favors stable summaries over dynamic presentation.
_Avoid_: Progress UI, prompt flow

**Doctor**:
A diagnostic view of whether configured CLI Update Profiles and local clikeep state are usable. In V0.1 it does not decide whether newer versions are available.
_Avoid_: Update checker, package audit

**Validation Gate**:
A decision point that tests whether clikeep needs to exist as its own CLI. The first gate compares Topgrade custom commands against clikeep's intended profile, status, log, diagnosis, and discovery capabilities.
_Avoid_: Milestone, implementation phase

**Package Manager Adapter**:
A bridge that delegates discovery or updates to an existing package manager or ecosystem tool. It should not reimplement that tool's upgrade logic.
_Avoid_: Package manager implementation

**cli-update-manager**:
An agent skill that guides safe use of clikeep. It helps discover candidates, confirm update commands, preview plans, and summarize results without bypassing user confirmation.
_Avoid_: clikeep implementation, autonomous updater
