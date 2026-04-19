# Tasks: Improve CLI Help Discovery

**Input**: Design documents from `/specs/077-cli-help-discovery/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-help-contract.md, quickstart.md

**Tests**: Tests are required for this feature because the plan and constitution call for focused `cmd/` regression coverage plus final `make test`.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated as an independent increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (`US1`, `US2`, `US3`)
- Every task includes an exact file path or directory path

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Lock the public command inventory and working slices before editing command metadata.

- [x] T001 Capture the full user-visible command coverage inventory and batching notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/research.md
- [x] T002 Align the implementation and validation checklist for the public command tree in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/quickstart.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared discovery language and regression seams before editing story-specific command groups.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [x] T003 Refresh the shared top-level discovery and automation guidance baseline in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go
- [x] T004 [P] Strengthen public-versus-hidden command coverage assertions in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract_test.go
- [x] T005 [P] Add shared help-output regression helpers for public command metadata in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root_test.go

**Checkpoint**: Shared discovery language and public-command regression seams are ready.

---

## Phase 3: User Story 1 - Choose The Right Command Path (Priority: P1) 🎯 MVP

**Goal**: Make every user-visible root, parent/group, and read-oriented public command explain when it should be used and how to choose the right path.

**Independent Test**: Run help-focused `cmd/` tests and verify that root, parent/group, and representative read-oriented leaf commands now explain purpose, mutation posture, and preferred machine-readable output guidance where supported.

### Tests for User Story 1

- [x] T006 [P] [US1] Add root and parent/group help regression coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go
- [x] T007 [P] [US1] Add discovery and configuration help regression coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_test.go

### Implementation for User Story 1

- [x] T008 [US1] Refresh root and family-entry help text in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/embed.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/version.go
- [x] T009 [P] [US1] Refresh discovery, configuration, and cluster-read examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/config_show.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_license.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go
- [x] T010 [P] [US1] Refresh read-oriented retrieval guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_variable.go
- [x] T011 [P] [US1] Refresh embedded-resource and version command examples in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/embed_list.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/embed_export.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/version.go

**Checkpoint**: User-visible discovery, routing, and read-oriented help should now be independently useful without touching state-changing semantics.

---

## Phase 4: User Story 2 - Understand Confirmation And Completion Semantics (Priority: P2)

**Goal**: Make applicable state-changing and verification-related commands describe waiting, acceptance, confirmation, and follow-up verification truthfully.

**Independent Test**: Verify that the help and examples for state-changing commands explain default waiting behavior, `--no-wait`, `--auto-confirm`, and realistic follow-up verification flows without changing the underlying runtime contract.

### Tests for User Story 2

- [x] T012 [P] [US2] Add state-changing help regression coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go
- [x] T013 [P] [US2] Add wait-and-verification help regression coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go

### Implementation for User Story 2

- [x] T014 [US2] Refresh run and deploy command-family semantics in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_processdefinition.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/embed_deploy.go
- [x] T015 [US2] Refresh cancel and delete command-family confirmation guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processdefinition.go
- [x] T016 [US2] Refresh verification and tree-inspection follow-up guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_processinstance.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go

**Checkpoint**: Applicable state-changing and follow-up verification help should now describe waiting and confirmation behavior clearly enough for unattended and operator-driven usage.

---

## Phase 5: User Story 3 - Keep Generated Docs In Sync With Help Metadata (Priority: P3)

**Goal**: Regenerate the public docs surfaces from the refreshed metadata and keep top-level guidance aligned with the updated command help.

**Independent Test**: Regenerate CLI docs and confirm the resulting public pages reflect the refreshed examples and command guidance without hand-editing generated output.

### Tests for User Story 3

- [ ] T017 [P] [US3] Add doc-parity regression coverage for public help anchors in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities_test.go

### Implementation for User Story 3

- [ ] T018 [US3] Update top-level workflow and discovery guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/use-cases.md
- [ ] T019 [US3] Regenerate the public CLI reference pages in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli using /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile
- [ ] T020 [US3] Sync README-derived documentation content in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md using /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile

**Checkpoint**: Generated docs and top-level guidance should now match the refreshed Cobra metadata.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, coverage audit, and repository-wide validation.

- [ ] T021 [P] Audit hidden/internal command exclusion and public coverage notes in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/research.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/quickstart.md
- [ ] T022 [P] Run focused command-help validation for the refreshed public tree and record the verification flow in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/077-cli-help-discovery/quickstart.md
- [ ] T023 Run repository-wide validation through /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/Makefile with `make test`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user story work.
- **User Story 1 (Phase 3)**: Depends on Foundational; establishes MVP discovery improvements.
- **User Story 2 (Phase 4)**: Depends on Foundational; can proceed after Phase 2, but should be integrated after the shared language from US1 is stable.
- **User Story 3 (Phase 5)**: Depends on the metadata changes from US1 and US2 because it regenerates docs from those sources.
- **Polish (Phase 6)**: Depends on all targeted story work being complete.

### User Story Dependencies

- **US1 (P1)**: No dependency on other user stories once Foundational work is complete.
- **US2 (P2)**: No strict dependency on US1 for execution, but it benefits from the shared vocabulary established in the earlier phases.
- **US3 (P3)**: Depends on completed metadata updates from US1 and US2.

### Within Each User Story

- Regression tests should be updated before or alongside the corresponding metadata edits.
- Parent/group command help should be refreshed before the story’s leaf-command batches when they share terminology.
- Generated docs should be regenerated only after the source metadata and top-level docs text are ready.

### Parallel Opportunities

- **Phase 2**: T004 and T005 can run in parallel after T003.
- **US1**: T006 and T007 can run in parallel; T009, T010, and T011 can run in parallel after T008.
- **US2**: T012 and T013 can run in parallel; T014, T015, and T016 can proceed in parallel once the story tests are in place.
- **US3**: T017 can run in parallel with T018; T019 depends on source metadata completion, and T020 depends on whether T018 changed README content.
- **Polish**: T021 and T022 can run in parallel before T023.

---

## Parallel Example: User Story 1

```bash
# Launch regression updates for root/group and discovery/config help together:
Task: "T006 Add root and parent/group help regression coverage in cmd/root_test.go and cmd/get_test.go"
Task: "T007 Add discovery and configuration help regression coverage in cmd/capabilities_test.go and cmd/config_test.go"

# Launch read-oriented command batches together after family-entry language is refreshed:
Task: "T009 Refresh discovery, configuration, and cluster-read examples in cmd/capabilities.go, cmd/config_show.go, cmd/get_cluster.go, cmd/get_cluster_license.go, and cmd/get_cluster_topology.go"
Task: "T010 Refresh read-oriented retrieval guidance in cmd/get_processdefinition.go, cmd/get_processinstance.go, cmd/get_resource.go, and cmd/get_variable.go"
Task: "T011 Refresh embedded-resource and version command examples in cmd/embed_list.go, cmd/embed_export.go, and cmd/version.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch state-changing and verification regression updates together:
Task: "T012 Add state-changing help regression coverage in cmd/run_test.go, cmd/deploy_test.go, cmd/delete_test.go, and cmd/cancel_test.go"
Task: "T013 Add wait-and-verification help regression coverage in cmd/expect_test.go and cmd/walk_test.go"

# Launch command-family metadata batches together after tests are in place:
Task: "T014 Refresh run and deploy command-family semantics in cmd/run.go, cmd/run_processinstance.go, cmd/deploy.go, cmd/deploy_processdefinition.go, and cmd/embed_deploy.go"
Task: "T015 Refresh cancel and delete command-family confirmation guidance in cmd/cancel.go, cmd/cancel_processinstance.go, cmd/delete.go, cmd/delete_processinstance.go, and cmd/delete_processdefinition.go"
Task: "T016 Refresh verification and tree-inspection follow-up guidance in cmd/expect.go, cmd/expect_processinstance.go, cmd/walk.go, and cmd/walk_processinstance.go"
```

---

## Parallel Example: User Story 3

```bash
# Launch doc-parity test work and top-level guidance updates together:
Task: "T017 Add doc-parity regression coverage for public help anchors in cmd/root_test.go and cmd/capabilities_test.go"
Task: "T018 Update top-level workflow and discovery guidance in README.md and docs/use-cases.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate updated help for root, parent/group, and read-oriented commands with focused `cmd/` tests.
5. Stop and review the discovery UX before widening the scope to semantics-heavy commands.

### Incremental Delivery

1. Finish Setup + Foundational to lock the public-tree inventory and shared language.
2. Deliver US1 to make the public command tree easier to navigate.
3. Deliver US2 to make waiting and confirmation semantics explicit on applicable commands.
4. Deliver US3 to regenerate docs and align top-level guidance with the refreshed metadata.
5. Finish with cross-cutting audits and `make test`.

### Parallel Team Strategy

1. One contributor handles the shared root/family language and public-command inventory.
2. A second contributor can refresh read-oriented leaf command help after the shared language lands.
3. A third contributor can work on state-changing semantics and matching regression tests in parallel after the foundational phase.
4. Once metadata changes settle, docs regeneration and README sync can run as a separate lane before final validation.

---

## Notes

- [P] tasks touch separate files or file groups and can be worked independently once dependencies are met.
- User story labels map directly to the clarified stories in `spec.md`.
- Every command in scope is user-visible; hidden/internal commands remain intentionally excluded.
- Command metadata is the source of truth; generated docs should always be regenerated, never hand-edited.
- Commit messages for downstream implementation must remain Conventional Commits and end with `#77`.
