# Implementation Plan: Accept Terminal States During Cancellation Confirmation

**Branch**: `develop` (unchanged) | **Date**: 2026-09-10 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/294-confirm-terminal-cancellation/spec.md`

**Feature context**: `setup-plan.sh --json` resolved this directory and returned `294-confirm-terminal-cancellation` as its fallback BRANCH identifier. `git branch --show-current` confirms the actual branch is `develop`.

## Summary

Correct false cancellation-confirmation timeouts by accepting completed, cancelled, terminated, and absent instances at the existing cancellation wait sites in all four adapters. Keep explicit cancelled-state expectations strict. Make terminal-root no-ops report success through bulk callers, handling established not-found locally in the cancellation precheck. Preserve cleanup stages, retries, flags, result models, and runtime output. Prove the behavior with adapter tests and one real cancellation path inside a process-definition cleanup regression.

## Technical Context

**Language/Version**: Go 1.26; `go.mod` selects toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2 CLI, generated Camunda/Operate clients, internal service contracts, shared waiter, domain states, and testify 1.11.1. No dependency changes.

**Storage**: Existing remote Camunda state; no local persistence or schema changes.

**Testing**: Go testing with existing strict client doubles, short configured confirmation deadlines, targeted package tests, and `make test` (`go test ./... -race -count=1`).

**Target Platform**: Existing c8volt CLI environments connected to Camunda 8.7, 8.8, 8.9, or 8.10.

**Project Type**: CLI and public Go facade over versioned internal services.

**Performance Goals**: Finish confirmation when all existing scoped members satisfy terminal cleanup acceptance, without waiting for a terminal member to transition again. Preserve polling cadence, timeout settings, and worker limits; no new timing target or work discovery.

**Constraints**: Keep explicit expectations, public models, retries, selection, runtime output, and opt-out semantics intact. No generated-client edits, coordinator, new evidence model, purge/progress/report redesign, or general cleanup refactoring.

**Scale/Scope**: Four adapters, eight wait-state lists, four cancellation prechecks/no-op responses; focused tests and documentation. Existing family scope defines work size.

## Constitution Check

| Gate | Before research | After design |
| --- | --- | --- |
| I. Operational proof | PASS: accept observed terminal outcomes only | PASS: every existing scoped member must qualify; later deletion verification remains required |
| II. Script-safe CLI | PASS: preserve interface contracts | PASS: no new flags/models; explicit cancelled expectation unchanged; success-value correction limited to terminal no-ops |
| III. Tests and validation | PASS: reproducible with controlled states | PASS: four-version matrix, real cancellation inside cleanup, strict expectations, targeted tests, then mandatory race suite before commit/merge |
| IV. Documentation | PASS: observable correction needs explanation | PASS: README and source long-help clarification plus generated references planned; runtime rendering unchanged |
| V. Small compatible changes | PASS: use existing owners | PASS: explicit state lists and local precheck handling; no new production abstraction or dependency |

The gates assess the design. Implementation tests have not run in this planning phase and remain mandatory before implementation is considered complete. No constitution exceptions are required.

## Project Structure

### Documentation (this feature)

```text
specs/294-confirm-terminal-cancellation/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/cancellation-confirmation.md
└── tasks.md                 # Future speckit-tasks output; not created here
```

### Source Code (repository root)

```text
internal/domain/state.go                         # Existing terminal definition; unchanged
internal/services/processinstance/
├── v87/service.go, service_test.go              # Cancellation and force-delete waits, no-ops
├── v88/service.go, service_test.go
├── v89/service.go, service_test.go
├── v810/service.go, service_test.go
├── waiter/waiter.go, waiter_test.go              # Matcher unchanged; regression coverage
└── bulk.go, bulk_test.go                        # Existing report propagation; no-op coverage
internal/services/processdefinition/
├── delete.go                                   # Existing workflow retained
└── delete_test.go                              # Focused cleanup regression
cmd/cancel_processinstance.go                    # Long-help clarification only
cmd/cancel_processinstance_test.go              # Existing output/flag regressions
cmd/expect_test.go                              # Strict expectation regression
README.md                                       # Operator semantics
docs/cli/                                      # Generated via make docs-content
```

**Structure Decision**: Keep cancellation mechanics in the versioned internal services. No new facade or CLI lifecycle logic, no public model changes, and no production refactoring in process-definition cleanup. Existing strict test fixtures own version-specific backend responses.

## Phase 0 — Research Outcome

[research.md](research.md) resolves all technical questions, including eight wait sites, absent prechecks, false no-op `Ok` values, and the forced-delete wait behavior. Explicit four-state literals are sufficient; the shared generic matcher stays unchanged.

## Phase 1 — Design

1. At both cancellation-related wait sites in each adapter, extend desired states with completed and absent. Do not alter the final deletion wait for absence.
2. Under the existing state-check guard, normalize only wrapped `d.ErrNotFound` to local absent state. Preserve all other errors and strict getters. Set `Ok: true` on the terminal no-op, retaining existing status wording/code and no cancellation submission.
3. Preserve cancellation's family discovery and all existing guards, including the unconditional forced-delete recovery wait. Do not reinterpret a family-discovery or submission failure as successful confirmation.
4. Add deterministic adapter coverage for all states and no-op response flags, plus no-wait/no-state-check regressions and strict explicit expectations. Assert requests and observations, not only final error values.
5. Add one process-definition cleanup test with a real 8.8 cancellation service and a completed descendant. Drive the existing orchestration through cancellation, draining, history deletion, and definition deletion/verification; retain later-failure coverage. A stub that simply returns cancellation success is insufficient.
6. Clarify operator documentation and regenerate CLI references. Runtime success/status rendering, flags, and examples retain their contracts; documentation may explain corrected acceptance.

See [data-model.md](data-model.md), [the interface contract](contracts/cancellation-confirmation.md), and [quickstart.md](quickstart.md).

## Validation and Delivery Sequence

Implement the primary cancellation/no-op correction across the four adapters first, then forced-delete confirmation and cleanup regression coverage, then explicit-expectation/CLI compatibility coverage and documentation. Keep commits grouped by purpose if committing is requested; use Conventional Commits with issue #294. Run targeted checks first and `make test` before commit or merge. Run gofmt on touched Go files and regenerate docs from source. Generate tasks separately with `$speckit-tasks`.

## Complexity Tracking

No violations or additional production abstractions. The focused cleanup test may reuse or wrap a real versioned service with existing test seams, but must not create a general testing framework for this fix.
