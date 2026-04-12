# Quickstart: Version-Aware Process-Instance Paging and Overflow Handling

## Prerequisites

- Work on branch `101-processinstance-paging`
- Keep the clarified spec as the source of truth for shared config, partial completion, cumulative counts, consistent continuation behavior, and stop/warn handling
- Preserve direct `--key` behavior for `cancel process-instance` and `delete process-instance`
- Reuse the existing process-instance command helpers in [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go) and shared test helpers in [`cmd/cmd_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go)
- Update `README.md` and regenerate `docs/cli/` rather than editing generated docs by hand

## Implementation Walkthrough

1. Add the shared default page-size config field in [`config/app.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config/app.go) and bind/default it through the existing Viper setup in [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go).
2. Extend the shared process-instance command helpers in [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go) so page-size resolution, overflow evaluation, cumulative counts, prompts, partial completion, and warning-stop behavior live in one place.
3. Update `get process-instance` to use the shared paging loop instead of one-shot search execution.
4. Update search-based `cancel process-instance` in [`cmd/cancel_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go) so it processes one fetched page at a time and uses the same continuation model before executing cancellations.
5. Update search-based `delete process-instance` in [`cmd/delete_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go) so it mirrors the same page loop and continuation behavior.
6. Extend the facade/service search seams in [`c8volt/process/api.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/api.go), [`c8volt/process/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go), and [`internal/services/processinstance/api.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/api.go) so the command layer can receive version-aware overflow metadata instead of only raw items.
7. Use native page metadata in [`internal/services/processinstance/v88/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go) and implement the fallback/indeterminate behavior in [`internal/services/processinstance/v87/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service.go).
8. Add or update regression coverage in [`cmd/get_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go), [`cmd/cancel_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go), [`cmd/delete_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go), [`c8volt/process/client_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go), [`internal/services/processinstance/v87/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go), and [`internal/services/processinstance/v88/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go).
9. Update `README.md`, regenerate CLI docs, and finish with `make test`.

## Suggested Verification Commands

```bash
go test ./cmd -run 'TestGetProcessInstanceSearchScaffold|TestCancelProcessInstanceSearchScaffold|TestDeleteProcessInstanceSearchScaffold' -count=1
go test ./cmd -run 'TestGetProcessInstance.*Paging|TestCancelProcessInstance.*Paging|TestDeleteProcessInstance.*Paging' -count=1
go test ./c8volt/process ./internal/services/processinstance/... -count=1
make docs-content
make docs
make test
```

## Manual Smoke Checks

```bash
./c8volt get process-instance --state active --config /tmp/c8volt-v88.yaml
./c8volt get process-instance --state active --count 250 --config /tmp/c8volt-v88.yaml
./c8volt get process-instance --state active --auto-confirm --config /tmp/c8volt-v88.yaml
./c8volt cancel process-instance --state active --auto-confirm --config /tmp/c8volt-v88.yaml
./c8volt delete process-instance --state completed --auto-confirm --config /tmp/c8volt-v88.yaml
```

## Verification Notes

- Build paging command tests on the shared capture helpers in [`cmd/cmd_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go) so request-order assertions, page payload fixtures, and captured pagination objects stay consistent across `get`, `cancel`, and `delete`.
- For multi-page command coverage, drive sequential search responses through `newProcessInstanceSearchCaptureServerWithResponses` and assert each captured `page` object via `decodeCapturedPISearchPages` before checking command output.
- Confirm that `get process-instance`, `cancel process-instance`, and `delete process-instance` all report the page size used, current-page count, cumulative count, and continuation state in a consistent format.
- Confirm that declining a continuation prompt after processed pages yields a non-error partial-completion summary.
- Confirm that an indeterminate overflow state stops with a warning rather than silently continuing or silently finishing.
- Confirm that direct `--key` flows for `cancel` and `delete` still bypass paging behavior.
