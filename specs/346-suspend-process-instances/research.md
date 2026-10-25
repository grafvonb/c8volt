# Research: Suspend Process Instances

**Feature**: [spec.md](spec.md) | **Issue**: #346 | **Date**: 2026-10-25

Research used repository source and the issue/documentation examined during specification. No runtime tests, integration fixtures, live target discovery, or service requests were performed for planning. Service and CLI research agents independently inspected the existing implementation; the decisions below resolve the design questions.

## 1. Reuse the existing single-instance suspension client

**Decision**: Use `SuspendProcessInstanceWithResponse(ctx, key, SuspendProcessInstanceJSONRequestBody{})` in the v810 process-instance adapter. Add the method to its narrow generated-client contract. Accept only a valid HTTP 204 response without a call error. No generated-client changes or regeneration are needed.

**Rationale**: `internal/clients/camunda/v810/camunda/client.gen.go` already implements `POST /process-instances/{processInstanceKey}/suspension`; its body has an optional operation reference that this feature does not need. `internal/services/processinstance/v810/contract.go` controls the adapter dependency. Validating response metadata follows the existing completion submission evidence pattern.

**Alternatives considered**: Direct HTTP from the CLI would break layering and duplicate authentication/diagnostics. Native batch suspension is outside scope. Treating every 2xx as acceptance would weaken the documented response contract.

## 2. Add an explicit suspension capability guard

**Decision**: Add `ValidateProcessInstanceSuspension() error` and a single-target suspension method to the existing process-instance service API and all version contracts. v88/v89 return `domain.ErrUnsupported` without I/O. The v810 guard succeeds locally. Guard single, collection, and dry-run entry paths before submission, pool dispatch, or empty-collection success.

**Rationale**: State-display support is not proof of mutation capability. `ValidateProcessInstanceStates` provides the existing no-I/O capability-check pattern, and `c8volt/ferrors.FromDomain` already converts unsupported errors. `internal/services/processinstance/factory.go` already selects the version.

**Alternatives considered**: A CLI-only version check would leave public-library calls unguarded. Calling the endpoint to discover support would send an unsupported request. A new service family or factory is unnecessary.

## 3. Preview explicit keys without preflight reads

**Decision**: Reuse the existing `DryRun` facade/call option. Suspension methods return preview results after local capability and input validation. The CLI requests a preview before confirmation, freezes the returned ordered unique keys, and submits those exact keys after approval. Neither preview nor ordinary execution performs a mandatory preflight read; default execution reads only after accepted submission.

**Rationale**: Input keys fully determine scope. State can change between any read and submission, so the mutation response must decide acceptance. Avoiding mandatory reads lets no-wait operate with suspension permission alone. Empty stdin already fails with `stdin contained no keys` in `cmd/cmd_stdin.go`. A direct public-library empty collection on v810 may return an empty result with no work, but the CLI must not reinterpret missing input that way.

**Alternatives considered**: Cancellation's `DryRunCancelOrDeletePlan` expands roots/descendants and is semantically wrong. A separate suspension-plan API/DTO adds no information beyond an immutable explicit key list. Preflight eligibility rejection would obscure backend rejection and add an unnecessary permission dependency.

## 4. Model independent submission and observation evidence

**Decision**: Introduce dedicated domain `ProcessInstanceSuspensionResult(s)` and public `process.SuspensionResult(s)`. Keep workflow status, submission evidence, confirmation evidence, last-observation presence/state, existing suspension date, submission response status, and per-key error evidence separate. Convert results even when an error is also returned.

**Rationale**: `domain.Reporter` and `process.Reporter` contain only key/ok/status code/status and cannot express accepted-but-unconfirmed or unknown acceptance. `StateUnknown` alone cannot distinguish an observed unknown state from no observation. Cancellation's `ProcessInstanceMutationFailure` describes root cancellation and is not reusable for suspension.

**Alternatives considered**: Extending shared reporter/error JSON would change unrelated contracts. A single success boolean or free-form status string would lose operational facts. Manufacturing a suspended timestamp or consulting state after rejected submission cannot prove request acceptance.

## 5. Preserve evidence across existing retries and local pacing

**Decision**: Reuse `services.RetryCamundaMutation` without broadening its retry policy. Keep attempt evidence outside its callback. Use attempt-local `httpc.BeginRequestRateAttempt` evidence to distinguish positively unsent local pacing failures from ambiguous transport failures. Only an actual valid 204 response may replace prior uncertainty with accepted submission. A later rejection must not erase an earlier ambiguous attempt.

**Rationale**: `internal/services/retry.go` retries temporary network errors, 429, and 503 RESOURCE_EXHAUSTED, with eight attempts by default. Cancellation during backoff returns a zero result. `internal/services/common/usertask_completion.go` and `scope_variable_submission.go` demonstrate retaining evidence independently. `http.Client.Timeout` can replace a pacing error, so the returned error alone is insufficient to prove an attempt was sent.

**Alternatives considered**: Disabling all existing submission retries would change established behavior. Retrying from the waiter could resubmit accepted mutations. Passing a converted HTTP status error into the retry callback would bypass its response-based 429/503 policy. State reconciliation after failed submission could create false acceptance.

## 6. Use a focused suspension waiter

**Decision**: Add `internal/services/processinstance/waiter/suspension.go`, depending on a narrow existing instance-read interface. Reuse `config.Backoff` timing, `NextDelay`, context deadlines, and lookup-log suppression. Confirm only a matching instance observed as SUSPENDED; stop unsuccessfully on COMPLETED, CANCELED, or TERMINATED. Typed not-found records ABSENT and continues bounded observation; other read failures stop without losing earlier evidence. Keep observation presence explicit.

**Rationale**: `waiter.WaitForProcessInstanceState` returns UNKNOWN on failure, uses substring-based absence detection, and cannot distinguish no observation from observed UNKNOWN. `domain.State.IsTerminal()` includes ABSENT, which is unsuitable here. A focused loop avoids changing existing cancellation/expect semantics. v810's existing `GetProcessInstance` performs direct keyed reads; no hierarchy or selection search is needed.

**Alternatives considered**: Reusing the generic waiter unchanged would misclassify observations. Globally changing its absence/terminal rules would enlarge scope and threaten cancellation compatibility. `toolx/poller` does not provide the required evidence and can write progress directly to process stderr; it is not the suspension execution path.

## 7. Keep collection scheduling inside the service layer

**Decision**: Add a version-neutral collection helper in `internal/services/processinstance/suspend.go`. Deduplicate and validate once, preserve first-input order, initialize a not-attempted report for every selected key, and use `toolx.DetermineNoOfWorkers` plus `pool.ExecuteSlice`. Retain all completed reports and explicit context failure, including cancellation before any callback starts. Honor fail-fast/no-worker-limit without erasing in-flight work.

**Rationale**: Existing `internal/services/processinstance/bulk.go` establishes service ownership. Pool results preserve indices but unstarted slots may be zero values, and pre-work cancellation must not accidentally return overall success. Parent and child keys remain independent queue entries.

**Alternatives considered**: A facade loop would violate the architecture. Cancellation's root/tree collection helper carries the wrong scope and output semantics. Building a single-item CLI first and replacing it later with collection support duplicates work.

## 8. Use existing output infrastructure, with an explicit stream contract

**Decision**: Follow `cmd/update_element_variables.go` and `cmd/cmd_views_element_update.go` for one-result partial failure handling. Use the shared envelope without schema changes. Effective mode is JSON, then keys-only, then human; quiet is orthogonal. Keys-only execution emits accepted keys, including accepted-but-unconfirmed targets; preview emits the selected keys. Human final result summaries are selected-format output on configured stdout; optional progress and diagnostic records use the configured logger, and prompts use plain configured stderr.

**Rationale**: `renderHumanLine` is an INFO logger helper, not a stdout result writer. The new view must deliberately use the stdout result writer for selected-format output and avoid its missing-context fallback for diagnostics. A partial JSON failure must render its payload before `exitAfterRenderedResult`, without a second error renderer. Known raw errors remain machine evidence but are not human text.

**Alternatives considered**: Copying the older cancellation runner could drop partial results. A direct stderr print would bypass logger configuration. Logging raw joined errors would leak transport details. Treating keys-only output as confirmation would differ from recent mutation conventions and discard useful accepted evidence.

## 9. Keep validation isolated and implementation units cohesive

**Decision**: Use generated-client stubs, test-owned HTTP servers, command subprocesses, and real terminal stdin tests. Reuse `testx` helpers and recent element-update diagnostics/terminal coverage. Update the Bash suspension fixture statically and regenerate CLI docs from source during implementation. Do not run the fixture or any live tests.

**Rationale**: The implementation testing boundary overrides integration requirements. `Makefile` separates `make test` from integration-tag targets; `testx/integration_env_require_stub.go` skips environment-backed helpers without the integration tag. Planning-only changes need documentation checks, not runtime tests.

**Alternatives considered**: A disposable live cluster still violates the boundary. Renderer-only tests cannot establish logger routing or prompt separation. Separate interface-only and adapter-only work units would leave the tree uncompilable.

## Resolved constraints for task generation

All research questions are resolved. No new dependency, client regeneration, public shared-envelope change, or general workflow framework is required. Keep interface additions, all adapters, and affected test doubles in a single compiling work unit. Relevant concrete stubs occur in `c8volt/process/client_test.go`, `c8volt/resource/client_test.go`, `internal/services/processinstance/dryrun_test.go`, `internal/services/processdefinition/delete_test.go`, `internal/services/ops/orphan_purge_test.go`, and the v810 adapter tests. Confirm the complete set through compiler checks during implementation.
