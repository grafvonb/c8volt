# Quickstart: Relative Day-Based Process-Instance Date Shortcuts

## Prerequisites

- Work on branch `095-processinstance-day-filters`
- Keep issue `#90` absolute date-filter behavior and issue `#93` management-command wiring as the canonical source for downstream semantics
- Use explicit temp configs in command tests that execute non-help paths so repository-local configuration cannot leak into results
- Set `app.camunda_version` explicitly in temp configs whenever a test exercises version-specific behavior
- Reuse the shared process-instance search capture helpers in [`cmd/cmd_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go) so relative-day command coverage asserts the derived canonical absolute-date request fields instead of open-coding JSON walking in each test
- Regenerate generated CLI docs rather than editing files under `docs/cli/` by hand

## Implementation Walkthrough

1. Start in [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go), where the shared process-instance search helpers already live for `get`, `cancel`, and `delete`.
2. Add shared flag variables and registration for `--start-date-newer-days`, `--start-date-older-days`, `--end-date-newer-days`, and `--end-date-older-days`, keeping help text aligned across all three commands.
3. Extend `validatePISearchFlags()` to parse non-negative integer day inputs, reject mixed absolute-plus-relative filters for the same field, reject invalid derived ranges, and preserve the explicit `--key` incompatibility for search-only filters.
4. Update `populatePISearchFilterOpts()` so relative day flags are converted into the existing absolute date fields on `process.ProcessInstanceFilter` before the facade call path is used.
5. Reuse the existing search helper calls already present in [`cmd/cancel_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go) and [`cmd/delete_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go) so search-based management commands inherit the same derived-bound logic automatically.
6. Add or update command tests in [`cmd/get_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go), [`cmd/cancel_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go), and [`cmd/delete_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go) to cover valid conversions, invalid values, absolute-plus-relative conflicts, invalid ranges, explicit `--key` conflicts, and v8.7 behavior, while keeping shared request-shape assertions in the common test helpers so later relative-day cases only need to vary CLI inputs and expected derived bounds.
7. Extend lower-level regression coverage only where needed to prove derived bounds still reach the existing canonical absolute date filter path in [`c8volt/process/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go), [`internal/services/processinstance/v87/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go), and [`internal/services/processinstance/v88/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go).
8. Update `README.md`, regenerate CLI docs with `make docs-content` and `make docs`, then run `make test`.

## Suggested Verification Commands

```bash
go test ./cmd -run 'TestGetProcessInstance(SearchScaffold|DateFilterScaffold)|TestCancelProcessInstance(SearchScaffold|Command_.*Date)|TestDeleteProcessInstance(SearchScaffold|Command_.*Date)' -count=1
go test ./cmd -run 'TestGetProcessInstance.*Days|TestCancelProcessInstance.*Days|TestDeleteProcessInstance.*Days' -count=1
go test ./c8volt/process ./internal/services/processinstance/... -count=1
make docs-content
make docs
make test
```

## Implemented Verification Notes

- Confirm the generated command help for `get process-instance`, `cancel process-instance`, and `delete process-instance` now shows all four `*-days` flags and examples that use them.
- Confirm README examples cover both absolute `YYYY-MM-DD` filters and relative day shortcuts for read and management flows.
- Keep generated `docs/cli/` pages in sync by changing Cobra help text first, then rerunning `make docs-content` and `make docs`.
- Finish the iteration with `make test`; do not mark the polish work complete if the repository-wide validation gate fails.

## Manual Smoke Checks

```bash
./c8volt get process-instance --start-date-older-days 7 --config /tmp/c8volt-v88.yaml
./c8volt get process-instance --start-date-older-days 30 --start-date-newer-days 7 --config /tmp/c8volt-v88.yaml
./c8volt cancel process-instance --state active --start-date-newer-days 30 --config /tmp/c8volt-v88.yaml
./c8volt delete process-instance --end-date-older-days 60 --end-date-newer-days 7 --auto-confirm --config /tmp/c8volt-v88.yaml
./c8volt cancel process-instance --key 2251799813711967 --start-date-older-days 7 --config /tmp/c8volt-v88.yaml
./c8volt get process-instance --start-date-older-days -1 --config /tmp/c8volt-v88.yaml
./c8volt get process-instance --end-date-newer-days 14 --config /tmp/c8volt-v87.yaml
```
