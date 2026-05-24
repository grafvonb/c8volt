# Tasks: Tenant Scope For Discovery And Explicit Admin Keys

**Input**: Design documents from `specs/156-tenant-admin-keys/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: Required by the specification. Write or update targeted tests before implementation for each story.

**Implementation Context**: Every Ralph implementation iteration must read `specs/ralph-implementation-rules.md`. Ralph launch instructions must include `--implementation-context specs/ralph-implementation-rules.md`.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested in a small Ralph iteration.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm local context and keep implementation bound to repository rules.

- [x] T001 Read `specs/ralph-implementation-rules.md` and verify no conflict with `specs/156-tenant-admin-keys/spec.md`
- [x] T002 [P] Review explicit key/stdin command mode handling in `cmd/get_processinstance*.go`, `cmd/walk_processinstance.go`, `cmd/expect_processinstance.go`, `cmd/cancel_processinstance.go`, and `cmd/delete_processinstance.go`
- [x] T003 [P] Review process-definition and resource direct-ID paths in `cmd/get_processdefinition.go`, `cmd/delete_processdefinition.go`, `cmd/get_resource.go`, `c8volt/resource/client.go`, and `internal/services/resource/`
- [x] T004 [P] Review selected-tenant option flow in `c8volt/foptions/options.go`, `internal/services/calloption.go`, `internal/services/common/`, and affected v88/v89 service packages

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared tenant semantics audit points before story-specific changes.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T005 Identify and record current local tenant mismatch checks or tenant-equality assumptions in `specs/156-tenant-admin-keys/progress.md`
- [x] T006 [P] Add or update shared test fixtures for tenant-a selected context with tenant-b returned metadata in `cmd/get_processinstance_test.go`
- [x] T007 [P] Add or update facade/service stubs for explicit tenant mismatch behavior in `c8volt/process/client_test.go` and `c8volt/resource/client_test.go`
- [x] T008 Verify no new c8volt-side authorization layer is needed and record the chosen repository-native path in `specs/156-tenant-admin-keys/progress.md`

**Checkpoint**: Audit notes and reusable test fixtures are ready for user story implementation.

---

## Phase 3: User Story 1 - Preserve Tenant-Scoped Discovery Boundaries (Priority: P1) MVP

**Goal**: Discovery/search-derived process-instance flows continue to apply the selected tenant and do not broaden c8volt-produced candidate sets across tenants.

**Independent Test**: A Camunda 8.8 or 8.9 discovery/search-derived flow with `--tenant tenant-a` only freezes candidates from tenant-scoped discovery and preserves that scope through preview.

### Tests for User Story 1

- [x] T009 [P] [US1] Add tenant-scoped process-instance search/list test in `cmd/get_processinstance_test.go`
- [x] T010 [P] [US1] Add search-derived `cancel pi --dry-run` tenant-scoped candidate test in `cmd/cancel_test.go`
- [x] T011 [P] [US1] Add search-derived `delete pi --dry-run` tenant-scoped candidate and dependency-scope test in `cmd/delete_test.go`

### Implementation for User Story 1

- [x] T012 [US1] Ensure `get pi` search/list mode continues passing selected tenant through existing filters/options in `cmd/get_processinstance_search.go` and affected process-instance services
- [x] T013 [US1] Ensure search-derived `cancel pi` preserves the tenant-scoped discovered candidate set in `cmd/cancel_processinstance.go` and `c8volt/process/client.go`
- [x] T014 [US1] Ensure search-derived `delete pi` preserves the tenant-scoped discovered candidate set and intended dependency scope in `cmd/delete_processinstance.go` and `c8volt/process/client.go`
- [x] T015 [US1] Run `go test ./cmd -run 'Test(GetProcessInstance|CancelProcessInstance|DeleteProcessInstance).*Tenant' -count=1`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Treat Explicit Process-Instance Keys As Admin Input (Priority: P2)

**Goal**: Direct process-instance key commands rely on Camunda backend authorization and do not reject solely because selected tenant and returned metadata differ.

**Independent Test**: With `--tenant tenant-a`, explicit tenant-b process-instance keys proceed when Camunda returns or accepts the target, while existing safety checks remain active.

### Tests for User Story 2

- [x] T016 [P] [US2] Add `get pi --key` selected-tenant mismatch test in `cmd/get_processinstance_test.go`
- [x] T017 [P] [US2] Add `walk pi --key` selected-tenant mismatch test in `cmd/walk_test.go`
- [x] T018 [P] [US2] Add `expect pi --key` selected-tenant mismatch test in `cmd/expect_test.go`
- [x] T019 [P] [US2] Add `cancel pi --key --dry-run` selected-tenant mismatch test in `cmd/cancel_test.go`
- [x] T020 [P] [US2] Add `delete pi --key --dry-run` selected-tenant mismatch test in `cmd/delete_test.go`

### Implementation for User Story 2

- [x] T021 [US2] Update direct process-instance lookup and enrichment paths to avoid c8volt-side tenant mismatch rejection in `c8volt/process/client.go` and affected `internal/services/processinstance/` packages
- [x] T022 [US2] Preserve existing direct-key cancellation preflight, dry-run, confirmation, force, and verification behavior in `cmd/cancel_processinstance.go` and `c8volt/process/client.go`
- [x] T023 [US2] Preserve existing direct-key deletion preflight, dry-run, dependency expansion, force, and verification behavior in `cmd/delete_processinstance.go` and `c8volt/process/client.go`
- [x] T024 [US2] Run `go test ./cmd ./c8volt/process ./internal/services/processinstance -run 'Test.*(Key|Direct).*Tenant|Test.*Tenant.*Key' -count=1`

**Checkpoint**: User Stories 1 and 2 work independently.

---

## Phase 5: User Story 3 - Align Explicit Definition, Resource, And Stdin Inputs (Priority: P3)

**Goal**: Process-definition keys, resource IDs, stdin keys, and direct flag values follow the same backend-authorized admin-input contract.

**Independent Test**: Explicit process-definition, resource, and stdin-key commands do not fail solely due to selected-tenant mismatch when Camunda authorizes the target.

### Tests for User Story 3

- [x] T025 [P] [US3] Add `get pd --key` and `get pd --xml` selected-tenant mismatch tests in `cmd/get_test.go`
- [x] T026 [P] [US3] Add `delete pd --key` selected-tenant mismatch dry-run or auto-confirm safety test in `cmd/delete_test.go`
- [x] T027 [P] [US3] Add `get resource --id` selected-tenant mismatch test in `cmd/get_test.go`
- [x] T028 [P] [US3] Add stdin key selected-tenant mismatch coverage for process-instance or process-definition bulk input in `cmd/delete_test.go`
- [x] T029 [P] [US3] Add facade/resource option propagation test in `c8volt/resource/client_test.go`

### Implementation for User Story 3

- [x] T030 [US3] Update process-definition direct-key behavior to avoid c8volt-side tenant mismatch rejection in `cmd/get_processdefinition.go`, `cmd/delete_processdefinition.go`, `c8volt/processdefinition/`, and `internal/services/processdefinition/`
- [x] T031 [US3] Update resource direct-ID behavior to avoid c8volt-side tenant mismatch rejection in `cmd/get_resource.go`, `c8volt/resource/`, and `internal/services/resource/`
- [x] T032 [US3] Ensure stdin keys remain classified as explicit admin input after validation and deduplication in `cmd/cmd_stdin.go`, `cmd/cmd_cli.go`, and affected command files
- [x] T033 [US3] Run `go test ./cmd ./c8volt/resource ./internal/services/processdefinition ./internal/services/resource -run 'Test.*(ProcessDefinition|Resource|Stdin).*Tenant|Test.*Tenant.*(ProcessDefinition|Resource|Stdin)' -count=1`

**Checkpoint**: User Stories 1, 2, and 3 work independently.

---

## Phase 6: User Story 4 - Document Tenant Semantics For Operators (Priority: P4)

**Goal**: Help text, README, command contract tests, and generated docs explain discovery-scoped tenant behavior versus backend-authorized explicit admin input.

**Independent Test**: Relevant help and generated docs describe the tenant contract consistently.

### Tests for User Story 4

- [x] T034 [P] [US4] Add command contract assertions for tenant flag/help wording in `cmd/command_contract_test.go`
- [x] T035 [P] [US4] Add command help assertions for affected process-instance, process-definition, and resource commands in `cmd/cmd_processinstance_test.go`, `cmd/get_test.go`, and `cmd/delete_test.go`

### Implementation for User Story 4

- [x] T036 [US4] Update root tenant flag description and affected command long help in `cmd/root.go`, process-instance command files, process-definition command files, and `cmd/get_resource.go`
- [x] T037 [US4] Update README tenant guidance in `README.md`
- [x] T038 [US4] Run `make docs-content` to regenerate generated CLI docs under `docs/cli/` and `docs/index.md`
- [x] T039 [US4] Run `go test ./docsgen ./cmd -count=1`

**Checkpoint**: All user-facing documentation and command help describe the tenant contract.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Verify the full feature, keep generated artifacts synchronized, and preserve implementation traceability.

- [ ] T040 Update `specs/156-tenant-admin-keys/progress.md` with codebase discoveries, validation results, and any follow-up risks
- [ ] T041 Run `go test ./cmd ./c8volt/process ./c8volt/resource ./internal/services/processinstance ./internal/services/processdefinition ./internal/services/resource -count=1`
- [ ] T042 Run `make test`
- [ ] T043 Verify generated docs and working tree status with `git status --short`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (P1)**: Depends on Foundational. This is the MVP.
- **User Story 2 (P2)**: Depends on Foundational and can proceed after direct process-instance fixtures are ready.
- **User Story 3 (P3)**: Depends on Foundational and can proceed after direct input semantics are confirmed.
- **User Story 4 (P4)**: Depends on behavior decisions from US1-US3 so help/docs describe actual behavior.
- **Polish**: Depends on selected user stories being complete.

### User Story Dependencies

- **US1**: Independent after Foundational; proves c8volt-produced scope remains tenant-bounded.
- **US2**: Independent after Foundational; proves explicit process-instance keys remain backend-authorized admin input.
- **US3**: Independent after Foundational; extends explicit admin-input semantics to definitions, resources, and stdin.
- **US4**: Cross-cutting documentation story after behavior is known.

### Within Each User Story

- Write or update tests first and confirm they fail for any current mismatched behavior.
- Preserve existing safety checks before removing any tenant mismatch block.
- Update command/facade/service behavior before help/docs.
- Run targeted validation at each story checkpoint.

## Parallel Opportunities

- T002, T003, and T004 can run in parallel.
- T006 and T007 can run in parallel after T005 identifies current audit points.
- US1, US2, and US3 test-writing tasks can run in parallel after Foundational if workers coordinate shared fixtures.
- US4 documentation and command contract tests can run in parallel once final wording is chosen.

## Parallel Example: User Story 2

```text
Task: "T016 [P] [US2] Add get pi --key selected-tenant mismatch test in cmd/get_processinstance_test.go"
Task: "T017 [P] [US2] Add walk pi --key selected-tenant mismatch test in cmd/walk_test.go"
Task: "T018 [P] [US2] Add expect pi --key selected-tenant mismatch test in cmd/expect_test.go"
Task: "T019 [P] [US2] Add cancel pi --key --dry-run selected-tenant mismatch test in cmd/cancel_test.go"
Task: "T020 [P] [US2] Add delete pi --key --dry-run selected-tenant mismatch test in cmd/delete_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational tasks.
2. Complete US1 for tenant-scoped discovery/search-derived behavior.
3. Run US1 targeted command and service tests.
4. Stop and validate before expanding to explicit direct-key behavior.

### Incremental Delivery

1. Foundation: audit tenant checks and prepare reusable mismatch fixtures.
2. US1: discovery/search-derived tenant scope.
3. US2: explicit process-instance admin input.
4. US3: explicit process-definition, resource, and stdin admin input.
5. US4: operator-facing help and docs.
6. Polish: broad validation and progress notes.

## Notes

- Commit subjects must use Conventional Commits format and append `#156` as the final token.
- Do not stage or commit unless the Ralph workflow explicitly reaches its commit step and validation passes.
- Generated CLI docs must be regenerated with `make docs-content`; do not hand-edit generated docs under `docs/cli/`.
- Do not rewrite Camunda 8.7 tenant behavior as part of this feature.
