# Quickstart: Push Supported Get Filters Into Search Requests

## Implemented Behavior Target

- `get process-instance` pushes `roots-only`, `children-only`, `incidents-only`, and `no-incidents-only` into the search request on `v8.8` and `v8.9`.
- `v8.7` keeps those four semantics as client-side fallback because the current Operate search request in this repo cannot represent them reliably.
- `--parent-key` keeps its existing request-side equality behavior on all supported versions.
- `--orphan-children-only` stays client-side on every version.
- Paging totals and continuation prompts become more accurate on `v8.8` and `v8.9` because the server returns already-filtered pages.

## Implementation Notes

- Start from the shared process filter types in `c8volt/process/model.go` and `internal/domain/processinstance.go`.
- Model parent-presence and incident-presence pushdown intent as optional `*bool` fields on the shared filter, leaving `ParentKey` reserved for exact parent equality.
- Keep CLI flag validation in `cmd/get_processinstance.go`; only the translation of supported list-mode flags into the shared filter should change.
- Add request-side predicate encoding in `internal/services/processinstance/v88` and `v89`, not in `cmd/`.
- Preserve `v8.7` behavior by omission: do not send unsupported predicates there.
- Keep the audit explicit for other `get` commands: `get process-definition --latest` is the additional qualifying seam already adopted in the repo, and the remaining audited `get` commands need bounded no-addition rationale.

## Verification Focus

1. Confirm `v8.8` request bodies include parent-presence and incident-presence predicates when those flags are used.
2. Confirm `v8.9` request bodies include the same pushed-down predicates.
3. Confirm `v8.7` request bodies do not claim unsupported predicates for those same flags.
4. Confirm the visible page and continuation behavior for supported versions no longer shows broad unfiltered pages before local trimming.
5. Confirm `--orphan-children-only` still uses the existing follow-up parent lookup flow.
6. Confirm the audited `get process-definition --latest` seam still sends `isLatestVersion` request-side on `v8.8` and `v8.9` while `v8.7` keeps client-side fallback.
7. Confirm the remaining audited `get` commands were ruled out explicitly rather than skipped silently.

## Suggested Verification Commands

```bash
go test ./c8volt/process -count=1
go test ./internal/services/processinstance/... -count=1
go test ./cmd -count=1
make test
```

Run the focused suites first to isolate shared-model, service, and command regressions, then finish with `make test` as the repository gate.

## Manual Smoke Ideas

Use the same process-instance dataset across versions to compare request capture and visible paging behavior:

```bash
./c8volt --config /tmp/c8volt-v88.yaml get pi --bpmn-process-id C88_SimpleUserTask_Process --roots-only --with-age
./c8volt --config /tmp/c8volt-v88.yaml get pi --bpmn-process-id C88_SimpleUserTask_Process --children-only
./c8volt --config /tmp/c8volt-v88.yaml get pi --bpmn-process-id C88_SimpleUserTask_Process --incidents-only
./c8volt --config /tmp/c8volt-v87.yaml get pi --bpmn-process-id C87_SimpleUserTask_Process --roots-only
./c8volt --config /tmp/c8volt-v89.yaml get pi --bpmn-process-id C89_SimpleUserTask_Process --no-incidents-only
./c8volt --config /tmp/c8volt-v88.yaml --json get process-definition --latest
./c8volt --config /tmp/c8volt-v87.yaml --json get process-definition --latest
```

Check that:

- `v8.8` and `v8.9` fetch already-filtered pages for the supported flags
- `v8.7` still returns the correct final filtered set via local fallback
- `orphan-children-only` behavior does not change
- continuation prompts on supported versions match the narrowed result set
- `get process-definition --latest` still uses request-side latest filtering on `v8.8` and `v8.9` and local latest selection on `v8.7`
