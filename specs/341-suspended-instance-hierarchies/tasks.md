# Tasks: Verify Suspended Instance Cancellation and Deletion

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [contract](contracts/process-instance-mutations.md), and [quickstart.md](quickstart.md).

**Branch**: `341-suspended-instance-hierarchies`

**Generated**: 2026-10-21

**Status**: T001 / US1, T002 / US2, and T003 / US3 completed with isolated regression validation and regenerated help/documentation.

**Tests**: Focused regressions are explicitly required. Each task includes its fixtures, any justified correction, formatting, and targeted checks. A regression may pass against current code: do not change passing behavior merely to manufacture a failing test.

**Authority**: The revised local spec and plan govern this work. Do not restore original GitHub issue requirements or update GitHub. Definition-level discovery/counting remains in #340.

## Format and Paths

Tasks use `- [ ] Tnnn [USn] description with file paths`. Paths are repository-relative. No `[P]` markers: shared fixtures and sequential review make parallel edits unnecessary.

## Work-Unit Rules

These are responsibilities within each task, not separate tasks:

- Read current feature artifacts, AGENTS.md, and `specs/ralph-implementation-rules.md` when running Ralph. Reuse existing owners and helpers; leave each unit compiling and validated.
- Run the focused regression against existing behavior before changing production code. A fix requires a demonstrated suspension-specific failure and stays with its regression in the same task. Passing behavior requires no runtime correction.
- Record general defects or fixes requiring excluded redesign separately. Do not widen #341, encode incorrect expectations, or mark a failing task complete.
- Preserve root cancellation, engine cascade, force restrictions, descendant-first attempts/conflict recovery, opt-out read counts, final confirmation, absence recovery, retries, public results, authorization, and retention. No new lifecycle package, ledger, model, result fields, reconciliation policy, or direct-library deduplication guarantee.
- Use isolated mocks, strict generated-client fakes, test-owned HTTP servers, and isolated command/subprocess fixtures. Inspect targets/helpers and inherited configuration before running. No live discovery, provisioning, deployment, mutation, cleanup, or completion gate.
- Comment new/materially changed tests and helpers; run targeted `gofmt`. Tests mutating command globals must not run in parallel. Capture command stdout/stderr separately.
- If a proven fix changes output/error paths, satisfy all applicable AGENTS.md execution-level logging, envelope, error-class, partial-result, and prompt checks. Otherwise do not add unrelated output hardening.
- Keep completion evidence with the work unit. Task generation does not implement, stage, commit, or run Ralph. Later authorized commits follow repository rules and end their subjects with ` #341`.

## Phase 1: Setup

No initialization is needed. Existing state support, adapters, and test infrastructure are available. Context reading and fixture inspection belong to each task; no setup checkbox is needed.

## Phase 2: Foundational

No new shared infrastructure, dependency, model, or interface is required. Extend existing fixtures only as needed by the current story.

## Phase 3: User Story 1 — Existing Suspended Cancellation (Priority: P1)

**Goal**: Prove suspended cancellation, mixed-tree confirmation, and existing CLI root deduplication without replacing the workflow.

**Independent test**: A suspended root and both mixed-tree combinations use root cancellation without resume; persistent suspended observations cannot satisfy normal confirmation. Overlapping CLI root/child input produces one successful non-retried root submission.

- [x] T001 [US1] Add suspended cancellation and CLI root-deduplication regressions in `internal/services/processinstance/v810/service_test.go`, `internal/services/processinstance/dryrun_test.go`, and `cmd/cancel_processinstance_test.go`; reuse `internal/services/processinstance/waiter/waiter_test.go` coverage, include only a reproducer-backed local correction in the existing responsible owner if necessary, and complete the US1 acceptance and validation below with fixture comments and targeted formatting.

**Acceptance and implementation**:

- Reuse `newCancellationFixture` and strict clients; add only needed per-key request/observation recording and parent fixtures.
- Cover suspended root, active root/suspended child, and suspended root/active child. Assert the cancellation key, one submission without retries, zero resume, and no separate child cancellation.
- Exercise selected-child force/no-force behavior through existing ancestry escalation.
- Cover a root appearing terminal while its discovered child is still suspended; wait for the child's terminal/absent observation. Also cover persistent suspension ending within the configured wait/context budget, preserving accepted-but-unconfirmed failure rather than success or an invented listener diagnosis.
- Cover no-wait submission; retain `CancelPreservesOptOutReadAndSubmissionBoundaries`, including combined no-wait/no-state-check zero reads. Reuse terminal/absent and unknown-state controls.
- Prove deduplication through the real planner plus command dispatch, or an isolated command HTTP fixture exercising both. A stub that simply returns a unique root is insufficient. Do not extend direct-library collection guarantees.
- Reuse or extend cancellation dry-run coverage for suspended scope: no prompt or mutation. Keep existing output and error contracts.

**Targeted checks during implementation**:

```sh
env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/v810 -run 'TestService_CancelAndDeleteProcessInstance/Cancel' -count=1
env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance -run 'DryRunCancelOrDeletePlan' -count=1
env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/waiter -run 'TestSuspendedStateWaits' -count=1
env GOENV=off GOFLAGS= GOWORK=off go test ./cmd -run 'TestCancelProcessInstance' -count=1
```

Put new service subtests under the existing parent with `Cancel` prefixes or select their actual names. Verify all intended tests ran; zero matches are not evidence. If shared/older-version runtime code changes, run corresponding controls in this unit.

**Checkpoint / MVP**: US1 validated independently; US2 and US3 remain required for full feature completion.

## Phase 4: User Story 2 — Existing Suspended Deletion Safeguards (Priority: P1)

**Goal**: Prove refusal, ordering, force recovery, and terminal prerequisites without reordering deletion.

**Independent test**: Non-force suspended CLI scope submits no mutation. Forced deletion records descendant-first attempts, conflict, cancellation recovery, terminal confirmation, and retry. Failure or persistent suspension blocks the dependent retry, including with no-wait.

- [x] T002 [US2] Extend suspended deletion regressions in `internal/services/processinstance/v810/service_test.go` and `cmd/delete_processinstance_test.go`, adding suspended classification coverage in `internal/services/processinstance/dryrun_test.go` only if missing; preserve existing ordering and no-wait prerequisites, include only a demonstrated suspension-specific local correction in the existing responsible owner if necessary, and complete the US2 acceptance and validation below with fixture comments and targeted formatting.

**Acceptance and implementation**:

- Reuse T001's fixture capabilities instead of rebuilding them; retain current terminal and absent-recovery expectations.
- Exercise actual CLI planning/non-force refusal for suspended scope; assert no cancellation/deletion or confirmation. Preserve direct-service conflict conventions rather than requiring zero direct-service delete attempts.
- Record the meaningful order for a small tree: descendant attempt before parent deletion, unfinished conflict before cancellation recovery, and terminal observation before the retry. A delete attempt before cancellation is expected; do not require a whole-tree pre-cancel phase.
- Cover successful normal/no-wait recovery and existing final absence confirmation. Cover both cancellation error and a prerequisite remaining suspended until bounded expiry: no dependent deletion retry and no completed-deletion claim.
- Preserve child force escalation and opt-out behavior. Do not introduce child-absence gates, general unsuccessful-response hardening, or altered absence retry counts.
- Reuse applicable terminal/absent/unknown, dry-run, force/worker, result/error, and output controls; avoid duplicating passing matrices.

**Targeted checks**:

```sh
env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/v810 -run 'TestService_CancelAndDeleteProcessInstance/(Delete|Force)' -count=1
env GOENV=off GOFLAGS= GOWORK=off go test ./cmd -run 'TestDeleteProcessInstance' -count=1
env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'TestService_(CancelAndDeleteProcessInstance|CancelProcessInstance|DeleteProcessInstance)' -count=1
```

Use `Delete`/`Force` prefixes for new service subtests or select their actual names. Run planner checks if extended. After shared fixture edits, run the combined 8.10 cancellation/deletion parent once to detect invalidation of T001; do not rerun unrelated suites.

**Checkpoint**: US2 validated independently; deletion sequence and existing safeguards preserved.

## Phase 5: User Story 3 — Controls and Suspension Help (Priority: P2)

**Goal**: Correct selector help while preserving existing runtime and compatibility contracts.

**Independent test**: Both commands list suspended with Camunda 8.10 applicability; older-version unsupported-state behavior remains unchanged. Dry-run and opt-out evidence from T001/T002 remains valid.

- [x] T003 [US3] Correct state-selector help in `cmd/cancel_processinstance.go` and `cmd/delete_processinstance.go`, add focused assertions in `cmd/command_help_focus_test.go`, reuse relevant controls in `cmd/processinstance_suspended_test.go`, update relevant `README.md` guidance, and regenerate `docs/cli/*` and `docs/index.md` through `make docs-content`; complete the US3 checks below without changing flags, defaults, results, or workflow semantics.

**Acceptance and implementation**:

- List suspended and its 8.10 applicability in both state descriptions. Preserve older-version rejection and existing force/no-wait meanings.
- Assert rendered help or metadata with existing conventions. Reuse state validation/search controls; do not change selector runtime logic.
- Keep README guidance concise: suspended is unfinished, cancellation does not require resume, and force deletion follows existing conflict recovery. Do not promise cancel-all-before-delete.
- Generate references and README-derived homepage from source; do not hand-edit generated output.
- Review the diff for unintended changes to signatures, schemas, defaults, error classes, prompts, authorization, retention, and older adapters. Reuse existing controls rather than creating a broad compatibility framework.

**Targeted checks**:

```sh
env GOENV=off GOFLAGS= GOWORK=off go test ./cmd -run 'TestAllCommandHelpFocusesOnActions|TestSuspended(StateValidationFeedback|SearchStateSelection|UnsupportedCommands)' -count=1
env -u MAKEFLAGS -u MFLAGS -u MAKEFILES GOENV=off GOFLAGS= GOWORK=off make docs-content
git diff --check
```

Include the actual new help-test name if separate from the existing test. Confirm intended cases executed and review generated changes. Prior dry-run/opt-out evidence remains valid unless a later executable change affects it.

**Checkpoint**: All three stories covered; help and docs match existing behavior.

## Phase 6: Polish and Cross-Cutting Completion

No standalone polish task. Formatting, relevant documentation, checks, and evidence are part of T001–T003. Any minimal runtime correction must include its affected documentation and checks in that same unit, not defer correctness to T003.

Review actual diffs against FR-008 before completion. A tests/help-only change does not automatically require `make test`; broaden only for shared runtime impact, unresolved failures, or another concrete reason under the constitution. Real engine/listener/exporter behavior remains unverified by fakes and is not a completion gate.

## Requirements and Outcome Coverage

| Requirement / outcome | Task and evidence |
| --- | --- |
| FR-001–002 / SC-001 | T001: suspended root, both mixed trees, force/root behavior, zero resume or separate child cancellation |
| FR-003 / SC-002 | T001: real planner plus CLI dispatch, one non-retried root submission |
| FR-004 / cancellation part of SC-003 | T001: suspended child remains unconfirmed; bounded failure and existing no-wait |
| FR-005 / deletion part of SC-003, SC-004 | T002: non-force refusal; preserved sequence; terminal prerequisite; no retry after failure |
| FR-006 / compatibility part of SC-005 | T001–T002: existing controls and older adapters; T003: help/validation and no unintended contract diff |
| FR-007 / help part of SC-005 | T003: source help, README, generated reference, version guidance |
| FR-008 | Every task: regression before any minimal fix; excluded redesign remains excluded |

## Dependencies and Execution Order

```text
Existing repository and #339 state support
  -> T001 / US1 cancellation
  -> T002 / US2 deletion
  -> T003 / US3 help and documentation
```

T002 follows T001 because they share fixture files; deletion remains independently testable. T003 follows behavioral review so documentation matches the verified flow. No new framework, #340 implementation, or live validation is a prerequisite.

## Parallel Execution Examples

- **US1**: No task split: planner, service, and dispatch evidence form one cancellation outcome.
- **US2**: Do not edit alongside US1; both use `internal/services/processinstance/v810/service_test.go` and possibly `dryrun_test.go`.
- **US3**: Keep source help, assertions, README, and generated docs together; generation follows source edits.
- Read-only review can be independent, but no extra tasks or delegation are created for hypothetical parallelism.

## Implementation Strategy and Consolidation Review

Deliver T001 as the MVP, T002 using the same fixtures, and T003 as the help/documentation correction. Task generation does not launch implementation or Ralph.

The plan suggests two broad work boundaries. Three tasks retain separate US1/US2 acceptance predicates and comply with Ralph's one-story-per-iteration rule. Shared fixture edits are sequential, not duplicated. Tests, any necessary fix, formatting, and targeted validation remain within each task. There are no standalone setup, model, fixture, verification, bookkeeping, or polish tasks.

T001 and T002 are complete; T003 remains unstarted. Implementation evidence is recorded in progress.md and durable handoff guidance in ralph-memory.md.
