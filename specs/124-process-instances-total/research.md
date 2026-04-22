# Research: Add Process-Instance Total-Only Output

## Decision 1: Implement `--total` as a command flag on `get process-instance`, not as a new shared render mode

- **Decision**: Add `--total` only to `get process-instance` and keep the existing shared output modes (`one-line`, `json`, `keys-only`, `tree`) unchanged.
- **Rationale**: The issue is scoped to one command family and one specific count-only behavior. The current contract discovery and render helpers already model shared modes globally, so introducing a new global render mode would broaden impact well beyond this feature.
- **Alternatives considered**:
  - Add a new global render mode such as `total-only`: rejected because it would require command-contract changes and wider command-family auditing for a behavior only this feature needs.
  - Add a new subcommand such as `get pi total`: rejected because it would create a parallel CLI path instead of extending the established Cobra command.

## Decision 2: Keep `--total` on the search/list path and reject it for direct `--key` lookups

- **Decision**: `--total` should only work when `get process-instance` is operating in list/search mode; combining it with `--key` must be rejected.
- **Rationale**: The spec and issue both describe the list command returning the number of found process instances. Direct `--key` lookups are intentionally strict single-resource fetches today, and reinterpreting them as count-only mode would blur that contract and complicate not-found semantics unnecessarily.
- **Alternatives considered**:
  - Allow `--total` with `--key` and print `0` or `1`: rejected because it changes a strict lookup into a count contract and weakens existing error behavior.
  - Allow `--total` with `--key` only when one key is supplied: rejected because it still creates two behavioral contracts for the same direct lookup path.

## Decision 3: Make `--total` mutually exclusive with `--json`, `--keys-only`, and `--with-age`

- **Decision**: Reject `--total` when combined with `--json`, `--keys-only`, or `--with-age`.
- **Rationale**: The clarified specification says `--total` should return only the numeric count. The current shared render model treats `--json` and `--keys-only` as distinct output contracts, and `--with-age` only makes sense for instance details. Explicit validation is clearer and smaller than silently overriding those flags.
- **Alternatives considered**:
  - Let `--json --total` return `{"total":12}`: rejected because the spec’s primary acceptance criteria describe a single numeric value and this would dilute the new flag into yet another response envelope.
  - Let `--total` silently override the other flags: rejected because hidden precedence rules are harder to explain, test, and discover.

## Decision 4: Carry backend-reported totals through the shared page model instead of teaching `cmd/` about version-specific response shapes

- **Decision**: Extend shared domain/public process-instance page models with reported-total metadata and whether that total is an exact count or a lower bound.
- **Rationale**: `internal/services/processinstance/v88/service.go` and `v89/service.go` already see `totalItems` and `hasMoreTotalItems`, while `v87/service.go` has access to the Operate payload total pointer. Today that information is reduced to `OverflowState` before reaching `cmd/`. A small shared metadata seam lets the command stay version-agnostic.
- **Alternatives considered**:
  - Compute count-only behavior separately in each versioned service with a new command-specific API: rejected because it would add parallel service entry points for one display concern.
  - Infer totals in `cmd/` from `OverflowState` plus item counts: rejected because that loses the clarified lower-bound semantics for capped totals.

## Decision 5: Preserve lower-bound totals exactly when the backend marks totals as capped

- **Decision**: When `hasMoreTotalItems=true` or the backend otherwise indicates the reported total is capped, `--total` should print that numeric lower bound unchanged.
- **Rationale**: This matches the explicit clarification recorded in the spec and avoids turning count-only mode into a full pagination/recount feature. It also preserves truthful reporting under the repository’s “Operational Proof Over Intent” principle.
- **Alternatives considered**:
  - Traverse all remaining pages to compute an exact total: rejected because it changes the cost profile and could still be misleading if the backend’s capped total semantics exist for server-side reasons.
  - Fail when the total is only a lower bound: rejected because the clarified requirement explicitly prefers returning the useful numeric lower bound.

## Decision 6: Keep default detail rendering unchanged and add a narrow count-only branch before the existing list view

- **Decision**: Implement count-only output as a small command branch in `get_processinstance.go` before `listProcessInstancesView(...)`, leaving `cmd/cmd_views_get.go` detail rendering unchanged for non-`--total` usage.
- **Rationale**: `listProcessInstancesView(...)` currently routes one-line, JSON, and keys-only detail output through shared helpers. Count-only output is a narrow exception, so short-circuiting before the detail renderer is the smallest compatible change.
- **Alternatives considered**:
  - Teach shared list render helpers about numeric-only mode: rejected because it would broaden a command-specific feature into a global rendering concern.
  - Modify `listProcessInstancesView(...)` to inspect `--total`: rejected because it couples shared detail rendering more tightly to one feature flag.

## Decision 7: Documentation updates must include both README and regenerated CLI reference

- **Decision**: Update `README.md` and regenerate `docs/cli/c8volt_get_process-instance.md` via `make docs-content`.
- **Rationale**: The constitution requires user-visible command behavior and examples to stay in sync with shipped behavior. The repository’s `Makefile` documents `make docs-content` as the supported path for regenerated CLI markdown.
- **Alternatives considered**:
  - Update only README: rejected because generated CLI docs would become stale.
  - Hand-edit `docs/cli/c8volt_get_process-instance.md`: rejected because the repository already has a documented generation path.

## Existing Technical Signals

- `cmd/get_processinstance.go` is the right command seam for `--total`, because it already owns process-instance search flag validation, keyed-vs-search branching, and final response rendering.
- `cmd/cmd_views_get.go` keeps current non-`--total` detail rendering centralized for one-line, JSON, and keys-only modes, which supports a small short-circuit design rather than a render-framework change.
- `internal/services/processinstance/v88/service.go` and `v89/service.go` already see `payload.Page.TotalItems` and `HasMoreTotalItems`, but only pass `OverflowState` upward today.
- `internal/services/processinstance/v87/service.go` already sees `payload.Total` from Operate search responses and can surface that as the best available reported total for count-only mode.
- `Makefile` already defines `make docs-content` as the supported CLI-doc regeneration path and `make test` as the repository-wide validation gate.

## Regression Anchors

- [`cmd/get_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance_test.go) is the best command-level anchor for `--total` output, zero-match behavior, and invalid flag combinations.
- [`cmd/cmd_processinstance_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_processinstance_test.go) contains reusable request and paging helpers if request-capture or shared command fixtures are needed.
- [`c8volt/process/client_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client_test.go) is the right seam for new shared page-metadata conversion coverage.
- [`internal/services/processinstance/v87/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87/service_test.go), [`v88/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service_test.go), and [`v89/service_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89/service_test.go) are the right seams for reported-total and lower-bound metadata behavior.
