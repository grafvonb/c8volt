# Quickstart: Process Definition Watch Mode

## Prerequisites

- Go toolchain from `go.mod`.
- Repository dependencies available.
- Optional Camunda integration profile only for manual smoke checks; unit and command tests should prove the core behavior without a real cluster.

## Targeted Validation

Run the generic watch runner tests:

```bash
go test ./toolx/... -run 'Watch|watch' -count=1
```

Expected outcome:

- First tick runs immediately.
- Default interval behavior is covered by command tests.
- Positive intervals are honored.
- Invalid intervals are rejected before lookup work.
- Context cancellation and timeout stop promptly.
- Consecutive transient failures respect the retry budget and reset after success.

Run process-definition service and facade tests:

```bash
go test ./internal/services/processdefinition/... -run 'SearchProcessDefinitions|Watch|Snapshot' -count=1
go test ./c8volt/process -run 'ProcessDefinition|Watch|Snapshot' -count=1
```

Expected outcome:

- Snapshot collection uses service-owned paging.
- Broad searches include all pages in each snapshot.
- Facade request/result mapping remains thin.
- Generated Camunda clients are not changed.

Run command behavior tests:

```bash
go test ./cmd -run 'TestGetProcessDefinition|TestCommandContract' -count=1
```

Expected outcome:

- `--watch` and `--watch-interval` are exposed in help/metadata.
- `--watch-interval` defaults to `1s` and rejects invalid, zero, and negative values.
- `--watch --json`, `--watch --keys-only`, `--watch --xml`, `--watch --quiet`, and `--watch --automation` are rejected.
- `--watch --key` and `--watch --key --stat` are allowed where existing stat rules allow them.
- `get process-definition --watch` without selectors watches all process definitions.
- Existing non-watch selector diagnostics still apply.
- Non-watch JSON, keys-only, quiet, and automation output contracts remain unchanged.
- Verbose and default human watch modes keep their human-output contracts.
- Retry budget reset and exhaustion behavior is covered.

Regenerate and validate docs when command metadata or help changes:

```bash
make docs-content
```

Expected outcome:

- Generated CLI docs include `--watch`, `--watch-interval`, default interval, human-output-only behavior, incompatible JSON/keys-only/XML/quiet/automation behavior, and output-mode notes.
- README-facing examples and notes match the command behavior.

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

- The first snapshot appears immediately.
- Subsequent snapshots appear approximately every second.
- The command stops cleanly on interrupt.

Watch a selected latest definition:

```bash
c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest --watch --watch-interval 2s
```

Expected outcome:

- Empty snapshots are successful while no selected definition is visible.
- Matching definitions appear in later snapshots when deployed.

Reject JSON watch:

```bash
c8volt get process-definition --watch --json
```

Expected outcome:

- The command fails before lookup work with a clear human-output-only incompatible-flags error.

Reject keys-only watch:

```bash
c8volt get process-definition --watch --keys-only
```

Expected outcome:

- The command fails before lookup work with a clear human-output-only incompatible-flags error.

Reject quiet watch:

```bash
c8volt get process-definition --watch --quiet
```

Expected outcome:

- The command fails before lookup work with a clear human-output-only incompatible-flags error.

Reject automation watch:

```bash
c8volt get process-definition --watch --automation
```

Expected outcome:

- The command fails before lookup work with a clear human-output-only incompatible-flags error.

Reject XML watch:

```bash
c8volt get process-definition --key <process-definition-key> --xml --watch
```

Expected outcome:

- The command fails before lookup work with a clear incompatible-flags error.

## Documentation Review

Review README, command metadata, and generated docs:

```bash
rg -n 'watch|watch-interval|process-definition|human-output|human output|keys-only|json|automation|quiet|xml' README.md docs/cli cmd docsgen
```

Expected outcome:

- User-facing docs match the implemented watch behavior.
- Machine-output guarantees are documented wherever command output contracts are listed.

## Final Validation Notes

Iteration 6 completed the documentation and validation pass on 2026-08-04:

- `make docs-content` passed and regenerated CLI reference content.
- `go test ./toolx/... -run 'Watch|watch' -count=1` passed.
- `go test ./internal/services/processdefinition/... -run 'SearchProcessDefinitions|Watch|Snapshot' -count=1` passed.
- `go test ./c8volt/process -run 'ProcessDefinition|Watch|Snapshot' -count=1` passed.
- `go test ./cmd -run 'TestGetProcessDefinition|TestCommandContract' -count=1` passed.
- `gofmt -w toolx/watch cmd/get_processdefinition.go cmd/cmd_views_get.go cmd/command_contract.go c8volt/process internal/domain/processdefinition.go internal/services/processdefinition/search.go` completed.
- `make test` passed.
