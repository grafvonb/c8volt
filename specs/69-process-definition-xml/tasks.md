# Tasks: Add Process Definition XML Command

**Input**: Design documents from `/specs/69-process-definition-xml/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/plan.md) (required), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/spec.md) (required for user stories), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/data-model.md), [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/contracts/cli-command-contract.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/quickstart.md)

**Tests**: Automated test tasks are REQUIRED for every story and shared change in this feature.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the existing process-definition command, facade, and test surface for the XML extension

- [ ] T001 Inspect the current process-definition command, facade wiring, and XML-capable services in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/api.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/api.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service.go
- [ ] T002 Inspect baseline command and processdefinition regression coverage in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared facade and CLI validation boundaries that all stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Extend the public process-definition facade contract for XML retrieval in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/api.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go
- [ ] T004 [P] Define the CLI flag and validation rules for XML mode in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_errors.go
- [ ] T005 [P] Add foundational regression coverage for facade wiring and XML flag validation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin in priority order or parallel where noted

---

## Phase 3: User Story 1 - Retrieve process definition XML by id (Priority: P1) 🎯 MVP

**Goal**: Deliver XML retrieval for a single process definition through the existing `get process-definition` command path

**Independent Test**: Run the command with a valid process definition key and confirm it prints only the XML payload; run it with a missing or unavailable definition and confirm it fails with the normal non-success behavior.

### Tests for User Story 1

- [ ] T006 [P] [US1] Add command tests for XML success and retrieval failure behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go
- [ ] T007 [P] [US1] Add versioned service regression tests for XML payload retrieval in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/v88/service_test.go

### Implementation for User Story 1

- [ ] T008 [US1] Expose XML retrieval through the process client implementation in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go
- [ ] T009 [US1] Implement the XML retrieval execution path in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go
- [ ] T010 [US1] Add a dedicated raw XML stdout renderer that bypasses the default list/detail views in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Use XML output in standard CLI workflows (Priority: P2)

**Goal**: Make XML output safe and predictable for shell redirection and conflicting-flag validation

**Independent Test**: Redirect XML output to a file and confirm the file contains only BPMN XML; verify `--xml` rejects conflicting output or list-style flags with clear validation semantics.

### Tests for User Story 2

- [ ] T011 [P] [US2] Add command tests for redirected XML output and conflicting flag combinations in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go
- [ ] T012 [P] [US2] Add facade-level regression tests for XML retrieval return values and error mapping in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/client.go

### Implementation for User Story 2

- [ ] T013 [US2] Tighten `--xml` validation so it requires `--key` and rejects incompatible render or filter combinations in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go
- [ ] T014 [US2] Ensure XML mode writes redirect-safe output without summaries or JSON wrapping in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go
- [ ] T015 [US2] Preserve non-XML `get process-definition` search and detail behavior while integrating XML mode in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Understand when to use XML retrieval (Priority: P3)

**Goal**: Make the XML option discoverable through help text and generated documentation

**Independent Test**: Review command help and generated docs to confirm the XML flag, required key usage, and redirect-oriented output behavior are documented without reading source code.

### Tests for User Story 3

- [ ] T016 [P] [US3] Add help-output assertions for the XML option and its usage guidance in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go

### Implementation for User Story 3

- [ ] T017 [US3] Update `get process-definition` help text to describe XML retrieval and its flag constraints in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go
- [ ] T018 [US3] Update user-facing process-definition usage examples if needed in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md
- [ ] T019 [US3] Regenerate CLI reference output for the XML flag in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/c8volt_get_process-definition.md and related generated files under /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and repository-wide consistency checks

- [ ] T020 [P] Run the quickstart validation steps in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/quickstart.md
- [ ] T021 Run targeted regression commands `go test ./cmd -run 'TestGet.*ProcessDefinition.*XML|TestGet.*ProcessDefinition.*Help' -count=1` and `go test ./c8volt/process ./internal/services/processdefinition/... -count=1` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T022 Regenerate user-facing CLI documentation with `make docs` from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
- [ ] T023 Run the repository validation command set, including `make test`, from /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: Depend on Foundational phase completion
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational - this is the MVP and delivers the core XML retrieval path
- **User Story 2 (P2)**: Starts after User Story 1 because redirect-safe behavior and conflicting-flag rules depend on the XML path existing
- **User Story 3 (P3)**: Starts after User Stories 1 and 2 because help and docs should reflect the final CLI contract

### Within Each User Story

- Tests are added or updated before story sign-off
- Facade and validation boundaries before command-path refinements
- XML retrieval behavior before redirect-safety polish
- Help text updates before generated docs
- Targeted validation before repository-wide validation

### Parallel Opportunities

- T004 and T005 can run in parallel once the affected XML surface is understood
- T006 and T007 can run in parallel within User Story 1
- T011 and T012 can run in parallel within User Story 2
- T018 and T019 can run in parallel within User Story 3 after help text is finalized
- T020 can run in parallel with T021 once implementation is complete

---

## Parallel Example: User Story 1

```bash
# Launch User Story 1 XML regression coverage together:
Task: "Add command tests for XML success and retrieval failure behavior in cmd/get_test.go"
Task: "Add versioned service regression tests for XML payload retrieval in internal/services/processdefinition/v87/service_test.go and internal/services/processdefinition/v88/service_test.go"
```

---

## Parallel Example: User Story 3

```bash
# Launch documentation work together after the final CLI contract is stable:
Task: "Update user-facing process-definition usage examples if needed in README.md and docs/index.md"
Task: "Regenerate CLI reference output for the XML flag in docs/cli/c8volt_get_process-definition.md and related generated files under docs/cli/"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Confirm XML retrieval works independently for success and failure cases

### Incremental Delivery

1. Add the XML facade and CLI path as the MVP
2. Add redirect-safe behavior and conflicting-flag validation
3. Update help and documentation for discoverability
4. Finish with targeted validation, docs generation, and `make test`

### Parallel Team Strategy

With multiple developers:

1. One developer completes Setup + Foundational work
2. After foundational work lands:
   - Developer A: User Story 1 XML retrieval path
   - Developer B: User Story 2 redirect and validation behavior
   - Developer C: User Story 3 help and docs refresh after the CLI contract stabilizes

---

## Notes

- [P] tasks touch different files or can be validated independently
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/69-process-definition-xml/spec.md)
- The current repository command uses `--key` for single process-definition lookup, so implementation and documentation tasks should align the issue’s `--id` wording with the existing CLI flag unless a deliberate rename is approved
- Do not hand-edit generated CLI reference pages beyond what is produced from Cobra metadata; regenerate them with `make docs`
