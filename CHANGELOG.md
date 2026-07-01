# Changelog

## Unreleased

## v0.2.0 Preview

- Added an installable Clikeep agent skill package with setup checks, explicit install confirmation, safe dry-run guidance, and skill validation coverage.
- Changed `clikeep update` to run profile updates concurrently by default, with `--jobs <n>` and `--sequential` for explicit concurrency control.
- Kept `--fail-fast` sequential by default so later profiles are not started after the first failure.
- Refined update run output for concurrent execution with run mode metadata and status-first progress lines.

## v0.1.0 Preview

- Added explicit profile-based update plans and sequential update execution.
- Added concise terminal summaries with per-tool logs for failures.
- Added readable `list`, `status`, and grouped `doctor` output.
- Added `clikeep update` as the primary verb, with `clikeep up` as an alias.
- Added JSON output for automation and color-aware text output for terminals.
- Added `clikeep version` and `clikeep --version`.
- Added `clikeep help`, default help output, and `clikeep self-update`.

Preview v0.1 does not include package-manager data sources, auto-discovery,
Homebrew tap support, or release artifacts beyond the Go install path.
