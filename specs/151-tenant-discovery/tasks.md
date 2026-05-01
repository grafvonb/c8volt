# Tasks: Tenant Discovery Command

**Input**: Design documents from `/specs/151-tenant-discovery/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/tenant-command.md

**Tests**: Required by the feature specification for list, single-tenant lookup, filtering, sorting, invalid flag-combination handling, JSON output, unsupported-version behavior, and existing `get` behavior preservation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the tenant slice without changing command behavior yet.

- [x] T001 [P] Review generated tenant client shapes and record any field mismatch in `specs/151-tenant-discovery/research.md`
- [x] T002 [P] Create tenant domain model skeleton in `internal/domain/tenant.go`
- [x] T003 [P] Create public tenant facade package skeleton in `c8volt/tenant/api.go`
- [x] T004 [P] Create internal tenant service package skeleton in `internal/services/tenant/api.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared model, service, facade, and root wiring needed by all tenant command stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T005 Implement tenant sort and literal name-filter helpers in `internal/domain/tenant.go`
- [ ] T006 [P] Add domain helper tests for tenant sorting and literal filtering in `internal/domain/tenant_test.go`
- [ ] T007 Define `internal/services/tenant.API` list and get operations in `internal/services/tenant/api.go`
- [ ] T008 Implement tenant service factory with `v87`, `v88`, and `v89` routing in `internal/services/tenant/factory.go`
- [ ] T009 [P] Add tenant factory version routing tests in `internal/services/tenant/factory_test.go`
- [ ] T010 Implement `v87` unsupported tenant service in `internal/services/tenant/v87/service.go`
- [ ] T011 [P] Add `v87` unsupported service tests in `internal/services/tenant/v87/service_test.go`
- [ ] T012 Define public tenant facade models in `c8volt/tenant/model.go`
- [ ] T013 Implement tenant facade conversion helpers in `c8volt/tenant/convert.go`
- [ ] T014 Implement tenant facade client in `c8volt/tenant/client.go`
- [ ] T015 Wire tenant facade into `c8volt.API` in `c8volt/contract.go`
- [ ] T016 Wire tenant service creation into `c8volt.New` in `c8volt/client.go`
- [ ] T017 [P] Add facade conversion and error-mapping tests in `c8volt/tenant/client_test.go`

**Checkpoint**: Foundation ready; tenant services and facade can be used by command stories.

---

## Phase 3: User Story 1 - List Tenants Compactly (Priority: P1) MVP

**Goal**: `c8volt get tenant` lists visible tenants in compact sorted human-readable output.

**Independent Test**: Run command tests with multiple upstream tenants in arbitrary order and verify compact sorted output with ID, name, and optional description.

### Tests for User Story 1

- [ ] T018 [P] [US1] Add `v88` tenant search service tests in `internal/services/tenant/v88/service_test.go`
- [ ] T019 [P] [US1] Add `v89` tenant search service tests in `internal/services/tenant/v89/service_test.go`
- [ ] T020 [P] [US1] Add command list output tests in `cmd/get_tenant_test.go`
- [ ] T021 [P] [US1] Add tenant list facade tests in `c8volt/tenant/client_test.go`

### Implementation for User Story 1

- [ ] T022 [US1] Implement `v88` generated `SearchTenants` service in `internal/services/tenant/v88/service.go`
- [ ] T023 [US1] Implement `v88` tenant conversion in `internal/services/tenant/v88/convert.go`
- [ ] T024 [US1] Implement `v89` generated `SearchTenants` service in `internal/services/tenant/v89/service.go`
- [ ] T025 [US1] Implement `v89` tenant conversion in `internal/services/tenant/v89/convert.go`
- [ ] T026 [US1] Add tenant list facade method in `c8volt/tenant/client.go`
- [ ] T027 [US1] Add `get tenant` command registration and read-only metadata in `cmd/get_tenant.go`
- [ ] T028 [US1] Add compact tenant list renderer in `cmd/cmd_views_get.go`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Show One Tenant by ID (Priority: P2)

**Goal**: `c8volt get tenant --key <tenant-id>` returns exactly one tenant or the comparable not-found outcome.

**Independent Test**: Run command tests for existing and missing tenant IDs and verify only the selected tenant is rendered.

### Tests for User Story 2

- [ ] T029 [P] [US2] Add `v88` get-by-ID service tests in `internal/services/tenant/v88/service_test.go`
- [ ] T030 [P] [US2] Add `v89` get-by-ID service tests in `internal/services/tenant/v89/service_test.go`
- [ ] T031 [P] [US2] Add keyed tenant command tests in `cmd/get_tenant_test.go`
- [ ] T032 [P] [US2] Add tenant lookup facade tests in `c8volt/tenant/client_test.go`

### Implementation for User Story 2

- [ ] T033 [US2] Implement `v88` generated `GetTenant` service path in `internal/services/tenant/v88/service.go`
- [ ] T034 [US2] Implement `v89` generated `GetTenant` service path in `internal/services/tenant/v89/service.go`
- [ ] T035 [US2] Add tenant lookup facade method in `c8volt/tenant/client.go`
- [ ] T036 [US2] Add `--key` handling and keyed-mode validation in `cmd/get_tenant.go`
- [ ] T037 [US2] Add single-tenant renderer in `cmd/cmd_views_get.go`

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Filter Tenant Lists by Name (Priority: P3)

**Goal**: `c8volt get tenant --filter <text>` applies literal contains filtering to tenant names.

**Independent Test**: Run command and domain tests with matching, non-matching, and pattern-like filter text, then verify no wildcard/glob/regex/query interpretation and that `--key` plus `--filter` is rejected.

### Tests for User Story 3

- [ ] T038 [P] [US3] Add command filter tests for matching and empty results in `cmd/get_tenant_test.go`
- [ ] T039 [P] [US3] Add wildcard/glob/regex/query literal filter tests in `cmd/get_tenant_test.go`
- [ ] T040 [P] [US3] Add `--key` plus `--filter` invalid-combination command test in `cmd/get_tenant_test.go`
- [ ] T041 [P] [US3] Add facade filter tests in `c8volt/tenant/client_test.go`

### Implementation for User Story 3

- [ ] T042 [US3] Add tenant filter field and list filtering path in `c8volt/tenant/model.go`
- [ ] T043 [US3] Apply literal name filtering before final sort in `c8volt/tenant/client.go`
- [ ] T044 [US3] Add `--filter` flag and reject `--key` plus `--filter` in `cmd/get_tenant.go`
- [ ] T045 [US3] Ensure filtered tenant list rendering reuses existing tenant list renderer in `cmd/cmd_views_get.go`

**Checkpoint**: User Stories 1, 2, and 3 are independently functional.

---

## Phase 6: User Story 4 - Structured Output and Version-Aware Support (Priority: P4)

**Goal**: Tenant discovery supports JSON output and unsupported-version behavior consistently across command modes.

**Independent Test**: Run JSON command tests for list and keyed modes, plus unsupported-version tests for `v8.7`.

### Tests for User Story 4

- [ ] T046 [P] [US4] Add JSON list and keyed command tests in `cmd/get_tenant_test.go`
- [ ] T047 [P] [US4] Add `v8.7` unsupported command tests in `cmd/get_tenant_test.go`
- [ ] T048 [P] [US4] Add generated-client sensitive-field exclusion assertions in `c8volt/tenant/client_test.go`
- [ ] T049 [P] [US4] Add existing `get` command preservation smoke test in `cmd/get_test.go`

### Implementation for User Story 4

- [ ] T050 [US4] Ensure tenant list JSON output uses the public tenant model in `cmd/cmd_views_get.go`
- [ ] T051 [US4] Ensure tenant keyed JSON output uses the public tenant model in `cmd/cmd_views_get.go`
- [ ] T052 [US4] Ensure unsupported tenant capability errors map through the existing command handler in `cmd/get_tenant.go`
- [ ] T053 [US4] Add command help examples for list, key, filter, and JSON modes in `cmd/get_tenant.go`

**Checkpoint**: All user stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, formatting, and validation across the completed feature.

- [ ] T054 [P] Run `gofmt` on tenant-related Go files in `cmd/`, `c8volt/tenant/`, `internal/domain/`, and `internal/services/tenant/`
- [ ] T055 Regenerate CLI documentation with `make docs-content`
- [ ] T056 [P] Review README tenant/get command mentions and update `README.md` only if the new command belongs in existing examples
- [ ] T057 Run targeted tenant validation with `go test ./internal/services/tenant/... ./c8volt/tenant ./cmd -run 'Test.*Tenant' -count=1`
- [ ] T058 Run full repository validation with `make test`
- [ ] T059 Confirm `specs/151-tenant-discovery/quickstart.md` scenarios match final command behavior

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational and is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational; can share service/facade types with US1.
- **User Story 3 (Phase 5)**: Depends on Foundational and can be implemented after list mode exists.
- **User Story 4 (Phase 6)**: Depends on the desired list/key/filter modes being present.
- **Polish (Phase 7)**: Depends on all desired user stories.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on other stories after Foundational.
- **User Story 2 (P2)**: No dependency on filtering; uses the same tenant model and service package.
- **User Story 3 (P3)**: Depends on list mode semantics from US1.
- **User Story 4 (P4)**: Depends on each command mode that needs JSON/unsupported coverage.

### Parallel Opportunities

- Setup skeleton tasks T002-T004 can run in parallel.
- Domain tests T006, factory tests T009, unsupported service tests T011, and facade tests T017 can be developed alongside their matching foundational files.
- Service tests for `v88` and `v89` can run in parallel within each story.
- Command tests and facade tests for the same story can run in parallel once shared models exist.
- README review and gofmt can run in parallel after implementation stabilizes.

---

## Parallel Example: User Story 1

```text
Task: "Add v8.8 tenant search service tests in internal/services/tenant/v88/service_test.go"
Task: "Add v8.9 tenant search service tests in internal/services/tenant/v89/service_test.go"
Task: "Add command list output tests in cmd/get_tenant_test.go"
Task: "Add tenant list facade tests in c8volt/tenant/client_test.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 only.
3. Validate with tenant service, facade, and command list tests.
4. Stop if needed with a usable `c8volt get tenant` list command.

### Incremental Delivery

1. Add list mode as MVP.
2. Add keyed lookup.
3. Add literal name filtering.
4. Add JSON/unsupported-version polish and docs.
5. Run targeted validation and `make test`.

### Commit Guidance

Every commit subject for this feature must use Conventional Commits and end with `#151`, for example `feat(tenant): add tenant discovery service #151`.
