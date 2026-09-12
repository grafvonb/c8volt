# Ralph Memory

Feature: 299-empty-selector-output
Started: 2026-09-11T09:51:18Z

## Codebase Patterns

- Empty selector completion is authoritative only after `PlanProcessInstanceMutationPages` returns successfully with aggregate `RequestedCount == 0`.
- Final process-instance output belongs in `cmd/cmd_views_processinstance.go`; JSON uses the shared succeeded envelope, while keys-only emits no bytes.
- `executeRootForProcessInstanceWithSeparateOutputs` plus the empty HTTP search fixture exercises real Cobra mode precedence without subprocess output merging.
- Quiet suppression belongs after `pickMode()` selects the human branch so explicit JSON remains visible and keys-only remains byte-empty.
- Exact empty-scope request coverage uses one `/v2/process-instances/search` for state/date selectors and `/v2/process-definitions/search` followed by one instance search for BPMN selectors; fixture default branches reject mutation or extra discovery.
- Real-terminal empty-scope coverage can run with zero `CmdTerminalExchange` values: pass the config and command matrix through environment variables, execute with buffered root streams in the child, and copy those streams to subprocess stdout/stderr for independent parent assertions.
- README guidance for selector-based cancellation and deletion belongs in the existing `Cancel And Delete Safely` workflow section; command-specific help and generated references remain sourced from Cobra metadata.
- `make docs-content` propagates Cobra metadata into `docs/cli/` and README content into `docs/index.md`; generated pages should be reviewed but not edited directly.

## Decisions

- US1 intentionally retains the existing human `found: 0` fallback; quiet suppression is deferred to T012 in US2.
- Empty normal JSON reuses `process.DeleteReports{}` / `process.CancelReports{}` and empty dry-run JSON reuses `newProcessInstanceDryRunSummary(operation, nil)`.

## Gotchas

- Command-local flags such as `--dry-run` and `--no-wait` must appear after the destructive subcommand path in direct Cobra test arguments.
- Do not use `renderCommandResult` for an empty no-wait result because it would classify the no-op as accepted.

## Reusable Commands

- `go test ./cmd -run 'Test(Delete|Cancel)ProcessInstanceEmptySelectorOutput|TestProcessInstance.*Empty' -count=1`
- `make test`

## Do Not Repeat

- Do not infer successful emptiness from report or preview slice length; aborted and nonempty workflows can also return empty slices.

## Current Handoff
- Feature complete; no handoff required.
