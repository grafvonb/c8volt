# Tasks: Align Camunda 8.10 Guidance

**Input**: Design documents from `/specs/277-align-v810-guidance/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/v810-guidance-contract.md, quickstart.md

**Tests**: Focused documentation and rendered-template assertions are required by the specification and plan. Write the named regression checks first, confirm the new expectation fails, then update the authored guidance source.

**Organization**: Tasks are grouped by user story so maintainer guidance, gateway semantics, fixture-history separation, and operator guidance can each be implemented and verified as a distinct outcome.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on another incomplete task
- **[Story]**: Maps the task to a specification user story
- Every task names the exact repository path it changes or uses for recorded evidence

## Phase 1: Setup (Shared Evidence)

**Purpose**: Establish the feature rules, baseline, and evidence record before changing guidance.

- [ ] T001 Read `AGENTS.md`, `specs/ralph-implementation-rules.md`, `specs/277-align-v810-guidance/spec.md`, `specs/277-align-v810-guidance/plan.md`, `specs/277-align-v810-guidance/research.md`, `specs/277-align-v810-guidance/data-model.md`, `specs/277-align-v810-guidance/contracts/v810-guidance-contract.md`, `specs/277-align-v810-guidance/quickstart.md`, and `.specify/memory/constitution.md`, then initialize `specs/277-align-v810-guidance/progress.md` with the conflict check and constitution result
- [ ] T002 Record feature baseline `d7f87d5d`, current branch, editable guidance paths, generated derivatives, integration-only guidance, and protected historical/runtime paths in `specs/277-align-v810-guidance/progress.md`

---

## Phase 2: Foundational (Authoritative Contract Baseline)

**Purpose**: Confirm the final delivered behavior against which every documentation edit will be judged.

**⚠️ CRITICAL**: Complete this phase before editing any user-story guidance.

- [ ] T003 Verify canonical V810 identity, aliases, V88 default, support sets, and active baseline in `toolx/version.go` and `toolx/camunda_baseline.go`, then record the exact no-change contract in `specs/277-align-v810-guidance/progress.md`
- [ ] T004 Verify gateway diagnostics and C810 selection in `cmd/config_diagnostics.go`, `toolx/fixture_compatibility.go`, `cmd/embed_files.go`, and `internal/services/ops/smoke_test_service.go`, then record the active-behavior baseline in `specs/277-align-v810-guidance/progress.md`

**Checkpoint**: The #273/#275 final behavior, guidance classifications, and protected scope are explicit and ready for story work.

---

## Phase 3: User Story 1 - Maintain Version-Aware Features Correctly (Priority: P1) 🎯 MVP

**Goal**: Make every current maintainer and architecture inventory include V810 while preserving V88 as the default and retaining version-neutral layering rules.

**Independent Test**: Review all supported-runtime inventories in the four active guidance sources and confirm they include V87, V88, V89, and V810, name V810 as newest where applicable, and retain V88 as default.

### Implementation for User Story 1

- [ ] T005 [P] [US1] Add `v810` to the version-specific adapter and API-difference guidance in `AGENTS.md` without changing the existing layer boundaries
- [ ] T006 [P] [US1] Update supported factory, adapter, generated-client, capability-review, fixture-family, and testing inventories for V810 while preserving explicit V88 default and stable-only exceptions in `specs/ralph-implementation-rules.md`
- [ ] T007 [P] [US1] Extend the supported-version tradeoff and version-gated architecture statements through Camunda 8.10 in `.specify/memory/architecture.md`
- [ ] T008 [P] [US1] Update V810 target, external boundary, generated-client provenance, adapter inventory, and service-client facts in `.specify/memory/architecture-repo-facts.md`
- [ ] T009 [US1] Run the maintainer inventory scan from `specs/277-align-v810-guidance/quickstart.md` and record exact US1 evidence plus any intentional non-enumerating exceptions in `specs/277-align-v810-guidance/progress.md`

**Checkpoint**: A maintainer can discover all four supported runtime lines, the V810-owned implementation boundaries, and the unchanged V88 default from current repository guidance alone.

---

## Phase 4: User Story 2 - Interpret Gateway Compatibility Consistently (Priority: P1)

**Goal**: Give each Camunda 8.10 gateway observation one unambiguous match or diagnostic outcome without changing runtime behavior.

**Independent Test**: Verify plain, patch, and prerelease 8.10 values are documented as matches; another major/minor as a mismatch diagnostic; and empty or unparseable values as unverifiable diagnostics, with no new hard-failure promise.

### Tests for User Story 2

- [ ] T010 [P] [US2] Add a failing help-contract assertion for the complete V810 match, mismatch, and unverifiable diagnostic matrix in `cmd/config_test.go`, and confirm it fails before authored help changes

### Implementation for User Story 2

- [ ] T011 [P] [US2] Replace ambiguous hard-rejection language in FR-006 and SC-002 with match, diagnostic non-match, and unverifiable-diagnostic semantics in `specs/273-camunda-v810-support/spec.md`
- [ ] T012 [US2] Clarify the authored `config test-connection` help for same-line patch/prerelease matches, different-line warnings, and empty/unparseable warnings in `cmd/config_test_connection.go`
- [ ] T013 [US2] Run `go test ./cmd -run 'TestConfigTestConnection.*(Help|V810GatewayReleaseLineWarnings)' -count=1` and record the passing gateway matrix in `specs/277-align-v810-guidance/progress.md`
- [ ] T014 [US2] Compare `specs/273-camunda-v810-support/spec.md`, `specs/273-camunda-v810-support/contracts/version-selection.md`, and `cmd/config_test_connection.go` against `specs/277-align-v810-guidance/contracts/v810-guidance-contract.md`, then record the no-hard-failure consistency result in `specs/277-align-v810-guidance/progress.md`

**Checkpoint**: Normative requirements, detailed contract, authored help, and existing behavior define the same six gateway outcomes.

---

## Phase 5: User Story 3 - Distinguish Active C810 Guidance from History (Priority: P1)

**Goal**: Prove active V810 guidance selects C810 exclusively while historical C89 implementation records and intentional integration scope remain preserved.

**Independent Test**: Scan active #273 design artifacts for fixture mapping, compare historical files with baseline, and run existing fixture-selection tests; active guidance must permit only C810 while historical records retain the #275 supersession context.

### Verification for User Story 3

- [ ] T015 [US3] Audit active C810 guidance in `specs/273-camunda-v810-support/spec.md`, `specs/273-camunda-v810-support/plan.md`, `specs/273-camunda-v810-support/research.md`, `specs/273-camunda-v810-support/data-model.md`, `specs/273-camunda-v810-support/quickstart.md`, and `specs/273-camunda-v810-support/contracts/service-compatibility.md`; correct only a concrete active fallback and record the result in `specs/277-align-v810-guidance/progress.md`
- [ ] T016 [US3] Verify `specs/273-camunda-v810-support/tasks.md`, `specs/273-camunda-v810-support/progress.md`, and `specs/273-camunda-v810-support/ralph-memory.md` retain their baseline content and existing #275 supersession context, then record the protected-history result in `specs/277-align-v810-guidance/progress.md`
- [ ] T017 [US3] Run focused C810 mapping, embedded-family, and smoke-selection checks anchored in `toolx/fixture_compatibility_test.go`, `cmd/embed_test.go`, `embedded/fs_test.go`, and `internal/services/ops/smoke_test_test.go`, then record commands and outcomes in `specs/277-align-v810-guidance/progress.md`

**Checkpoint**: Current V810 fixture guidance has one C810 decision, historical C89 records remain truthful, and live integration remains intentionally limited to stable profiles through V89.

---

## Phase 6: User Story 4 - See One Coherent Operator Contract (Priority: P2)

**Goal**: Make shipped configuration guidance and generated operator documentation agree on identity, aliases, default, baseline, gateway semantics, and update model.

**Independent Test**: Review the authored and generated operator sources, render the configuration template, and confirm every relevant source presents the correct V810 contract without introducing another identity or changing V88 default.

### Tests for User Story 4

- [ ] T018 [US4] Add a failing rendered-template assertion for supported versions `8.7, 8.8, 8.9, 8.10` and V88 default disclosure in `cmd/config_test.go`, and confirm it fails before the template update

### Implementation for User Story 4

- [ ] T019 [US4] Add Camunda 8.10 and explicit V88 default guidance to `config/templates/config.example.yaml` while retaining its intentional example selection
- [ ] T020 [P] [US4] Review `README.md`, `api/README.md`, `cmd/root.go`, and `cmd/version.go` against the operator identity contract, change only concrete drift in those paths, and record the no-change or correction result in `specs/277-align-v810-guidance/progress.md`
- [ ] T021 [US4] Run `make docs-content` after the US2 help change, retain only source-driven updates in `docs/index.md` and `docs/cli/`, and verify `docs/cli/c8volt_config_test-connection.md` exposes the complete gateway diagnostic matrix
- [ ] T022 [US4] Run the operator-contract scans and `go test ./cmd ./docsgen -count=1` from `specs/277-align-v810-guidance/quickstart.md`, then record the passing US4 evidence in `specs/277-align-v810-guidance/progress.md`

**Checkpoint**: Operators see one coherent V810 configuration, baseline, gateway, and embedded-family contract across authored and generated guidance.

---

## Phase 7: Polish & Cross-Cutting Validation

**Purpose**: Format touched code, prove protected scope, and complete the repository delivery gate.

- [ ] T023 Run `gofmt` on touched Go files `cmd/config_test.go` and `cmd/config_test_connection.go`, then review their diff for documentation-only behavior
- [ ] T024 Run every content and classification scan in `specs/277-align-v810-guidance/quickstart.md` and record final maintainer, gateway, fixture, operator, and integration-scope evidence in `specs/277-align-v810-guidance/progress.md`
- [ ] T025 Verify zero diff from `d7f87d5d` under `internal/clients/camunda/`, `internal/services/`, `c8volt/`, `embedded/processdefinitions/`, `integration/`, and `api/`, recording the protected-scope result in `specs/277-align-v810-guidance/progress.md`
- [ ] T026 Run `git diff --check` and `go test ./cmd ./docsgen -count=1`, resolving only #277 guidance, generated-document, formatting, or documentation-assertion failures in the paths listed by `specs/277-align-v810-guidance/plan.md`
- [ ] T027 Run `make test` and record the full race-enabled repository result in `specs/277-align-v810-guidance/progress.md`
- [ ] T028 Audit the complete diff against `specs/277-align-v810-guidance/spec.md`, `specs/277-align-v810-guidance/contracts/v810-guidance-contract.md`, and the issue #277 scope; confirm no rewritten history, runtime change, new V810 identity, baseline change, BPMN change, or integration expansion in `specs/277-align-v810-guidance/progress.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; starts immediately.
- **Foundational (Phase 2)**: Depends on Setup and blocks guidance edits until the delivered runtime contract is recorded.
- **US1 (Phase 3)**: Depends on Foundational; no dependency on another story.
- **US2 (Phase 4)**: Depends on Foundational; no dependency on another story for authored requirements/help.
- **US3 (Phase 5)**: Depends on Foundational; verification-only and independent of US1/US2 edits.
- **US4 (Phase 6)**: Depends on Foundational. Template work is independent, while generated documentation task T021 waits for US2 authored help T012.
- **Polish (Phase 7)**: Depends on all selected stories and generated documentation being complete.

### User Story Dependency Graph

```text
Setup → Foundation ─┬→ US1 (maintainer guidance) ───────┐
                    ├→ US2 (gateway semantics) ──┐     │
                    ├→ US3 (C810/history proof) ─┼─────┼→ Polish and delivery gate
                    └→ US4 (operator contract) ──┘     │
                              T021 also waits on US2 T012
```

### Within Each User Story

- Write and run each required failing documentation assertion before changing its authored source.
- Update authored sources before regenerating derived documentation.
- Complete the story's focused scan/test and record evidence before its checkpoint.
- Preserve historical and integration-only records throughout; they are comparison baselines, not cleanup targets.

### Parallel Opportunities

- US1 tasks T005–T008 touch separate guidance files and can run in parallel after Foundation.
- US2 test task T010 and normative specification task T011 can run in parallel; T012 follows the failing test.
- US3 can begin in parallel with US1 and US2 because it is a read/validation slice over different paths.
- US4 operator-source review T020 can run in parallel with the test-first template work T018–T019.
- Different stories can proceed concurrently after Foundation, except generated documentation T021 must include the completed US2 authored help.

---

## Parallel Example: User Story 1

```text
Task T005: "Update V810 adapter guidance in AGENTS.md"
Task T006: "Update V810 implementation rules in specs/ralph-implementation-rules.md"
Task T007: "Update supported-version synthesis in .specify/memory/architecture.md"
Task T008: "Update observable V810 facts in .specify/memory/architecture-repo-facts.md"
```

## Parallel Example: User Story 2

```text
Task T010: "Add the failing gateway help contract assertion in cmd/config_test.go"
Task T011: "Clarify FR-006 and SC-002 in specs/273-camunda-v810-support/spec.md"
```

## Parallel Example: User Story 3

```text
Task T015: "Audit active #273 C810 guidance paths" can run alongside US1 task T005 and US2 task T010.
```

US3 tasks remain sequential within the story because T015–T017 append evidence to the same `specs/277-align-v810-guidance/progress.md` record.

## Parallel Example: User Story 4

```text
Task T018: "Add the failing rendered-template assertion in cmd/config_test.go"
Task T020: "Review README.md, api/README.md, cmd/root.go, and cmd/version.go"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Complete Setup and Foundational phases.
2. Complete T005–T009 for US1.
3. Stop and run the US1 independent inventory check.
4. Demonstrate that current maintainer guidance now discovers V810 correctly while preserving V88 default.

### Incremental Delivery

1. **Foundation**: Freeze authoritative behavior and artifact classifications.
2. **US1**: Correct maintainer and architecture guidance.
3. **US2**: Align gateway requirements and authored help with diagnostics.
4. **US3**: Prove C810 is active and C89 history remains preserved.
5. **US4**: Correct the shipped template and regenerate operator documentation.
6. **Polish**: Run protected-scope, focused, generated-doc, and full repository gates.

### Parallel Team Strategy

After Foundation:

- Maintainer A can execute US1 across durable guidance and architecture memory.
- Maintainer B can execute US2 test-first gateway clarification.
- Maintainer C can execute the US3 active/history audit.
- US4 template testing and operator-source review can begin concurrently, with generated docs waiting for US2 authored help.

---

## Notes

- `[P]` tasks operate on different paths and have no dependency on another incomplete task.
- Story labels map directly to the four independently testable scenarios in `spec.md`.
- Historical #273 tasks, progress, and Ralph memory must not be rewritten.
- Integration-only V87–V89 matrices are intentional because live V810 integration remains out of scope.
- Generated documentation must be regenerated from source and never hand-edited.
- Commit after each task or coherent task group using Conventional Commits and issue #277 when implementation begins.
