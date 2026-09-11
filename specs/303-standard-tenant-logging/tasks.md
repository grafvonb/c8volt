# Tasks: Standard Tenant Logging for Delete and Cancel

**Input**: Design documents from `specs/303-standard-tenant-logging/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/tenant-logging.md`, and `quickstart.md`.

**Tests**: Required by FR-008, SC-004, the implementation plan, and the project constitution. Use existing fixtures; add coverage only where current tests do not prove the stated contract.

**Organization**: Tasks are grouped by user story. The two-line routing correction delivers the shared behavior; later stories validate additional independent acceptance dimensions rather than introduce duplicate implementations.

## Format: `[ID] [P?] [Story] Description`

- `[P]` identifies independent file edits that can proceed together after their stated prerequisites.
- `[US1]`, `[US2]`, and `[US3]` map to the specification's user stories.
- Paths are relative to the repository root. Tests mutating command globals must use existing reset/cleanup patterns, not `t.Parallel()`.

## Phase 1: Setup

**Purpose**: Confirm the existing feature context and toolchain; no project scaffolding or dependencies are needed.

- [x] T001 Verify branch `codex/303-standard-tenant-logging`, read the feature artifacts and `AGENTS.md`, and confirm the Go toolchain against `go.mod`; record the implementation environment and any blockers in `specs/303-standard-tenant-logging/quickstart.md`. For Ralph execution, also read `specs/ralph-implementation-rules.md` and surface any conflict before implementation.

## Phase 2: Foundational

**Purpose**: Establish the regression baseline and existing test support before changing either emitter.

- [x] T002 Inspect `cmd/ops_progress_render.go`, `cmd/processinstance_mutation_progress_test.go`, `cmd/ops_progress_test.go`, and `cmd/cmd_confirmation_terminal_test.go` for the existing logger, cleanup, and terminal fixtures; run `go test ./cmd -run 'Test(ProcessInstanceMutation|CancelProcessInstance|DeleteProcessInstance|Confirmation)' -count=1` and record the baseline in `specs/303-standard-tenant-logging/quickstart.md` without treating a baseline pass as proof of the new behavior.

**Checkpoint**: Existing helper semantics and test fixtures are understood; all story tasks depend on this phase. No new shared production infrastructure is required.

## Phase 3: User Story 1 — Recognize Tenant Information and Warnings (Priority: P1, MVP)

**Goal**: Restore standard INFO/WARN reporting while retaining text, ordering, and deduplication.

**Independent Test**: Both emitters report mixed informational and warning tenant lines with an attached plain logger at INFO level, with exact expected message ordering, correct severity, no stdout records, and no duplicate reporting.

### Tests

- [ ] T003 [US1] Add `TestProcessInstanceMutationTenantSeverity` cases in `cmd/processinstance_mutation_progress_test.go` for both `printProcessInstanceMutationTenantContext` and `renderProcessInstanceMutationTenantContextStderr`, attaching `logging.New` through `logging.ToContext` with separate stdout/stderr buffers; cover configured/override/scope/affected/unknown-tenant messages with fixed expected text and severity, validating timestamp prefixes separately. Follow the data-model constraint "Never recompute warning classification or normalize tenant identifiers during rendering." Run `go test ./cmd -run '^TestProcessInstanceMutationTenantSeverity$' -count=1` and confirm the severity assertions fail on the original raw writes before T004.

### Implementation

- [ ] T004 [US1] Replace the raw write in each of the two tenant-context loops in `cmd/processinstance_mutation_progress.go` with `printOpsDurableLine(cmd, line.Text, line.Warn)`, preserving every existing guard and mark-before-emission call. Enforce "Preserve line order and current producer deduplication." and "Do not attach new tenant evidence or issue requests from rendering." from `data-model.md`; leave `cmd/ops_progress_render.go`, tenant line producers, dry-run renderers, and all backend behavior unchanged.
- [ ] T005 [US1] Extend `cmd/processinstance_mutation_progress_test.go` with same-path and cross-path repeat-reporting cases, absent/zero context, and no-logger configured-stderr fallback; assert exact raw fallback bytes and zero stdout. Run `go test ./cmd -run 'TestProcessInstanceMutation(Tenant|Progress)' -count=1` to prove both the new severity cases and existing ordering/provenance regressions pass.

**Checkpoint**: US1 provides the smallest demonstrable fix. The remaining stories and final validation are still required to accept issue #303.

## Phase 4: User Story 2 — Apply Configured Log Format and Level (Priority: P2)

**Goal**: Prove both paths honor plain/JSON log formatting and INFO/WARN/ERROR thresholds through the shared helper.

**Independent Test**: For each emitter and log format, INFO includes information and warnings, WARN includes only warnings, and ERROR emits zero tenant records, with no raw fallback.

### Tests and Integration Validation

- [ ] T006 [US2] Extend the attached-logger cases in `cmd/processinstance_mutation_progress_test.go` under the `TestProcessInstanceMutationTenant` prefix to cover both emitters × plain/JSON formats × info/warn/error levels; validate standard plain prefixes plus exact ordered text/severity, decode JSON `time`, `level`, and `msg` records and require EOF after expected records, and assert empty stdout. Reuse the T004 integration; do not add a second formatter or change `toolx/logging`.
- [ ] T007 [US2] Add filtered-first-report regression cases in `cmd/processinstance_mutation_progress_test.go`: suppress all lines at ERROR, retain the command's rendered state while attaching a permissive logger, and verify subsequent reporting does not replay messages. Assert "Filtering never causes raw fallback." from `data-model.md`, then run `go test ./cmd -run 'TestProcessInstanceMutationTenant' -count=1` and confirm the format/level subtests actually execute.

**Checkpoint**: Format and threshold behavior is independently verified. US2 uses the same production integration completed by US1; its new tests may already pass after T004 and must not motivate unnecessary production changes.

## Phase 5: User Story 3 — Preserve Command Output and Execution Behavior (Priority: P2)

**Goal**: Preserve command output, tenant eligibility, interaction, and request behavior across both commands.

**Independent Test**: Execute delete/cancel with separate streams and attached logging across supported output modes, retaining exact result contracts, prompt behavior, and discovery/mutation counts.

### Command Regression Coverage

- [ ] T008 [P] [US3] Extend attached-logger execution coverage in `cmd/cancel_processinstance_test.go` and existing selector cases in `cmd/cancel_processinstance_selector_test.go` for ordinary/verbose human, quiet, automation, auto-confirm, dry-run, JSON-result, keys-only, quiet+JSON, quiet+keys, and supported no-wait combinations. Preserve "Result envelopes and nested tenantContext payloads remain unchanged." from `data-model.md`; capture streams separately, decode one result envelope plus EOF, assert exact human/preview and keys bytes, and verify tenant logging adds no discovery or mutation calls. Reuse existing empty/sparse-page, explicit-key, abort, and error cases; add only missing assertions or cases and retain zero-byte empty keys output and prompt/mutation-free empty scope.
- [ ] T009 [P] [US3] Extend equivalent delete execution coverage in `cmd/delete_processinstance_test.go` and `cmd/delete_processinstance_selector_test.go` with attached logging and independent streams for the same modes/combinations as T008; retain exact result schemas and human/dry-run wording, one JSON envelope plus EOF, zero-byte empty keys output, unchanged request counts, sparse-page continuation, explicit-key behavior, abort/error handling, and empty-scope prompt/mutation suppression.
- [ ] T010 [P] [US3] Extend or reuse real-terminal fixtures in `cmd/cmd_confirmation_terminal_test.go` to verify both commands with terminal stdin and independently captured stdout/stderr, configured and inherited stderr destinations, confirmation acceptance/abort, and prompt-free empty scope. Include attached/normal command logging in plain and JSON formats where tenant context is eligible; assert prompts retain exact plain text and never enter stdout, and retain keys-only paging with one key per stdout line. Do not replace terminal checks with mocks or change prompt implementation.
- [ ] T011 [US3] Run `go test ./cmd -run 'Test(Cancel|Delete)ProcessInstance' -count=1` and `go test ./cmd -run 'TestConfirmation' -count=1`; document the exercised existing/new cases and any unsupported terminal checks in `specs/303-standard-tenant-logging/quickstart.md`, verifying the T008–T010 matrix rather than counting skipped or unmatched tests as passes.

**Checkpoint**: Both workflows satisfy compatibility criteria; tests retain existing behavior beyond the two corrected emission sites.

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Align documentation and complete required validation without broadening the production change.

- [ ] T012 [P] Clarify eligible selector tenant diagnostics in the tenant-context paragraph of `README.md`: standard INFO/WARN logging honors configured format/level, existing mode eligibility remains unchanged, and JSON logs are distinct from JSON command results.
- [ ] T013 [P] Add the same concise logging clarification to command source metadata in `cmd/cancel_processinstance.go` and `cmd/delete_processinstance.go`; preserve command names, flags, examples, tenant policy, and prompt wording, and adjust relevant metadata assertions in `cmd/cmd_processinstance_test.go` if needed.
- [ ] T014 Run `make docs-content` using `Makefile` after T012–T013; inspect regenerated `docs/cli/` pages and README-derived documentation for only the intended clarification, without hand-editing generated files.
- [ ] T015 Run gofmt on all touched Go files, the three targeted commands in `specs/303-standard-tenant-logging/quickstart.md`, and `git diff --check`; then run the required `make test` from `Makefile` (`go test ./... -race -count=1`) and record actual outcomes in `specs/303-standard-tenant-logging/quickstart.md` before any implementation commit. Resolve failures within scope and explicitly report environmental blockers.
- [ ] T016 Review the final diff against `specs/303-standard-tenant-logging/contracts/tenant-logging.md` and FR-001–FR-009 in `specs/303-standard-tenant-logging/spec.md`; confirm only the two emission calls plus tests/docs changed, all acceptance evidence is recorded, and update completion checkboxes in `specs/303-standard-tenant-logging/tasks.md` only for finished work.

## Dependencies & Execution Order

```text
T001 → T002 → US1 (T003 → T004 → T005)
                    ├→ US2 (T006 → T007)
                    └→ US3 (T008 || T009 || T010 → T011)
US1 + US2 + US3 → (T012 || T013) → T014 → T015 → T016
```

- US1 depends only on setup/foundation and supplies the shared production correction.
- US2 and US3 depend on completed US1 for final acceptance, but verify distinct behaviors and can proceed independently of one another.
- T008, T009, and T010 own separate files and can be edited concurrently once US1 is complete. Reuse established fixtures rather than introducing competing shared helpers.
- T012 and T013 are independent edits after story completion. Documentation generation waits for both.
- US1 and US2 tasks within each story touch the same files and must remain sequential. Do not mark them parallel just because individual subtests cover different cases.
- The final full test run follows all code, test, and metadata edits. If subsequent changes invalidate its evidence, rerun the relevant checks before completion.

## Parallel Execution Examples

### User Story 1

No safe parallel implementation split: T003 establishes the failing regression, T004 fixes both shared loops, and T005 validates behavior in the same test file. Execute sequentially.

### User Story 2

T006 and T007 edit `cmd/processinstance_mutation_progress_test.go` sequentially. Once US1 is complete, this sequence can proceed alongside US3's independent command test files.

### User Story 3

After US1, edit cancel coverage (T008), delete coverage (T009), and real-terminal coverage (T010) independently, then integrate and run T011. Parallel editing does not authorize `t.Parallel()` for package-global flag tests.

## Requirement Coverage

| Requirement / outcome | Primary tasks |
| --- | --- |
| FR-001, FR-002; SC-001 severity | T003–T005 |
| FR-003; SC-002 format/filtering | T006–T007 |
| FR-004; SC-001 order/deduplication | T003, T005, T007 |
| FR-005, FR-006; SC-003 modes/results | T008–T011 |
| FR-007; SC-004 execution/prompts | T008–T011 |
| FR-008; attached-logger and execution evidence | T003, T005–T011, T015 |
| FR-009; bounded scope | T004, T016 |
| Documentation and full validation gates | T012–T016 |

## Implementation Strategy

Start with T001–T005 for the US1 MVP: demonstrate correct tenant severity through the existing helper with unchanged wording and fallback. Continue with US2 format/filtering evidence and US3 command compatibility. Complete documentation and the full race-enabled suite before treating the issue as done or committing implementation changes. No database, service, facade, API, or new logging infrastructure work is required.

## Notes

All tasks start unchecked. Generating this list does not execute implementation or establish runtime test results. Optional commit and Ralph hooks remain separate actions. Keep future commits small and Conventional Commit formatted with issue #303 in the subject, subject to the constitution's required pre-commit validation.
