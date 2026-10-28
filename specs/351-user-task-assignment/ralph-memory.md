# Ralph Memory

Feature: 351-user-task-assignment
Started: 2026-10-28T15:41:53Z

## Codebase Patterns

- Native user-task adapters live in `internal/services/usertask/{v810,v89,v88}`; all three now implement `AssignUserTask` and `UnassignUserTask`. Generated interfaces and unrelated service/facade fixtures include fail-closed assignment methods.
- `internal/services/common/usertask_submission.go` owns private native response evidence. `CompleteUserTaskSubmission` remains a compatibility wrapper; `AssignUserTaskSubmission` and `UnassignUserTaskSubmission` map to assignment-specific domain results. No completion/domain aliases escape the assignment boundary.
- Existing mutation retry policy is unchanged: preserve original call errors, last status and sticky ambiguity outside retry return values. Only documented 204 with request metadata and no overriding call error establishes acceptance.
- Adapter assignment evaluates options once, snapshots override, and creates fresh assignee/override pointers per attempt. Default override is explicitly false; optional action remains nil. Unassignment is bodyless DELETE, with no reads or other operations.

- `internal/services/usertask/assignment.go` owns both collection actions through one private coordinator. Public `c8volt/task` methods only map options, delegate, convert all partial results and apply `ferrors.FromDomain`. Both existing constructors and the root embedded task API remain compatible.
- The coordinator copies/deduplicates keys with `Keys.Unique`, evaluates call options once into a value copy (not the caller-retained config pointer), then supplies a frozen option that copies into each adapter config. Reusing caller option closures per key would change intent. Both actions reject no-wait; only assignment permits explicit override.
- Ordered results are initialized before cancellation/preview/scheduling. A started marker distinguishes absent pool slots; join only native started errors and caller cancellation, ignoring pool scheduling-cancellation artifacts. Submissions receive the original caller context while fail-fast uses the pool context and an atomic stop flag.
- `validUserTaskKey` is the shared exact ASCII grammar (renamed from `validCompletionKey` in `completion.go` without changing completion validation behavior). No facade/service trimming of keys or nonblank identities.

## Decisions

- Iteration 1 completed T001 and iteration 2 completed T002, both in the Foundational phase. Public assignment/unassignment collection APIs now work; CLI input/report work (T003) and command delivery (T004) remain separate US1 tasks.
- Kept native response state as a private record and mechanically mapped it into existing completion and new assignment domain types. Existing completion messages and behavior are preserved.
- All validation used isolated mocks or test-owned HTTP servers. Real-engine listener execution and patch-specific engine behavior remain unverified; they are not implementation gates.

## Gotchas

- `AllowOverride` lacks `omitempty` in every generated client; a nil pointer is not safe protection. `Action` serializes as null when unset, and tests assert the generated body exactly.
- A new ambiguous attempt after rejection must change submission to unknown, including 504, network timeout, and in-flight cancellation. Backoff cancellation without a new ambiguous attempt may retain rejected. Later rejection cannot clear ambiguity; later valid 204 can.
- Retry logging is globally throttled by operation; assignment adapter tests use fresh isolated subprocesses for each plain/structured format so logging assertions cannot pass from suppression alone.
- Existing test fixtures such as `completionClient` embed generated interfaces. Explicit rejecting assignment methods were added so unrelated flows fail closed on forbidden calls.

- Use `testing/synctest` plus gates for fail-fast with accepted in-flight requests and queued remainder: `synctest.Wait()` lets the rejection and scheduling stop finish while the successful request remains blocked, without timing assumptions. Worker bounds and out-of-order completion use barriers/counters.

## Reusable Commands

- Use the `isolated_check` clean-environment helper from `quickstart.md` before test execution; inspect selected test families first. During iteration 1 its exact environment filtering was run via `/tmp/c8volt-351-check.py` (temporary, recreate from the guide if absent).
- Passed targeted native checks: `isolated_check go test ./internal/services/usertask/... -run 'Test(Assign|Unassign|Complete)UserTask' -count=1` and `isolated_check go test ./internal/services/common -run 'Test(Assign|Unassign|Complete)UserTask' -count=1`.
- Passed affected facade checks: `isolated_check go test ./c8volt/task -count=1`.
- Passed broader affected-package race check: `isolated_check go test ./internal/services/common ./internal/services/usertask/... ./c8volt/task -race -count=1`.
- Passed repository-wide compilation only: `isolated_check go test ./... -run '^$'`; no runtime coverage is claimed for that check. The final full race suite remains owned by T004.
- Iteration 2 passed targeted collection/facade checks: `isolated_check go test ./internal/services/usertask/... -run 'Test(Assign|Unassign)UserTasks' -race -count=1`; `isolated_check go test ./c8volt/task -run 'Test(Assign|Unassign|Complete)UserTasks|TestAssignmentPublicModels' -count=1`; `isolated_check go test ./c8volt/foptions -run 'TestUserTaskAssignmentOverride' -count=1`. Collection families run in the neutral service package; adapter packages use singular native test names and were covered by the broader check below.
- Iteration 2 passed `isolated_check go test ./internal/services/usertask/... ./c8volt/task ./c8volt/foptions -race -count=1`, preserving completion/read/resolve/variable regressions and native assignment tests. Also passed `isolated_check go test ./c8volt -run '^TestNew_' -count=1` and repository compilation `isolated_check go test ./... -run '^$'`. Full repository runtime/race validation remains T004.
- Touched Go files were formatted with `gofmt`; `git diff --check` passed.

## Do Not Repeat

- Do not add reads, tenant discovery, completion preflight, polling, compensation, generic 504/409 retries, or generated-client edits to assignment/unassignment.
- Do not reuse completion-named result types or interpret failed/not-submitted assignment as an already-completed task.
- Do not run live-environment tests or seek a target. Keep machine error evidence in results; never send raw per-key errors to normal human logging.

## Current Handoff

- Next dependency-ready work: T003 [US1], shared local input-to-report contract for assignment/unassignment. Recheck `tasks.md` before selecting it. T001 and T002 are complete; no blocking dependency remains for T003. T004 depends on T003 and has not started.
- T003 owns strict CLI key parsing/presence tracking, frozen input and assignment result views without exposing new flags. Preserve completion/legacy variable parsing and all JSON, keys-only, configured logger and quiet contracts. T004 owns command registration/approval, full command/terminal/diagnostic checks, docs and final full-suite validation.
- Public APIs are ready: `AssignUserTasks` and `UnassignUserTasks` return assignment-specific ordered results even on error. `WithUserTaskAssignmentOverride` is explicit opt-in; unassignment rejects it. Existing `stubTaskAPI` methods fail closed; new command fixtures must provide intended assignment behavior explicitly.
