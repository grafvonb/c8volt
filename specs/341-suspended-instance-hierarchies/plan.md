# Implementation Plan: Verify Suspended Instance Cancellation and Deletion

**Branch**: `341-suspended-instance-hierarchies` | **Date**: 2026-10-21 | **Spec**: [spec.md](spec.md)

**Status**: Narrowed planning only; no implementation or task generation performed.

## Summary

Add focused regression evidence that existing cancellation and deletion handle SUSPENDED correctly. Preserve the current root cancellation, engine cascade, descendant-first deletion, conflict-triggered force recovery, wait semantics, retry helpers, and opt-outs. Correct the small cancel/delete state-help omission. Make a production fix only if a new regression demonstrates a suspension-specific defect, and keep that fix local to its existing owner.

This plan supersedes the earlier broad design. It does not introduce a shared mutation package, execution ledger, outcome model, retry policy, or new deletion schedule. The confirmed ACTIVE-only definition discovery gap belongs to #340.

## Technical Context

**Language/Version**: Existing repository Go 1.26 / go1.26.2 toolchain; no upgrade.

**Primary Dependencies**: Existing Cobra commands, domain states, service adapters, traversal/wait helpers, and generated-client test doubles. No new dependency.

**Storage**: No storage or schema change.

**Testing**: Existing Go unit tests, generated-client mocks, test-owned HTTP fakes, and isolated command execution. Reuse existing fixtures before adding helpers.

**Target Platform**: Existing CLI/library platforms and Camunda adapters 8.8, 8.9, and 8.10. Suspension-specific cases belong to 8.10; older adapters retain existing behavior.

**Project Type**: CLI and public Go library.

**Performance Goals**: No new performance target. Preserve current request, retry, worker, and wait behavior.

**Constraints**: No implementation in this revision. Future implementation must preserve public contracts, opt-out request counts, authorization, output, and deletion order. Isolated validation only.

**Scale/Scope**: Small suspended-root and mixed-tree fixtures; current CLI overlap planning; force-recovery prerequisites; relevant compatibility and help checks. Large-tree pagination, general concurrency hardening, and library overlap redesign are separate concerns.

## Constitution Check

| Principle | Before revision | After narrowed design |
| --- | --- | --- |
| Operational proof | Preserve current terminal/absent confirmation | Focused suspended observations cannot falsely satisfy existing waits |
| CLI-first/script-safe | Preserve existing commands and results | Only help clarification; no public signature, schema, error, or flag-semantic change |
| Proportionate validation | Planning is documentation-only | Targeted isolated checks during implementation; no automatic full-suite or live gate |
| Documentation matches behavior | Identify existing help omission | Correct source metadata and relevant README guidance; regenerate CLI docs when implemented |
| Small compatible changes | Reject unsupported redesign | Reuse current workflow and fixtures; local fixes require a reproducing regression |

All gates pass for this design. This is not a claim that implementation tests have passed.

## Project Structure

### Documentation

```text
specs/341-suspended-instance-hierarchies/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── contracts/process-instance-mutations.md
```

No tasks.md is created in this revision.

### Existing Source Owners

- `internal/services/processinstance/v810/service_test.go`: suspended cancellation and deletion recovery cases, reusing existing fixtures.
- `internal/services/processinstance/dryrun_test.go` and adjacent command tests: existing root planning and dispatch evidence.
- `internal/services/processinstance/waiter/waiter_test.go`: existing suspended/nonterminal coverage; extend only for a missing acceptance case.
- `internal/services/processinstance/v88/service_test.go` and `v89/service_test.go`: existing compatibility controls, not new suspension behavior.
- `cmd/cancel_processinstance.go`, `cmd/delete_processinstance.go`: state-help correction only unless a focused regression proves another relevant defect.
- `cmd/*_test.go`, `README.md`, generated `docs/cli/*`: affected help and command acceptance evidence.

**Structure Decision**: Keep tests and any proven fix with their existing owners. No new package, interface, public model, facade conversion, or general-purpose fixture framework is planned. Source paths identify likely owners, not a mandate to edit every file.

## Design and Execution Boundaries

1. **Characterize existing cancellation**: Extend the existing 8.10 fake-client fixture with SUSPENDED state sequences. Cover suspended root, both mixed parent/child combinations, force escalation, and normal confirmation. Assert no resume or additional child cancellation. Keep engine cascading as an engine responsibility.
2. **Protect CLI deduplication**: Exercise overlapping root/child selection through existing planning and dispatch with successful responses and no retries. Assert one root submission. Do not expand direct library collection semantics or create cross-invocation guarantees.
3. **Characterize existing deletion**: Preserve descendant-first traversal. A deletion attempt may conflict before cancellation recovery; fixtures must expect that existing sequence. Assert terminal confirmation before the retry and no retry after cancellation failure or a wait that remains suspended. Cover no-wait without adding new child-absence gates or a whole-tree pre-cancel phase.
4. **Keep controls stable**: Retain current NoStateCheck/NoWait read and submission boundaries, including combined opt-outs; retain terminal and absent behavior. Do not alter existing absence-recovery request counts, retry policy, final confirmation, or public results to satisfy the discarded design.
5. **Evidence before fixes**: Run the new focused regression against current behavior. If it passes, no runtime change is needed. If it fails because of a suspension-specific defect, include the smallest local correction and its regression in one compiling unit. A general defect or any fix requiring the excluded redesign must be recorded separately, not silently folded into #341.
6. **Help only where missing**: Add suspended and 8.10 applicability to cancel/delete state-selector source help. Update relevant README guidance and regenerate CLI references with `make docs-content`. Keep the documented children-first/conflict-recovery behavior accurate. No implementation or generation occurs during this planning revision.

## Validation Strategy

| Evidence | Requirements | Expected result |
| --- | --- | --- |
| Suspended root and both mixed trees | FR-001, FR-002 | Existing root cancellation and force rules; zero resume/additional child cancellation |
| Overlapping CLI selection | FR-003 | One planned root and one submission without retries |
| Persistently suspended observations / context expiry | FR-004 | Bounded unconfirmed failure, no false completion |
| Non-force refusal / force conflict recovery / no-wait | FR-005 | Existing sequence; terminal wait before retry; no retry after failed prerequisite |
| Existing opt-out, terminal, absent, unknown, dry-run and older-version controls | FR-006 | No semantic or request-boundary changes |
| State help and generated reference | FR-007 | Accurate suspended selector and version guidance |
| Diff review against reproducing regression | FR-008 | No speculative production changes or excluded redesign |

Run the smallest matching tests first; [quickstart.md](quickstart.md) supplies commands and prerequisites. Extend compatibility checks only according to the actual diff. A tests/help-only change does not automatically require `make test`. Broader validation is warranted only by shared runtime changes, unresolved targeted failures, or another concrete risk under the constitution.

Existing logging, output, empty-result, and prompt tests remain compatibility controls. If a proven fix changes those boundaries, all applicable AGENTS.md execution-level validation requirements apply; do not use this narrow plan to waive them. Otherwise do not create an unrelated output-hardening project.

## Phase Deliverables and Work-Unit Guidance

Research from the completed read-only reassessment is consolidated in [research.md](research.md); no new research unknowns or technology choices require additional research agents. [data-model.md](data-model.md) and [contract](contracts/process-instance-mutations.md) explicitly retain existing models and interfaces. [quickstart.md](quickstart.md) defines isolated future validation.

Future tasks should normally group into two cohesive units: (1) cancellation and deletion suspension regression coverage, existing CLI root planning, and any minimal proven fix with targeted checks; (2) state-help/README clarification, generated references, and relevant help checks. Keeping the documentation unit separate is justified by its independently verifiable result; consolidate if the implementation is small. Do not create standalone tasks for reading context, broad refactoring, or live validation. Do not manufacture new test cases where existing coverage already proves a required control.

## Complexity Tracking

No added architectural complexity or constitutional exception. Existing helpers, data models, result types, and retries are sufficient for the narrowed scope unless a focused regression demonstrates otherwise. Such evidence does not automatically authorize the earlier redesign.

## Coverage Limitations

The reassessment inspected repository source, existing tests, official documentation, and upstream Camunda 8.10.4 engine tests. Neither upstream tests nor local runtime tests were executed in this planning revision. Fakes establish c8volt behavior only; real listener and exporter behavior remains unverified. Historical #339 evidence is preserved as historical, and #342 does not become a live implementation gate.
