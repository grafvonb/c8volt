# Quickstart: Slow Analysis Progress After Confirmation

## Prerequisites

- Go toolchain from `go.mod` available.
- Feature branch or worktree with `specs/265-slow-analysis-progress` active.
- Existing test fixtures for slow-process analysis and activity sink available.

## Focused Validation

Run shared progress policy tests:

```bash
go test ./cmd -run 'TestOps.*Progress|TestFormatOps.*Progress|TestOpsETA' -count=1
```

Expected outcome:

- Shared progress channel tests keep machine modes stdout-safe.
- New milestone pacing tests prove default human durable milestones require both elapsed time and forward progress.
- Boundary tests cover no elapsed time, no counter movement, elapsed plus counter movement, and verbose/debug behavior.

Run slow-analysis command progress tests:

```bash
go test ./cmd -run 'TestOpsAnalyseSlowProcessInstances.*Progress|TestOpsAnalyseSlowProcessInstances.*Preflight' -count=1
```

Expected outcome:

- Default human mode keeps transient workflow activity visible.
- Default human mode can emit sparse durable post-confirmation milestones without stdout leakage.
- Verbose and debug modes keep durable detailed progress.
- JSON, keys-only, quiet, and automation modes remain free of human progress text.

Run slow-analysis service progress tests if service progress event behavior changes:

```bash
go test ./internal/services/ops -run 'TestSlowProcessAnalysis.*Progress|TestSlowProcessAnalysis.*Preflight' -count=1
```

Expected outcome:

- Services continue to emit structured preflight, page, and frozen-scope events.
- Services do not own human milestone pacing or output-mode routing.

## Documentation Validation

If command help text changes, regenerate generated CLI docs:

```bash
make docs-content
```

Expected outcome:

- `README.md`, generated CLI docs, and command metadata describe the same user-visible behavior.
- If command help does not change, record in tasks/progress that docs generation was not required.

## Full Validation

Run the repository full test target before completion:

```bash
make test
```

Expected outcome:

- `go test ./... -race -count=1` passes through the project make target.
- No progress text appears in machine stdout contract tests.
- Existing activity-priority behavior remains intact for nested runtime element lookups.
