# Ralph Memory

Feature: 311-complete-user-tasks
Started: 2026-10-24T06:09:57Z

## Codebase Patterns

- Native completion is `API.CompleteUserTask(ctx, key, variables, opts...)` on usertask and all three adapter contracts. Each adapter maps generated nullable fields and snapshots variables with `toolx.CopyJSONMap`; nil remains nil, `{}` stays an object, and numbers retain precision.
- `internal/services/common/usertask_completion.go` owns shared native retry/evidence handling for v88/v89/v810. Adapters retain explicit generated request/response mappings. The common helper uses existing `RetryCamundaMutation`, with no task reads, assignment, legacy fallback, scope writes or polling.
- Domain statuses/submission values and field invariants are in `internal/domain/usertask_completion.go`. The collection type exists, but the coordinator/public models are still T002 work.
- Existing service/facade test doubles now reject unexpected `CompleteUserTask` calls. Adapter completion mocks embed the generated interface so non-completion calls fail closed; an explicit rejecting transport also prevents fallback traffic.

## Decisions

- Only a valid HTTP 204 proves acceptance. Unexpected success statuses and missing HTTP metadata are malformed/unknown. Transport errors override incomplete response evidence, including a partial 204.
- Preserve last actual HTTP status across retries and backoff cancellation. Ambiguity is sticky until a later 204; a later rejection never erases it. Cancellation/deadline after only explicit rejection can retain rejected submission evidence.
- Reuse the structured resource-exhaustion parser already used by job completion (renamed `completionResourceExhausted` without behavioral change). Generic 503/proxy failures remain unknown even when the retry policy recognizes a marker.
- Retry progress uses a constant operation label, configured service logger only with verbose, and honors workflow-detail suppression. Compact calls pass nil; existing transport DEBUG is untouched.
- T001 was intentionally kept separate from T002 per the task-boundary review. Initial untracked feature artifacts and the existing AGENTS.md active-plan pointer are included with this first coordinated work-unit commit.

## Gotchas

- `RetryCamundaMutation` returns a zero result when cancellation interrupts backoff. Never derive completion evidence solely from its returned response. Pass original call errors into its callback; prematurely classifying a 429/503 as an error disables the existing HTTP retry branch.
- `httpc.HttpStatusErr` requires response request URL metadata; generated-client mocks must populate it. Raw errors are deliberately retained for machine evidence only, never logged by the native completion path.
- Network timeout errors retain their original cause and wrap `domain.ErrGatewayTimeout` for the existing facade timeout class. A final caller cancellation takes precedence over an earlier timeout during retries.
- Mutation retry logging has a process-global throttle. The adapter logging tests use fresh test subprocesses for each plain/structured format, assert an actual INFO record, and avoid sleeps/global modifications.

## Reusable Commands

Use the quickstart clean environment wrapper for every runtime check:

```sh
isolated() {
  env -i PATH="$PATH" HOME="$HOME" TMPDIR="${TMPDIR:-/tmp}" \
    GOENV=off GOFLAGS= GOWORK=off "$@"
}
isolated go test ./internal/services/usertask/... -run 'Complete|Completion' -count=1
isolated env GORACE=atexit_sleep_ms=0 go test ./internal/services/usertask/... ./c8volt/task ./internal/services/common -race -count=1
isolated go test ./internal/services/job/... -run 'Completion|Complete' -count=1
isolated go test ./... -run '^$'
git diff --check
```

All commands above passed in iteration 1. The final command compiling Go packages does not execute their tests. Untagged integration helpers skip live checks; no integration tags or live targets were used.

## Do Not Repeat

- Do not add native polling, task/variable reads, assignment, tenant filtering, new retries, or generated-client edits. Acceptance is the only native success claim.
- Do not log raw completion Error strings or joined backend errors. T002 must preserve partial evidence and classified errors; T003/T004 own safe rendering and configured aggregate logging.
- Do not claim command-level diagnostics, public collection behavior, or live-engine completion verification from T001 tests. The full suite and CLI documentation gate remain part of T004 delivery.

## Current Handoff

- Next dependency-ready work is Phase 2 Foundational T002: collection coordinator plus thin public facade and their validation. Recheck tasks.md before selecting it; T001 is complete, T003 depends on T002, and T004 depends on T003. No blocking dependency remains for T002.
