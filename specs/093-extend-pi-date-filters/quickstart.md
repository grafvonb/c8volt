# Quickstart: Extend Process-Instance Management Date Filters

## Prerequisites

- Work on branch `093-extend-pi-date-filters`
- Keep issue `#90` behavior as the canonical source for date-filter semantics and validation helpers
- Use explicit temp configs in command tests that execute non-help paths so repository-local configuration cannot leak into results
- Set `app.camunda_version` explicitly in temp configs when a test exercises version-specific behavior
- Regenerate generated CLI docs rather than editing files under `docs/cli/` by hand

## Implementation Walkthrough

1. Expose the four date flags on [`cmd/cancel_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go) and [`cmd/delete_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go) using the same shared flag variables and help text already defined in [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go).
2. Reuse the existing process-instance search validation helpers so cancel/delete reject invalid dates, invalid ranges, and explicit `--key` plus date-filter combinations before any action is taken.
3. Keep search-based selection routed through the existing shared helpers that already compose `process.ProcessInstanceFilter` values for process-instance searches.
4. Add or extend targeted tests in [`cmd/cancel_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go), [`cmd/delete_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go), and any nearby command test files needed to exercise search-based selection and invalid combinations.
5. Reuse the existing issue `#90` versioned service coverage unless command work exposes a missing regression that must be filled lower in the stack.
6. Update `README.md`, then regenerate docs with `make docs-content` and `make docs`.
7. Run `make test`.

## Suggested Verification Commands

```bash
go test ./cmd -run 'TestCancelProcessInstanceCommand_RejectsInvalidSearchState|TestDeleteProcessDefinitionCommand_RequiresTargetSelector' -count=1
go test ./cmd ./c8volt/process ./internal/services/processinstance/... -count=1
make docs-content
make docs
make test
```

## Manual Smoke Checks

```bash
./c8volt cancel process-instance --state active --start-date-before 2026-03-31 --config /tmp/c8volt-v88.yaml
./c8volt delete process-instance --state completed --end-date-after 2026-01-01 --end-date-before 2026-01-31 --auto-confirm --config /tmp/c8volt-v88.yaml
./c8volt cancel process-instance --key 2251799813711967 --start-date-after 2026-01-01 --config /tmp/c8volt-v88.yaml
./c8volt delete process-instance --start-date-after 2026-02-30 --config /tmp/c8volt-v88.yaml
./c8volt cancel process-instance --end-date-before 2026-03-31 --config /tmp/c8volt-v87.yaml
```
