# Quickstart: Day-Based Process Instance Date Filters

## Prerequisites

- Work on branch `090-process-instance-date-filters`
- Use an explicit temp config in command tests when executing non-help paths; for this feature, use `writeTestConfigForVersion(...)` in `cmd/get_processinstance_test.go` and always pass `--config`
- Prefer `--json` in command assertions so process-instance output checks stay stable while the feature adds new search flags
- For fresh `get process-instance` command tests, use a fresh `Root()` execution path instead of the shared pre-`SetArgs()` flag reset helper; Cobra can round-trip `StringSlice` defaults into a literal `[]` and falsely trip `--key` exclusivity

## Implementation Walkthrough

1. Extend shared process-instance filter models in [`c8volt/process/model.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go) and [`internal/domain/processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go) with start/end date bounds.
2. Wire new Cobra flags and validation in [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go), keeping `--key` incompatible with search-only date filters.
3. Update facade/domain mapping in [`c8volt/process/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go) and related helpers so date bounds flow into versioned services.
4. Implement v8.8 request mapping in [`internal/services/processinstance/v88/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go) using native inclusive datetime filter operators.
7. Update `README.md`, regenerate CLI docs with `make docs-content` and `make docs`, then run `make test`.

## Suggested Verification Commands

```bash
go test ./cmd -run 'TestGetProcessInstance(SearchScaffold_UsesTempConfigAndCapturesSearchRequest|DateFilterScaffold|InvalidDateFormatHelper|InvalidStartDateRangeHelper|DateFiltersWithKeyHelper)$' -count=1
go test ./c8volt/process ./internal/services/processinstance/...
make docs-content
make docs
make test
```

## Manual Smoke Checks

```bash
./c8volt get process-instance --start-date-after 2026-01-01 --config /tmp/c8volt-v88.yaml
./c8volt get process-instance --end-date-before 2026-03-31 --state completed --config /tmp/c8volt-v88.yaml
./c8volt get process-instance --start-date-after 2026-01-31 --start-date-before 2026-01-01 --config /tmp/c8volt-v88.yaml
./c8volt get process-instance --key 2251799813685255 --start-date-after 2026-01-01 --config /tmp/c8volt-v88.yaml
```
