# Ralph Memory

Feature: 351-user-task-assignment
Started: 2026-10-28T15:41:53Z

## Codebase Patterns

- Native user-task adapters live in `internal/services/usertask/{v810,v89,v88}`; all three now implement `AssignUserTask` and `UnassignUserTask`. Generated interfaces and unrelated service/facade fixtures include fail-closed assignment methods.
- `internal/services/common/usertask_submission.go` owns private native response evidence. `CompleteUserTaskSubmission` remains a compatibility wrapper; `AssignUserTaskSubmission` and `UnassignUserTaskSubmission` map to assignment-specific domain results. No completion/domain aliases escape the assignment boundary.
- Existing mutation retry policy is unchanged: preserve original call errors, last status and sticky ambiguity outside retry return values. Only documented 204 with request metadata and no overriding call error establishes acceptance.
- Adapter assignment evaluates options once, snapshots override, and creates fresh assignee/override pointers per attempt. Default override is explicitly false; optional action remains nil. Unassignment is bodyless DELETE, with no reads or other operations.

## Decisions

- Iteration 1 completed T001 only in the Foundational phase. T002 is a separate substantial, independently verifiable collection/facade outcome; no CLI modes or public collection methods are exposed yet.
- Kept native response state as a private record and mechanically mapped it into existing completion and new assignment domain types. Existing completion messages and behavior are preserved.
- All validation used isolated mocks or test-owned HTTP servers. Real-engine listener execution and patch-specific engine behavior remain unverified; they are not implementation gates.

## Gotchas

- `AllowOverride` lacks `omitempty` in every generated client; a nil pointer is not safe protection. `Action` serializes as null when unset, and tests assert the generated body exactly.
- A new ambiguous attempt after rejection must change submission to unknown, including 504, network timeout, and in-flight cancellation. Backoff cancellation without a new ambiguous attempt may retain rejected. Later rejection cannot clear ambiguity; later valid 204 can.
- Retry logging is globally throttled by operation; assignment adapter tests use fresh isolated subprocesses for each plain/structured format so logging assertions cannot pass from suppression alone.
- Existing test fixtures such as `completionClient` embed generated interfaces. Explicit rejecting assignment methods were added so unrelated flows fail closed on forbidden calls.

## Reusable Commands

- Use the `isolated_check` clean-environment helper from `quickstart.md` before test execution; inspect selected test families first. During iteration 1 its exact environment filtering was run via `/tmp/c8volt-351-check.py` (temporary, recreate from the guide if absent).
- Passed targeted native checks: `isolated_check go test ./internal/services/usertask/... -run 'Test(Assign|Unassign|Complete)UserTask' -count=1` and `isolated_check go test ./internal/services/common -run 'Test(Assign|Unassign|Complete)UserTask' -count=1`.
- Passed affected facade checks: `isolated_check go test ./c8volt/task -count=1`.
- Passed broader affected-package race check: `isolated_check go test ./internal/services/common ./internal/services/usertask/... ./c8volt/task -race -count=1`.
- Passed repository-wide compilation only: `isolated_check go test ./... -run '^$'`; no runtime coverage is claimed for that check. The final full race suite remains owned by T004.
- Touched Go files were formatted with `gofmt`; `git diff --check` passed.

## Do Not Repeat

- Do not add reads, tenant discovery, completion preflight, polling, compensation, generic 504/409 retries, or generated-client edits to assignment/unassignment.
- Do not reuse completion-named result types or interpret failed/not-submitted assignment as an already-completed task.
- Do not run live-environment tests or seek a target. Keep machine error evidence in results; never send raw per-key errors to normal human logging.

## Current Handoff

- Next dependency-ready work: T002, Foundational phase, shared collection assignment/unassignment coordinator and thin public facade/options. Recheck `tasks.md` before selection. T001 is complete; no blocking dependency remains for T002.
- Preserve full-input validation, frozen keys/options, ordered initialized results, started-slot tracking, separate fail-fast scheduling stop and caller context, no-wait/override rejection, and partial facade conversion. T003 depends on T002 and T004 depends on T003; neither has started.
