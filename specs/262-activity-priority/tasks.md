# Tasks: Preserve High-Level Activity

**Input**: Design documents from `/specs/262-activity-priority/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Required by FR-010, FR-011, SC-001, SC-002, SC-003, and the repository constitution. Write or update focused tests before each implementation slice where practical.

**Organization**: Tasks are grouped by user story so the high-level activity fix, fallback activity wording, and script-safe output checks can be delivered and validated independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and does not depend on incomplete tasks in the same phase
- **[Story]**: Maps to the user story from `spec.md`
- Every task names the concrete repository file or directory it affects

---

## Phase 1: Setup (Shared Discovery)

**Purpose**: Confirm the existing activity ownership points and reusable test utilities before changing behavior.

- [x] T001 [P] Review current shared activity writer behavior and nested-scope tests in `toolx/logging/activity.go` and `toolx/logging/activity_test.go`
- [x] T002 [P] Review current HTTP fallback activity behavior and label tests in `internal/services/httpc/round_trippers.go` and `internal/services/httpc/round_trippers_test.go`
- [x] T003 [P] Review reusable activity sink test support in `testx/activitysink/activity_sink.go`
- [x] T004 [P] Review representative command progress emitters in `cmd/get_processinstance_paging.go`, `cmd/get_processinstance_total.go`, `cmd/get_processinstance_orphan.go`, `cmd/processinstance_mutation_progress.go`, and `cmd/ops_analyse_slow_process_instances.go`
- [x] T005 [P] Review service-level bulk and waiter activity call sites in `internal/services/processinstance/bulk.go`, `internal/services/processinstance/variables.go`, `internal/services/processinstance/waiter/waiter.go`, `internal/services/processdefinition/delete.go`, `internal/services/incident/waiter/waiter.go`, `internal/services/job/waiter/waiter.go`, `internal/services/resource/v88/service.go`, and `internal/services/resource/v89/service.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the shared activity hierarchy primitives that every story uses.

**CRITICAL**: No user story implementation can begin until the shared activity API and test sink support exist.

- [ ] T006 Add activity importance values, scoped activity metadata, and priority selection rules in `toolx/logging/activity.go`
- [ ] T007 Add backward-compatible helper APIs for priority-aware start and update operations in `toolx/logging/activity.go`
- [ ] T008 Add unit tests for priority ordering, equal-priority tie breaking, scope stop fallback, idempotent stop behavior, and disabled writer behavior in `toolx/logging/activity_test.go`
- [ ] T009 Update activity sink test support to record priority-aware starts and updates while preserving existing callers in `testx/activitysink/activity_sink.go`
- [ ] T010 Add tests for context helper compatibility and priority-aware activity helper routing in `toolx/logging/activity_test.go`

**Checkpoint**: The shared activity writer can track multiple active scopes and select the visible message by importance without changing existing call sites.

---

## Phase 3: User Story 1 - Keep Workflow Progress Visible (Priority: P1) MVP

**Goal**: Long-running commands keep the command-level workflow message visible while nested HTTP, waiter, poller, and lookup activity scopes run.

**Independent Test**: Simulate representative delete, run, get/search, and ops analysis paths with nested activity and verify the visible or recorded activity remains on workflow progress until that workflow scope ends.

### Tests for User Story 1

- [ ] T011 [P] [US1] Add activity hierarchy contract tests for workflow-over-wait and workflow-over-http examples in `toolx/logging/activity_test.go`
- [ ] T012 [P] [US1] Add process-instance mutation activity tests for delete and cancel progress not being overwritten by nested loads or waits in `cmd/cancel_test.go` and `cmd/delete_test.go`
- [ ] T013 [P] [US1] Add bulk run activity tests for process-instance creation progress not being overwritten by individual create or confirmation activity in `cmd/run_test.go`
- [ ] T014 [P] [US1] Add process-instance get/search progress tests for paging, totals, and orphan discovery activity stability in `cmd/get_processinstance_test.go`, `cmd/cmd_paging_totals_test.go`, and `cmd/get_test.go`
- [ ] T015 [P] [US1] Add ops slow-process activity tests proving runtime element search activity does not replace analysis progress in `cmd/ops_analyse_slow_process_instances_test.go`
- [ ] T016 [P] [US1] Add service waiter nesting tests for wait activity staying below workflow and above HTTP in `internal/services/processinstance/waiter/waiter_test.go`, `internal/services/incident/waiter/waiter_test.go`, and `internal/services/job/waiter/waiter_test.go`

### Implementation for User Story 1

- [ ] T017 [US1] Promote process-instance paging and search activity to workflow importance in `cmd/get_processinstance_paging.go`
- [ ] T018 [US1] Promote process-instance total and orphan discovery activity to workflow importance in `cmd/get_processinstance_total.go` and `cmd/get_processinstance_orphan.go`
- [ ] T019 [US1] Promote process-instance mutation preflight, frozen-scope, and execution progress to workflow importance in `cmd/processinstance_mutation_progress.go`
- [ ] T020 [US1] Promote ops slow-process discovery, analysis, and timeline progress to workflow importance in `cmd/ops_analyse_slow_process_instances.go`
- [ ] T021 [US1] Mark service-level process-instance bulk create, get, cancel, delete, and variable update scopes as batch importance in `internal/services/processinstance/bulk.go` and `internal/services/processinstance/variables.go`
- [ ] T022 [US1] Mark process-definition delete impact and delete execution scopes as batch importance in `internal/services/processdefinition/delete.go`
- [ ] T023 [US1] Mark deployment confirmation scopes as batch importance in `internal/services/resource/v88/service.go` and `internal/services/resource/v89/service.go`
- [ ] T024 [US1] Mark process-instance, incident, and job waiter scopes as wait importance in `internal/services/processinstance/waiter/waiter.go`, `internal/services/incident/waiter/waiter.go`, and `internal/services/job/waiter/waiter.go`
- [ ] T025 [US1] Mark shared poller activity as wait importance in `toolx/poller/poller.go`
- [ ] T026 [US1] Run targeted US1 validation from `specs/262-activity-priority/quickstart.md` with `go test ./toolx/logging ./cmd ./internal/services/processinstance/... ./internal/services/incident/... ./internal/services/job/... ./internal/services/processdefinition/... ./internal/services/resource/... ./toolx/poller -run 'Activity|Progress|Indicator|RunProcessInstance|DeleteProcessInstance|Ops|GetProcessInstance|Wait' -count=1`

**Checkpoint**: User Story 1 is independently functional when representative long-running commands preserve high-level workflow progress through nested work.

---

## Phase 4: User Story 2 - Preserve Useful Fallback Activity (Priority: P2)

**Goal**: Simple Camunda-backed commands still show compact, resource-aware fallback activity when no higher-level activity scope exists.

**Independent Test**: Exercise known Camunda endpoint paths through HTTP activity label tests and verify no known c8volt path falls back to generic API wording.

### Tests for User Story 2

- [ ] T027 [P] [US2] Extend HTTP label table tests for all known labels in `specs/262-activity-priority/contracts/http-activity-labels.md` inside `internal/services/httpc/round_trippers_test.go`
- [ ] T028 [P] [US2] Add HTTP fallback priority tests proving HTTP activity is visible without higher scopes and hidden below workflow or wait scopes in `internal/services/httpc/round_trippers_test.go`
- [ ] T029 [P] [US2] Add representative simple command activity tests for cluster, tenant, resource, incident, job, variable, element-instance, and user-task operations in `cmd/get_test.go`, `cmd/get_tenant_test.go`, `cmd/get_job_test.go`, `cmd/get_incident_test.go`, `cmd/get_element_test.go`, `cmd/get_processinstance_variable_filter_test.go`, and `cmd/get_processinstance_user_tasks_test.go`

### Implementation for User Story 2

- [ ] T030 [US2] Start HTTP transport activity with HTTP fallback importance in `internal/services/httpc/round_trippers.go`
- [ ] T031 [US2] Add resource-aware fallback labels for deployment, license, resource deletion, batch operation, element-instance, variable, user-task, and tenant endpoints in `internal/services/httpc/round_trippers.go`
- [ ] T032 [US2] Normalize version-prefixed and legacy Camunda paths consistently for fallback labels in `internal/services/httpc/round_trippers.go`
- [ ] T033 [US2] Ensure unknown HTTP methods and paths keep generic fallback wording only when no known label matches in `internal/services/httpc/round_trippers.go`
- [ ] T034 [US2] Run targeted US2 validation from `specs/262-activity-priority/quickstart.md` with `go test ./internal/services/httpc ./cmd -run 'HTTPActivity|ActivityLabel|GetCluster|GetTenant|GetJob|GetIncident|GetVariable|GetResource|UserTask|ElementInstance' -count=1`

**Checkpoint**: User Story 2 is independently functional when simple commands have useful fallback activity without reintroducing high-level progress flicker.

---

## Phase 5: User Story 3 - Keep Scripted Output Stable (Priority: P3)

**Goal**: JSON, keys-only, quiet, automation, debug, and verbose modes keep their existing durable output contracts while transient activity behavior improves for interactive human terminals.

**Independent Test**: Run representative command tests in machine-oriented modes and verify stdout and stderr contain no transient spinner or activity text.

### Tests for User Story 3

- [ ] T035 [P] [US3] Add or extend JSON and keys-only cleanliness tests for paged process-instance output in `cmd/cmd_json_assertions_test.go`
- [ ] T036 [P] [US3] Add or extend quiet and automation activity suppression tests for process-instance mutation progress in `cmd/cancel_test.go` and `cmd/delete_test.go`
- [ ] T037 [P] [US3] Add or extend machine-mode fallback suppression tests for simple Camunda-backed commands in `cmd/get_test.go` and `cmd/get_tenant_test.go`
- [ ] T038 [P] [US3] Add writer-level tests proving durable log, warning, prompt, and final output clear transient activity cleanly in `toolx/logging/activity_test.go`

### Implementation for User Story 3

- [ ] T039 [US3] Preserve root command activity writer gating for quiet, automation, JSON, keys-only, and non-interactive modes in `cmd/root.go`
- [ ] T040 [US3] Ensure priority-aware activity helper calls remain no-ops when no activity sink is present in `toolx/logging/activity.go`
- [ ] T041 [US3] Audit debug and verbose paths so endpoint details remain durable debug logs rather than default activity text in `internal/services/httpc/round_trippers.go` and `cmd/root.go`
- [ ] T042 [US3] Run targeted US3 validation from `specs/262-activity-priority/quickstart.md` with `go test ./cmd ./toolx/logging -run 'JSON|KeysOnly|Quiet|Automation|Machine|ActivityWriter' -count=1`

**Checkpoint**: User Story 3 is independently functional when machine-oriented outputs remain deterministic and transient activity remains terminal-only.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validate the complete UX, update documentation only if user-visible documented behavior changed, and leave a concise implementation handoff.

- [ ] T043 [P] Review README and generated CLI docs for documented activity wording and update `README.md` only if visible documented behavior changed
- [ ] T044 [P] Regenerate CLI docs with `make docs-content` only if command help or generated documentation changed in `cmd/`
- [ ] T045 [P] Run documentation search from `specs/262-activity-priority/quickstart.md` across `README.md`, `docs/cli`, and `cmd`
- [ ] T046 Run focused quickstart validation commands from `specs/262-activity-priority/quickstart.md`
- [ ] T047 Run full repository validation from `specs/262-activity-priority/quickstart.md` with `make test`
- [ ] T048 Record final validation notes and any manual smoke gaps in `specs/262-activity-priority/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion; blocks all user stories because shared priority-aware activity primitives must exist first.
- **User Story 1 (Phase 3)**: Depends on Foundational; delivers the MVP fix for high-level activity flicker.
- **User Story 2 (Phase 4)**: Depends on Foundational; can be implemented in parallel with US1 after the shared activity API exists, but final validation should also run after US1 to prove fallback activity stays lower priority.
- **User Story 3 (Phase 5)**: Depends on Foundational; can be implemented in parallel with US1 or US2, but final validation should run after both to cover all new call sites.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational; no dependency on US2 or US3.
- **User Story 2 (P2)**: Starts after Foundational; independent fallback label coverage, with final interaction checks against US1.
- **User Story 3 (P3)**: Starts after Foundational; independent output-mode safety coverage, with final interaction checks against US1 and US2.

### Within Each User Story

- Write or update tests before implementation tasks in that story.
- Implement shared call-site migrations after the activity hierarchy API exists.
- Run the story-specific targeted validation before moving to the next story.
- Keep durable INFO, WARN, prompt, debug, verbose, and final outcome wording stable unless a test proves it must change.

---

## Parallel Opportunities

- Phase 1 review tasks T001-T005 can run in parallel.
- Foundational tests T008 and T010 can be drafted in parallel after T006 and T007 are understood.
- US1 tests T011-T016 can run in parallel because they target different packages and command families.
- US1 implementation tasks T017-T025 can be split by command/service ownership after T006-T010 are complete.
- US2 tests T027-T029 can run in parallel; US2 implementation stays mostly within `internal/services/httpc/round_trippers.go`.
- US3 tests T035-T038 can run in parallel across command and writer packages.
- Polish documentation checks T043-T045 can run in parallel after user-visible behavior stabilizes.

---

## Parallel Example: User Story 1

```bash
Task: "T012 [P] [US1] Add process-instance mutation activity tests for delete and cancel progress not being overwritten by nested loads or waits in cmd/cancel_test.go and cmd/delete_test.go"
Task: "T015 [P] [US1] Add ops slow-process activity tests proving runtime element search activity does not replace analysis progress in cmd/ops_analyse_slow_process_instances_test.go"
Task: "T016 [P] [US1] Add service waiter nesting tests for wait activity staying below workflow and above HTTP in internal/services/processinstance/waiter/waiter_test.go, internal/services/incident/waiter/waiter_test.go, and internal/services/job/waiter/waiter_test.go"
```

## Parallel Example: User Story 2

```bash
Task: "T027 [P] [US2] Extend HTTP label table tests for all known labels in specs/262-activity-priority/contracts/http-activity-labels.md inside internal/services/httpc/round_trippers_test.go"
Task: "T029 [P] [US2] Add representative simple command activity tests for cluster, tenant, resource, incident, job, variable, element-instance, and user-task operations in cmd/get_test.go, cmd/get_tenant_test.go, cmd/get_job_test.go, cmd/get_incident_test.go, and cmd/get_variable_test.go"
```

## Parallel Example: User Story 3

```bash
Task: "T035 [P] [US3] Add or extend JSON and keys-only cleanliness tests for paged process-instance output in cmd/cmd_json_assertions_test.go"
Task: "T038 [P] [US3] Add writer-level tests proving durable log, warning, prompt, and final output clear transient activity cleanly in toolx/logging/activity_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup review.
2. Complete Phase 2 shared activity hierarchy primitives and tests.
3. Complete Phase 3 User Story 1 tests and implementation.
4. Stop and validate with T026 before touching fallback label expansion or machine-mode polish.

### Incremental Delivery

1. Shared hierarchy foundation makes nested activity selectable by user value.
2. US1 gives operators stable high-level progress during long-running work.
3. US2 improves fallback wording for simple commands without competing with US1.
4. US3 proves script-safe output contracts stayed clean.
5. Polish runs quickstart and full repository validation.

### Ralph Execution Guidance

- Prefer one story slice per Ralph iteration after the foundational phase.
- Do not add bespoke per-command progress logic when a central emitter or shared service activity importance can cover the behavior.
- Keep generated client code under `internal/clients/camunda` untouched.
- Use existing `testx` helpers before creating new test infrastructure.
- Commit after each completed logical slice with a Conventional Commit scope and `#262` in the subject when the user asks to commit.
