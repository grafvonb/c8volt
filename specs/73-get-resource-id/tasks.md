# Tasks: Add Resource Get Command By Id

**Input**: Design documents from `/specs/73-get-resource-id/`
**Prerequisites**: [plan.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/plan.md), [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/spec.md), [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/research.md), [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/data-model.md), [contracts/cli-command-contract.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/contracts/cli-command-contract.md), [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/quickstart.md)

**Tests**: Automated test tasks are REQUIRED for every story and shared change in this feature.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture the existing `get` command, resource facade, and regression surfaces before implementation

- [ ] T001 Review the current `get` command and view helpers in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`
- [ ] T002 Review the current resource facade and service lookup surface in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/contract.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/api.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/resource.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/api.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared public API and validation boundaries that all user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Extend the public resource facade contract and model for single-resource retrieval in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/api.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/model.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/contract.go`
- [ ] T004 [P] Define the `get resource` command shape, required `--id` flag, and validation/error boundaries in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_errors.go`
- [ ] T005 [P] Add foundational regression coverage for facade wiring and required-flag validation in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory_test.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in priority order or parallel where noted

---

## Phase 3: User Story 1 - Retrieve one resource by id (Priority: P1) 🎯 MVP

**Goal**: Deliver a working `c8volt get resource --id <id>` command that returns the normal single-resource object/details view

**Independent Test**: Run the command with a valid resource id and confirm it renders one resource; run it with a missing or unknown id and confirm it fails with the normal non-success behavior.

### Tests for User Story 1

- [ ] T006 [P] [US1] Add command tests for resource lookup success and not-found failure behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`
- [ ] T007 [P] [US1] Add versioned service regression tests for single-resource retrieval and malformed-payload handling in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go`

### Implementation for User Story 1

- [ ] T008 [US1] Implement the public single-resource retrieval method and domain-to-facade mapping in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/model.go`
- [ ] T009 [US1] Add the `get resource` Cobra command and route it through `cli.GetResource(...)` in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go`
- [ ] T010 [US1] Add a single-resource renderer for resource details/object output in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go`
- [ ] T011 [US1] Preserve current malformed-response error behavior for both supported versions while exposing lookup through the CLI path in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service.go`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Reuse existing get-command conventions (Priority: P2)

**Goal**: Make the new resource lookup command behave consistently with existing `get` commands for validation, output, and exit handling

**Independent Test**: Compare `c8volt get resource --id <id>` with other single-item `get` commands and confirm required-flag validation, output rendering, and failure handling follow the same conventions.

### Tests for User Story 2

- [ ] T012 [P] [US2] Add command tests for missing, empty, and invalid `--id` validation behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`
- [ ] T013 [P] [US2] Add facade and service regression tests for consistent error mapping and single-item resource output shape in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client_test.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go`

### Implementation for User Story 2

- [ ] T014 [US2] Tighten `--id` validation so the command fails before any lookup when the identifier is missing, empty, or whitespace-only in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go`
- [ ] T015 [US2] Align resource detail rendering with existing single-item `get` views and JSON/list behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_get.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go`
- [ ] T016 [US2] Ensure aggregate CLI wiring exposes the new resource capability without changing unrelated `get` behavior in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go`, `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/contract.go`, and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/api.go`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Discover the new resource lookup workflow (Priority: P3)

**Goal**: Make the new command discoverable through help text and generated documentation

**Independent Test**: Review `c8volt get --help`, `c8volt get resource --help`, and generated CLI docs to confirm users can discover the command and its required `--id` usage without reading source code.

### Tests for User Story 3

- [ ] T017 [P] [US3] Add help-output assertions for the new resource command and required `--id` guidance in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go`

### Implementation for User Story 3

- [ ] T018 [US3] Update `get` and `get resource` help text to describe the single-resource lookup workflow in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get.go` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go`
- [ ] T019 [US3] Update user-facing usage guidance if needed in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md` and `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/index.md`
- [ ] T020 [US3] Regenerate CLI reference output for the new command in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli/`

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and repository-wide consistency checks

- [ ] T021 [P] Run the quickstart validation steps in `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/quickstart.md`
- [ ] T022 Run targeted regression commands `go test ./cmd -run 'TestGet.*Resource' -count=1` and `go test ./c8volt/resource ./internal/services/resource/... -count=1` from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt`
- [ ] T023 Regenerate user-facing CLI documentation with `make docs` from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt`
- [ ] T024 Run the repository validation command set, including `make test`, from `/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: Depend on Foundational phase completion
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational - this is the MVP and delivers the core resource lookup behavior
- **User Story 2 (P2)**: Starts after User Story 1 because command-convention hardening depends on the core command and renderer existing
- **User Story 3 (P3)**: Starts after User Stories 1 and 2 because help and docs should reflect the final shipped CLI contract

### Within Each User Story

- Tests are added or updated before story sign-off
- Facade and command contracts before command-path polish
- Core lookup behavior before help and documentation refresh
- Targeted validation before repository-wide validation
- Generated docs after help text is finalized

### Parallel Opportunities

- `T004` and `T005` can run in parallel after the shared surface is understood
- `T006` and `T007` can run in parallel within User Story 1
- `T012` and `T013` can run in parallel within User Story 2
- `T019` and `T020` can run in parallel within User Story 3 after help text is finalized
- `T021` can run in parallel with `T022` once implementation is complete

---

## Parallel Example: User Story 1

```bash
# Launch User Story 1 lookup regression coverage together:
Task: "Add command tests for resource lookup success and not-found failure behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go"
Task: "Add versioned service regression tests for single-resource retrieval and malformed-payload handling in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch User Story 2 convention-hardening coverage together:
Task: "Add command tests for missing, empty, and invalid --id validation behavior in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go"
Task: "Add facade and service regression tests for consistent error mapping and single-item resource output shape in /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client_test.go, /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v87/service_test.go, and /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/v88/service_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Confirm `c8volt get resource --id <id>` works independently for success and not-found cases

### Incremental Delivery

1. Add the public resource lookup capability and CLI command as the MVP
2. Harden validation and renderer behavior so the command matches other `get` flows
3. Update help and generated docs for discoverability
4. Finish with targeted validation, docs generation, and `make test`

### Parallel Team Strategy

With multiple contributors:

1. One contributor completes Setup + Foundational work
2. After foundational work lands:
   - Contributor A: User Story 1 lookup path
   - Contributor B: User Story 2 validation and renderer hardening
   - Contributor C: User Story 3 help and docs refresh after the CLI contract stabilizes

---

## Notes

- [P] tasks touch different files or can be validated independently
- [US1], [US2], and [US3] map directly to the user stories in [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/73-get-resource-id/spec.md)
- Keep malformed success responses as errors by preserving the existing `internal/services/common.RequirePayload` behavior in the underlying versioned services
- Do not hand-edit generated CLI reference pages beyond what is produced from Cobra metadata; regenerate them with `make docs`
- Suggested MVP scope: Phase 3 / User Story 1
