# Ralph Memory

Feature: 262-activity-priority
Started: 2026-07-28T09:17:49Z

## Codebase Patterns
- `toolx/logging/activity.go` now tracks active activity scopes with a monotonically increasing ID/order and selects the visible scope by highest `ActivityImportance`, then newest equal-priority scope. Stopping a visible scope redraws the next-best remaining scope; stopping the last active scope clears the indicator.
- Legacy `StartActivity`/`StopActivity` remains available and records legacy scopes at `ActivityImportanceBatch`; unscoped `UpdateActivity` updates the newest active scope. New helpers are `StartActivityWithImportance(ctx, msg, importance)` and `UpdateActivityWithImportance(ctx, msg, importance)`, and they fall back to legacy interfaces when a sink does not implement the priority-aware optional interfaces.
- `ActivityImportance` is an `int` alias so external test sinks can implement priority-aware interfaces without importing `toolx/logging`; ordered constants are `ActivityImportanceHTTP`, `ActivityImportanceWait`, `ActivityImportanceBatch`, and `ActivityImportanceWorkflow`.
- `testx/activitysink.Sink` preserves `Snapshot`, `Messages`, and `Updates`, and now also records priority-aware `Starts()` and `PriorityUpdates()` with idempotent scoped stop functions.
- Command progress emitters already centralize high-level activity updates in `cmd/get_processinstance_paging.go`, `cmd/get_processinstance_total.go`, `cmd/get_processinstance_orphan.go`, `cmd/processinstance_mutation_progress.go`, and `cmd/ops_analyse_slow_process_instances.go`; these are the right places to promote workflow importance.
- Service-level bulk work in `internal/services/processinstance/bulk.go`, variable updates in `internal/services/processinstance/variables.go`, process-definition delete planning/execution in `internal/services/processdefinition/delete.go`, deployment confirmation in `internal/services/resource/v88/service.go` and `internal/services/resource/v89/service.go`, and waiters/poller all use unprioritized `logging.StartActivity`.
- `internal/services/httpc.LogTransport` starts activity around each request with `httpActivityMessage(req)` and already strips hosts/version prefixes. It covers topology, process-instance, incident, job, process-definition, and basic resource labels, but not every path required by `contracts/http-activity-labels.md`.

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

## Do Not Repeat

## Current Handoff
- Next iteration should start User Story 1 only, beginning with T011-T016 tests before implementation. Promote command-level progress through `StartActivityWithImportance`/`UpdateActivityWithImportance` using `ActivityImportanceWorkflow`, service bulk scopes with `ActivityImportanceBatch`, and waiters/poller with `ActivityImportanceWait`; keep HTTP fallback for US2.
