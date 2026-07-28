# Ralph Memory

Feature: 262-activity-priority
Started: 2026-07-28T09:17:49Z

## Codebase Patterns
- `toolx/logging/activity.go` currently has a single visible message plus `refs`; nested `StartActivity` calls only increment refs and do not own independently selectable scopes. `UpdateActivity` rewrites the shared message only when the writer is active.
- Backward-compatible context helpers are `ToActivityContext`, `ActivityFromContext`, `StartActivity`, and `UpdateActivity`. Existing call sites expect `StartActivity(ctx, msg)` to return an idempotent stop function and no-op when no sink exists.
- `testx/activitysink.Sink` records start messages and update messages separately through `Snapshot`, `Messages`, and `Updates`; it currently has no notion of priority, scope IDs, or visible-message selection.
- Command progress emitters already centralize high-level activity updates in `cmd/get_processinstance_paging.go`, `cmd/get_processinstance_total.go`, `cmd/get_processinstance_orphan.go`, `cmd/processinstance_mutation_progress.go`, and `cmd/ops_analyse_slow_process_instances.go`; these are the right places to promote workflow importance.
- Service-level bulk work in `internal/services/processinstance/bulk.go`, variable updates in `internal/services/processinstance/variables.go`, process-definition delete planning/execution in `internal/services/processdefinition/delete.go`, deployment confirmation in `internal/services/resource/v88/service.go` and `internal/services/resource/v89/service.go`, and waiters/poller all use unprioritized `logging.StartActivity`.
- `internal/services/httpc.LogTransport` starts activity around each request with `httpActivityMessage(req)` and already strips hosts/version prefixes. It covers topology, process-instance, incident, job, process-definition, and basic resource labels, but not every path required by `contracts/http-activity-labels.md`.

## Decisions
- Phase 2 should keep legacy `ActivitySink`/`ActivityUpdater` compatibility while adding optional priority-aware interfaces so existing call sites and tests continue to compile.
- Priority-aware writer tests should exercise the concrete writer directly for visible selection and the context helpers for compatibility routing.

## Gotchas
- `activityWriter` tests force `enabled=true` through `newActivityWriter` to bypass terminal detection; `NewActivityWriterEnabled` still gates on `isInteractiveTerminal`.
- `toolx/poller.WaitForCompletion` can either use context activity or print dots to `os.Stderr` when `noProgress` is false and no activity sink exists; preserve that fallback unless a later task explicitly changes it.
- Command transient updates are guarded by `ops.ProgressChannel.TransientAllowed` or command output mode checks. Do not bypass these guards when promoting workflow importance.

## Reusable Commands
- `go test ./toolx/logging ./internal/services/httpc ./testx/activitysink -count=1`

## Do Not Repeat

## Current Handoff
- Next iteration should start Phase 2 with T006-T010: add priority/metadata and backward-compatible helper APIs in `toolx/logging/activity.go`, extend `testx/activitysink`, and cover priority ordering, tie-breaking, stop fallback, idempotent stops, disabled writer behavior, and context helper routing before touching US1 call sites.
