# Ralph Memory

Feature: 311-complete-user-tasks
Started: 2026-10-24T06:09:57Z

## Codebase Patterns

- Native completion is `API.CompleteUserTask(ctx, key, variables, opts...)` on usertask and all three adapter contracts. Each adapter maps generated nullable fields and snapshots variables with `toolx.CopyJSONMap`; nil remains nil, `{}` stays an object, and numbers retain precision.
- `internal/services/common/usertask_completion.go` owns shared native retry/evidence handling for v88/v89/v810. Adapters retain explicit generated request/response mappings. The common helper uses existing `RetryCamundaMutation`, with no task reads, assignment, legacy fallback, scope writes or polling.
- Domain statuses/submission values and field invariants are in `internal/domain/usertask_completion.go`. The collection coordinator and equivalent public models are implemented; all validation failures return initialized empty Items arrays.
- Existing service/facade test doubles now reject unexpected `CompleteUserTask` calls. Adapter completion mocks embed the generated interface so non-completion calls fail closed; an explicit rejecting transport also prevents fallback traffic.

- `usertask.CompleteUserTasks` owns validation, stable uniqueness, JSON snapshotting, preview, workers and ordered partial results. It delegates only `CompleteUserTask`; facade `task.API.CompleteUserTasks` copies/maps and converts through `ferrors.FromDomain`. Existing constructors/root embedding support completion without the variable-update dependency.
- Coordinator uses `pool.ExecuteSlice`, atomic stop state and started/result/error slots. Pool context controls scheduling; original caller context reaches native calls. Real callback errors and caller cancellation are joined; synthetic fail-fast cancellation is ignored. Native adapters own status/submission evidence; the coordinator preserves it and stamps the selected key.
- Public completion fields have explicit JSON tags; `items` is always initialized by conversion and never omitted. Native stubs for new tests embed nil API interfaces so forbidden reads/writes fail closed. The command's shared task stub now explicitly rejects unexpected collection completion.

- Completion-local CLI input is `userTaskCompletionInput` from `parseUpdateUserTaskCompletionInput(cmd, args)`: owned stable unique Keys, optional snapshotted Variables, and WantedWorkers. It uses configured stdin only for `-`, validates every key, and reuses the existing variable parser/JSON confirmation guard without changing variable mode.
- Views use `newUserTaskCompletionPreview(input)` locally and `newUserTaskCompletionReport(input, results)` for complete ordered service evidence. `renderUpdateUserTaskCompletion(cmd, report, operationErr)` writes selected-format results only: succeeded preview, explicit accepted execution, or one original-class failure envelope. The runner must log `formatUserTaskCompletionFailure` once for non-JSON errors and preserve the original error for exit.

## Decisions

- Only a valid HTTP 204 proves acceptance. Unexpected success statuses and missing HTTP metadata are malformed/unknown. Transport errors override incomplete response evidence, including a partial 204.
- Preserve last actual HTTP status across retries and backoff cancellation. Ambiguity is sticky until a later 204; a later rejection never erases it. Cancellation/deadline after only explicit rejection can retain rejected submission evidence.
- Reuse the structured resource-exhaustion parser already used by job completion (renamed `completionResourceExhausted` without behavioral change). Generic 503/proxy failures remain unknown even when the retry policy recognizes a marker.
- Retry progress uses a constant operation label, configured service logger only with verbose, and honors workflow-detail suppression. Compact calls pass nil; existing transport DEBUG is untouched.
- T001 was intentionally kept separate from T002 per the task-boundary review. Initial untracked feature artifacts and the existing AGENTS.md active-plan pointer are included with this first coordinated work-unit commit.

- T002 validates exactly 16 ASCII digits, nonempty selection, nonnegative workers and no-wait rejection before submission. Zero workers uses existing defaults; explicit workers win over no-worker-limit. Canceled/expired valid requests preserve not-attempted slots, including canceled previews.
- Facade payload copying uses `CopyJSONMap`, not the older recursive variable-map copier: invalid/cyclic inputs must return classified errors rather than recurse indefinitely. Service independently snapshots direct callers; workers share a private read-only map and native adapters snapshot their request bodies.

- T003 keeps command registration and execution deferred to T004. Completion human/verbose results use configured stdout directly; quiet suppresses only human results, JSON wins over keys, execution keys include accepted items only. Preview counts never imply submission. Zero-accepted execution says no acceptance established.
- Human reasons use allowlisted status/submission values and standard HTTP labels, with at most three first-seen reason groups plus other failures. Unknown acceptance remains unknown even when the last response was a rejection. Raw error evidence is preserved only in machine output.

## Gotchas

- `RetryCamundaMutation` returns a zero result when cancellation interrupts backoff. Never derive completion evidence solely from its returned response. Pass original call errors into its callback; prematurely classifying a 429/503 as an error disables the existing HTTP retry branch.
- `httpc.HttpStatusErr` requires response request URL metadata; generated-client mocks must populate it. Raw errors are deliberately retained for machine evidence only, never logged by the native completion path.
- Network timeout errors retain their original cause and wrap `domain.ErrGatewayTimeout` for the existing facade timeout class. A final caller cancellation takes precedence over an earlier timeout during retries.
- Mutation retry logging has a process-global throttle. The adapter logging tests use fresh test subprocesses for each plain/structured format, assert an actual INFO record, and avoid sleeps/global modifications.

- `ferrors.FromDomain` preserves the established facade class/text contract, not arbitrary original cause inspection. Tests compare facade classes/exit codes and retained item evidence; the service tests assert joined original causes.
- Synchronization barriers/counters prove worker concurrency and reverse response order without sleeps. Separate fail-fast tests prove queued slots and original-context in-flight acceptance. Cancellation after an accepted native result keeps acceptance plus the caller error.

- pflag StringSlice drops empty `--key=` occurrences, including repeated ones beside valid keys. T004 must call `trackUserTaskCompletionKeyInput(fs.Lookup("key"))` immediately after registering the key flag, including in re-registration test helpers. The wrapper delegates legacy values, implements SliceValue and clears occurrence evidence on Replace for existing reset helpers; only the completion parser rejects those empties. T003 installs it on private test commands only.
- The shared variable parser decodes JSON numbers as float64; `CopyJSONMap` snapshots normalize them to json.Number. T003 tests preserve the existing parser contract while checking nil versus nonnil empty maps and nested values. No shared parser was changed.
- The report builder consumes complete service outcomes, never fabricates missing evidence. Use existing local/bootstrap error handling before report rendering when no valid frozen selection exists. Do not call a second error renderer after a completion envelope.

## Reusable Commands

Use the quickstart clean environment wrapper for every runtime check:

```sh
isolated() {
  env -i PATH="$PATH" HOME="$HOME" TMPDIR="${TMPDIR:-/tmp}" \
    GOENV=off GOFLAGS= GOWORK=off "$@"
}
isolated go test ./internal/services/usertask/... -run 'Complete|Completion' -count=1
isolated env GORACE=atexit_sleep_ms=0 go test ./internal/services/usertask/... ./c8volt/task ./internal/services/common -race -count=1
isolated go test ./c8volt/task -run 'Complete|Completion' -count=1
isolated go test ./cmd -run 'UpdateUserTask|GetUserTask|HasUserTasks|UserTaskHTTPFallback' -count=1
isolated go test ./... -run '^$'
isolated go test ./cmd -run 'TestUpdateUserTaskCompletionInput|TestUpdateUserTaskCompletionView' -count=1
isolated go test ./cmd -run 'TestUpdateUserTask|TestUpdateProcessInstanceCommand_InvalidVarsFailBeforeMutation' -count=1
isolated env GORACE=atexit_sleep_ms=0 go test ./cmd -run 'TestUpdateUserTaskCompletion' -race -count=1
git diff --check
```

The adapter/coordinator and facade focused checks, complete usertask/task race suites, affected CLI regressions and repository compilation passed in iteration 2. Iteration 1 also validated the common helper and job consumer. The final command compiling Go packages does not execute their tests. Untagged integration helpers skip live checks; no integration tags or live targets were used.

Iteration 3 passed the focused input/view checks, existing user-task command/terminal/output regressions plus shared variable-input failures, and focused completion race checks. These are local completion preparation tests plus existing variable-mode execution regressions, not completion command acceptance. No live environment was contacted.

## Do Not Repeat

- Do not add native polling, task/variable reads, assignment, tenant filtering, new retries, or generated-client edits. Acceptance is the only native success claim.
- Do not log raw completion Error strings or joined backend errors. T002 preserves partial evidence and classified errors; T003 provides safe rendering/aggregate formatting; T004 must wire configured logging and exits.
- Do not claim command-level completion diagnostics or live-engine completion verification from T001–T003 tests. The full suite and CLI documentation gate remain part of T004 delivery.

## Current Handoff

- Next dependency-ready work is Phase 3 / US1 T004: register and execute completion, wire the existing input/report helpers and empty-key occurrence tracker, then deliver command-level diagnostics, terminal/exit/output coverage, metadata/docs generation, compatibility checks and the final isolated full race suite. Recheck tasks.md before selection. T001–T003 are complete; no blocking dependency remains. Do not duplicate local parsing or preview logic. No actual completion command flag or lifecycle has been exposed yet.
