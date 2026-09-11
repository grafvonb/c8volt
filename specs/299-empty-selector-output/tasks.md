---
description: "Implementation tasks for issue #299 empty selector result output"
---

# Tasks: Empty Selector Result Output

**Input**: Design documents from `specs/299-empty-selector-output/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/cli-output.md](contracts/cli-output.md), [quickstart.md](quickstart.md)

**Tests**: Required by the feature specification and constitution. Write regression tests before the corresponding fix; demonstrate the existing machine-output and quiet defects. Existing compatibility tests should remain passing.

**Organization**: Tasks are grouped by user story. This existing Go project needs no new dependencies, storage, public models, or service infrastructure.

## Format: `[ID] [P?] [Story] Description`

- `[P]` identifies independent file edits that can be performed together after the prerequisites listed below are complete.
- `[US1]`, `[US2]`, and `[US3]` map to the stories in the specification.
- All file paths are relative to the repository root. Task completion includes the stated validation; leave tasks unchecked until the work is verified.

## Path Conventions

Production CLI changes belong in `cmd/`; existing public report types are in `c8volt/process/model.go`; terminal helpers are in `testx/`; feature evidence belongs in `specs/299-empty-selector-output/`. Generated references under `docs/cli/` must be regenerated from command metadata.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the active feature and implementation constraints without modifying unrelated work.

- [x] T001 Confirm branch `codex/299-empty-selector-output`, feature selection in `.specify/feature.json`, toolchain in `go.mod`, and the constraints in `AGENTS.md`, `.specify/memory/constitution.md`, and `specs/299-empty-selector-output/plan.md`; if execution uses Ralph, also read `specs/ralph-implementation-rules.md` and surface any conflict before implementation. Record setup or validation blockers in `specs/299-empty-selector-output/quickstart.md`.

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish reusable command-test assertions before separate story cases are authored.

- [x] T002 Reuse existing HTTP fixtures and command runners in `cmd/cmd_processinstance_test.go`; add only the shared empty-result assertion support needed by both selector suites to decode one envelope and require EOF on a second decode, compare operation-specific payloads, and inspect separate stdout/stderr. Preserve fixture request checks and reset helpers; do not introduce a production abstraction or parallel tests that mutate package globals.

**Checkpoint**: Shared test support is available; no production behavior has changed.

## Phase 3: User Story 1 - Consume empty results in automation (Priority: P1) — MVP

**Goal**: Both commands return one successful JSON no-op or zero keys-only bytes for normal execution and dry-run, including automation controls.

**Independent Test**: Against empty local HTTP fixtures, run delete/cancel × normal/dry-run × JSON/keys-only, then auto-confirm, automation, and supported no-wait variants. Require successful exit, exactly one envelope followed by EOF or zero stdout bytes, no human text, and no confirmation/mutation calls.

### Tests for User Story 1

- [x] T003 [P] [US1] Add `TestDeleteProcessInstanceEmptySelectorOutput` cases in `cmd/delete_processinstance_selector_test.go` for normal execution and dry-run with JSON and keys-only, auto-confirm, automation with explicit machine output, and supported no-wait combinations; capture streams separately, assert canonical command identity, no `found: 0`, and no confirmation/mutation calls. Run the new test to demonstrate the current output defect before T007.
- [x] T004 [P] [US1] Add matching `TestCancelProcessInstanceEmptySelectorOutput` cases in `cmd/cancel_processinstance_selector_test.go` for normal execution and dry-run with JSON and keys-only, auto-confirm, automation with explicit machine output, and supported no-wait combinations; capture streams separately and assert successful no-op semantics with no confirmation/mutation calls. Run the new test to demonstrate the current output defect before T008.
- [x] T005 [P] [US1] Add focused empty-result view contract tests in `cmd/cmd_views_processinstance_test.go` for both operations and execution types, using the field constraints from `specs/299-empty-selector-output/data-model.md`: `outcome` is "`succeeded`, including normal execution with no-wait"; `command` is "Canonical `delete process-instance` or `cancel process-instance`"; `class`, `detail` are "Absent for successful empty discovery"; `tenantContext` is "Existing attached context when available; no additional lookup"; normal `Items` obeys "Zero reports. Existing `json:"items,omitempty"` serialization omits this field, giving `{}`." For dry-run assert `operation` is "`delete` or `cancel`", all five count fields are "`0`", the four collection fields are "`null`, preserving the existing constructor's nil-slice serialization", `traversalOutcome` is "`complete`", `scopeComplete` is "`true`", `warning` is "Empty string", and `mutationSubmitted` is "`false`". Assert no synthetic entry and unchanged mode precedence.

### Implementation for User Story 1

- [x] T006 [US1] Add a focused empty-selector result view helper in `cmd/cmd_views_processinstance.go` that resolves `pickMode()`, renders JSON with `process.DeleteReports{}` or `process.CancelReports{}` through `renderSucceededResult`, reuses `newProcessInstanceDryRunSummary(operation, nil)` and successful dry-run JSON rendering for previews, and emits nothing for keys-only; preserve existing human `found: 0` output as the fallback until US2 adds quiet suppression. Reuse `cmd/cmd_views_contract.go` and `cmd/cmd_views_processinstance_dryrun.go` without changing their general semantics or the public report types; never use no-wait-sensitive accepted rendering for a no-op.
- [x] T007 [P] [US1] Replace both direct empty-result writes in `cmd/delete_processinstance_selector.go` with the shared view helper after T006; retain the data-model rule "Only successful planning with `RequestedCount == 0` enters the empty-result path." Preserve existing early returns, tenant evidence, abort handling, activity cleanup, and error propagation so empty results render exactly once without new discovery or mutation work.
- [x] T008 [P] [US1] Replace the direct empty-result write in `cmd/cancel_processinstance_selector.go` with the shared view helper after T006; retain the successful aggregate `planned.RequestedCount == 0` guard and existing error handling, cleanup, and empty report/preview returns. Do not detect emptiness from individual pages or report length and do not change nonempty mutation behavior.
- [x] T009 [US1] Format Go files touched in T002–T008 and run `go test ./cmd -run 'Test(Delete|Cancel)ProcessInstanceEmptySelectorOutput|TestProcessInstance.*Empty' -count=1` with new view tests named to match; verify all command/execution combinations and T005 payload constraints, and record results in `specs/299-empty-selector-output/quickstart.md`.

**Checkpoint**: US1 provides the machine-output MVP. Quiet suppression and the full compatibility/release checks remain required for issue completion.

## Phase 4: User Story 2 - Understand an empty result interactively (Priority: P2)

**Goal**: Preserve the compact human summary and suppress it in quiet mode without losing requested machine-readable results.

**Independent Test**: For both commands and execution types, require exactly `found: 0` plus one newline on ordinary human stdout, no informational empty-result message on either stream when quiet, one envelope for quiet+JSON, and zero bytes for quiet+keys-only.

### Tests for User Story 2

- [x] T010 [P] [US2] Extend `TestDeleteProcessInstanceEmptySelectorOutput` in `cmd/delete_processinstance_selector_test.go` with human, quiet human, quiet+JSON, and quiet+keys-only cases for normal execution and dry-run; verify the exact human line and absence of duplicate or stderr summaries, and demonstrate the quiet failure before T012.
- [x] T011 [P] [US2] Extend `TestCancelProcessInstanceEmptySelectorOutput` in `cmd/cancel_processinstance_selector_test.go` with the same human and quiet combinations for normal execution and dry-run; capture streams separately and demonstrate the quiet failure before T012.

### Implementation for User Story 2

- [x] T012 [US2] Add explicit `flagQuiet` suppression only to the human branch of the empty-result helper in `cmd/cmd_views_processinstance.go`, keeping JSON and keys-only precedence intact; use the existing raw output helper for unprefixed `found: 0` rather than changing logger behavior or calling the full human dry-run summary renderer.
- [x] T013 [US2] Format the US2 changes and run `go test ./cmd -run 'Test(Delete|Cancel)ProcessInstanceEmptySelectorOutput|TestProcessInstance.*Empty' -count=1`; verify the entire output matrix in `specs/299-empty-selector-output/contracts/cli-output.md` and record the results in `specs/299-empty-selector-output/quickstart.md`.

**Checkpoint**: Both output stories are independently verifiable. Ordinary human output is unchanged, quiet suppresses the informational summary, and machine output remains intact.

## Phase 5: User Story 3 - Preserve existing selection and execution behavior (Priority: P2)

**Goal**: Prove output changes do not alter selection, requests, prompts, failures, or nonempty execution.

**Independent Test**: Compare exact discovery requests for successful empty scopes and retain representative invalid-selector, discovery-error, sparse-page, abort, nonempty, and explicit-key behavior. Real terminal stdin with no supplied answer must complete without prompting or mutation.

### Tests for User Story 3

- [ ] T014 [P] [US3] Extend compatibility coverage in `cmd/delete_processinstance_selector_test.go`: assert exactly one instance-search request for simple state/date selectors and exactly definition-validation plus instance-search requests for BPMN selectors, in normal and dry-run empty cases; reject unexpected requests. Preserve or add focused cases proving sparse intermediate pages continue, discovery errors do not yield successful no-ops, and aborted/nonempty workflows with empty reports are not reclassified. Retain existing nonempty and explicit-key regression expectations.
- [ ] T015 [P] [US3] Extend matching compatibility coverage in `cmd/cancel_processinstance_selector_test.go` for exact one/two-request empty discovery baselines and zero mutation requests; preserve or add missing-selector, discovery-failure, sparse-page, abort, nonempty, and explicit-key guards so only successful aggregate zero selection produces the new no-op result.
- [ ] T016 [P] [US3] Add `TestConfirmationEmptySelectorResults` in `cmd/cmd_confirmation_terminal_test.go` using `testx.NewCmdTerminalRunner` from `testx/cmd_terminal_runner.go`: exercise delete/cancel × normal/dry-run with real terminal stdin, no input exchanges, no auto-confirm/automation in the interactive cases, separate stdout/stderr, and the existing bounded timeout. Cover human and machine output plus quiet, verify prompt-free successful completion, and assert local fixture request counts and absence of mutations. Reuse configured/inherited output destinations from the existing harness without altering prompt routing.

### Integration and Verification for User Story 3

- [ ] T017 [US3] Format US3 test changes and run `go test ./cmd -run 'Test(Delete|Cancel)ProcessInstance' -count=1` followed by `go test ./cmd -run 'TestProcessInstance|TestConfirm|TestConfirmation' -count=1`; verify the existing exit-code, explicit-key, sparse-page, nonempty, abort, and activity tests remain passing and record actual coverage/results in `specs/299-empty-selector-output/quickstart.md`. Resolve any regression within the scoped call sites in `cmd/delete_processinstance_selector.go` and `cmd/cancel_processinstance_selector.go`; do not broaden error-envelope, tenant, or service behavior.

**Checkpoint**: All three stories satisfy their independent acceptance criteria. No service or public model changes are needed.

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Align user documentation and complete release validation.

- [ ] T018 [P] Document empty selector results in `README.md`, covering JSON success/no-op, zero-byte keys-only, unchanged human summary, quiet suppression, and dry-run semantics; keep examples aligned with `specs/299-empty-selector-output/contracts/cli-output.md`.
- [ ] T019 [P] Update `Long`/`Example` metadata in `cmd/delete_processinstance.go` and `cmd/cancel_processinstance.go` with the same empty-result behavior and extend destructive-help assertions in `cmd/cmd_processinstance_test.go`; preserve existing flags, aliases, and nonempty examples.
- [ ] T020 Run `gofmt` on all Go files changed for this feature, run the affected help tests in `cmd/cmd_processinstance_test.go`, and execute `make docs-content` from `Makefile`; review generated `docs/cli/` and README-derived documentation for accurate empty-result behavior without hand-editing generated files.
- [ ] T021 Execute the complete guide in `specs/299-empty-selector-output/quickstart.md`, including any targeted checks invalidated by later edits, `make test` (`go test ./... -race -count=1`), and `git diff --check`; inspect the final diff for scope and constitution compliance and record outcomes/blockers in that guide. Mark completion only when required validation passes; use Conventional Commits referencing #299 if committing, and preserve unrelated work.

## Dependencies & Execution Order

### Phase Dependencies

```text
T001 setup → T002 test support
                 ↓
US1: (T003, T004, T005) → T006 → (T007, T008) → T009
                 ↓
US2: (T010, T011) → T012 → T013
                 ↓
US3: (T014, T015, T016) → T017
                 ↓
Polish: (T018, T019) → T020 → T021
```

All story work depends on setup and foundational support. US2 extends the shared helper delivered by US1. US3 validates the complete behavior, including US2 quiet mode. This explicit order avoids concurrent edits to shared command and selector test files; independent testability means each story has its own observable acceptance checks, not that they require duplicate infrastructure.

### Within Each User Story

Tests precede the associated behavior change. T003/T004 demonstrate the existing bug; T005 can initially fail to compile until the new helper exists. US2 tests demonstrate quiet suppression is missing. US3 compatibility cases are expected to remain passing. T007 and T008 require T006; each story's validation waits for all its edits to complete.

### Parallel Opportunities and Examples

Parallel markers describe authoring work, not permission to add `t.Parallel()` to global-state command tests. Do not run checks against files being edited by another task.

- **US1**: After T002, author delete tests (T003), cancel tests (T004), and view tests (T005) in their three separate files. After T006, integrate the two delete branches (T007) alongside the cancel branch (T008).
- **US2**: After T009, author delete human/quiet cases (T010) alongside cancel human/quiet cases (T011); integrate the shared quiet fix only after both are ready.
- **US3**: After T013, author delete invariants (T014), cancel invariants (T015), and terminal coverage (T016) together in separate files, then run T017 after they are complete.
- **Polish**: After T017, README work (T018) can proceed alongside command metadata/help work (T019); generation and final validation are sequential.

## Requirement Coverage

| Requirements / outcomes | Tasks |
| --- | --- |
| FR-001–FR-004; SC-001, machine portion of SC-002, SC-006 | T003–T009 |
| FR-005–FR-006; human/quiet portion of SC-002 | T010–T013 |
| FR-007; SC-003–SC-004 | T003–T004, T010–T011, T014–T017 |
| FR-008–FR-009; SC-005 | T007–T008, T014–T017 |
| FR-010 and constitution documentation/validation gates | T018–T021 |

## Implementation Strategy

### MVP First

Complete T001–T009 to deliver US1: scripts receive a valid successful empty JSON result or no keys for both commands and execution types. Validate this checkpoint independently. The full issue still requires quiet behavior, compatibility proofs, documentation, and race-enabled validation.

### Incremental Delivery

Add US2 after the machine-output checkpoint, then prove US3 operational invariants. Finish documentation and the full suite before treating issue #299 as complete. Keep the small production change inside the existing selector and view files; tests may cover a broader execution matrix without refactoring unrelated code.

## Notes

- No backend mechanics, discovery loops, tenant filtering, authorization, mutation targets, generated clients, or shared schema changes are planned.
- No prompt-routing or logger-formatting change is planned. Terminal coverage verifies empty scopes never reach confirmation.
- Execute checks appropriate to each increment and the full race-enabled suite before committing or completing implementation; do not equate task generation with passing tests.
- All tasks are initially unchecked. Task generation does not start implementation or the optional Ralph loop.
