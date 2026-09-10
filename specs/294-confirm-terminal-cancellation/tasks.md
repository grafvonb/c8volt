---
description: "Implementation tasks for issue #294"
---

# Tasks: Accept Terminal States During Cancellation Confirmation

**Input**: Design documents in `specs/294-confirm-terminal-cancellation/`.
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contract](contracts/cancellation-confirmation.md), and [quickstart.md](quickstart.md).
**Tests**: Required explicitly by FR-009/FR-010 and the constitution. Write regression cases before the matching production edits; demonstrate failures caused by the original bug. Compatibility controls may already pass and must continue passing.
**Organization**: Tasks follow US1 (P1), US2 (P2), US3 (P3). All paths below are relative to the repository root. No new dependencies, public models, or generated-client changes are required.

## Format: `[ID] [P?] [Story] Description`

`[P]` permits parallel execution only within the dependency-ready wave described below, on distinct files. Story labels map to the specification. All tasks start unchecked because this command generates work rather than implementing it.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify the existing environment without scaffolding a new project.

- [ ] T001 Verify the Go/toolchain requirements in `go.mod`, active feature in `.specify/feature.json`, and repository guidance in `AGENTS.md`; run the existing targeted checks from `specs/294-confirm-terminal-cancellation/quickstart.md` and record baseline outcomes there. Keep the current branch and dependencies unchanged.


## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the acceptance matrix and existing test seams; no production infrastructure is needed.

- [ ] T002 Map contract rows A–I to the four versioned service suites and the focused cleanup test in `specs/294-confirm-terminal-cancellation/quickstart.md`; inspect existing strict clients in `internal/services/processinstance/v88/service_test.go`, shared waiter semantics in `internal/services/processinstance/waiter/waiter.go`, and cleanup seams in `internal/services/processdefinition/delete_test.go`. Record actual test prefixes and bounded polling configuration for deterministic tests; reuse local fixtures without a general test framework.


**Checkpoint**: Baseline and test seams are understood; all story work depends on T002.

## Phase 3: User Story 1 — Confirm a Family That No Longer Needs Cancellation (Priority: P1, MVP)

**Goal**: Confirm terminal families and return true success for terminal-root no-ops.
**Independent Test**: For all four adapters, an active root with a completed descendant, natural completion, or disappearance after family discovery succeeds once every scoped member qualifies. Active/unknown members prevent success; terminal/absent root no-ops submit zero cancellation requests and bulk reports success.

### Tests for User Story 1

- [ ] T003 [P] [US1] Extend `internal/services/processinstance/v87/service_test.go` under `TestService_CancelProcessInstance` with contract A–E: completed descendant, active-to-completed during polling, disappearance after keys are discovered, mixed terminal family, completed root with still-active descendant during confirmation, active/unknown timeout and interruption controls, and all four terminal-root no-ops (including real not-found mapping). Assert zero cancellation requests and the data-model constraint "Terminal-root no-op has `Ok=true`, code 200, existing status wording; no added fields". Use short configured waits and strict request counters; run this package with `-run TestService_CancelProcessInstance -count=1` and capture expected regression failures before its implementation task.

- [ ] T004 [P] [US1] Extend `internal/services/processinstance/v88/service_test.go` under `TestService_CancelProcessInstance` with contract A–E: completed descendant, active-to-completed during polling, disappearance after keys are discovered, mixed terminal family, completed root with still-active descendant during confirmation, active/unknown timeout and interruption controls, and all four terminal-root no-ops (including real not-found mapping). Assert zero cancellation requests and the data-model constraint "Terminal-root no-op has `Ok=true`, code 200, existing status wording; no added fields". Use short configured waits and strict request counters; run this package with `-run TestService_CancelProcessInstance -count=1` and capture expected regression failures before its implementation task.

- [ ] T005 [P] [US1] Extend `internal/services/processinstance/v89/service_test.go` under `TestService_CancelProcessInstance` with contract A–E: completed descendant, active-to-completed during polling, disappearance after keys are discovered, mixed terminal family, completed root with still-active descendant during confirmation, active/unknown timeout and interruption controls, and all four terminal-root no-ops (including real not-found mapping). Assert zero cancellation requests and the data-model constraint "Terminal-root no-op has `Ok=true`, code 200, existing status wording; no added fields". Use short configured waits and strict request counters; run this package with `-run TestService_CancelProcessInstance -count=1` and capture expected regression failures before its implementation task.

- [ ] T006 [P] [US1] Extend `internal/services/processinstance/v810/service_test.go` under `TestService_CancelProcessInstance` with contract A–E: completed descendant, active-to-completed during polling, disappearance after keys are discovered, mixed terminal family, completed root with still-active descendant during confirmation, active/unknown timeout and interruption controls, and all four terminal-root no-ops (including real not-found mapping). Assert zero cancellation requests and the data-model constraint "Terminal-root no-op has `Ok=true`, code 200, existing status wording; no added fields". Use short configured waits and strict request counters; run this package with `-run TestService_CancelProcessInstance -count=1` and capture expected regression failures before its implementation task.


### Implementation for User Story 1

- [ ] T007 [P] [US1] Update `internal/services/processinstance/v87/service.go` cancellation-family desired states to COMPLETED, CANCELED, TERMINATED, ABSENT, honoring "Cleanup accepts exactly the four terminal states; active/unknown do not qualify" and "Every included key must satisfy its wait; discovery timing and membership rules stay unchanged". Under the existing state-check guard only, map `errors.Is(err, d.ErrNotFound)` to local absent state; return other errors unchanged and set `Ok: true` on the existing terminal no-op response. Keep status code/text, getters, retry calls, family discovery, and NoWait/NoStateCheck guards intact. Run gofmt and the matching cancellation tests after the paired test task.

- [ ] T008 [P] [US1] Update `internal/services/processinstance/v88/service.go` cancellation-family desired states to COMPLETED, CANCELED, TERMINATED, ABSENT, honoring "Cleanup accepts exactly the four terminal states; active/unknown do not qualify" and "Every included key must satisfy its wait; discovery timing and membership rules stay unchanged". Under the existing state-check guard only, map `errors.Is(err, d.ErrNotFound)` to local absent state; return other errors unchanged and set `Ok: true` on the existing terminal no-op response. Keep status code/text, getters, retry calls, family discovery, and NoWait/NoStateCheck guards intact. Run gofmt and the matching cancellation tests after the paired test task.

- [ ] T009 [P] [US1] Update `internal/services/processinstance/v89/service.go` cancellation-family desired states to COMPLETED, CANCELED, TERMINATED, ABSENT, honoring "Cleanup accepts exactly the four terminal states; active/unknown do not qualify" and "Every included key must satisfy its wait; discovery timing and membership rules stay unchanged". Under the existing state-check guard only, map `errors.Is(err, d.ErrNotFound)` to local absent state; return other errors unchanged and set `Ok: true` on the existing terminal no-op response. Keep status code/text, getters, retry calls, family discovery, and NoWait/NoStateCheck guards intact. Run gofmt and the matching cancellation tests after the paired test task.

- [ ] T010 [P] [US1] Update `internal/services/processinstance/v810/service.go` cancellation-family desired states to COMPLETED, CANCELED, TERMINATED, ABSENT, honoring "Cleanup accepts exactly the four terminal states; active/unknown do not qualify" and "Every included key must satisfy its wait; discovery timing and membership rules stay unchanged". Under the existing state-check guard only, map `errors.Is(err, d.ErrNotFound)` to local absent state; return other errors unchanged and set `Ok: true` on the existing terminal no-op response. Keep status code/text, getters, retry calls, family discovery, and NoWait/NoStateCheck guards intact. Run gofmt and the matching cancellation tests after the paired test task.

- [ ] T011 [US1] Extend `internal/services/processinstance/bulk_test.go` to assert "Bulk cancellation propagates the corrected no-op success value": terminal and absent no-op results remain successful in reports and totals with existing status fields. Reuse existing API doubles and leave `internal/services/processinstance/bulk.go` unchanged; run `go test ./internal/services/processinstance -run "Test.*Cancel" -count=1`.

- [ ] T012 [US1] Run cancellation tests across `internal/services/processinstance/v87/service_test.go`, `internal/services/processinstance/v88/service_test.go`, `internal/services/processinstance/v89/service_test.go`, and `internal/services/processinstance/v810/service_test.go` with `go test ./internal/services/processinstance/... -run "Test.*Cancel" -count=1`; record the US1 matrix results in `specs/294-confirm-terminal-cancellation/quickstart.md` and confirm all original failing cases now pass.


**Checkpoint**: US1 is a demonstrable MVP; complete all stories and final checks before declaring issue #294 finished.

## Phase 4: User Story 2 — Continue Existing Forced Cleanup (Priority: P2)

**Goal**: Prevent a second narrow cancellation wait from blocking forced deletion and prove process-definition cleanup with a completed descendant.
**Independent Test**: Forced-delete recovery accepts completed/absent observations in every adapter but retains final deletion verification. One controlled 8.8 cleanup run invokes real cancellation, drains active instances, deletes history, and completes definition deletion; downstream failures remain failures.

### Tests for User Story 2

- [ ] T013 [P] [US2] Extend `internal/services/processinstance/v87/service_test.go` under `TestService_DeleteProcessInstance` for wrong-state force recovery with completed and absent cancellation-confirmation observations (contract I). Ensure the follow-up wait is actually exercised, assert delete retry and final absence verification, preserve its existing unconditional recovery wait with NoWait, and retain later deletion-failure outcomes. Run matching tests and demonstrate the old two-state follow-up wait fails before the paired production edit.

- [ ] T014 [P] [US2] Extend `internal/services/processinstance/v88/service_test.go` under `TestService_DeleteProcessInstance` for wrong-state force recovery with completed and absent cancellation-confirmation observations (contract I). Ensure the follow-up wait is actually exercised, assert delete retry and final absence verification, preserve its existing unconditional recovery wait with NoWait, and retain later deletion-failure outcomes. Run matching tests and demonstrate the old two-state follow-up wait fails before the paired production edit.

- [ ] T015 [P] [US2] Extend `internal/services/processinstance/v89/service_test.go` under `TestService_DeleteProcessInstance` for wrong-state force recovery with completed and absent cancellation-confirmation observations (contract I). Ensure the follow-up wait is actually exercised, assert delete retry and final absence verification, preserve its existing unconditional recovery wait with NoWait, and retain later deletion-failure outcomes. Run matching tests and demonstrate the old two-state follow-up wait fails before the paired production edit.

- [ ] T016 [P] [US2] Extend `internal/services/processinstance/v810/service_test.go` under `TestService_DeleteProcessInstance` for wrong-state force recovery with completed and absent cancellation-confirmation observations (contract I). Ensure the follow-up wait is actually exercised, assert delete retry and final absence verification, preserve its existing unconditional recovery wait with NoWait, and retain later deletion-failure outcomes. Run matching tests and demonstrate the old two-state follow-up wait fails before the paired production edit.


### Implementation for User Story 2

- [ ] T017 [P] [US2] Extend only the forced-delete recovery cancellation desired-state list in `internal/services/processinstance/v87/service.go` to COMPLETED, CANCELED, TERMINATED, ABSENT. Preserve the unconditional intermediate wait, final absent-only deletion wait and its existing NoWait guard, mutation retries, and failure propagation; run gofmt and this package with `-run TestService_DeleteProcessInstance -count=1` after its paired regression task.

- [ ] T018 [P] [US2] Extend only the forced-delete recovery cancellation desired-state list in `internal/services/processinstance/v88/service.go` to COMPLETED, CANCELED, TERMINATED, ABSENT. Preserve the unconditional intermediate wait, final absent-only deletion wait and its existing NoWait guard, mutation retries, and failure propagation; run gofmt and this package with `-run TestService_DeleteProcessInstance -count=1` after its paired regression task.

- [ ] T019 [P] [US2] Extend only the forced-delete recovery cancellation desired-state list in `internal/services/processinstance/v89/service.go` to COMPLETED, CANCELED, TERMINATED, ABSENT. Preserve the unconditional intermediate wait, final absent-only deletion wait and its existing NoWait guard, mutation retries, and failure propagation; run gofmt and this package with `-run TestService_DeleteProcessInstance -count=1` after its paired regression task.

- [ ] T020 [P] [US2] Extend only the forced-delete recovery cancellation desired-state list in `internal/services/processinstance/v810/service.go` to COMPLETED, CANCELED, TERMINATED, ABSENT. Preserve the unconditional intermediate wait, final absent-only deletion wait and its existing NoWait guard, mutation retries, and failure propagation; run gofmt and this package with `-run TestService_DeleteProcessInstance -count=1` after its paired regression task.

- [ ] T021 [US2] Add a focused `TestDeleteProcessDefinitions` regression in `internal/services/processdefinition/delete_test.go` using the real 8.8 process-instance cancellation service with controlled backend responses: active root, completed descendant, cancellation confirmation, drained statistics, history deletion, and definition deletion/verification. Enter through existing deletion orchestration; do not stub cancellation as a canned success. Assert sequence and final success, reuse existing downstream failure controls, and temporarily restore only the old cancellation state list to prove this test fails, then restore the fix. Keep `internal/services/processdefinition/delete.go` production orchestration unchanged.

- [ ] T022 [US2] Run `go test ./internal/services/processinstance/... -run "TestService_(CancelProcessInstance|DeleteProcessInstance)" -count=1` and `go test ./internal/services/processdefinition -run "Test(CleanupProcessDefinitionDeletePlanForceScope|DeleteProcessDefinitions)" -count=1`; record the cleanup proof and later-failure outcomes in `specs/294-confirm-terminal-cancellation/quickstart.md`.


**Checkpoint**: Cleanup reaches its verified final state, and cancellation success does not hide downstream failure.

## Phase 5: User Story 3 — Preserve Explicit Expectations and Operator Contracts (Priority: P3)

**Goal**: Preserve explicit cancelled-state checks, opt-outs, error handling, and runtime output contracts.
**Independent Test**: Explicit canceled matches CANCELED/TERMINATED and rejects COMPLETED/ABSENT in all four versions. Existing no-wait/no-state-check request behavior, read/submission errors, retries, output formats, and exit behavior remain intact.

### Compatibility tests for User Story 3

- [ ] T023 [P] [US3] Extend or reuse cases in `internal/services/processinstance/v87/service_test.go` for contract F–H: invoke the real service-backed explicit expectation path for all four terminal states; preserve canceled/terminated equivalence and completed/absent rejection; verify NoWait and NoStateCheck individually and together by read/submission counts, strict getter not-found behavior, unrelated read errors, submission retry/error outcomes, and family-discovery errors before polling. Use `TestService_CancelProcessInstance`/`TestService_DeleteProcessInstance` subtests where relevant and document any additional expectation test prefix in the validation task. Do not change production matching or flag behavior.

- [ ] T024 [P] [US3] Extend or reuse cases in `internal/services/processinstance/v88/service_test.go` for contract F–H: invoke the real service-backed explicit expectation path for all four terminal states; preserve canceled/terminated equivalence and completed/absent rejection; verify NoWait and NoStateCheck individually and together by read/submission counts, strict getter not-found behavior, unrelated read errors, submission retry/error outcomes, and family-discovery errors before polling. Use `TestService_CancelProcessInstance`/`TestService_DeleteProcessInstance` subtests where relevant and document any additional expectation test prefix in the validation task. Do not change production matching or flag behavior.

- [ ] T025 [P] [US3] Extend or reuse cases in `internal/services/processinstance/v89/service_test.go` for contract F–H: invoke the real service-backed explicit expectation path for all four terminal states; preserve canceled/terminated equivalence and completed/absent rejection; verify NoWait and NoStateCheck individually and together by read/submission counts, strict getter not-found behavior, unrelated read errors, submission retry/error outcomes, and family-discovery errors before polling. Use `TestService_CancelProcessInstance`/`TestService_DeleteProcessInstance` subtests where relevant and document any additional expectation test prefix in the validation task. Do not change production matching or flag behavior.

- [ ] T026 [P] [US3] Extend or reuse cases in `internal/services/processinstance/v810/service_test.go` for contract F–H: invoke the real service-backed explicit expectation path for all four terminal states; preserve canceled/terminated equivalence and completed/absent rejection; verify NoWait and NoStateCheck individually and together by read/submission counts, strict getter not-found behavior, unrelated read errors, submission retry/error outcomes, and family-discovery errors before polling. Use `TestService_CancelProcessInstance`/`TestService_DeleteProcessInstance` subtests where relevant and document any additional expectation test prefix in the validation task. Do not change production matching or flag behavior.

- [ ] T027 [P] [US3] Add or extend state/incident compatibility cases in `internal/services/processinstance/waiter/waiter_test.go` for "Explicit expectation matching remains independent; no state/incident relaxation"; explicit canceled rejects completed/absent and accepts canceled/terminated, with existing incident requirements preserved. Leave `internal/services/processinstance/waiter/waiter.go` and `internal/domain/state.go` unchanged; run `go test ./internal/services/processinstance/waiter -run TestWaitForProcessInstance -count=1`.

- [ ] T028 [P] [US3] Extend the existing strict state-mismatch command regression in `cmd/expect_test.go` for completed/absent versus canceled/terminated and verify existing output/error envelopes and exit behavior through command execution; run `go test ./cmd -run TestExpect -count=1`. Keep `cmd/expect_processinstance.go` matching and command semantics unchanged.

- [ ] T029 [P] [US3] Review and extend only missing relevant assertions in `cmd/cancel_processinstance_test.go` for unchanged human/JSON/supported machine output, prompts, dry-run, inherited opt-out flags, and activity. Reuse existing fixtures and renderer assertions; run `go test ./cmd -run TestCancel -count=1`. Corrected terminal no-op success values are expected; no runtime wording or public-field additions are permitted.


### Integration validation for User Story 3

No new production interface is needed; compatibility failures must be resolved within the bounded adapter changes rather than by weakening expectations.

- [ ] T030 [US3] Run `go test ./internal/services/processinstance/... -count=1` and `go test ./cmd -run "Test(Cancel|Expect)" -count=1`; verify every version has F–H coverage and update actual runnable prefixes and outcomes in `specs/294-confirm-terminal-cancellation/quickstart.md` so no added test is omitted by a filter.


**Checkpoint**: All three stories meet their independent acceptance criteria.

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Explain the correction and complete repository validation.

- [ ] T031 Clarify terminal cancellation acceptance versus explicit canceled expectations in `README.md` and the existing long-help metadata in `cmd/cancel_processinstance.go`; preserve flags, examples, and runtime rendering. Run gofmt on touched Go files and `make docs-content` to regenerate `docs/cli/c8volt_cancel_process-instance.md` and other generator-owned outputs; inspect the diff and never hand-edit generated references.

- [ ] T032 Execute the final validation guide in `specs/294-confirm-terminal-cancellation/quickstart.md`, run `git diff --check` and the full `make test` target from `Makefile` (`go test ./... -race -count=1`), and record exact outcomes or unavailable checks in the guide. Confirm eight cancellation lists and four prechecks/no-op responses are covered and no generated clients, public models, retries, or generic matcher changed. Do not consider implementation complete if required validation is failing or skipped.

- [ ] T033 Reconcile acceptance coverage against FR-001–FR-010 and SC-001–SC-005 in `specs/294-confirm-terminal-cancellation/spec.md`, update completed checkboxes in `specs/294-confirm-terminal-cancellation/tasks.md` only with evidence, and verify `specs/294-confirm-terminal-cancellation/quickstart.md` reflects the final test names. Keep work on the current branch; commit only when authorized and after required checks, using a Conventional Commit subject referencing #294.


## Dependencies & Execution Order

```text
T001 → T002 → US1 (T003–T012) → US2 (T013–T022) → US3 (T023–T030) → T031 → T032 → T033
```

- US1: T003–T006 are parallel test files. T007–T010 are parallel production files after their respective T003–T006 cases have demonstrated the bug; do not edit a version's production file while its baseline regression is being demonstrated. T011 follows the four production fixes; T012 follows all US1 work.
- US2: Begin after T012 to avoid overlapping adapter edits. T013–T016 are parallel test files; T017–T020 pair respectively with them and form the next production wave. T021 follows all four fixes and must restore the correction after its deliberate regression check; no concurrent validation or editing of the v88 service during that check. T022 follows T021.
- US3: T023–T029 operate on distinct files and may run in parallel after T022. T030 follows all seven compatibility tasks; production behavior is not intentionally changed in this story.
- T031–T033 are sequential after all stories. If any earlier commit is authorized, run the constitution-required full test target before that commit as well.
- Parallel markers describe scheduling opportunities, not authorization to launch agents. Default implementation may run sequentially. These dependencies deliberately serialize stories sharing versioned service files while keeping each story independently verifiable.

## Parallel Examples

### User Story 1

After T002, write the v87 tests (T003) and v88 tests (T004) in separate files concurrently. After their failing checks, the respective v87 and v88 production changes (T007/T008) can proceed concurrently. The same pattern applies to v89/v810.

### User Story 2

After T012, write recovery cases T013/T014 in the separate v87/v88 test files concurrently, then implement T017/T018 after each test prerequisite. Finish the single real-cleanup regression T021 separately.

### User Story 3

After T022, the v87 compatibility suite (T023), shared waiter tests (T027), and command expectation tests (T028) can proceed concurrently because their files differ. Run the integrated validation T030 after the entire wave.

## Requirement Coverage

| Requirements | Tasks | Evidence |
| --- | --- | --- |
| FR-001–FR-003, SC-001/SC-003 | T003–T010, T012 | Four-version terminal family matrix and nonterminal controls |
| FR-004, SC-002 | T003–T012 | Zero-submission no-ops, Ok/status assertions, bulk success |
| FR-005, FR-010, SC-004 | T013–T022 | Recovery waits and real cancellation in complete definition cleanup |
| FR-006, SC-003 | T023–T028, T030 | Adapter-backed strict expectations, waiter and command checks |
| FR-007–FR-008, SC-005 | T023–T032 | Flag/request/error compatibility, output checks, docs, full race suite |
| FR-009 | T003–T010, T013–T020, T023–T026, T030 | All four supported adapters covered |

## Implementation Strategy

Deliver US1 first as the smallest useful increment: terminal family confirmation and genuine no-op success across all supported versions. Validate it independently before adding forced-cleanup proof in US2, then lock down operator compatibility in US3. Complete documentation and full validation before calling the issue done. No model creation, migrations, new endpoints, or general cleanup refactoring are needed.

## Notes

Use existing fixture ownership and `testx` helpers where applicable. Preserve unrelated working-tree changes and feature context. A passing test command with zero matching tests is insufficient; confirm test selection when names change. This file is a work plan, not evidence that implementation or tests have already completed.
