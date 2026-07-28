# Tasks: Retry Transient Camunda Read Failures

**Input**: Design documents from `specs/261-retry-camunda-reads/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Test tasks are included because the feature specification defines measurable retry behavior and the c8volt constitution requires automated validation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each behavior slice.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Repository root paths are used.
- Production HTTP retry code belongs under `internal/services/httpc/`.
- Existing mutation retry behavior under `internal/services/retry.go` must be mirrored for style, not replaced.

## Phase 1: Setup (Shared Context)

**Purpose**: Confirm feature scope and nearby repository patterns before changing code.

- [x] T001 Read feature and repository guidance in `specs/261-retry-camunda-reads/spec.md`, `specs/261-retry-camunda-reads/plan.md`, `specs/261-retry-camunda-reads/contracts/http-read-retry-contract.md`, `AGENTS.md`, and `specs/ralph-implementation-rules.md`
- [x] T002 [P] Review current shared HTTP service and transport chain in `internal/services/httpc/service.go` and `internal/services/httpc/round_trippers.go`
- [x] T003 [P] Review existing retry timing and logging style in `internal/services/retry.go` and `internal/services/retry_test.go`
- [x] T004 [P] Review final HTTP response/error-body mapping in `internal/services/httpc/httpmap.go` and `internal/services/common/response.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the shared retry transport surface and test harness needed by all user stories.

**CRITICAL**: No user story implementation should begin until this phase is complete.

- [x] T005 Create `internal/services/httpc/read_retry.go` with a package-local `ReadRetryTransport`, retry policy defaults, delay helper, context-aware sleep helper, and base transport delegation shape
- [x] T006 Create `internal/services/httpc/http_read_retry_test.go` with package-local round trip test doubles and a fast retry policy helper for deterministic tests
- [x] T007 Update transport unwrapping support in `internal/services/httpc/service.go` so activity sink wiring still reaches `LogTransport` after the retry transport is installed

**Checkpoint**: HTTP retry transport surface exists, compiles, and is ready for story-specific behavior.

---

## Phase 3: User Story 1 - Continue After Transient Read Failure (Priority: P1) MVP

**Goal**: Operators running read-heavy commands recover automatically when one safe GET/HEAD read fails transiently and a later attempt succeeds.

**Independent Test**: Simulate a GET/HEAD request that fails once with a transient server-side or transport failure and then succeeds; verify the caller receives the successful response.

### Tests for User Story 1

> Write these tests first and verify they fail before implementation.

- [ ] T008 [US1] Add GET 500-then-200 retry success test in `internal/services/httpc/http_read_retry_test.go`
- [ ] T009 [US1] Add HEAD 503-then-200 retry success test in `internal/services/httpc/http_read_retry_test.go`
- [ ] T010 [US1] Add temporary network error retry success test in `internal/services/httpc/http_read_retry_test.go`
- [ ] T011 [US1] Add compact retry log assertion using existing activity wording in `internal/services/httpc/http_read_retry_test.go`

### Implementation for User Story 1

- [ ] T012 [US1] Implement GET/HEAD retry decisions for transient transport errors and HTTP 429/500/502/503/504 in `internal/services/httpc/read_retry.go`
- [ ] T013 [US1] Implement bounded exponential backoff, jitter, `Retry-After` handling, and context-aware retry sleep in `internal/services/httpc/read_retry.go`
- [ ] T014 [US1] Implement compact rate-limited read retry logging with `httpActivityMessage` labels in `internal/services/httpc/read_retry.go`
- [ ] T015 [US1] Install the retry transport in the shared client path from `internal/services/httpc/service.go`
- [ ] T016 [US1] Run focused US1 validation for `internal/services/httpc/read_retry.go` and `internal/services/httpc/http_read_retry_test.go` with `go test ./internal/services/httpc -run 'ReadRetry|LogTransport' -count=1`

**Checkpoint**: User Story 1 is functional and testable independently.

---

## Phase 4: User Story 2 - Preserve Business Errors (Priority: P2)

**Goal**: Expected semantic outcomes such as not-found, invalid request, permission failure, and conflict remain final and are not hidden by retry loops.

**Independent Test**: Simulate non-transient GET/HEAD responses and verify each response returns immediately with one transport call.

### Tests for User Story 2

> Write these tests first and verify they fail before implementation.

- [ ] T017 [US2] Add 400/401/403/404/409 non-retry tests in `internal/services/httpc/http_read_retry_test.go`
- [ ] T018 [US2] Add final response body preservation test for exhausted retry responses in `internal/services/httpc/http_read_retry_test.go`
- [ ] T019 [US2] Add context cancellation during retry sleep test in `internal/services/httpc/http_read_retry_test.go`

### Implementation for User Story 2

- [ ] T020 [US2] Ensure non-transient GET/HEAD responses return without retry in `internal/services/httpc/read_retry.go`
- [ ] T021 [US2] Preserve the final response status, request, headers, and body after exhausted retries in `internal/services/httpc/read_retry.go`
- [ ] T022 [US2] Ensure canceled contexts stop retry sleep promptly in `internal/services/httpc/read_retry.go`
- [ ] T023 [US2] Run focused US2 validation for `internal/services/httpc/read_retry.go` and `internal/services/httpc/http_read_retry_test.go` with `go test ./internal/services/httpc -run 'ReadRetry' -count=1`

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Keep Mutations Safe (Priority: P3)

**Goal**: The generic read retry layer never replays search requests or state-changing requests, and existing mutation retry behavior remains unchanged.

**Independent Test**: Simulate transient failures for POST search and mutation methods and verify the generic read retry layer performs one call only.

### Tests for User Story 3

> Write these tests first and verify they fail before implementation.

- [ ] T024 [US3] Add POST search non-retry test in `internal/services/httpc/http_read_retry_test.go`
- [ ] T025 [US3] Add DELETE/PATCH/PUT/non-search POST non-retry tests in `internal/services/httpc/http_read_retry_test.go`
- [ ] T026 [P] [US3] Add or confirm mutation retry regression coverage in `internal/services/retry_test.go`

### Implementation for User Story 3

- [ ] T027 [US3] Enforce the GET/HEAD-only method gate in `internal/services/httpc/read_retry.go`
- [ ] T028 [US3] Verify existing mutation retry behavior remains owned by `internal/services/retry.go` without adding generic mutation replay in `internal/services/httpc/read_retry.go`
- [ ] T029 [US3] Run focused US3 validation for `internal/services/httpc/read_retry.go`, `internal/services/httpc/http_read_retry_test.go`, `internal/services/retry.go`, and `internal/services/retry_test.go` with `go test ./internal/services/httpc ./internal/services -run 'ReadRetry|RetryCamundaMutation' -count=1`

**Checkpoint**: All user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish validation, formatting, and documentation alignment.

- [ ] T030 [P] Add a short compact transient read retry troubleshooting note in `README.md`
- [ ] T031 Run `gofmt` on `internal/services/httpc/read_retry.go`, `internal/services/httpc/http_read_retry_test.go`, and any touched Go files
- [ ] T032 Run package validation for `internal/services/httpc/read_retry.go` and `internal/services/httpc/http_read_retry_test.go` with `go test ./internal/services/httpc -count=1`
- [ ] T033 Run mutation retry regression validation for `internal/services/retry.go` and `internal/services/retry_test.go` with `go test ./internal/services -run 'RetryCamundaMutation' -count=1`
- [ ] T034 Run customer-path regression validation for process-instance orphan behavior in `cmd/` and `internal/services/processinstance/` with `go test ./cmd ./internal/services/processinstance/... -run 'Orphan|Retry' -count=1`
- [ ] T035 Run full repository validation for all touched paths with `make test`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational; recommended MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational and can start after the retry transport surface exists, but should be validated after US1 to preserve recovery behavior.
- **User Story 3 (Phase 5)**: Depends on Foundational and can start after the method-gating surface exists, but should be validated after US1 to ensure safety does not remove recovery behavior.
- **Polish (Phase 6)**: Depends on all implemented user stories.

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational; no dependency on US2 or US3.
- **User Story 2 (P2)**: Can start after Foundational; validates semantic non-retry behavior against the shared retry transport.
- **User Story 3 (P3)**: Can start after Foundational; validates non-GET/HEAD and mutation safety against the shared retry transport.

### Within Each User Story

- Tests must be written and observed failing before implementation.
- Retry policy and helpers before shared service wiring.
- Method/status decisions before body-preservation and logging refinements.
- Focused package validation before broader command/service validation.

### Parallel Opportunities

- T002, T003, and T004 can run in parallel during setup.
- T026 can run in parallel with T024/T025 because it touches `internal/services/retry_test.go`, not `internal/services/httpc/http_read_retry_test.go`.
- T030 can run in parallel with final Go validation if retry log wording is already finalized.
- After T005-T007, US2 and US3 tests can be drafted while US1 implementation proceeds, but final implementation should keep one owner for `internal/services/httpc/read_retry.go` to avoid file conflicts.

---

## Parallel Example: User Story 3

```bash
# Launch in parallel because these touch different files:
Task: "Add POST search and mutation non-retry tests in internal/services/httpc/http_read_retry_test.go"
Task: "Add or confirm mutation retry regression coverage in internal/services/retry_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 foundational retry transport surface.
3. Complete Phase 3 User Story 1 tests and implementation.
4. Stop and validate with `go test ./internal/services/httpc -run 'ReadRetry|LogTransport' -count=1`.

### Incremental Delivery

1. Add US1 transient GET/HEAD retry success.
2. Add US2 semantic non-retry and final error preservation.
3. Add US3 search/mutation safety and mutation retry regression.
4. Finish formatting, focused validation, customer-path validation, and `make test`.

### Notes

- Keep retry behavior central in `internal/services/httpc/`; do not add process-instance-specific retry branches.
- Do not retry POST search requests in this issue.
- Do not move or replace existing Camunda mutation retry behavior in `internal/services/retry.go`.
- Keep retry information off stdout so JSON, keys-only, quiet, and automation output contracts remain stable.
