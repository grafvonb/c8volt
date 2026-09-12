# Research: Terminal Cancellation Confirmation

## Decision 1 — Broaden only cancellation-specific wait requests

**Decision**: Update the desired-state literals at the two cancellation-related wait sites in each versioned process-instance service to include `COMPLETED`, `CANCELED`, `TERMINATED`, and `ABSENT`. Keep the existing waiter functions and their matching rules unchanged.

**Evidence**: `internal/services/processinstance/v87/service.go` (lines 411, 493), `v88/service.go` (538, 616), `v89/service.go` (526, 593), and `v810/service.go` (545, 614) each contain a two-state list. The first site confirms the cancellation family; the second confirms cancellation before retrying forced deletion. `internal/domain/state.go:40` already defines all four states as terminal. Line references describe the planning baseline.

**Rationale**: Explicit desired states at the operation boundary prevent a cleanup-specific rule from weakening `expect process-instance --state canceled`. Eight small literal changes preserve version ownership and avoid new dependencies or abstractions.

**Alternatives considered**: Changing `statesEquivalent` globally would incorrectly equate completion/absence with cancellation. A shared coordinator, generic terminal-wait mode, or helper for eight literals is unnecessary for this bounded correction.

## Decision 2 — Make existing terminal-root no-ops successful to callers

**Decision**: Set `Ok: true` on the existing terminal-state cancellation no-op response in all four adapters, retaining status code 200 and existing status text. In the cancellation precheck only, recognize `errors.Is(err, d.ErrNotFound)` as local `StateAbsent` and take that same no-op path. Propagate every other read error.

**Evidence**: Each `CancelProcessInstance` precheck returns read errors before `IsTerminal()` and the terminal response omits `Ok`. `internal/services/processinstance/bulk.go` propagates `resp.Ok` into reports; `internal/services/processdefinition/delete.go` counts false reports as failures. In 8.7 the empty-result mapping is in `internal/services/common/response.go`; in newer adapters the HTTP not-found mapping is in `internal/services/httpc/httpmap.go`.

**Rationale**: A nil error and HTTP 200 alone do not satisfy FR-004 through bulk cleanup. Missing-root acceptance must be local to cancellation so ordinary reads continue reporting not-found. This corrects existing outcome values without changing the public model.

**Alternatives considered**: Leaving `Ok` false preserves the bug. Changing getters to always return absence would alter unrelated public behavior. Treating arbitrary errors or text matches as absence would hide failures.

## Decision 3 — Preserve lifecycle ordering and opt-outs exactly

**Decision**: Keep the `NoStateCheck` guard, `NoWait` guard around cancellation family confirmation, mutation retries, family discovery, and later deletion absence verification unchanged. Preserve the existing unconditional cancellation-state wait in forced deletion, including when `NoWait` is set.

**Evidence**: Each adapter discovers the family after submitting cancellation. `DeleteProcessInstance` has a cancellation confirmation inside its wrong-state/force recovery path distinct from its final absence wait. `internal/services/processinstance/waiter/waiter.go` already interprets disappearance during a wait and keeps cancelled/terminated equivalence separate.

**Rationale**: The issue explicitly excludes flag and retry redesign. A family-discovery failure before confirmation remains an error; accepting absence during confirmation does not authorize bypassing discovery failures or later cleanup verification.

**Alternatives considered**: Moving discovery earlier, freezing new search plans, skipping recovery waits, or suppressing submission errors would expand scope.

## Decision 4 — Prove the correction at adapter and cleanup boundaries

**Decision**: Extend existing versioned service tests using their strict generated-client doubles and short configured waits. Add strict explicit-expectation regression coverage, plus one focused process-definition cleanup test that delegates cancellation through a real versioned service with controlled backend responses.

**Evidence**: The versioned `service_test.go` files provide `newTestService`, strict clients, and cancellation/deletion tests. The v88 cancellation test currently bypasses both state checks and waiting. `internal/services/processdefinition/delete_test.go` already exercises stage sequencing and failure propagation but its canned successful cancellation callback cannot reproduce the bug. `cmd/expect_test.go` includes a strict state-mismatch test.

**Rationale**: A completed descendant must be observed by real cancellation confirmation; returning a fabricated successful cancel response would not detect the original regression. Use one supported history-deletion adapter (8.8) for the focused cleanup integration; all four adapters get service-level coverage.

**Alternatives considered**: Only testing a state predicate misses wiring and missing `Ok`; four full cleanup suites add unnecessary duplication. Live clusters are optional supplementary verification, not required for deterministic regression tests.

## Decision 5 — Document the observable correction without redesign

**Decision**: Add a concise README explanation and cancellation command long-help clarification about terminal cleanup acceptance versus explicit cancelled-state expectations. Regenerate CLI references with `make docs-content`; preserve runtime rendering, flags, examples, and public result shapes.

**Rationale**: The constitution requires documentation for user-visible behavior corrections. Help explains that successful cleanup confirmation is not proof every instance was cancelled; this is a documentation clarification, not a runtime output redesign.

**Alternatives considered**: Calling the fix internal-only would conceal an observable change and bypass the documentation gate. Hand-editing generated references conflicts with repository guidance.

All planning unknowns are resolved from repository evidence. No new technology choice or external dependency is required.
