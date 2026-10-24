# Research: Complete User Tasks by Explicit Key

**Date**: 2026-10-24 | **Issue**: #311 | **Specification**: [spec.md](spec.md)

Research used the current repository, vendored API contract, and three read-only research agents covering services/adapters, command/output behavior, and isolated validation. No live service or runtime test was used. All design unknowns are resolved below.

## 1. Extend the existing command with a focused mode

**Decision**: Add `--complete` and dispatch from `cmd/update_usertask.go` into `cmd/update_usertask_complete.go` before mandatory variable parsing and variable planning. Keep completion rendering in `cmd/cmd_views_usertask_complete.go`. Preserve the ordinary variable-update path and its schemas.

**Rationale**: Completion does not have a variable-diff or verification lifecycle. The existing path calls `planUpdateUserTaskVariables`, `userTaskVariablePlanHasChanges`, and `ExecuteUserTaskVariableUpdates`, all of which have incompatible no-op or scope semantics. The dedicated mode also satisfies command-file cohesion rules.

**Alternatives considered**: Adding a boolean to variable planning risks skipping completion when values already match. Copying job batch mode wholesale imports discovery, state reconciliation, and unrelated job policies.

**Evidence**: `cmd/update_usertask.go`, `cmd/update_usertask_variables.go`, `cmd/update_job_batch.go`.

## 2. Validate the entire explicit selection locally

**Decision**: Use a completion-local parser with flag keys preceding stdin keys, stable deduplication, and the existing user-task stdin key convention (exactly 16 decimal digits) applied to every input source. Ignore blank stdin lines but reject malformed/nonempty tokens and empty flag tokens. Read stdin only for positional `-`, through the configured input reader. Reject an empty final selection; valid flags plus empty stdin remain valid. Validate all input before any completion call.

**Rationale**: `mergeAndValidateKeys` validates stdin keys but not flag keys. `readKeysIfDash` rejects empty stdin before combining it with flags and reads process stdin directly. Copying either unchanged would miss acceptance cases. Job batch parsing provides the full-input pattern but its positive-int64 grammar is broader than the user-task convention.

**Alternatives considered**: Changing the shared reader or variable mode would broaden scope and alter existing behavior. Adopting job's key grammar would silently change the user-task convention. No discovery fallback is permitted.

**Evidence**: `cmd/cmd_cli.go`, `cmd/cmd_stdin.go`, `cmd/update_job_batch.go:parseUpdateJobExplicitSelection`.

## 3. Optional variables retain their JSON identity

**Decision**: Call `parseUpdateVariablesFromFlags` only when a source flag is present. Omitted source produces a nil map; explicit `{}` produces a nonnil empty map. Reject two sources, unreadable files, invalid JSON, non-objects, and trailing JSON content. Snapshot nested values using `toolx.CopyJSONMap` and the existing task facade copying pattern; validate before workers start.

**Rationale**: The existing parser already enforces object payloads but requires a source. Keeping omission distinct from an empty object proves that neither case suppresses completion. A shared read-only snapshot prevents accidental mutation between worker submissions.

**Alternatives considered**: Mandatory variables would exclude native completion without data. Comparing effective variables would add reads and scope semantics explicitly excluded by the issue.

**Evidence**: `cmd/update_variables_payload.go`, `c8volt/task/convert.go`, `internal/services/usertask/update.go`, `toolx` JSON-copy helpers.

## 4. One collection facade and an existing-area coordinator

**Decision**: Add `CompleteUserTasks` to `c8volt/task.API`, with keys, optional variables, wanted workers, and public call options. Delegate mechanically to a shared function in `internal/services/usertask/completion.go`. Add scalar `CompleteUserTask` to the existing service interface and each version adapter contract. Preserve constructors and factories.

**Rationale**: Existing `GetUserTasks` already demonstrates facade-to-area-function delegation. No new composed service or constructor is necessary because completion needs only the existing user-task service. The coordinator owns validation, stable uniqueness, snapshotting, worker policy, result ordering, and aggregation.

**Alternatives considered**: Separate single and batch public operations duplicate a workflow already required for collections. A separate plan service would add an abstraction with no backend planning work: the preview is simply the locally validated immutable request, not a discovery or variable-diff plan. CLI/facade worker pools violate ownership rules.

**Evidence**: `c8volt/task/{api.go,client.go}`, `internal/services/usertask/api.go`, `toolx/pool`, `toolx.DetermineNoOfWorkers`.

## 5. Native completion is supported in all three versions

**Decision**: Use generated `CompleteUserTaskWithResponse` in v88, v89, and v810. Send `Action=nil`; send `Variables=nil` for omission and a pointer to the object for supplied variables. Treat documented HTTP 204 as acceptance without requiring a response JSON payload. Preserve non-success status/body evidence for machine errors.

**Rationale**: All three generated clients expose the same request shape. `Action` and `Variables` have no `omitempty` tags: nil produces wire `null`, which the vendored API explicitly allows. “Without additional variables” does not promise literal wire-field omission. No client regeneration is needed. Native completion returns no actual tenant metadata, so do not invent a resolved tenant or read a task merely to enrich output.

**Alternatives considered**: Hand-edited generated types or hand-built requests to omit nullable fields are unnecessary. Tasklist fallback, assignment, scope writes, and state/variable polling are excluded.

**Evidence**: `internal/clients/camunda/{v88,v89,v810}/camunda/client.gen.go`; `api/camunda/zeebe/gateway-protocol/src/main/proto/v2/user-tasks.yaml` (completion operation, nullable fields, and 204 response).

## 6. Reuse finite mutation retries while preserving uncertainty

**Decision**: Reuse `services.RetryCamundaMutation`: existing 429, RESOURCE_EXHAUSTED 503, and temporary/timeout network-error retries only. Keep its finite attempts and delay policy; do not add state reconciliation or replay rules. Track last HTTP evidence and earlier ambiguous attempts in the adapter closure. A final 204 establishes acceptance; an earlier ambiguous attempt followed by rejection cannot establish that nothing was accepted. A 504 is uncertain, not confirmed rejection.

**Rationale**: The issue explicitly permits applicable existing backoff and HTTP diagnostics. One logical completion per key may include existing transport attempts; it is not an exactly-once HTTP-delivery guarantee. The retry helper returns a zero value when canceled during backoff, so evidence must not depend solely on its returned response. Keep caller cancellation observable without throwing away already accepted results.

**Alternatives considered**: Job collection completion's one-shot/reconciliation policy belongs to its different lifecycle and cannot be imported selectively as a new user-task polling workflow. Generic mutation retry changes would affect unrelated commands.

**Logging decision**: Pass no retry-progress logger in compact mode and the configured logger for verbose functional retry messages. The helper already supports nil. Its known bounded reasons contain no raw errors; use a constant operation label without resource keys. Existing transport DEBUG instrumentation remains independently enabled and redacted, including when verbose is off.

**Evidence**: `internal/services/retry.go`, `internal/services/httpc`, `internal/services/job/completion/execute.go`.

## 7. Fail-fast stops scheduling, not submitted requests

**Decision**: Use `pool.ExecuteSlice` and existing worker selection. Prepopulate ordered result slots as not attempted. Follow the job completion pattern with a started flag, recorded callback error, and stop state: scheduling observes the pool context, but started requests use the original caller context. Aggregate real operation errors and caller cancellation, not synthetic pool cancellation caused by fail-fast.

**Rationale**: The pool cancels its derived context on failure and leaves zero values in unscheduled slots. Naive reuse would incorrectly cancel in-flight work or lose selected keys. An observed failure prevents further submissions, while requests already started may still be accepted.

**Alternatives considered**: New goroutine/channel scheduling duplicates existing helpers. Compacting results to only completed callbacks loses unattempted evidence.

**Evidence**: `toolx/pool`, `internal/services/job/completion/execute.go`, `internal/services/usertask/update.go`.

## 8. Separate structured evidence, safe diagnostics, and exit status

**Decision**: Define completion-specific public/domain outcome types with execution status and separate submission evidence. Keep all per-key results on errors. Build human summaries from typed counts/statuses and known HTTP status labels, grouping repeated causes into at most three named groups plus an “other” count. Never interpolate raw backend errors into human output. Use one shared JSON envelope and preserve the original classified error for exit after rendering.

**Rationale**: `httpc.HttpStatusErr` intentionally carries method/URL/body evidence. `ferrors.FromDomain` can lose the original typed chain while preserving classification, so capture structured evidence before conversion. Existing variable renderers expose raw `item.Error`; they are not safe completion templates. Existing `renderHumanLine` and job bulk views log human result prose; this feature explicitly reserves selected-format completion results for stdout.

**Alternatives considered**: Logging `err.Error()` merely relocates the leak. Reusing the variable result schema blurs scope writes and acceptance-only completion. Calling `handleCommandError` after rendering partial JSON produces duplicate output.

**Evidence**: `cmd/cmd_views_job_logging.go`, `cmd/update_job_batch.go:finishUpdateJobBatch`, `cmd/cmd_views_usertask_update.go`, `cmd/cmd_errors.go`, `c8volt/ferrors/errors.go`.

## 9. Explicit accepted rendering and unchanged guardrails

**Decision**: Render full completion acceptance with `renderAcceptedResult`, not a helper that infers acceptance from `flagNoWait`. Preview uses succeeded with an explicit preview payload. Preserve JSON requirements for dry-run/auto-confirm/automation, rejection of JSON+verbose, JSON precedence over keys-only, quiet machine-output behavior, and existing exit-code overrides. Keys-only preview prints selected keys; execution prints accepted keys only.

**Rationale**: Completion rejects `--no-wait` but is always acceptance-only. Machine consumers must distinguish acceptance from independently verified completion.

**Alternatives considered**: Reusing generic no-wait-dependent rendering would falsely advertise confirmed success. Printing every attempted key on failure would make failed keys indistinguishable from accepted keys in pipelines.

**Evidence**: `cmd/update_usertask_variables.go:validateUpdateUserTaskJSONConfirmation`, `cmd/command_contract.go`, `cmd/cmd_views_contract.go`.

## 10. Isolated validation and proportionate planning checks

**Decision**: Use test-owned HTTP servers, generated-client mocks, facade/service stubs, command subprocesses, and real-terminal stdin tests. Keep completion assertions separate from variable-mode human-stream regressions. Sanitize inherited test environments and prohibit live tags/targets. Review artifacts now; run focused tests and a final race suite during implementation because the change spans interfaces, adapters, command execution, and concurrency.

**Rationale**: Existing variable tests provide useful fixtures but some debug assertions only check substrings. Stronger logging precedents validate severity, framing, destinations, redaction, no duplicate emission, and unchanged stdout. No environment-backed implementation gate is permitted.

**Alternatives considered**: Pipe-only stdin cannot prove terminal prompt routing. Renderer-only tests cannot prove configured logger use or original exit classes. Live engine checks are a separate explicitly requested workflow.

**Evidence**: `cmd/update_job_batch_logging_test.go`, `cmd/root_api_diagnostics_test.go`, `cmd/get_processdefinition_watch_timeout_test.go`, `testx/cmd_terminal_runner.go`, `testx/integration_env_require_stub.go`, `Makefile`, `docsgen/main.go`.

## Human wording clarification within #311

The vendored native completion contract (`api/camunda/zeebe/gateway-protocol/src/main/proto/v2/user-tasks.yaml`) declares the operation non-eventually-consistent and HTTP 204 as successful task completion. The request waits for blocking task listeners; delayed processing can return 504. This supersedes the earlier acceptance-only human wording rationale: successful responses are backend confirmation, while independent task-state reads and downstream waiting remain out of scope.

Keep the existing `accepted` machine identifiers, fields, envelope outcomes, keys selection, and error/exit behavior for compatibility. Limit human success claims to accepted items and preserve uncertainty after timeout/cancellation. No polling or job-command changes are needed. `--no-wait` still skips variable verification only: completion has no additional client waiter, and the flag cannot bypass the native request or its listeners.

## Pragmatic simplification and T005

Completion reuses payload source selection but decodes its own object with `json.Decoder.UseNumber` and an EOF check; legacy variable updates retain their decoder. The fresh decoded map needs no parser copy, and the facade delegates snapshot ownership to the coordinator. Native adapters keep their copy for direct calls and generated-client isolation. This supersedes the earlier parser/copy recommendation in section 3.

Shared HTTP classification, retry and cancellation tests now exercise `common.CompleteUserTaskSubmission` once. Each version retains generated-wire, payload-ownership, invalid-payload, logging-option and response-delegation checks. No machine schema, outcome identifiers, polling or scheduling behavior changes.
