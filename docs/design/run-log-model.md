# Run Log Model

## Config-State Separation

Run evidence belongs under clikeep state, not in the profile config.

V0.1 state uses:

```text
~/.local/state/clikeep/
```

State should store:

- latest run pointer
- run summaries
- per-tool logs
- latest result evidence used by `status`

State should not redefine profile intent. The source of truth for managed tools and confirmed update commands remains the config file.

## V0.1 Storage Shape

Each `clikeep up` execution creates one run directory:

```text
~/.local/state/clikeep/runs/<run-id>/
```

The run directory contains:

```text
run-summary.json
<tool>.log
```

`<run-id>` should be stable enough for display and lookup, such as a timestamp with enough precision to avoid collisions.

## Per-Tool Logs

Each selected tool gets its own log file when clikeep attempts or skips it.

The per-tool log should capture:

- tool name
- resolved command and path
- start time
- end time
- stdout
- stderr
- exit code
- timeout or start failure
- skip reason when skipped

## Run Summary

`run-summary.json` is the durable index for the run.

It should capture:

- run id
- start time
- end time
- selected tools
- per-tool status
- exit code or failure kind
- elapsed time
- log path
- overall result

Reason: terminal output should stay concise while logs preserve enough evidence for `status`, failure diagnosis, and future troubleshooting.

## Status View

V0.1 `clikeep status` reads:

- current configuration
- the latest run summary

It should show:

- configured tools
- whether each tool has a confirmed Profile
- latest per-tool result when available
- latest run time
- latest log path when available
- missing or stale local state when detected

V0.1 `status` should not scan all historical runs.

Not included in V0.1:

- history list
- time-range filtering
- trend analysis
- retention policy
- cross-run aggregation

Reason: the first status view should answer the daily question: what is configured, and what happened last time? Historical reporting can wait until the run model proves useful.
