# Tasks: Suspended Process-Instance State

**Input**: Design documents in `specs/339-suspended-instance-state/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/process-instance-state.md](contracts/process-instance-state.md), [quickstart.md](quickstart.md), and [planning-context.md](planning-context.md).

**Tests**: Required by issue #339. Tests, fixtures, implementation, gofmt and targeted validation are part of each work unit, not separate incomplete commits. Within each unit, first add the relevant regression test and demonstrate its failure where feasible, then implement and pass the checks. Keep any new interface and all required implementations/test doubles together before running compilation-dependent tests.

**Organization**: Six dependency-ordered work units grouped by user story. Paths are repository-relative. No `[P]` markers: the units share state models, adapter files, facade tests or generated documentation. Parallel test execution opportunities below do not authorize concurrent edits or extra Ralph stories.

## Phase 1: Setup (Shared Infrastructure)

No initialization task is needed: the existing Go module, generated clients, command tree and test harness already exist. Context reading, branch checks and progress tracking are responsibilities of every work unit. Read `AGENTS.md`, `.specify/memory/constitution.md` and `specs/ralph-implementation-rules.md` before execution; stop on a genuine conflict. Use branch `339-suspended-instance-state` and issue suffix ` #339` on associated Conventional Commit subjects.

## Phase 2: Foundational (Blocking Prerequisites)

No standalone infrastructure or interface-only task is needed. Story 1 supplies the state vocabulary; Story 2 owns the capability contract together with every adapter and affected test double. No new dependencies, public mutation methods, persistence, client regeneration or server upgrade are required. Keep `internal/clients/camunda/v810/camunda/provenance.json` and generated clients unchanged.

## Phase 3: User Story 1 — Recognize suspended instances accurately (Priority: P1)

**Goal**: Public and internal state representations and existing instance/state reports retain SUSPENDED.

**Independent test**: Feed suspended instance results into lookup and expectation report conversion and assert SUSPENDED survives, parses in supported case variants, and remains nonterminal.

- [x] T001 [US1] Add `StateSuspended` and public valid-state parsing in `internal/domain/state.go` and `c8volt/process/state.go`; preserve state through lookup/expectation report conversion in `c8volt/process/convert.go` and `c8volt/process/client.go` as needed. Enforce the model constraints “Suspension is nonterminal” and “Keep domain/public terminal sets unchanged, including their existing different treatment of ABSENT.” Add or extend `internal/domain/state_test.go`, `c8volt/process/state_test.go`, `c8volt/process/client_test.go` and `cmd/cmd_views_processinstance_test.go` for uppercase/lowercase/mixed-case/whitespace parsing, instance/state/expectation reports, canceled/cancelled/terminated compatibility, unknown backend values and ABSENT behavior. Verify existing compact rendering of the state without adding columns. Document the additive read/report behavior in `README.md`; update affected command metadata and regenerate docs with `make docs-content` if the exposed valid-value text changes in this unit. Run gofmt and targeted domain, facade and command tests; record actual commands/results in `specs/339-suspended-instance-state/progress.md`. Do not add SUSPENDED to creation-observable states until T004.

**Validation**: `go test ./internal/domain -run 'State' -count=1`; `go test ./c8volt/process -run 'State|ProcessInstance|Expectation' -count=1`; targeted tests for touched rendering under `go test ./cmd -run '<actual-render-test-pattern>' -count=1`. Replace placeholder patterns with actual test names and verify tests ran.

**Checkpoint**: State recognition is a testable MVP; unsupported-version filtering, creation visibility and timestamp behavior are not yet claimed complete.

## Phase 4: User Story 2 — Select and wait for suspended instances (Priority: P1)

**Goal**: Existing selectors and expectations support suspension on 8.10 and reject explicit suspension consistently on 8.8/8.9.

**Independent test**: Search a known mixture of states, verify exact request filtering and returned populations, then exercise successful suspension expectations and unsatisfied terminal expectations. On both older versions assert unsupported errors and zero backend requests through public entry points.

- [x] T002 [US2] Implement the explicit suspension capability contract in `internal/services/processinstance/api.go`, `internal/services/processinstance/v88/contract.go`, `internal/services/processinstance/v89/contract.go` and `internal/services/processinstance/v810/contract.go`, with `ValidateProcessInstanceStates(domain.States) error` implemented by each adapter. In the matching `service.go` and `bulk.go` files, reject case-insensitive suspended members on v88/v89 with wrapped `domain.ErrUnsupported` before page searches, single/bulk state waits and single/bulk combined expectations initiate requests or workers; v810 permits the new capability. Guard `internal/services/processinstance/search.go` page traversal and totals before direct-incident branching or empty shortcuts; preserve `internal/services/processinstance/wait.go` delegation and `c8volt/ferrors` classification. Keep `internal/services/processinstance/waiter/waiter.go` free of explicit-capability guards. Verify v810 serialized SUSPENDED equality mapping in `internal/services/processinstance/v810/convert.go` without altering CANCELED-to-TERMINATED normalization. Update every affected test double in this same compiling unit, including `c8volt/process/client_test.go` and service tests. Add regression coverage in `internal/services/processinstance/search_test.go`, versioned `service_test.go`/`bulk_test.go`, waiter tests and facade tests: direct-incident empty/nonempty paths, page/total search, mixed desired states, empty-key waits, state-plus-incident conditions, suspension success, terminal-wait non-success, exact ACTIVE and ALL selection, old unfiltered/default behavior, and zero backend calls for rejected explicit requests. Preserve the model constraint “Unset filters and existing ALL selection keep their existing representation and behavior; do not expand an implicit ALL into a list triggering the older-version guard.” Keep facade logic mechanical; document capability behavior in `README.md`. Run gofmt and relevant service/facade tests before recording results in `specs/339-suspended-instance-state/progress.md`.

- [x] T003 [US2] Complete CLI selection, feedback and completion in `cmd/get_processinstance.go`, `cmd/get_processinstance_validation.go`, `cmd/expect_processinstance.go` and a focused state-completion helper file under `cmd/` if needed. Use command-appropriate public state values; describe the 8.10 restriction and provide completion without configuration, bootstrap or network access. Audit `cmd/ops_analyse_slow_process_instances_validation.go` and other hard-coded process-instance state lists for affected read/expectation paths; retain command-specific restrictions and do not broaden mutation eligibility, purge scopes or definition statistics. Add actual execution-path regression coverage in `cmd/get_processinstance_paging_test.go`, `cmd/expect_test.go`, `cmd/completion_test.go` and a focused `cmd/processinstance_suspended_test.go`: supported case variants, old-version unsupported error/exit classification and zero unsupported requests, state-plus-incident expectations, nonempty results, sparse pages, explicit keys and errors; empty searches in human/JSON/keys-only/quiet/quiet+JSON/quiet+keys and applicable automation/auto-confirm combinations. Capture stdout/stderr separately, decode exactly one envelope and require EOF, assert exact human output and zero-byte empty keys output, verify request counts and no added prompts/mutations; retain real-terminal-stdin coverage for applicable prompt eligibility and paging paths. Keep `cmd/cmd_views_processinstance.go` default rows and output precedence stable. Update `README.md` and command metadata, run `make docs-content`, gofmt, and targeted command tests, and record results in `specs/339-suspended-instance-state/progress.md`.

**Validation**: T002 uses `go test ./internal/services/processinstance/... -run 'State|Expectation|Search|Suspended' -count=1` and `go test ./c8volt/process -run 'State|Expectation|Search|Suspended' -count=1`. T003 uses `go test ./cmd -run 'GetPI|GetProcessInstance|Expect|Completion|Suspended' -count=1`; refine patterns to include every newly added test and run the closest relevant existing regressions. Generated documents are regenerated, never hand-edited.

**Checkpoint**: State filtering and expectations are supported through CLI and facade with consistent version errors. T002 and T003 remain separate because service-wide capability guarantees and CLI stream/completion contracts are independently verifiable outcomes.

## Phase 5: User Story 3 — Confirm creation when execution is suspended (Priority: P2)

**Goal**: The first valid suspended observation confirms creation without implying completion or breaking older-version creation.

**Independent test**: Supply SUSPENDED as the first creation lookup result and require visibility success without another poll or transition; repeat existing creation scenarios on 8.8/8.9.

- [ ] T004 [US3] Extend `ObservableProcessInstanceCreationStates` in `internal/domain/state.go` to include SUSPENDED and preserve its cloned-list behavior. Keep creation's implicit observation calls in `internal/services/processinstance/v88/service.go`, `internal/services/processinstance/v89/service.go` and `internal/services/processinstance/v810/service.go` separate from T002's explicit-request guards; retain the existing generic waiter. Add deterministic tests in `internal/domain/state_test.go`, versioned `service_test.go`, `internal/services/processinstance/waiter/waiter_test.go`, facade tests and `cmd/run_test.go` for first-observed suspension, no premature execution-completion claim, older-version defaults, delayed visibility, lookup errors, bounded timeout/retries, terminal observations and no-wait. Use controlled responses/attempts rather than sleeps. Align creation visibility wording in `cmd/run.go`, `cmd/run_processinstance.go` and `README.md`, regenerate with `make docs-content`, run gofmt and targeted domain/service/facade/run-command tests, and record results in `specs/339-suspended-instance-state/progress.md`.

**Validation**: `go test ./internal/domain -run 'ObservableProcessInstanceCreation|State' -count=1`; `go test ./internal/services/processinstance/... -run 'CreateProcessInstance|State|Expectation' -count=1`; the matching facade/run tests with `-count=1`. Require explicit rejected-state tests from T002 still to pass; implicit creation must not be treated as an unsupported user request.

**Checkpoint**: Suspended visibility and terminal completion remain distinct, with no older-version regression.

## Phase 6: User Story 4 — Read current suspension metadata (Priority: P2)

**Goal**: Preserve optional suspension timestamps and clear stale values when a new response lacks them.

**Independent test**: Convert and render one instance twice, first with a timestamp then with nil/absent metadata; require presence then omission, without modifying default human rows.

- [ ] T005 [US4] Add `SuspendedDate string` beside existing dates in `internal/domain/processinstance.go` and `c8volt/process/model.go`, using `json:"suspendedDate,omitempty"` on the public model; map it through `internal/services/processinstance/v810/convert.go` with `formatTimePtr` and both `fromDomainProcessInstance`/`toDomainProcessInstance` in `c8volt/process/convert.go`. Enforce “Present timestamp formatted UTC RFC3339Nano; absent/null/zero maps to empty and is omitted” and “Every read/search result is a fresh snapshot.” Older adapters leave the field empty. Add tests in `internal/services/processinstance/v810/convert_test.go`, `c8volt/process/convert_test.go`, `c8volt/process/client_test.go` and `cmd/processinstance_suspended_test.go` for non-UTC/fractional input, nil/zero values, two-way mapping, nested/enriched instance payloads, sequential read and search snapshots, present-to-absent structured rendering, and unchanged default human rows/keys output. State must remain SUSPENDED even with no timestamp; derive no cumulative duration or history. Do not introduce process-instance watch or change `cmd/get_processdefinition_watch.go`, which contains no instance metadata path. Update structured-output guidance in `README.md` and any affected command metadata, regenerate CLI docs when metadata changes, run gofmt and relevant converter/facade/command tests, and record results in `specs/339-suspended-instance-state/progress.md`.

**Validation**: `go test ./internal/services/processinstance/v810 -run 'Convert|ProcessInstance|Suspended' -count=1`; `go test ./c8volt/process -run 'Convert|ProcessInstance|Suspended' -count=1`; `go test ./cmd -run 'Suspended|ProcessInstance' -count=1`, narrowed to actual affected tests when appropriate. Decode structured output and verify absence, not a retained old value or serialized zero date.

**Checkpoint**: All four stories have deterministic acceptance coverage and associated documentation.

## Phase 7: Integration & Cross-Cutting Validation

**Purpose**: Prove the integrated behavior against the required server and retain reusable evidence/fixtures for #342.

- [ ] T006 Add `TestGetFamilySuspendedState` in `integration/cli/get_processinstance_suspended_test.go` and narrowly reusable fixture helpers in `integration/cli/suspended_fixture_test.go`, reusing `integration/cli/batch_test.go`, `integration/cli/isolated_seed_test.go` and existing harness/evidence conventions. Validate the actual server patch **8.10.4 before mutations**, independently of the existing minor-line gate; retain the generated-client **8.10.3** provenance unchanged. Exercise public state lookup and CLI suspended/ACTIVE/ALL filtering, state/combined expectations, bounded terminal non-success and timestamp presence followed by absence after resumption, scoped to isolated held instances. Use existing generated suspension/resumption calls only as fixture setup, and resume owned instances before normal cleanup to avoid relying on #341; record cleanup failures distinctly. Ensure the test is discovered by the existing `integration-cli-get` slice (the `TestGetFamily` prefix) and update `integration/README.md` plus `specs/339-suspended-instance-state/quickstart.md` with verified commands. Format the fixture/test files, prove test discovery, run focused live validation on the designated disposable profile, and run `make test` after all implementation units as the shared state/interface changes affect multiple packages and concurrent waits. Record tested revision/dirty status, observed server version, configured minor/profile, exact client provenance, stdout/stderr, exit/results and cleanup outcome in existing integration evidence and `specs/339-suspended-instance-state/validation.md`. If live infrastructure/permissions are unavailable, explicitly record the reason and leave live acceptance pending; do not mark this task complete or count a skip, no-tests run or another patch version as proof. Resolve relevant failures, then review generated docs and unchanged generated-client/provenance diff; do not rerun successful tests solely for prose edits.

**Validation**: Follow [quickstart.md](quickstart.md), first confirming that `go test -tags=integration ./integration/cli -list '^TestGetFamilySuspendedState$'` lists the new test. Use `C8VOLT_IT_RELEASE_VERSION=8.10` for existing harness configuration, not `8.10.4`; the focused scenario additionally verifies the full observed patch. The final race suite is required for this integrated runtime change, not for generation of this task document.

## Dependencies & Execution Order

```text
T001 (US1)
  └─ T002 (US2 service capability)
       └─ T003 (US2 CLI contract)
            └─ T004 (US3 creation visibility)
                 └─ T005 (US4 timestamp)
                      └─ T006 (integration and final validation)
```

T002 depends on the new state vocabulary. T003 depends on service-enforced capability guarantees. T004's visibility regression deliberately runs after explicit guards exist. T005 logically depends only on the instance representation, but follows T004 to avoid shared conversion, adapter, command-test and documentation conflicts. T006 depends on every story and disposable live infrastructure. No circular or forward prerequisite is introduced.

## Parallel Execution Examples by Story

No implementation tasks are marked parallel. Do not split a cohesive unit or create new tasks merely to provide parallel work.

| Story | Safe opportunity within its completed edit set | Ordering constraint |
| --- | --- | --- |
| US1 | Domain and facade test commands may run independently | Finish vocabulary/conversion edits first |
| US2 | Version-package tests may run independently once every interface/stub compiles | T003 follows T002; shared command globals follow existing test serialization |
| US3 | Version-package creation tests may run independently | Keep explicit-guard regressions and observable-state changes in the same verified result |
| US4 | Converter and facade test commands may run independently | Finish both mapping directions and schema fields first |

Live tests sharing fixtures/profile and generated-document writes remain serialized. Parallel test commands are optional execution choices, not extra work units or authorization to spawn agents.

## Requirement Coverage

| Requirements / acceptance | Owning tasks |
| --- | --- |
| FR-001–003: representation, aliases, terminal/unknown semantics | T001; regression preservation in T002/T004 |
| FR-004–008: filtering, expectations, version guards, defaults | T002–T003; live proof T006 |
| FR-009: creation observation | T004 |
| FR-010–011: nullable timestamp and clearing | T005; live proof T006 |
| FR-012: output and empty-result compatibility | T001, T003, T005 |
| FR-013: validation/help/completion/docs | T003, with behavior-specific docs in T001/T004/T005 |
| SC-001–006 | T001–T005 deterministic acceptance; T006 integrated live evidence |
| Exact client baseline and 8.10.4 server target, reusable #342 fixtures | T006; immutable baseline enforced in every unit |

## Implementation Strategy

Deliver US1 first as the recognition MVP, without claiming the full issue complete. Continue by service capability and CLI acceptance, creation visibility, then timestamp refresh. Each unit must compile and pass its relevant tests before being checked off; its changed behavior and documentation ship together. Keep tests, fixtures and required stubs with their owning behavior. Ralph executes one permitted story/work unit at a time, with T006 as the explicit final integration phase.

Retain the six drafted units after consolidation review: T002 cannot split interfaces from adapters/stubs without risking compilation; T002 and T003 have different independently testable service/CLI outcomes; T004 concerns implicit creation rather than explicit state selection; T005 is the separate metadata story; T006 requires live infrastructure and reusable integration evidence. No routine context-reading, branch-check, formatting-only, documentation-only or progress-only tasks are introduced.

There are no new suspend/resume commands, mutation-eligibility changes, purge expansion, process-definition statistics, date filter/sort flags or watch modes. Runtime tests were not executed when this task list was generated.
