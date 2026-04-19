# Quickstart: Version-Aware Process-Instance Paging and Overflow Handling

## Shipped Behavior

- Search-based `get process-instance`, `cancel process-instance`, and `delete process-instance` now execute one page at a time instead of silently truncating at the first page.
- The shared config key is `app.process_instance_page_size`; when unset or invalid, it normalizes to the existing default `1000`.
- `--count` overrides the shared config for the current command only.
- `--auto-confirm` continues across additional pages without prompting; otherwise the CLI prompts between pages.
- Direct `--key` flows for `cancel process-instance` and `delete process-instance` still bypass paging.
- Camunda `8.8` uses native page metadata for overflow detection. Camunda `8.7` uses the Operate fallback and stops with a warning when overflow remains indeterminate.

## Verification Focus

1. Reuse the shared process-instance helpers in [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go) and the shared capture helpers in [`cmd/cmd_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go) when extending or debugging paging behavior.
2. Treat [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md) as the source for user-facing guidance and regenerate `docs/cli/` from Cobra metadata instead of editing generated docs by hand.
3. Keep output aligned across `get`, `cancel`, and `delete`: page size used, current-page count, cumulative count, and continuation or warning state should stay in the same operator-facing shape.
4. Preserve the non-error partial-completion path when a user declines continuation after one or more pages.
5. Preserve the warning-stop path for Camunda `8.7` full pages without a trustworthy total.

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
./c8volt cancel process-instance --state active --count 250 --config /tmp/c8volt-v88.yaml
./c8volt cancel process-instance --state active --auto-confirm --config /tmp/c8volt-v88.yaml
./c8volt delete process-instance --state completed --count 250 --auto-confirm --config /tmp/c8volt-v88.yaml
./c8volt delete process-instance --state completed --auto-confirm --config /tmp/c8volt-v88.yaml
```

## Verification Notes

- Build paging command tests on the shared capture helpers in [`cmd/cmd_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go) so request-order assertions, page payload fixtures, and captured pagination objects stay consistent across `get`, `cancel`, and `delete`.
- For multi-page command coverage, drive sequential search responses through `newProcessInstanceSearchCaptureServerWithResponses` and assert each captured `page` object via `decodeCapturedPISearchPages` before checking command output.
- Confirm that `get process-instance`, `cancel process-instance`, and `delete process-instance` all report the page size used, current-page count, cumulative count, and continuation state in a consistent format.
- Confirm that declining a continuation prompt after processed pages yields a non-error partial-completion summary.
- Confirm that an indeterminate overflow state stops with a warning rather than silently continuing or silently finishing.
- Confirm that direct `--key` flows for `cancel` and `delete` still bypass paging behavior.
