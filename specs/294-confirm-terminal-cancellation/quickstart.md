# Validation Guide: Terminal Cancellation Confirmation

## Prerequisites

Use the repository root, Go 1.26 with toolchain go1.26.2 available, and the existing module dependencies. Regression tests use controlled clients and require no live Camunda credentials. This guide describes validation after implementation; the planning phase has not added or run the proposed regressions.

## Targeted checks

```sh
go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 ./internal/services/processinstance/v810 -run 'TestService_(CancelProcessInstance|DeleteProcessInstance)' -count=1
go test ./internal/services/processinstance/waiter -run 'TestWaitForProcessInstance' -count=1
go test ./internal/services/processinstance -run 'Test.*Cancel' -count=1
go test ./internal/services/processdefinition -run 'Test(CleanupProcessDefinitionDeletePlanForceScope|DeleteProcessDefinitions)' -count=1
go test ./cmd -run 'Test(Cancel|Expect)' -count=1
```

Place new cases under these existing test-name prefixes, or update these commands when generating tasks. A passing command with no matching tests is not evidence. Use `go test <package> -list '<pattern>'` to confirm coverage if naming changes.

## Scenarios and expected outcomes

Use the [acceptance matrix](contracts/cancellation-confirmation.md) and [state table](data-model.md). In every version, verify a completed descendant, active-to-completed transition, disappearance after family keys are known, each terminal-root no-op, and active/unknown timeout controls. Assert cancellation request counts and `Ok` values for no-ops; preserve existing status text and HTTP code.

Exercise forced-delete recovery so a second narrow cancellation wait cannot reintroduce the timeout. Preserve existing opt-out guards, including the forced-delete recovery wait. A terminal cancellation outcome must not bypass final history absence verification.

For the focused process-definition regression, configure a real 8.8 process-instance service with controlled backend responses: active root, completed descendant, successful root cancellation, drained active statistics, successful history deletion, and the existing definition deletion/verification result. Enter through existing process-definition deletion orchestration so the test proves the entire path. Confirm that replacing the four-state cancellation set with the old two-state set makes the regression fail. Never substitute a canned successful cancellation callback for this proof.

Check explicit cancelled-state expectations against all four terminal states; completed/absent must fail to match within the configured bounded wait. Confirm normal read not-found errors and unrelated errors remain errors. Run existing command regressions for JSON/human output, prompts, dry-run, flags, and activity.

## Documentation and final validation

Clarify README and cancellation command long help, then run:

```sh
make docs-content
git diff --check
make test
```

Run `gofmt -w` on each touched Go file before these checks. Inspect generated documentation changes for only the intended clarification and any known generator effects. Do not hand-edit generated CLI references. The full test target runs `go test ./... -race -count=1` and must pass before implementation commit/merge. Record any unavailable validation explicitly.

## Baseline — Iteration 1 (2026-09-10)

- Environment verified: `go.mod` requires Go 1.26 and selects toolchain go1.26.2; `go version` reported `go1.26.2 darwin/arm64`.
- Active feature verified: `.specify/feature.json` selects `specs/294-confirm-terminal-cancellation`.
- Repository guidance verified from `AGENTS.md` and `specs/ralph-implementation-rules.md`; branch remained `develop` and dependencies were unchanged.
- Versioned cancellation/deletion command passed for v87, v88, v89, and v810. The current filter selected no tests in v89 and v810, which is baseline evidence only and must be addressed by the later test-mapping task.
- Waiter cancellation-state command passed.
- Process-instance bulk cancellation command passed.
- Process-definition cleanup/deletion command passed.
- Command cancellation/expectation command passed.
- Full repository gate `make test` (`go test ./... -race -count=1`) passed before the coordinated setup commit.

## Acceptance and test-seam map — Iteration 2 (2026-09-10)

The versioned cancellation tests do not share one top-level name. Use these actual runnable prefixes when adding contract cases:

| Version | Cancellation cases | Forced-delete cases | Local fixture seam |
| --- | --- | --- | --- |
| 8.7 | `TestService_CancelProcessInstance` | `TestService_DeleteProcessInstance` | `newTestService` with strict Camunda/Operate clients; override only expected generated-client calls |
| 8.8 | `TestService_CancelProcessInstance` | `TestService_DeleteProcessInstance` | `newTestService`, `newStrictCamundaClient`, and `newStrictOperateClient`; unexpected calls fail immediately |
| 8.9 | `TestService_CancelAndDeleteProcessInstance` | `TestService_CancelAndDeleteProcessInstance` | `newTestService` with a strict version-local Camunda client; cancellation and deletion are subtests of the combined prefix |
| 8.10 | `TestService_CancelAndDeleteProcessInstance` | `TestService_CancelAndDeleteProcessInstance` | `newTestService` with a strict version-local Camunda client; cancellation and deletion are subtests of the combined prefix |

The contract rows map to those suites as follows:

| Contract row | Four-version service coverage | Additional focused seam |
| --- | --- | --- |
| A — completed descendant | Cancellation prefix in every version; model the existing family search/get sequence and assert every discovered key is confirmed | Real 8.8 cancellation is reused by the process-definition cleanup proof |
| B — active becomes completed | Cancellation prefix in every version; return ACTIVE on the first state read and COMPLETED on the second | Shared waiter already supports repeated state reads without changing equivalence rules |
| C — known member disappears | Cancellation prefix in every version; discover the member first, then return the adapter's established not-found response during confirmation | `TestWaitForProcessInstanceState` owns generic absent-during-wait behavior |
| D — terminal/absent root no-op | Cancellation prefix in every version; use strict cancellation counters to prove zero submissions and assert `Ok`, code, and status | `TestCancelProcessInstancesEmitsCompletionFacts` in `bulk_test.go` is the report-propagation seam |
| E — active/unknown remains | Cancellation prefix in every version; exhaust bounded retries and separately interrupt context | `TestWaitForProcessInstanceState` owns timeout and context controls |
| F — explicit canceled expectation | Add adapter-backed cases under each version's actual cancellation/deletion prefix, then retain shared waiter and command strictness tests | `TestWaitForProcessInstanceExpectation_StateAndIncidentCompatibility`, `TestWaitForProcessInstanceState`, and `TestExpectProcessInstanceCommand_StateMismatchRemainsStrict` |
| G — no-wait/no-state-check | Cancellation and forced-delete prefixes in every version; assert reads, submission, discovery, and wait counts separately and combined | Existing `CancelNoWait` cases are controls, including subtests under the combined 8.9/8.10 prefix |
| H — unrelated read/submission errors | Cancellation prefix in every version, with strict unexpected-call failures and request counters preserving retry/error boundaries | Existing strict getters and command error-envelope tests remain controls |
| I — forced-delete recovery | Forced-delete prefix in every version; exercise the intermediate cancellation wait, delete retry, and final absence verification | Add a `TestDeleteProcessDefinitions...` case in `internal/services/processdefinition/delete_test.go` that enters `cleanupProcessDefinitionDeletePlanForceScope` with a real 8.8 cancellation service; retain `TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterCancellationFailure`, `...StopsAfterDrainFailure`, and `...StopsAfterHistoryFailure` as downstream boundaries |

All four versioned suites already expose `waitTestConfig`, configured with fixed backoff, `InitialDelay: 1ms`, `MaxRetries: 2`, and `Timeout: 25ms`. Use it for deterministic state transitions and retry exhaustion. Use an already-canceled context or a test-owned cancellation callback for interruption cases instead of lengthening the timeout. The cleanup test can return drained statistics immediately; its real cancellation service should use the same bounded configuration.

The shared waiter intentionally remains unchanged: it fans family waits across unique keys and requires every scheduled key to return success; only its existing not-found detection maps an observation to ABSENT; and state equivalence remains limited to CANCELED/TERMINATED. In `delete_test.go`, the existing callback-only `cleanupProcessInstanceAPI` is suitable for downstream failure controls but cannot prove row A. The focused cleanup regression therefore needs a small file-local adapter/client double that delegates cancellation to the real 8.8 service while reusing `processDefinitionStageSequence`, `controlledProcessDefinitionAPI`, and the current force-cleanup entry point. Do not introduce a shared test framework.

Prefix inventory was verified with `go test -list`: the original `TestService_(CancelProcessInstance|DeleteProcessInstance)` filter selects 8.7/8.8 only, while 8.9/8.10 require `TestService_CancelAndDeleteProcessInstance`. Until the later suites are renamed, use this coverage-safe command:

```sh
go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 ./internal/services/processinstance/v810 -run 'TestService_(CancelProcessInstance|DeleteProcessInstance|CancelAndDeleteProcessInstance)$' -count=1
```

## US1 v8.7 proof — Iteration 3 (2026-09-10)

- Before T007, `go test ./internal/services/processinstance/v87 -run TestService_CancelProcessInstance -count=1` failed the new terminal-root checks because `Ok` was false and the absent precheck propagated `ErrNotFound`. The first family cases also exposed that v8.7 cancellation routed its existing family walk through the intentionally unsupported public direct getter.
- T007 now uses the existing tenant-safe traversal adapter for that same family walk, accepts COMPLETED/CANCELED/TERMINATED/ABSENT during cancellation confirmation, locally maps only wrapped `ErrNotFound` to ABSENT in the guarded precheck, and returns the existing no-op response with `Ok=true`.
- The focused cancellation command passes all contract A–E cases, including completed and disappearing descendants, active-to-completed polling, a mixed terminal family, active/unknown timeout controls, context interruption, and all four terminal-root no-ops with zero cancellation submissions.
- `go test ./internal/services/processinstance/v87 -count=1`, `git diff --check`, and `make test` (`go test ./... -race -count=1`) passed after the correction.

## US1 v8.8 proof — Iteration 4 (2026-09-10)

- Before T008, `go test ./internal/services/processinstance/v88 -run TestService_CancelProcessInstance -count=1` failed the completed, naturally completed, disappeared, and mixed-terminal family cases against the old CANCELED/TERMINATED list; terminal-root responses had `Ok=false`, and the absent-root precheck propagated wrapped `ErrNotFound`.
- T008 accepts COMPLETED/CANCELED/TERMINATED/ABSENT during cancellation-family confirmation, locally maps only wrapped `ErrNotFound` to ABSENT inside the guarded precheck, and returns the existing no-op response with `Ok=true`; the separate forced-delete recovery wait remains unchanged for US2.
- The focused cancellation command now passes all v8.8 contract A–E cases, including active/unknown timeout controls, context interruption, and four zero-submission terminal-root no-ops.
- `go test ./internal/services/processinstance/v88 -count=1`, `git diff --check`, and `make test` (`go test ./... -race -count=1`) passed after the correction.

## US1 v8.9 proof — Iteration 5 (2026-09-10)

- Before T009, `go test ./internal/services/processinstance/v89 -run TestService_CancelAndDeleteProcessInstance -count=1` failed completed, naturally completed, disappeared, and mixed-terminal family cases against the old CANCELED/TERMINATED list; terminal-root responses had `Ok=false`, and the absent-root precheck propagated wrapped `ErrNotFound`.
- T009 accepts COMPLETED/CANCELED/TERMINATED/ABSENT during cancellation-family confirmation, locally maps only wrapped `ErrNotFound` to ABSENT inside the guarded precheck, and returns the existing no-op response with `Ok=true`; the separate forced-delete recovery wait remains unchanged for US2.
- The combined cancellation/deletion prefix now passes all v8.9 contract A–E cases, including active/unknown timeout controls, context interruption, and four zero-submission terminal-root no-ops.
- `go test ./internal/services/processinstance/v89 -count=1`, `git diff --check`, and `make test` (`go test ./... -race -count=1`) passed after the correction.
