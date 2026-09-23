# Tasks: User-Task Integration Coverage

**Input**: Design documents from `specs/324-user-task-integration/`
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [integration contract](contracts/integration.md), and [quickstart.md](quickstart.md).
**Tests**: This feature is integration coverage. Each task delivers executable assertions and their necessary setup/validation together; no product implementation is requested.
**Organization**: Four tasks in three story phases. Reuse the existing harness and models. Reading, validation, documentation and completion bookkeeping are not separate iteration-sized tasks.

## Format and Constraints

Tasks use `- [ ] Tnnn [USn] description with file paths`. No task is marked `[P]`: shared helper and scenario files make serial work safer. Paths are repository-relative.

Read `AGENTS.md`, `integration/AGENTS.md`, `specs/integration-test-responsibility.md`, `.specify/memory/constitution.md`, and the feature artifacts before implementation. Ralph execution additionally requires `specs/ralph-implementation-rules.md`; surface genuine conflicts rather than silently relaxing requirements.

For every task, reuse existing profile/version gates, command execution, evidence and retained/cleanup behavior. Record relevant commands and actual version outcomes concisely in `specs/324-user-task-integration/quickstart.md`; never claim unavailable or skipped coverage passed. Run gofmt on changed Go files. Reuse still-valid results; no full suite or repeated live runs merely to commit or mark completion.

## Phase 1: Setup (Existing Infrastructure)

No initialization task is needed: the Go integration package, version selectors, subprocess harness, Make targets and embedded models already exist. Verify the active feature is #324 while loading context; do not create new scaffolding, dependencies, runner or BPMN files.

## Phase 2: Foundational (Existing Infrastructure)

No separate foundation work is necessary. The small seed extension belongs to its first consumer in T001, and is then reused by the other stories. Preserve old helper defaults and signatures where wrappers suffice.

## Phase 3: User Story 1 - Trust Local Filtering and Effective Display (Priority: P1, MVP)

**Goal**: Prove real local-variable selection, process-only exclusion, shadowing and independent display on every selected supported minor.

**Independent test**: `TestGetFamilyUserTaskVariables` standalone subtests observe process `incident=99`, task-local `incident=1`, and process-only values before verifying exact selected task keys and display output. On 8.7, assert native search and effective display are unsupported. Do not seed supported-version scenarios there.

### Executable coverage and supporting implementation

- [ ] T001 [US1] Add the standalone `TestGetFamilyUserTaskVariables` entry point and scenario-owned seed/readiness helpers in `integration/cli/get_usertask_variables_test.go`; minimally extend/delegate the existing start helper in `integration/cli/deploy_embed_run_test.go` only where needed. Use the profile's existing `SimpleUserTask` and `SimpleUserTaskWithIncident`, exact deployment selectors, typed `hasIncident=false`, process `incident=99`, `customer="alice"`, structured/Unicode payload, and the authoritative run marker. Preserve these data-model constraints: "Existing integrationProfile; prefix matches 8.8/C88, 8.9/C89, 8.10/C810; 8.7 unsupported"; "Existing fixture/deployment records; matching parent-child family"; "Preserve typed JSON; marker cannot be overridden; no alteration of shared defaults"; "Small scenario-local structure only if useful; all task ownership validated". Establish bounded readiness for task identity, process value, effective local value and scopes before exclusions, retaining last observations on failure. Implement the complete compact six-operator table from `specs/324-user-task-integration/contracts/integration.md`, including serialized numeric membership strings, process-only exclusion, and 8.7 unsupported checks without direct API setup or fixture self-tests. Preserve the evidence constraint "Reuse evidenceRecord and existing family report; retain last readiness observation and assertion failures" with distinct profile/case/attempt labels. Validate focused standalone cases on available selected versions and any changed seed-payload behavior (false/numeric preservation, marker precedence, unchanged defaults); exercise existing `TestGetFamilyUserTask` if shared seed semantics changed. Missing required seeded state fails; record unavailable versions explicitly instead of weakening assertions.
- [ ] T002 [US1] Extend `integration/cli/get_usertask_variables_test.go` with matching-filter identity comparisons with/without `--with-vars`, effective shadowing and process-only values, human `--var-value-limit` shortening, full received JSON values and truncation metadata; use existing `task.UserTasks` and `task.VariableEnrichedUserTasks` shapes and decode exactly one envelope followed by EOF. Preserve the data-model constraint "Effective scopeKey for local values equals element-instance key, process values equal owning PI; no assumption that caller PI remains the variable owner in child". Update the user-task manifest in `integration/cli/all_commands_test.go` with `var`, `var-exists`, `var-like`, `with-vars`, and `var-value-limit`. Validate the newly added display cases and `TestCommandInventory`; report unrelated inventory drift separately. Confirm the baseline entry point matches the existing `Makefile` get selector without changing Make targets. Include evidence and readiness/error behavior from T001; do not repeat the full operator matrix across output modes.

**Checkpoint**: Standalone feature proof is independently runnable through the existing get target. T001 and T002 form one cohesive delivery unit if both fit the implementation iteration.

## Phase 4: User Story 2 - Verify Variables in Called Processes (Priority: P2)

**Goal**: Prove the same scope distinction for a discovered child task without relying on a C89-only scenario engine.

**Independent test**: The called-process subtest deploys the matching child/parent, starts the parent, discovers and verifies the child PI, then checks local match, process-value exclusion and effective display against that child's key.

- [ ] T003 [US2] Add called-process subtests to `integration/cli/get_usertask_variables_test.go`, reusing completed standalone seed/assertion helpers. Select and deploy the profile-matching `SimpleUserTaskWithIncident` followed by `SimpleParentWithIncidentSubprocess`, start the exact parent definition with the same explicit values, discover the child via existing walk/PI commands, and verify actual child definition/version, parent relationship, task ownership and variable readiness before applying `--pi-key` to the child. Assert `incident=1` matches, `incident=99` is excluded and effective display preserves the task identity and expected child-process/local scopes. Treat caller values as copied; do not assume later caller updates propagate. Reuse T002's scope constraint and existing evidence types, avoiding a second full operator/output table or optional extra topologies. Validate the called-process subtest on available selected 8.8/8.9/8.10 profiles, with precise version evidence and failures for wrong child or readiness timeout; reuse prior standalone results unless helper changes invalidate them.

**Checkpoint**: Called-process coverage can run independently using its own setup, after the shared helper implementation exists.

## Phase 5: User Story 3 - Trust Bounded Searches Across Versions (Priority: P2)

**Goal**: Prove complete filtered paging and counts in dirty clusters, using current baseline/volume entry points and honest version outcomes.

**Independent test**: The existing volume get entry point seeds at least three matching tasks, traverses with batch size one, compares unrestricted JSON/keys/total, verifies a limit of two, and checks unmatched output.

- [ ] T004 [US3] Add filtered user-task volume scenarios in `integration/cli/volume_get_test.go`, using T001's helpers from `integration/cli/get_usertask_variables_test.go` and minimally extending `integration/cli/volume_seed_test.go` for explicit seed values while preserving existing count/default callers. Seed at least three `SimpleUserTaskWithIncident` matches with `hasIncident=false` on each supported selected profile; keep C87 on expected-unsupported handling. Scope by exact deployed definition key/state and local predicate, independently discover seeded task keys by owned PI, and snapshot preexisting matches before seeding when the definition scope is reused. Preserve "Keep ownership classes separate; stable expected union; no duplicate seeded keys"; compare the stable preexisting-plus-seeded union without global count assumptions, process-only marker filtering or deletion of unrelated data, and report scope churn explicitly. Verify batch-size-one traversal, duplicate-free keys before set conversion, JSON/keys/total agreement, limit two without ordering assumptions, and empty human/JSON/zero-byte keys output. Append evidence to the existing volume family report, including failed assertions and actual selected-version outcomes. Update `integration/README.md` and `specs/324-user-task-integration/quickstart.md` with focused commands and concise actual validation results; keep product README, generated CLI docs, BPMN files and Make recipes unchanged. Validate the affected volume path and existing caller regressions where shared helpers changed, confirm `integration-cli-get-volume` and `integration-test-all` select it by inspecting their existing selectors, then review the complete diff and run `git diff --check`. Do not rerun all earlier live scenarios unless relevant changes invalidate their evidence.

**Checkpoint**: All three stories are implemented, selected by existing targets, and accompanied by truthful coverage evidence. Unavailable versions remain a stated validation gap, not a passing result.

## Phase 6: Polish & Cross-Cutting Concerns

No standalone polish task. Manifest work belongs to T002; documentation, target inclusion review and final diff checks belong to T004. No additional iteration is needed just to recopy validation records, regenerate unaffected product docs, or recheck unchanged runtime code.

## Dependencies & Execution Order

```text
Existing harness and context
  -> T001: US1 local filtering, seed/readiness, version handling
  -> T002: US1 display and manifest (MVP complete)
       -> T003: US2 called-process scope
       -> T004: US3 volume, documentation and final review
```

T003 and T004 both depend on US1 helpers; the volume scenario does not semantically depend on called-process behavior. Execute T003 then T004 by default because shared files/helpers and live clusters make parallel implementation/execution unattractive. T004's final cross-story review follows all story work.

## Parallel Execution Examples by Story

- **US1**: T002 consumes T001 helpers; execute serially. An independent read-only review of the manifest can overlap implementation, but does not warrant a separate task.
- **US2**: Single cohesive task; no parallel authoring needed. Do not run tests concurrently against shared versioned definitions.
- **US3**: Single cohesive task spanning the volume caller, seed helper and documentation; keep those changes together. Independent evidence review can overlap only when it does not mutate the same cluster/files.

No `[P]` markers or parallel agents are required for this task list.

## Implementation Strategy

1. Deliver US1 (T001-T002) as the MVP, with observable scope proof and aligned inventory.
2. Add US2 (T003), reusing rather than duplicating assertions.
3. Add US3 (T004), including documentation and final review.

Write scenario assertions before filling their missing setup/helper support where practical. Existing product behavior is the subject under test, so a mandatory red product test is not a prerequisite for adding integration coverage. Do not alter production semantics to make a test pass. If live evidence contradicts a contract expectation, preserve and report it for a separate product decision.

Use focused commands from `quickstart.md`, selecting actual configured profiles. Ensure test selectors execute the intended cases; a build-only or zero-test result is not live verification. Run shared clusters serially and separate evidence directories between Go invocations. Full aggregate runs are optional release validation, not another completion gate after focused checks pass.

## Requirement Coverage

| Requirements | Tasks |
| --- | --- |
| FR-001, FR-002: CLI/version/model boundaries | T001, T003, T004 |
| FR-003: local filtering, display, shortening, six operators | T001, T002 |
| FR-004: called-process ownership and scopes | T003 |
| FR-005: paging, limits, totals and empty results | T004 |
| FR-006, FR-007: seed compatibility, owned data, readiness | T001, T003, T004 |
| FR-008: truthful execution evidence | T001-T004 |
| FR-009: existing target selection | T002, T004 |
| FR-010: five manifest flags | T002 |
| FR-011: focused repository-native scope | All tasks; final diff review in T004 |

All SC-001 through SC-006 are addressed by the story checkpoints and their independent tests. No implementation or live validation has been performed by task generation.
