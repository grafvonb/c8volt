---
description: "Implementation tasks for issue #291 force-cleanup progress"
---

# Tasks: Force-Cleanup Progress During Process-Definition Purge

**Input**: Design documents in `specs/291-force-cleanup-progress/`.
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [progress contract](contracts/force-cleanup-progress.md), [quickstart.md](quickstart.md).
**Tests**: Required by spec FR-014/FR-015 and the project constitution. Add regression tests before the associated implementation and demonstrate the intended failure, then make them pass before completing the work unit. Do not commit an intermediate failing test-only task.
**Organization**: Three user-story phases follow setup and shared prerequisites. This is an existing Go repository: no project scaffolding, dependencies, or new runtime configuration are needed.

## Format and Path Conventions

Tasks use `- [ ] TNNN [P?] [USn?] Description`. All paths are relative to the repository root. `[P]` identifies disjoint-file work that can run concurrently within its stated dependency wave; it does not waive prerequisites. User-story labels correspond to the specification priorities.

Implementation must read `AGENTS.md`, `.specify/memory/constitution.md`, and `specs/ralph-implementation-rules.md`. Any Ralph invocation must include `--implementation-context specs/ralph-implementation-rules.md`. Keep the current branch unless the user explicitly requests a switch. Preserve unrelated changes. If committing is later authorized, use Conventional Commits with `#291` as the final subject token and run `make test` first.

## Phase 1: Setup

**Purpose**: Establish the existing behavior and an implementation evidence log.

- [ ] T001 Read the feature artifacts and repository rules, confirm the actual checkout and clean ownership boundaries, and create `specs/291-force-cleanup-progress/progress.md` recording the service callback chain, affected files, and planned validation; preserve the feature selection in `.specify/feature.json` and the active-plan reference in `AGENTS.md`.
- [ ] T002 Run the existing focused APD command, semantic reporter, facade conversion, and process-definition service checks listed in `specs/291-force-cleanup-progress/quickstart.md`; record baseline results and any pre-existing failures in `specs/291-force-cleanup-progress/progress.md` without changing unrelated code.

**Checkpoint**: Baseline behavior and implementation scope are recorded; no backend mutation is needed.

## Phase 2: Foundational — Shared Progress Facts and Reduction

**Purpose**: Supply the stage event and reusable aggregate reduction used by all stories.

- [ ] T003 Add stage-envelope contract tests in `internal/domain/ops_progress_test.go` for phase/resource fields, optional total and planned affected scope, nil versus known zero, and unchanged existing completion serialization, using the field names in `specs/291-force-cleanup-progress/data-model.md`.
- [ ] T004 Add `OpsProgressEventKindStage`, `OpsStageProgress`, and the optional `Stage` envelope payload in `internal/domain/ops_progress.go`; keep existing variants unchanged, represent unavailable counts with nil, and document nonnegative count and payload invariants so T003 passes.
- [ ] T005 [P] Add tests first in `c8volt/ops/model_test.go`, then expose the public stage kind/payload in `c8volt/ops/progress_model.go` and map it mechanically in `c8volt/ops/convert.go`; prove nil/zero preservation and independent optional-count pointers without adding stage fields to final result or report models.
- [ ] T006 [P] Add callback mapping tests first in `c8volt/foptions/options_test.go`, then extend `c8volt/foptions/options.go` with the corresponding public stage payload and conversion; verify nil callbacks, pointer copies, and preservation of all existing completion fields and dispositions.
- [ ] T007 [P] Add focused reduction regressions in `cmd/ops_semantic_progress_test.go`, then extract the completion-to-aggregate update from `opsSemanticProgressReporter.ingestCompletionLocked` into a small pure helper in `cmd/ops_semantic_progress.go`; keep total bounds, failures, unknown affected coverage, and existing single-scope reporter output behavior unchanged.
- [ ] T008 Run domain, ops/foptions conversion, and ordinary semantic-reporter tests from `specs/291-force-cleanup-progress/quickstart.md`; format touched Go files and record passing checks in `specs/291-force-cleanup-progress/progress.md` before using the new event in user-story work.

**Checkpoint**: The additive callback survives both public mapping surfaces, and the existing reporter still passes its regressions.

## Phase 3: User Story 1 — See the Actual Force-Cleanup Stage (Priority: P1, MVP)

**Goal**: Show truthful cancellation, draining, history-deletion, and definition-deletion activity with separate counters, including before the first completion.

**Independent Test**: Run the real command against a fake backend with multiple affected roots and definitions, shared-root expansion, and barriers holding cancellation and draining. Observe activity before releasing each barrier and verify all entered stages and per-completion counters. Repeat non-force and no-cleanup cases.

### Tests for User Story 1

- [ ] T009 [P] [US1] Add service stage-entry tests in `internal/services/processdefinition/delete_test.go` proving entry before each actual force-cleanup operation, no synthetic waiting denominator, skipped empty cleanup, and a once-only definition entry on the ordinary non-force path after the first item's validation; retain the serial-probe and requested-worker assertions.
- [ ] T010 [P] [US1] Extend `TestPurgeAllProcessDefinitionsForceCleanupDeduplicatesProcessInstanceRoots` in `internal/services/ops/all_process_definitions_purge_test.go` to capture the actual full nested sequence and assert unique-root totals, planned affected scope, cancellation/history/definition completion phases, and no double counting of FrozenScope events; add no-cleanup and non-force sequence cases.
- [ ] T011 [P] [US1] Create `cmd/ops_purge_all_processdefinitions_progress_test.go` with coordinator tests for zero-at-entry totals, distinct root/definition counters, planned versus completed affected counts, nil/zero coverage, waiting without counts, unrelated/empty/unentered phase rejection, late historical completions without activity rollback, concurrent completions, and idempotent activity cleanup.
- [ ] T012 [P] [US1] Add a real nested command acceptance test in `cmd/ops_purge_all_processdefinitions_test.go` using the existing CLI configuration/fake-server facilities and `testx/activitysink`; execute Cobra through the actual facade and services, use synchronized barriers to observe first-root and drain activity, and prove the old definition-only callback behavior fails the test. Keep new test names under `TestOpsPurgeAllProcessDefinitions`.

### Implementation for User Story 1

- [ ] T013 [US1] Emit typed stage entries at cancellation, drain, and history boundaries in `internal/services/processdefinition/delete.go:cleanupProcessDefinitionDeletePlanForceScope`, and before the first force/preplanned resource request in `DeleteProcessDefinitionResources`; use existing unique roots and only trustworthy planned affected scope, preserve every request, preview, wait, completion, and worker option, and emit nothing for unentered stages.
- [ ] T014 [US1] Cover the ordinary non-force worker path in `internal/services/processdefinition/delete.go` with a private per-run synchronized once-only entry hook reached immediately before the first validated resource deletion in `deleteProcessDefinition`; use the unique bulk total, preserve per-item validation and scheduling, and do not reroute it through `DeleteProcessDefinitionResources`.
- [ ] T015 [US1] Implement the focused coordinator in `cmd/ops_purge_all_processdefinitions_progress.go` with a dormant constructor, generic real-execution activity, explicit current stage, at most three stage aggregates, separate planned affected count, exact phase routing, and serialized state/output updates using the T007 reducer and existing formatters/policy/printers; preserve established failure, verbose, mode suppression, and single-stage definition milestone behavior while leaving cross-stage refinements to US2.
- [ ] T016 [US1] Wire the coordinator into `cmd/ops_purge_all_processdefinitions.go`, move progress lifecycle dispatch into `cmd/ops_purge_all_processdefinitions_progress.go`, remove eager definition-only startup and the frozen-candidate deletion label, retain the old outer activity only for preview/dry-run calls, and explicitly close real-execution activity before final result/error rendering on every return path so only one workflow activity owner exists.
- [ ] T017 [US1] Run the US1 tests in `cmd/ops_purge_all_processdefinitions_test.go`, `cmd/ops_purge_all_processdefinitions_progress_test.go`, `internal/services/processdefinition/delete_test.go`, and `internal/services/ops/all_process_definitions_purge_test.go`; verify ordinary definition reporter regressions still pass and record the stage-visibility checkpoint in `specs/291-force-cleanup-progress/progress.md`.

**Checkpoint**: Stage visibility is independently demonstrated through real command callbacks, including ordinary non-force deletion. This is the MVP demonstration; complete US2/US3 and final gates before releasing the full issue.

## Phase 4: User Story 2 — Retain Useful Progress and Failure Evidence (Priority: P2)

**Goal**: Preserve workflow-wide 10-second pacing, immediate failures, verbose outcomes, and one accurate final historical-progress record.

**Independent Test**: Drive the contract's multi-stage timeline with a fake clock, failures before the first interval, long idle draining, pending earlier-stage counts, and repeated close. Check exact durable records independently of transient repainting.

### Tests for User Story 2

- [ ] T018 [P] [US2] Add deterministic pacing and closure tests in `cmd/ops_purge_all_processdefinitions_progress_test.go` covering 9.999s/10s boundaries, clock start at first actual stage, no clock reset or flush on transitions, no timer-only drain lines, warning activation without clock reset, per-stage dirty retention, single- and multi-stage final records, clean short-run silence, and repeated close/post-close callbacks.
- [ ] T019 [P] [US2] Add nested default/verbose/debug command tests in `cmd/ops_purge_all_processdefinitions_test.go` proving immediate cancellation/history/definition failures, exact one semantic outcome per item per stage, submitted versus confirmed wording with existing no-wait behavior, and no duplicate aggregate informational lines in verbose/debug output.

### Implementation for User Story 2

- [ ] T020 [US2] Complete workflow-wide durable dispatch in `cmd/ops_purge_all_processdefinitions_progress.go`: one clock anchored to first stage entry, completion-driven 10-second milestones, immediate warnings that do not reset that clock, stage-local dirty clearing, and verbose/debug item outcomes replacing paced lines; reuse `cmd/ops_progress_mode.go`, `cmd/ops_progress_render.go`, and `cmd/ops_progress_milestones.go` without changing other command policies.
- [ ] T021 [US2] Implement the final historical-progress record in `cmd/ops_purge_all_processdefinitions_progress.go` so activated default mode emits at most one line joining only dirty entered mutation-stage aggregates in execution order, retains the ordinary format for a single stage, excludes waiting and already-reported work, never repaints an old activity or claims success, and stops activity exactly once.
- [ ] T022 [US2] Run US2 fake-clock, nested output, and ordinary semantic-reporter regressions with race detection where callbacks are concurrent; record the contract timeline, warning behavior, final-record results, and absence of new timers in `specs/291-force-cleanup-progress/progress.md` using the commands in `specs/291-force-cleanup-progress/quickstart.md`.

**Checkpoint**: Durable evidence covers nested work without stage-transition noise, stale activity, duplicate item outcomes, or lost earlier-stage counts.

## Phase 5: User Story 3 — Preserve Existing Purge Contracts (Priority: P3)

**Goal**: Prove the progress correction leaves scripts, confirmation, mutation scope, backend mechanics, and final results unchanged.

**Independent Test**: Execute equivalent nested cleanup scenarios across supported modes and success/error paths, compare final payloads/reports/exits, and assert the existing backend request sequence and worker controls. Apply mode precedence including automation plus verbose and JSON plus verbose.

### Compatibility Tests and Enforcement for User Story 3

- [ ] T023 [P] [US3] Extend `cmd/ops_purge_all_processdefinitions_test.go` with real nested JSON, automation, quiet, verbose-combination, interactive-confirmation, declined-confirmation, dry-run, auto-confirmed, empty, non-force, error/interruption, and no-wait cases; assert zero added human progress in machine modes, immediate quiet warnings, activity cleanup, and unchanged final stdout/envelopes/reports/exits. Retain existing unsupported keys-only behavior and shared policy tests in `cmd/ops_progress_test.go`; do not introduce APD keys-only support.
- [ ] T024 [P] [US3] Extend `internal/services/processdefinition/delete_test.go` with cancellation failure, drain failure/timeout/interruption, history failure, first definition request-shape rejection, and fail-fast cases; assert no later unentered stage, unchanged read/mutation requests and preview rechecks, original worker settings, serial-probe behavior, and actual operational wait/confirmation semantics.
- [ ] T025 [P] [US3] Extend the APD service-boundary regression in `c8volt/ops/client_test.go` to prove stage forwarding leaves frozen keys, tenant scope, force/dry-run/no-wait/worker/fail-fast options, result/report mapping, and error conversion unchanged; cover nil callbacks and preserve the real-service sequence assertions in `internal/services/ops/all_process_definitions_purge_test.go` without changing those shared test files in this task.
- [ ] T026 [US3] Verify the US3 matrix and repair any exposed progress-specific regression at its owning boundary in `cmd/ops_purge_all_processdefinitions_progress.go`, `cmd/ops_purge_all_processdefinitions.go`, or `c8volt/ops/convert.go`; leave backend operations and final renderers unchanged, rerun the affected tests, and record the compatibility evidence in `specs/291-force-cleanup-progress/progress.md`.

**Checkpoint**: All three stories pass independently focused acceptance tests. Output support, safety decisions, requests, and final outcomes remain compatible.

## Phase 6: Polish and Cross-Cutting Validation

**Purpose**: Align documentation and complete repository gates without unrelated cleanup.

- [ ] T027 [P] Update progress guidance in `README.md` and `docs/ops/purge-all-process-definitions.md` to describe the four actual force-cleanup stages, root versus definition counts, completion-driven pacing, verbose item outcomes, quiet/machine suppression, and discovery-only `--batch-size`.
- [ ] T028 [P] Update APD command metadata/help in `cmd/ops_purge_all_processdefinitions.go` and its help assertions in `cmd/ops_purge_all_processdefinitions_test.go` to match the implemented stage behavior while preserving flags, aliases, capability notes, and safe examples.
- [ ] T029 Run `make docs-content` to regenerate `docs/cli/c8volt_ops_purge_all-process-definitions.md` and `docs/index.md` from command metadata and README; review generated changes and rerun the APD help test rather than hand-editing generated output.
- [ ] T030 Review touched declarations in `cmd/ops_purge_all_processdefinitions.go`, `cmd/ops_purge_all_processdefinitions_progress.go`, `cmd/ops_semantic_progress.go`, and their service/facade counterparts for correct ownership, comments, bounded state, and no new dependency or backend mechanics; apply targeted `gofmt` and record the cohesion review in `specs/291-force-cleanup-progress/progress.md`.
- [ ] T031 Execute the focused conversion/service/command checks and concurrent race checks in `specs/291-force-cleanup-progress/quickstart.md`, confirm the patterns include the new real nested tests, and record exact commands/results in `specs/291-force-cleanup-progress/progress.md`; optional live demonstration is not a prerequisite and requires test-owned fixtures.
- [ ] T032 Run `make test` and `git diff --check`, verify every FR/SC mapping below has passing evidence, and update `specs/291-force-cleanup-progress/progress.md` and `specs/291-force-cleanup-progress/tasks.md` with actual completion status; explicitly record unavailable or failing validation instead of marking the feature complete.

**Checkpoint**: All required validation and documentation gates pass. No commit, branch change, live mutation, or Ralph run is authorized merely by generation of this task list.

## Dependencies and Execution Order

### Phase graph

```text
Setup T001–T002
    ↓
Foundation T003 → T004 → (T005 ∥ T006 ∥ T007) → T008
    ↓
US1 (T009 ∥ T010 ∥ T011 ∥ T012)
    → T013 → T014 → T015 → T016 → T017
    ↓
US2 (T018 ∥ T019) → T020 → T021 → T022
    ↓
US3 (T023 ∥ T024 ∥ T025) → T026
    ↓
Polish (T027 ∥ T028) → T029 → T030 → T031 → T032
```

- T001 precedes T002. T008 requires T003–T007 to pass and blocks story work.
- Within US1, add all regression tests first. T013/T014 share `internal/services/processdefinition/delete.go` and are sequential. T015 relies on T004–T007 and the event contract; T016 requires the service entries and coordinator. T017 requires the whole US1 slice.
- US2 consumes US1's coordinator and callback wiring. T020/T021 share its implementation file and are sequential; T022 requires both test and implementation tasks.
- US3 validates the integrated US1/US2 behavior. Its tests may be authored after the foundational interface is fixed, but the passing checkpoint depends on T022. T026 is the sequential integration/fix gate.
- T029 requires both documentation-source tasks. T030–T032 follow integration and docs regeneration.
- Do not run separate commands that mutate shared package flags or clocks concurrently in one test process. `[P]` describes disjoint file editing, not permission to add `t.Parallel()` to global-state command tests.

### Parallel examples per user story

| Wave | Concurrent work | Why it is safe |
|---|---|---|
| Foundation, after T004 | T005 ops mapping; T006 foptions mapping; T007 shared reducer | Separate source/test files with a fixed domain event contract. |
| US1, after T008 | T009 service tests; T010 ops workflow tests; T011 coordinator tests; T012 real command tests | Four separate test files. Implementation follows the red-test wave. |
| US2, after T017 | T018 coordinator timing tests; T019 nested command output tests | Separate test files; both consume the completed coordinator interface. |
| US3, after T022 | T023 command compatibility; T024 service safety; T025 facade forwarding | Separate packages/files; compare the same agreed event contract. |
| Polish, after T026 | T027 authored documentation; T028 command help/tests | No overlapping files; regenerate docs only after both finish. |

Story completion is intentionally sequential because these are three aspects of the same command lifecycle. Each has its own focused test criteria; claiming fully independent implementation would hide the shared coordinator dependency. The parallel examples are optional scheduling guidance, not a request to launch agents.

## Requirement Coverage

| Requirements / outcomes | Implementation and acceptance tasks |
|---|---|
| FR-001–FR-004; SC-001/SC-002: actual stages and exact per-stage counts | T003–T016, T017 |
| FR-005: truthful planned/observed affected counts | T003–T007, T009–T011, T013, T015 |
| FR-006/FR-007: drain activity and one workflow owner | T009–T017, T023/T024 |
| FR-008; SC-003: completion-driven workflow pacing and final flush | T018–T022 |
| FR-009/FR-010; SC-004: failures and per-item lifecycle outcomes | T019–T022, T023/T024 |
| FR-011; SC-005: machine, quiet, and unsupported-mode contracts | T015/T016, T023–T026 |
| FR-012; SC-006: unchanged mutation, confirmation, reports, exits | T009/T010, T013/T014, T023–T026 |
| FR-013: discovery-only batch size and compact wording | T015, T020/T021, T027–T029 |
| FR-014: real nested command callback sequence | T010/T012, T016/T017, T019, T023 |
| FR-015: failure/edge-case coverage and documentation | T009–T012, T018/T019, T023–T025, T027–T032 |

## Implementation Strategy

### MVP First

Complete setup and foundation, then US1. Its proof is visible stage progression through a real nested command execution, including activity before the first completion and throughout draining. Preserve existing definition-only behavior while introducing the new coordinator. Use this checkpoint for a focused demonstration, not to claim all #291 acceptance criteria are complete.

### Incremental Delivery

1. Establish and test the additive event and reusable reducer without changing existing callers' results.
2. Implement and validate service entries and command stage visibility (US1).
3. Complete exact cross-stage pacing, immediate item evidence, and historical final flushing (US2).
4. Prove and enforce compatibility across modes, failures, safety controls, and public boundaries (US3).
5. Align authored/generated documentation and run the full race-enabled suite before release or an authorized commit.

Pair red-test tasks with their dependent implementation inside a validated work unit. Do not commit known-failing intermediate checkpoints. If an implementation task exposes a conflict with the feature artifacts or mandatory Ralph rules, stop and record the conflict rather than weakening a requirement.

## Task Counts

32 tasks total: 2 setup, 6 foundational, 9 US1, 5 US2, 4 US3, and 6 polish/validation. All start unchecked; generation of this document is not evidence that implementation or tests have completed.
