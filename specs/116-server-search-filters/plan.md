# Implementation Plan: Push Supported Get Filters Into Search Requests

**Branch**: `116-server-search-filters` | **Date**: 2026-04-19 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/spec.md)
**Input**: Feature specification from `/specs/116-server-search-filters/spec.md`

## Summary

Move request-capable `get` filter narrowing ahead of page fetches so supported versions send those predicates directly in the versioned process-instance search request instead of fetching broad pages and trimming them locally afterward. The design keeps the CLI surface unchanged, extends the shared process-instance filter model with explicit request-side capability fields for parent-presence and incident-presence semantics, applies those fields in the `v88` and `v89` search builders where the generated Camunda clients support them, preserves existing local fallback in `v87` where the search shape cannot express the same meaning reliably, and proves the result with request-capture regressions plus paging-behavior coverage.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, shared process facade under `c8volt/process`, public/domain models under `c8volt/process` and `internal/domain`, generated Camunda clients under `internal/clients/camunda/v88/camunda` and `v89/camunda`, existing helpers in `internal/services/common`, versioned process-instance services under `internal/services/processinstance/{v87,v88,v89}`  
**Storage**: File-based YAML config plus environment variables; no persistent datastore changes  
**Testing**: focused `go test ./cmd -count=1`, `go test ./c8volt/process -count=1`, `go test ./internal/services/processinstance/... -count=1`, final repository validation with `make test`; all four validation commands passed on 2026-04-19 during the polish iteration  
**Target Platform**: Cross-platform CLI for local and CI use against supported Camunda `8.7`, `8.8`, and `8.9` environments  
**Project Type**: CLI  
**Performance Goals**: Reduce unnecessary process-instance overfetching on supported versions, keep paging totals and continuation prompts aligned with the filtered result set, avoid user-visible regressions in keyed lookup, wait, or mutation flows, and preserve current throughput on versions that must stay on client-side fallback  
**Constraints**: Preserve existing `get process-instance` flag semantics and validation, keep `--orphan-children-only` client-side, apply pushdown only where the active version has a reliable request-side representation, audit other relevant `get` command families and record both already-adopted qualifying seams and bounded no-addition rationale for non-qualifying commands, reuse the existing versioned service/facade architecture, update source and regenerate only if generated artifacts actually need to change, and finish with `make test`  
**Scale/Scope**: CLI filter parsing and paging flow in [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go), shared public filter types in [`c8volt/process/model.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go), facade mapping in [`c8volt/process/convert.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/convert.go) and [`c8volt/process/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/client.go), domain filter types in [`internal/domain/processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/domain/processinstance.go), versioned search builders and tests under [`internal/services/processinstance/v87/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v87), [`internal/services/processinstance/v88/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88), and [`internal/services/processinstance/v89/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v89), plus command regressions under [`cmd/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature changes where filtering happens so displayed page counts and continuation prompts better reflect real matching state instead of optimistic local intent after overfetching.
- **CLI-First, Script-Safe Interfaces**: Pass. Existing commands and flags remain stable; only the request-construction path and pagination accuracy improve underneath the same CLI contract.
- **Tests and Validation Are Mandatory**: Pass. The plan requires request-capture coverage in versioned services, command paging regressions, facade-model coverage for new filter fields, and final `make test`.
- **Documentation Matches User Behavior**: Pass. No user-facing docs change is expected unless implementation reveals help or examples that become inaccurate once paging behavior changes; that decision is explicitly re-checked during implementation.
- **Small, Compatible, Repository-Native Changes**: Pass. The design extends existing process-instance filter models and versioned search builders rather than inventing a parallel query layer or command-specific request logic.

## Project Structure

### Documentation (this feature)

```text
specs/116-server-search-filters/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── process-instance-search-filters.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── get_processinstance.go
├── cmd_processinstance_test.go
├── get_processinstance_test.go
├── cancel_test.go
└── delete_test.go

c8volt/process/
├── api.go
├── client.go
├── convert.go
├── filter.go
├── model.go
└── client_test.go

internal/domain/
└── processinstance.go

internal/services/processinstance/
├── api.go
├── factory.go
├── v87/
│   ├── service.go
│   └── service_test.go
├── v88/
│   ├── service.go
│   ├── convert.go
│   └── service_test.go
└── v89/
    ├── service.go
    ├── convert.go
    └── service_test.go

internal/clients/camunda/
├── v88/camunda/client.gen.go
└── v89/camunda/client.gen.go
```

**Structure Decision**: Keep request-capable filter semantics centralized in the shared process-instance filter model and the versioned `SearchForProcessInstancesPage` builders. `cmd/get_processinstance.go` should only translate CLI flags into the shared filter and retain the client-side-only fallback rules, while `v88` and `v89` decide how to encode server-side predicates and `v87` continues to reject unsupported pushdown by omission rather than command-local branching.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/research.md).

- Confirm which current local filters in `get process-instance` are applied after fetch and whether any other `get` command family repeats the same late-filtering pattern.
- Confirm the exact shared model gap between CLI flags (`roots-only`, `children-only`, `incidents-only`, `no-incidents-only`) and the current `ProcessInstanceFilter` shape, which today only carries `ParentKey`, state, process selectors, and date bounds.
- Confirm the request-side predicate support in `v88` and `v89` generated Camunda clients, especially `hasIncident` equality and `parentProcessInstanceKey` existence semantics.
- Confirm that `v87` Operate search can express none of the new parent-presence or incident-presence semantics reliably enough for request-side pushdown, so fallback should remain local there.
- Confirm the best regression anchors for paging-behavior proof, request-body capture, and shared filter-model mapping.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/quickstart.md)
- [contracts/process-instance-search-filters.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/116-server-search-filters/contracts/process-instance-search-filters.md)

- Extend the shared `ProcessInstanceFilter` model with explicit optional `*bool` request-capable fields for parent presence and incident presence so the CLI can describe roots/children/incidents semantics without overloading `ParentKey` equality.
- Keep `--orphan-children-only` out of the shared request model because it depends on verifying missing parents after fetch and remains intentionally client-side.
- Apply request-side filter encoding in `v88` and `v89` only when the generated Camunda client supports the needed predicate shape, including equality for `hasIncident` and existence checks on `parentProcessInstanceKey`.
- Leave `v87` on the current client-side narrowing path for roots/children/incidents semantics because the Operate request builder in this repo only supports parent-key equality and lacks the reliable existence/incident filter shapes needed for the same meaning.
- Audit other `get` command families for equivalent late-filtering seams and record bounded no-addition rationale if no other server-capable post-fetch filters are found.

### Authoritative Filter Pushdown Boundary

| Filter semantic | CLI flag(s) | Shared filter addition | `v8.7` plan | `v8.8` plan | `v8.9` plan |
|--------|-------------|------------------------|-------------|-------------|-------------|
| Root instances only | `--roots-only` | `HasParent=*bool(false)` | Keep local fallback | Encode request-side parent-key absence filter | Encode request-side parent-key absence filter |
| Child instances only | `--children-only` | `HasParent=*bool(true)` | Keep local fallback | Encode request-side parent-key presence filter | Encode request-side parent-key presence filter |
| Has incidents | `--incidents-only` | `HasIncident=*bool(true)` | Keep local fallback | Encode request-side incident equality filter | Encode request-side incident equality filter |
| No incidents | `--no-incidents-only` | `HasIncident=*bool(false)` | Keep local fallback | Encode request-side incident equality filter | Encode request-side incident equality filter |
| Explicit parent lookup | `--parent-key` | existing `ParentKey=<key>` | Keep existing request-side equality | Keep existing request-side equality | Keep existing request-side equality |
| Orphan children only | `--orphan-children-only` | none | Client-side only | Client-side only | Client-side only |

This matrix is the authoritative planning boundary for later tasks. Any implementation that silently broadens `v8.7` support without a reliable request-side predicate, or that leaves `v8.8`/`v8.9` on local filtering for the supported semantics above, is incomplete.

### Audit Boundary For Other `get` Commands

| Command family | Current audit outcome | Planning rule |
|--------|------------------------|---------------|
| `get process-instance` | Confirmed in-scope; late local filtering already exists for roots/children/incidents/orphan children | Implement request-side pushdown where supported and keep client fallback only where required |
| `get process-definition --latest` | Confirmed additional qualifying seam | Keep the existing `v8.8`/`v8.9` request-side `isLatestVersion` pushdown, retain `v8.7` client-side fallback, and make that adoption explicit in audit notes and regression coverage |
| Other `get` command families | No other equivalent server-capable post-fetch filter seam confirmed by the implementation audit | Record bounded no-addition rationale instead of silently treating the audit as process-instance-only |

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Extend the shared public/domain process-instance filter types and facade conversions so request-capable parent-presence and incident-presence semantics can flow from the CLI into versioned services without changing existing flag meanings.
2. Update `cmd/get_processinstance.go` to translate `--roots-only`, `--children-only`, `--incidents-only`, and `--no-incidents-only` into the new shared filter fields for search mode while preserving local fallback only where the active version cannot honor them request-side.
3. Implement the new request-side predicate encoding in `internal/services/processinstance/v88` and `v89`, including service-level regressions that capture request bodies for parent-presence and incident-presence filters.
4. Preserve `v87` behavior by keeping those semantics client-side there, and add regressions proving the request body does not claim unsupported filters while the final result set still honors the flags.
5. Audit the rest of the `get` command surface for equivalent late-filtering seams, documenting the already-adopted `get process-definition --latest` seam and explicit no-addition rationale for the remaining non-qualifying cases.
6. Add or update command-level paging regressions so continuation prompts and total-so-far counts reflect the filtered server result set on supported versions, then finish with focused Go tests and final `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design improves the truthfulness of page-count and continuation messaging by moving supported narrowing into the authoritative request.
- **CLI-First, Script-Safe Interfaces**: Still passes. Flags, output modes, and command usage stay the same while the request body and paging behavior become more accurate.
- **Tests and Validation Are Mandatory**: Still passes with shared-model tests, versioned request-capture tests, command paging regressions, and `make test`.
- **Documentation Matches User Behavior**: Still passes. The current design expects no doc edits, but tasks must add them if the implementation reveals user-visible help text that becomes inaccurate once paging semantics improve.
- **Small, Compatible, Repository-Native Changes**: Still passes. The design stays inside current `cmd`, `c8volt/process`, and versioned service seams with no new abstraction layer.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |

## Implementation Status

- Shipped behavior matches the planned boundary: `get process-instance` pushes parent-presence and incident-presence predicates request-side on `v8.8` and `v8.9`, while `v8.7` keeps the existing client-side fallback.
- `--orphan-children-only` remains on the follow-up lookup seam across all supported versions.
- The broader audit outcome is finalized: `get process-definition --latest` remains the only additional qualifying seam, with request-side adoption on `v8.8` and `v8.9` and tested fallback on `v8.7`.

## Verification Record

- Passed `go test ./c8volt/process -count=1`
- Passed `go test ./internal/services/processinstance/... -count=1`
- Passed `go test ./cmd -count=1`
- Passed `make test`
