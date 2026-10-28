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

- `cmd/update_usertask_input.go` now owns `userTaskKeyValue`, `trackUserTaskKeyInput` and `collectUserTaskMutationKeys`. Completion delegates only strict key collection; its payload decoding and variable-mode behavior are unchanged. The real command and reset-contract tests use the neutral tracker name.
- Assignment input reads new selectors from command-local flags: `resolveUpdateUserTaskAssignmentMode` returns `assign`, `unassign`, or empty for legacy dispatch; it rejects mode conflicts and override presence outside assignment. `parseUpdateUserTaskAssignmentInput` validates controls, snapshots unique keys and opaque identity, and returns automatic workers (zero) when omitted. The real command now registers these flags and resolves conflicts before legacy dispatch.
- `newUserTaskAssignmentPreview` and `newUserTaskAssignmentReport` own copied arrays/intent and assignment-specific public results. Pointer fields preserve explicit false override on assign and omit both assignment fields on unassign. Missing service items remain absent; validation errors use the existing payload-free envelope.
- `renderUpdateUserTaskAssignment` emits JSON/keys before checking quiet, requires a configured logger for human output, and lists every selected key for bulk previews. `%q` escapes identity control characters without changing backend bytes. `formatUserTaskAssignmentFailure` groups at most three known reasons plus remainder; timed-out/canceled unknown submissions retain explicit uncertainty. No completion-conflict inference or raw error interpolation enters human text.

- `cmd/update_usertask_assignment.go` owns local preview/approval and action-specific facade dispatch. It uses `attachExplicitKeysTenantContext` and `collectExplicitAdminInputOptions`, adding only explicit assignment override. It never calls completion preparation, reads, scheduling or polling.
- The runner renders selected results once, logs one bounded non-JSON ERROR, then calls `exitAfterRenderedResult` with the original operation error. A failed write gets one compact configured diagnostic and retains operation-error precedence. `renderUserTaskAssignmentPlan` stays in the view file; bulk approval lists every frozen key, and the prompt safely quotes opaque identity bytes.
- Assignment command tests share a mutation-only backend that rejects all reads and unrelated routes. Existing completion-named subprocess/terminal/envelope/log-record utilities are command-agnostic and reused; the completion backend/preflight fixture is never reused for assignment.

## Decisions

- Iterations 1–2 completed Foundational T001/T002; iteration 3 completed US1 T003. Iteration 4 completed T004, exposing both modes with actual command, logger, terminal and documentation coverage. All feature tasks are complete.
- Kept native response state as a private record and mechanically mapped it into existing completion and new assignment domain types. Existing completion messages and behavior are preserved.
- All validation used isolated mocks or test-owned HTTP servers. Real-engine listener execution and patch-specific engine behavior remain unverified; they are not implementation gates.

## Gotchas

- `AllowOverride` lacks `omitempty` in every generated client; a nil pointer is not safe protection. `Action` serializes as null when unset, and tests assert the generated body exactly.
- A new ambiguous attempt after rejection must change submission to unknown, including 504, network timeout, and in-flight cancellation. Backoff cancellation without a new ambiguous attempt may retain rejected. Later rejection cannot clear ambiguity; later valid 204 can.
- Retry logging is globally throttled by operation; assignment adapter tests use fresh isolated subprocesses for each plain/structured format so logging assertions cannot pass from suppression alone.
- Existing test fixtures such as `completionClient` embed generated interfaces. Explicit rejecting assignment methods were added so unrelated flows fail closed on forbidden calls.

- Use `testing/synctest` plus gates for fail-fast with accepted in-flight requests and queued remainder: `synctest.Wait()` lets the rejection and scheduling stop finish while the successful request remains blocked, without timing assumptions. Worker bounds and out-of-order completion use barriers/counters.

- `renderHumanLine` has a legacy stdout fallback when no logger is attached. The assignment renderer explicitly checks `logging.FromContext` first, returns the setup error and writes neither stream. Future runner setup must attach the configured logger before any human view call.
- Existing `bulkCompletionDiagnosticRecords` supports the active `plain-time` and `json` formats and returns `[2]string` level/message pairs. A bare `plain` fixture uses a different timestamp and is not compatible with that helper; use `plain-time` for active-format checks.

- `TestAllCommandHelpFocusesOnActions` forbids presentation wording such as stdout, stderr and zero bytes in command Short/Long text. Keep detailed stream contracts in README; update parent-help assertions in both job and process-instance help tests when the update group description changes.
- Darwin's real-terminal runner reports support and fails allocation errors rather than skipping them. Both new actions passed actual terminal stdin tests with redirected stdout and separately captured configured/inherited stderr; no terminal coverage was skipped.

## Reusable Commands

- Use the `isolated_check` clean-environment helper from `quickstart.md` before test execution; inspect selected test families first. During iteration 1 its exact environment filtering was run via `/tmp/c8volt-351-check.py` (temporary, recreate from the guide if absent).
- Passed targeted native checks: `isolated_check go test ./internal/services/usertask/... -run 'Test(Assign|Unassign|Complete)UserTask' -count=1` and `isolated_check go test ./internal/services/common -run 'Test(Assign|Unassign|Complete)UserTask' -count=1`.
- Passed affected facade checks: `isolated_check go test ./c8volt/task -count=1`.
- Passed broader affected-package race check: `isolated_check go test ./internal/services/common ./internal/services/usertask/... ./c8volt/task -race -count=1`.
- Passed repository-wide compilation only: `isolated_check go test ./... -run '^$'`; no runtime coverage is claimed for that check. The final full race suite passed in iteration 4.
- Iteration 2 passed targeted collection/facade checks: `isolated_check go test ./internal/services/usertask/... -run 'Test(Assign|Unassign)UserTasks' -race -count=1`; `isolated_check go test ./c8volt/task -run 'Test(Assign|Unassign|Complete)UserTasks|TestAssignmentPublicModels' -count=1`; `isolated_check go test ./c8volt/foptions -run 'TestUserTaskAssignmentOverride' -count=1`. Collection families run in the neutral service package; adapter packages use singular native test names and were covered by the broader check below.
- Iteration 2 passed `isolated_check go test ./internal/services/usertask/... ./c8volt/task ./c8volt/foptions -race -count=1`, preserving completion/read/resolve/variable regressions and native assignment tests. Also passed `isolated_check go test ./c8volt -run '^TestNew_' -count=1` and repository compilation `isolated_check go test ./... -run '^$'`. Full repository runtime/race validation passed in iteration 4.
- Touched Go files were formatted with `gofmt`; `git diff --check` passed.

- Iteration 3 passed `isolated_check go test ./cmd -run 'TestUpdateUserTask(Assignment|Unassignment|Completion)Input|Test.*UserTaskAssignment.*(Report|Render|View)' -count=1` and broader `isolated_check go test ./cmd -run 'Test(UpdateUserTask|UserTaskAssignment|UserTaskVariableUpdate|CommandCapabilityForCommand_UpdateUserTaskContract|ParseUpdate)' -count=1`. The latter covers existing completion/variable command, output, diagnostic and terminal families plus shared tracker reset metadata. Test fixtures use private commands or test-owned HTTP/subprocess/terminal dependencies. No new assignment command or terminal coverage is claimed before T004.

- Iteration 4 passed `isolated_check go test ./cmd -run 'TestUpdateUserTask(Assignment|Unassignment)|TestCommandCapabilityForCommand_UpdateUserTaskContract' -count=1` and `isolated_check go test ./cmd -run 'TestUpdateUserTaskAssignmentCommand(Guardrails|AwaitsResponse)' -count=1`.
- Passed legacy/new command regressions: `isolated_check go test ./cmd -run 'Test(UpdateUserTask|GetUserTask|GetProcessInstance.*UserTask|UserTaskVariableUpdate|CommandCapabilityForCommand_UpdateUserTaskContract|ParseUpdate)' -count=1`.
- The first full race run found only two stale parent-help expectations and the action-focused help wording check. After correcting those, `isolated_check go test ./cmd -run 'Test.*(Help|Contract|Capability)' -race -count=1` passed. A final `isolated_check make test` passed repository-wide, closing the failed quality gate. `isolated_check make docs-content` regenerated CLI references and the homepage from source.
- Full-suite isolation review: 58 live-test files remain guarded by `integration`; the eight untagged files under integration paths contain only local parser/fixture/unit checks and one test-owned HTTP server. No live suite/target was enabled. Keep the clean-environment helper when reproducing checks.

## Do Not Repeat

- Do not add reads, tenant discovery, completion preflight, polling, compensation, generic 504/409 retries, or generated-client edits to assignment/unassignment.
- Do not reuse completion-named result types or interpret failed/not-submitted assignment as an already-completed task.
- Do not run live-environment tests or seek a target. Keep machine error evidence in results; never send raw per-key errors to normal human logging.

## Current Handoff

- Feature complete; no handoff required.
