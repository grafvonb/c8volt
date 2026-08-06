# Quickstart: Process Definition Watch Repaint

## Prerequisites

- Go toolchain from `go.mod`.
- Repository dependencies available.
- Optional Camunda profile only for manual smoke checks; command tests should prove the core behavior without a real cluster.

## Targeted Validation

Run focused command tests:

```bash
go test ./cmd -run 'TestGetProcessDefinitionWatch|TestValidateGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestProcessDefinitionSelectorValidationHelpContract' -count=1
```

Expected outcome:

- Watch output uses the normal process-definition human result body.
- Watch output does not contain `snapshot 1:`, `snapshot 2:`, or other watch-only labels.
- Repaint behavior is asserted without requiring a real terminal.
- Slow refresh warnings are emitted once per slow streak, suppressed during continuing slow refreshes, and reset after an on-time refresh.
- Verbose mode can expose per-refresh timing/status outside the result body.
- Incompatible watch output modes still fail before lookup work.
- Help and command metadata no longer describe appended repeated snapshots.

Run generic watch-loop tests if timing behavior changes:

```bash
go test ./toolx/... -run 'Watch|watch' -count=1
```

Expected outcome:

- First refresh still runs immediately.
- Refreshes remain serial.
- Cancellation, timeout, retry reset, and retry exhaustion behavior still pass.

Regenerate generated docs after command help or metadata changes:

```bash
make docs-content
```

Expected outcome:

- Generated CLI docs describe repaint behavior, normal-result-body parity, slow-refresh warnings, and incompatible output modes.
- Generated docs no longer claim watch prints repeated terminal snapshot blocks.

Run broader validation before commit readiness:

```bash
make test
```

Expected outcome:

- Full race-enabled repository test suite passes.

## Optional Manual Smoke Checks

Use a configured Camunda profile and an interactive terminal.

Watch all process definitions:

```bash
c8volt get process-definition --watch --watch-interval 1s
```

Expected outcome:

- The terminal shows one refreshed result view at a time.
- The visible result body looks like `c8volt get process-definition`.
- No `snapshot N:` labels appear in the result body.
- The command stops cleanly on interrupt.

Watch a selected latest definition:

```bash
c8volt get pd --bpmn-process-id <bpmn-process-id> --latest --watch --watch-interval 2s
```

Expected outcome:

- Each refreshed result body looks like the equivalent non-watch `get pd` command.
- The view repaints rather than appending old results.

Check slow-refresh warning behavior with a broad or statistics-heavy lookup:

```bash
c8volt get process-definition --watch --stat --watch-interval 1s
```

Expected outcome:

- If a refresh takes longer than the interval, the operator sees a clear warning.
- Continuing slow refreshes do not repeatedly flood default human output.
- Verbose mode provides more detailed timing/status outside the result body.

Reject machine-oriented watch modes:

```bash
c8volt get process-definition --watch --json
c8volt get process-definition --watch --keys-only
c8volt get process-definition --watch --xml
c8volt get process-definition --watch --quiet
c8volt get process-definition --watch --automation
```

Expected outcome:

- Each command fails before lookup work with a clear incompatible-flags error.
