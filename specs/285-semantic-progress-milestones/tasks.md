# Tasks: Semantic Progress Milestones for Long-Running Commands

**Input**: Design documents from `/specs/285-semantic-progress-milestones/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the feature specification and constitution. Story test tasks must be written before their implementation tasks and should fail until the story behavior is implemented.

**Ralph Context**: Every Ralph implementation iteration for this feature MUST include `--implementation-context specs/ralph-implementation-rules.md`.

**Organization**: Tasks are grouped by user story so exact live progress, durable evidence, and script-safe compatibility can be implemented and verified as incremental operator-facing slices.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it changes different files and does not depend on another incomplete task in the phase
- **[Story]**: Maps to the user story in [spec.md](./spec.md)
- Every task names concrete repository paths

## Phase 1: Setup (Shared Implementation Context)

**Purpose**: Establish persistent implementation tracking for the locked #285 feature before code changes.

- [x] T001 Create the #285 implementation log with branch, artifact links, validation commands, and the required Ralph context in `specs/285-semantic-progress-milestones/progress.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the wording-free completion contract and the smallest command reporter scaffold used by every story.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T002 [P] Add failing completion-kind, disposition, identity, failure-detail, and nil-versus-zero affected-count mapping tests in `internal/domain/ops_progress_test.go`, `c8volt/foptions/options_test.go`, and `c8volt/ops/model_test.go`
- [x] T003 Implement the canonical completion fact and mechanical callback propagation in `internal/domain/ops_progress.go`, `internal/services/calloption.go`, `c8volt/foptions/options.go`, `c8volt/ops/progress_model.go`, and `c8volt/ops/convert.go`
- [x] T004 [P] Add failing reporter-construction, output-policy, aggregate-invariant, and idempotent-close tests in a focused `cmd/ops_semantic_progress_test.go`
- [x] T005 Implement the focused reporter scaffold, family vocabulary, and mode-policy inputs in `cmd/ops_semantic_progress.go`, `cmd/ops_progress_mode.go`, and `cmd/ops_progress_render.go`
- [x] T006 Run the foundational domain, facade, and command tests with `-race` and record the results in `specs/285-semantic-progress-milestones/progress.md`

**Checkpoint**: Completion facts cross domain/facade boundaries without wording, and a command-owned reporter can safely accept facts without changing user-visible output yet.

---

## Phase 3: User Story 1 - Understand Live Work Without Spinner Churn (Priority: P1) MVP

**Goal**: Eligible workflows keep one workflow-priority activity whose exact completed, failed, and fully trustworthy affected counters update after every real completion.

**Independent Test**: Run a frozen multi-item operation with concurrent out-of-order completions and nested HTTP/wait/batch activity; verify the workflow activity remains selected and monotonic counters match every completion.

### Tests for User Story 1

- [x] T007 [P] [US1] Add concurrent out-of-order aggregate, affected-coverage invalidation, and workflow-priority activity tests in `cmd/ops_semantic_progress_test.go` and `toolx/logging/activity_test.go`
- [x] T008 [P] [US1] Add process-instance create/cancel/delete completion-fact tests covering success, failure, fail-fast unscheduled work, and affected-count availability in `internal/services/processinstance/bulk_test.go`
- [ ] T009 [P] [US1] Add direct-key, stdin-key, and search-selected cancel/delete live-activity tests in `cmd/processinstance_mutation_progress_test.go`, `cmd/cancel_processinstance_selector_test.go`, and `cmd/delete_processinstance_selector_test.go`
- [ ] T010 [P] [US1] Add basic and all-process-definition delete completion tests, including the serial first capability probe and force cleanup, in `internal/services/processdefinition/delete_test.go` and `internal/services/ops/all_process_definitions_purge_test.go`
- [ ] T011 [P] [US1] Add per-definition deployment visibility and no-wait acceptance progress tests in a new `internal/services/resource/payload/payload_test.go` and in `internal/services/resource/v87/service_test.go`, `internal/services/resource/v88/service_test.go`, `internal/services/resource/v89/service_test.go`, and `internal/services/resource/v810/service_test.go`
- [ ] T012 [P] [US1] Add retention, orphan, and incident-selected purge live completion tests in `internal/services/ops/retention_policy_test.go`, `internal/services/ops/orphan_purge_test.go`, and `internal/services/ops/incident_purge_test.go`
- [ ] T013 [P] [US1] Add real-time repair and smoke-test stage completion tests in `internal/services/ops/repair_test.go` and `internal/services/ops/smoke_test_test.go`
- [ ] T014 [P] [US1] Add justified secondary-workflow tests for bulk starts, slow analysis, and multi-key expect while pinning transient-only exclusions for plain search/watch/walk in `cmd/run_test.go`, `cmd/ops_analyse_slow_process_instances_progress_test.go`, `cmd/expect_test.go`, and `internal/services/processinstance/waiter/waiter_test.go`

### Implementation for User Story 1

- [ ] T015 [US1] Implement mutex-protected completion ingestion, monotonic completed/failed/affected aggregation, and explicit workflow activity ownership in `cmd/ops_semantic_progress.go`
- [x] T016 [US1] Emit exactly one structured completion fact from each executed process-instance create/cancel/delete worker and disable legacy timer progress when the callback is installed in `internal/services/processinstance/bulk.go`
- [ ] T017 [US1] Route direct, stdin, and search cancel/delete mutations through the same reporter without changing planning, confirmation, result ordering, or final summaries in `cmd/processinstance_mutation_progress.go`, `cmd/cancel_processinstance.go`, `cmd/cancel_processinstance_selector.go`, `cmd/delete_processinstance.go`, and `cmd/delete_processinstance_selector.go`
- [ ] T018 [P] [US1] Emit completion facts for every basic process-definition deletion, including the serial first probe and concurrent remainder, in `internal/services/processdefinition/delete.go`
- [ ] T019 [US1] Start fresh confirmed deletion reporters for basic and all-process-definition commands while leaving discovery pages separate in `cmd/delete_processdefinition.go`, `cmd/ops_purge_all_processdefinitions.go`, and `internal/services/ops/all_process_definitions_purge.go`
- [ ] T020 [P] [US1] Track each returned process definition's first visibility without extra backend requests and emit accepted/confirmed deployment facts in `internal/services/resource/payload/payload.go`, `internal/services/resource/v87/service.go`, `internal/services/resource/v88/service.go`, `internal/services/resource/v89/service.go`, `internal/services/resource/v810/service.go`, `cmd/deploy_processdefinition.go`, and `cmd/embed_deploy.go`
- [ ] T021 [P] [US1] Propagate live deletion completions through retention, orphan, and incident-selected purge requests in `internal/services/ops/retention_policy.go`, `internal/services/ops/orphan_purge.go`, `internal/services/ops/incident_purge.go`, and `cmd/ops_processinstance_purge_progress.go`
- [ ] T022 [US1] Move repair progress emission to worker return points and expose smoke-test deploy/start/walk/cleanup stage facts in `internal/services/ops/repair.go`, `internal/services/ops/repair_progress.go`, `internal/services/ops/smoke_test_service.go`, `cmd/ops_repair_progress.go`, and `cmd/ops_explicit_large_work_progress.go`
- [ ] T023 [US1] Add completion facts and reporter wiring for bulk starts, slow-analysis frozen work, and multi-key expect while retaining the assessed exclusions in `internal/services/processinstance/bulk.go`, `internal/services/processinstance/waiter/waiter.go`, `internal/services/ops/slow_process_analysis.go`, `cmd/run_processinstance.go`, `cmd/ops_analyse_slow_process_instances_progress.go`, and `cmd/expect_processinstance.go`
- [ ] T024 [US1] Run all US1 reporter, activity, process-instance, process-definition, resource, ops, run, analysis, and expect tests with `-race` and record the exact commands/results in `specs/285-semantic-progress-milestones/progress.md`

**Checkpoint**: User Story 1 is complete when all required families expose truthful live semantic completion and one stable workflow activity without durable-line behavior being required yet.

---

## Phase 4: User Story 2 - Retain Durable Evidence During Long Operations (Priority: P2)

**Goal**: Default human runs produce completion-driven 10-second aggregate milestones, immediate failure warnings, and one final flush, while verbose mode produces per-item outcomes instead of aggregate milestone duplication.

**Independent Test**: Drive a fake-clock sequence before and after 10 seconds with successes and failures; verify pacing, warning timing, affected-count omission, verbose replacement, activity clear/restore, and idempotent final flush.

### Tests for User Story 2

- [ ] T025 [P] [US2] Add fake-clock tests for clean sub-10-second silence, the first completion at 10 seconds, rapid-completion suppression, durable activation, immediate failures, and exactly-once final flush in `cmd/ops_semantic_progress_test.go` and `cmd/ops_progress_test.go`
- [ ] T026 [P] [US2] Add verbose identity/outcome replacement, cumulative affected/failed rendering, and quiet warning-severity tests in `cmd/ops_semantic_progress_test.go` and `cmd/ops_progress_test.go`
- [ ] T027 [P] [US2] Add concurrent durable-write clear/redraw and nested workflow-priority arbitration tests in `toolx/logging/activity_test.go` and a new `testx/activitysink/activity_sink_test.go`
- [ ] T028 [P] [US2] Add process-instance command milestone tests for default, verbose, failures, force cleanup, final flush, and no duplicate timer output in `cmd/processinstance_mutation_progress_test.go`, `cmd/cancel_processinstance_test.go`, and `cmd/delete_processinstance_test.go`
- [ ] T029 [P] [US2] Add process-definition deletion, deployment, and all-definition purge milestone tests in `cmd/delete_test.go`, `cmd/deploy_test.go`, and `cmd/ops_purge_all_processdefinitions_test.go`
- [ ] T030 [P] [US2] Add retention, orphan, incident purge, repair, smoke-test, run, analysis, and expect durable-behavior tests in `cmd/ops_execute_retention_policy_test.go`, `cmd/ops_purge_orphan_processinstances_test.go`, `cmd/ops_purge_processinstances_with_incidents_test.go`, `cmd/ops_repair_incident_test.go`, `cmd/ops_repair_processinstance_test.go`, `cmd/ops_execute_smoke_test_test.go`, `cmd/run_test.go`, `cmd/ops_analyse_slow_process_instances_progress_test.go`, and `cmd/expect_test.go`

### Implementation for User Story 2

- [ ] T031 [US2] Replace the 30-second snapshot pacer with a mutex-safe completion-driven 10-second cadence, durable activation, dirty tracking, and idempotent finish in `cmd/ops_progress_milestones.go` and `cmd/ops_semantic_progress.go`
- [ ] T032 [US2] Implement compact aggregate, per-item lifecycle, affected-count, immediate failure, and final-flush rendering on the activity-aware diagnostic path in `cmd/ops_progress_render.go` and `cmd/ops_semantic_progress.go`
- [ ] T033 [US2] Wire reporter finish around every eligible facade call and ensure planning scopes stop before prompts in `cmd/processinstance_mutation_progress.go`, `cmd/delete_processdefinition.go`, `cmd/deploy_processdefinition.go`, `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_processinstance_purge_progress.go`, `cmd/ops_repair_progress.go`, `cmd/ops_explicit_large_work_progress.go`, `cmd/run_processinstance.go`, `cmd/ops_analyse_slow_process_instances_progress.go`, and `cmd/expect_processinstance.go`
- [ ] T034 [US2] Suppress legacy process-instance timer and smoke-test informational progress whenever structured semantic progress is installed while retaining it for non-callback callers in `internal/services/processinstance/bulk.go` and `internal/services/ops/smoke_test_service.go`
- [ ] T035 [US2] Run the US2 fake-clock, activity, and required command-family milestone tests with `-race` and record exact results in `specs/285-semantic-progress-milestones/progress.md`

**Checkpoint**: User Story 2 is complete when long/failing human runs leave compact durable evidence, fast clean runs remain silent, and verbose output contains exactly one line per completion.

---

## Phase 5: User Story 3 - Preserve Command and Automation Contracts (Priority: P3)

**Goal**: Equivalent inputs share lifecycle semantics while JSON, keys-only, quiet, and automation behavior remains script-safe and all mutation/result contracts stay unchanged.

**Independent Test**: Execute equivalent direct, stdin, and search scopes in waited/no-wait and every output mode; compare lifecycle wording, prompt/activity boundaries, results, reports, exit behavior, and stdout parseability.

### Tests for User Story 3

- [ ] T036 [P] [US3] Add completion disposition and conversion tests proving accepted no-wait work is `submitted`, waited work is `confirmed`, failures stay `failed`, and service facts contain no rendered command wording in `internal/domain/ops_progress_test.go`, `c8volt/foptions/options_test.go`, and `c8volt/ops/model_test.go`
- [ ] T037 [P] [US3] Add direct-key, stdin-key, and search parity tests for cancel/delete with waited, no-wait, force, failed, and affected-unknown scopes in `cmd/processinstance_mutation_progress_test.go`, `cmd/cancel_processinstance_selector_test.go`, and `cmd/delete_processinstance_selector_test.go`
- [ ] T038 [P] [US3] Add JSON, keys-only, quiet, automation, prompt-boundary, stdout-parseability, and unchanged report/result tests for process-definition, deployment, purge, repair, and smoke commands in `cmd/delete_test.go`, `cmd/deploy_test.go`, `cmd/ops_purge_all_processdefinitions_test.go`, `cmd/ops_execute_retention_policy_test.go`, `cmd/ops_purge_orphan_processinstances_test.go`, `cmd/ops_purge_processinstances_with_incidents_test.go`, `cmd/ops_repair_incident_test.go`, `cmd/ops_repair_processinstance_test.go`, and `cmd/ops_execute_smoke_test_test.go`

### Implementation for User Story 3

- [ ] T039 [US3] Extend output policy so quiet uses a direct activity-aware stderr path for immediate failures while automation, JSON, and keys-only suppress every human progress line in `cmd/ops_progress_mode.go`, `cmd/ops_progress_render.go`, and `cmd/ops_semantic_progress.go`
- [ ] T040 [US3] Map submitted, confirmed operation-specific, and failed vocabulary per command family without moving wording into services in `cmd/ops_semantic_progress.go`, `cmd/processinstance_mutation_progress.go`, `cmd/ops_repair_progress.go`, and `cmd/ops_explicit_large_work_progress.go`
- [ ] T041 [US3] Normalize direct/stdin/search reporter setup and ensure destructive planning activity stops before confirmation and mutation starts a fresh clock/activity in `cmd/cancel_processinstance.go`, `cmd/cancel_processinstance_selector.go`, `cmd/delete_processinstance.go`, `cmd/delete_processinstance_selector.go`, `cmd/delete_processdefinition.go`, `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_repair_incident.go`, and `cmd/ops_repair_processinstance.go`
- [ ] T042 [US3] Run US3 lifecycle, input-parity, machine-output, quiet, automation, prompt, result, and report regression tests with `-race` and record exact results in `specs/285-semantic-progress-milestones/progress.md`

**Checkpoint**: User Story 3 is complete when progress is truthful and consistent without changing any final result, machine stdout, safety, ordering, report, or exit contract.

---

## Phase 6: Polish & Cross-Cutting Validation

**Purpose**: Finish documentation, generated artifacts, repository-native file cohesion, and broad validation.

- [ ] T043 Inventory declarations in every touched `cmd/*.go` file, move the reporter lifecycle into focused files where required, add mandated comments, and run `gofmt` on all touched Go paths; record the review in `specs/285-semantic-progress-milestones/progress.md`
- [ ] T044 Update operator-facing progress guidance and affected command help/examples in `README.md`, `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`, `cmd/delete_processdefinition.go`, `cmd/deploy_processdefinition.go`, `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_execute_retention_policy.go`, `cmd/ops_purge_orphan_processinstances.go`, `cmd/ops_purge_processinstances_with_incidents.go`, `cmd/ops_repair_incident.go`, `cmd/ops_repair_processinstance.go`, and `cmd/ops_execute_smoketest.go`
- [ ] T045 Regenerate CLI documentation with `make docs-content` and verify generated changes under `docs/cli/` and `docs/index.md` rather than hand-editing them
- [ ] T046 [P] Review the implemented behavior against `specs/285-semantic-progress-milestones/quickstart.md` and `specs/285-semantic-progress-milestones/contracts/semantic-progress-contract.md`, updating those files only if implementation details changed without weakening the specification
- [ ] T047 Run targeted `-race` tests for all changed packages under `cmd/`, `c8volt/foptions/`, `c8volt/ops/`, `internal/domain/`, `internal/services/processinstance/`, `internal/services/processdefinition/`, `internal/services/resource/`, `internal/services/ops/`, and `toolx/logging/`; record commands/results in `specs/285-semantic-progress-milestones/progress.md`
- [ ] T048 Run `make test` from the repository root and record the full-suite result in `specs/285-semantic-progress-milestones/progress.md`
- [ ] T049 Review `git diff --check`, confirm changes are scoped to issue #285, mark completed work in `specs/285-semantic-progress-milestones/tasks.md`, and finalize reusable codebase notes in `specs/285-semantic-progress-milestones/progress.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; creates persistent implementation tracking.
- **Foundational (Phase 2)**: Depends on Setup and blocks all stories.
- **User Story 1 (Phase 3)**: Depends on Foundational; establishes facts, aggregation, and live workflow activity.
- **User Story 2 (Phase 4)**: Depends on US1 completion events and reporter aggregation.
- **User Story 3 (Phase 5)**: Depends on US1 and US2 behavior so compatibility is tested against the finished progress stream.
- **Polish (Phase 6)**: Depends on all desired stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: MVP. It is independently demonstrable through exact live activity and can ship without durable lines.
- **User Story 2 (P2)**: Builds only on US1's structured facts and aggregate; it is independently verified with a fake clock and diagnostic capture.
- **User Story 3 (P3)**: Hardens lifecycle wording and mode/input compatibility after the progress behavior exists; it must complete before release.

### Within Each User Story

- Add failing story tests before implementation.
- Add service facts before command reporter wiring for the same family.
- Start the reporter only after a frozen scope exists and destructive confirmation succeeds.
- Run targeted `-race` validation before marking the story checkpoint complete.
- Do not change result ordering, worker counts, or backend execution to satisfy progress tests.

### Parallel Opportunities

- T002 and T004 can run in parallel because they add tests in different packages.
- US1 service/command test tasks T007-T014 can run in parallel by package before implementation.
- After T015 establishes the reporter, process-instance, process-definition, deployment, and ops implementations T016, T018, T020, and T021 can proceed in parallel; command wiring must coordinate shared `cmd` files.
- US2 test tasks T025-T030 can run in parallel by package.
- US3 test tasks T036-T038 can run in parallel by contract area.
- Documentation review T046 can run beside final package validation after behavior stabilizes.

## Parallel Examples

### User Story 1

```text
Task T008: Add process-instance service completion tests.
Task T010: Add process-definition delete/purge completion tests.
Task T011: Add deployment visibility completion tests.
Task T012: Add ops cleanup completion tests.
Task T013: Add repair and smoke-test completion tests.
```

### User Story 2

```text
Task T025: Add fake-clock pacing/final-flush tests.
Task T028: Add process-instance durable milestone tests.
Task T029: Add process-definition/deployment durable tests.
Task T030: Add ops/run/analysis/expect durable tests.
```

### User Story 3

```text
Task T036: Add lifecycle disposition conversion tests.
Task T037: Add direct/stdin/search parity tests.
Task T038: Add machine-output and prompt-boundary tests.
```

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational tasks.
2. Complete User Story 1 across required command families.
3. Stop and validate exact live activity independently under `-race`.
4. Demonstrate that nested HTTP/wait/batch activity cannot replace the workflow aggregate.

### Incremental Delivery

1. **US1**: Structured completion facts and exact live aggregate activity.
2. **US2**: Completion-driven 10-second milestones, immediate failures, verbose detail, and final flush.
3. **US3**: Lifecycle wording, input parity, quiet/automation/machine safety, and prompt boundaries.
4. **Polish**: Docs, generated CLI content, file cohesion, targeted validation, and `make test`.

### Ralph Iteration Discipline

- Every Ralph launch or iteration must include `--implementation-context specs/ralph-implementation-rules.md`.
- Complete one task or one tightly coupled validation slice per Ralph iteration.
- Read `spec.md`, `plan.md`, `tasks.md`, `research.md`, `data-model.md`, `quickstart.md`, `contracts/semantic-progress-contract.md`, and `progress.md` before implementation.
- Do not stage or commit until the current task's relevant validation passes.
- Commit subjects must use Conventional Commits and end with `#285`.
