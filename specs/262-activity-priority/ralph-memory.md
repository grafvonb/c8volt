# Ralph Memory

Feature: 262-activity-priority
Started: 2026-07-28T09:17:49Z

## Codebase Patterns
- `toolx/logging/activity.go` now tracks active activity scopes with a monotonically increasing ID/order and selects the visible scope by highest `ActivityImportance`, then newest equal-priority scope. Stopping a visible scope redraws the next-best remaining scope; stopping the last active scope clears the indicator.
- Legacy `StartActivity`/`StopActivity` remains available and records legacy scopes at `ActivityImportanceBatch`; unscoped `UpdateActivity` updates the newest active scope. New helpers are `StartActivityWithImportance(ctx, msg, importance)` and `UpdateActivityWithImportance(ctx, msg, importance)`, and they fall back to legacy interfaces when a sink does not implement the priority-aware optional interfaces.
- `ActivityImportance` is an `int` alias so external test sinks can implement priority-aware interfaces without importing `toolx/logging`; ordered constants are `ActivityImportanceHTTP`, `ActivityImportanceWait`, `ActivityImportanceBatch`, and `ActivityImportanceWorkflow`.
- `testx/activitysink.Sink` preserves `Snapshot`, `Messages`, and `Updates`, and now also records priority-aware `Starts()` and `PriorityUpdates()` with idempotent scoped stop functions.
- Command progress emitters in `cmd/get_processinstance_paging.go`, `cmd/get_processinstance_total.go`, `cmd/get_processinstance_orphan.go`, `cmd/processinstance_mutation_progress.go`, and `cmd/ops_analyse_slow_process_instances.go` now use workflow importance for transient starts and updates.
- Service-level bulk work in `internal/services/processinstance/bulk.go`, variable updates in `internal/services/processinstance/variables.go`, process-definition delete planning/execution in `internal/services/processdefinition/delete.go`, and deployment confirmation in `internal/services/resource/v88/service.go` and `internal/services/resource/v89/service.go` now use batch importance.
- Process-instance, incident, and job waiters plus `toolx/poller.WaitForCompletion` now use wait importance. Poller fallback dots to `os.Stderr` remain unchanged when no activity sink exists.
- `internal/services/httpc.LogTransport` starts each request as `ActivityImportanceHTTP` through the priority-aware sink when available, so request fallback stays below workflow/batch/wait scopes while remaining visible for simple commands.
- `internal/services/httpc.httpActivityMessage` strips hosts, query strings, and a leading `/v2` prefix, then maps all known labels from `contracts/http-activity-labels.md`, including legacy `/deployments`, `/resources/{key}/deletion`, `/process-instances/search`, `/process-definitions/search`, and `/variables/search`; unknown method/path combinations keep the generic Camunda API fallback wording.
- US2 command tests use small command-helper facade stubs to record HTTP-priority fallback starts through the command context for cluster, tenant, resource, incident, job, variable, element-instance, and user-task representatives. The actual endpoint label table and priority value are enforced in `internal/services/httpc/round_trippers_test.go`.
- Root activity indicator gating now explicitly disables transient indicators when `--json` or `--keys-only` is active, in addition to `--no-indicator`, quiet, automation, JSON log format, and non-interactive writer gating.
- US3 machine-output tests cover paged process-instance JSON and keys-only streams, cancel/delete quiet and automation suppression, simple cluster and tenant fallback suppression, durable debug endpoint details, writer clearing before log/warn/prompt/final output, and priority helper no-op behavior without an activity sink.

## Decisions
- Priority-aware APIs are additive optional interfaces: `PriorityActivitySink` and `PriorityActivityUpdater`; context helpers probe these interfaces before falling back to `ActivitySink`/`ActivityUpdater`.
- Priority-aware writer tests exercise the concrete writer directly for visible selection and the context helpers for compatibility routing.

## Gotchas
- `activityWriter` tests force `enabled=true` through `newActivityWriter` to bypass terminal detection; `NewActivityWriterEnabled` still gates on `isInteractiveTerminal`.
- Hidden lower-priority updates must not redraw the visible spinner; `UpdateActivityWithImportance` only redraws when the updated scope is currently selected.
- `toolx/poller.WaitForCompletion` can either use context activity or print dots to `os.Stderr` when `noProgress` is false and no activity sink exists; preserve that fallback unless a later task explicitly changes it.
- Command transient updates are guarded by `ops.ProgressChannel.TransientAllowed` or command output mode checks. Do not bypass these guards when promoting workflow importance.

## Reusable Commands
- `go test ./toolx/logging ./internal/services/httpc ./testx/activitysink -count=1`
- `go test ./toolx/logging ./internal/services/httpc ./testx/activitysink ./toolx/poller -count=1`
- `go test ./toolx/logging ./cmd ./internal/services/processinstance/... ./internal/services/incident/... ./internal/services/job/... ./internal/services/processdefinition/... ./internal/services/resource/... ./toolx/poller -run 'Activity|Progress|Indicator|RunProcessInstance|DeleteProcessInstance|Ops|GetProcessInstance|Wait' -count=1`
- `go test ./cmd ./toolx/logging -run 'JSON|KeysOnly|Quiet|Automation|Machine|ActivityWriter|IndicatorEnabled|NoOpWithoutContextSink' -count=1`
- `go test ./cmd ./toolx/logging -count=1`

## Do Not Repeat

## Current Handoff
- Feature complete; no handoff required.
