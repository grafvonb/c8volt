# Implementation Plan: Filter User Tasks by Local Variables

**Branch**: `codex/310-user-task-variable-filtering` | **Date**: 2026-09-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/310-user-task-variable-filtering/spec.md`, backed by issue #310.

## Summary

Add the three existing variable-search flags to `get user-task`, reusing `get pi` grammar without changing it. Pass ordered predicates through the existing task facade/query and map them to native local-variable filters in the 8.8, 8.9, and 8.10 task adapters. Reuse current task traversal, counting, rendering, tenant scope, and optional effective-variable display. No new endpoint, query language, filtering loop, or result schema.

## Technical Context

**Language/Version**: Go 1.26, toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag, standard JSON handling, generated Camunda clients, domain validation, toolx, and testx. No new dependencies.

**Storage**: Camunda remains authoritative. No persistence or schema migration.

**Testing**: Go/testify, native HTTP request fixtures, service/facade doubles, command execution helpers, and real-terminal stdin tests with separate stdout/stderr.

**Target Platform**: Existing cross-platform CLI; terminal tests follow repository platform support.

**Project Type**: CLI plus public Go facade library.

**Performance Goals**: No additional requests for filtering beyond ordinary native task search/count. No per-task variable reads for filtering; optional display retains its existing selected-task bounds.

**Constraints**: Identical PI grammar, aliases, errors, and serialized-value conventions. Local-variable scope only. Preserve 8.7 unsupported behavior, 8.8–8.10 native support, version-local mappings, pipeline output, and established empty results. No generated-client edits or broad parser refactoring.

**Scale/Scope**: Three flags, additive filter inputs, one small parser orchestration extraction, three adapter mappings, targeted tests, and documentation. Existing pagination and count algorithms remain authoritative.

## Constitution Check

*Pre-research gates passed. Post-design re-evaluation also passes; no exceptions are required.*

| Principle | Design evidence |
| --- | --- |
| Operational proof over intent | Native selection; preserved exact-count/completion checks; errors never become empty matches or unfiltered fallback |
| CLI-first, script-safe interfaces | Existing flags/grammar and output contracts; key conflicts and validation before requests; real-terminal prompt checks |
| Proportional validation | Document checks for planning; targeted grammar, adapter, facade, traversal, command, and terminal checks for implementation; broader testing only if impact warrants it |
| Documentation matches behavior | Source help/metadata, six issue workflows, README, and `make docs-content` in implementation |
| Small compatible changes | Reuse parser, filter types, services, and views; bounded version-local mapping; no generic filtering framework |

Compatibility is additive for task search inputs and flags. No existing result field or PI behavior changes. Backend missing/null semantics are preserved rather than independently redefined.

## Project Structure

### Documentation (this feature)

```text
specs/310-user-task-variable-filtering/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli.md
│   └── facade-service.md
└── checklists/requirements.md
```

`tasks.md` is produced by the subsequent `$speckit-tasks` phase, not by this planning command.

### Source Code (repository root)

```text
cmd/
├── get_usertask.go                        # Flags, validation, request dispatch, help
├── get_usertask_variable_filter.go        # Small task-owned parser wrapper
├── get_processinstance_variable_filter.go # Explicit-input shared orchestration; retain PI wrapper/helpers
├── get_usertask_*test.go                  # Filter, execution, output, paging, display regression tests
├── get_processinstance_variable_filter_test.go # Existing grammar regression and parity fixtures
└── command_contract_test.go              # Help/capability expectations
c8volt/task/
├── model.go                              # Filter aliases/constants, additive search input
├── convert.go                            # Mechanical predicate mapping
└── *_test.go                             # Request conversion, error/options coverage
internal/domain/usertask.go               # Add existing domain filter-set field
internal/services/usertask/
├── search_test.go                        # Predicates survive existing traversal/counting
├── v87/native_test.go                    # Retained unsupported behavior
└── v88/, v89/, v810/
    ├── search.go                         # Attach localVariables
    ├── variable_filter.go                # Small matching-version mapper
    └── search_test.go                    # Native request/error regressions
README.md
docs/cli/                                # Regenerated from command source
```

**Structure Decision**: Keep flags and dispatch in the command, clause mechanics in focused parser support, facade conversion thin, and generated types in matching service adapters. Existing `get_usertask_search.go`, `get_usertask_vars.go`, and view code should require no production behavior changes. Do not add an independent mode or lifecycle.

## Phase 0: Research Outcomes

[research.md](research.md) resolves parser reuse, model ownership, native capabilities, value encoding, paging/display boundaries, and validation strategy. Read-only parallel research checked adapter/schema compatibility while local research checked CLI/paging paths.

The current PI parser reads command globals only in its orchestration. Extract an explicit-input helper there, retaining its existing lower-level functions, errors, and wrapper. Both commands supply their own raw flags; no mutable cross-command state or grammar rewrite.

Public task aliases reuse existing process predicate records, following the existing task variable-record alias. The domain query reuses the existing domain predicate set. Adapter-local mappers mirror the established PI conversion for their own generated versions. This avoids importing another area's service or broadening PI adapter changes.

## Phase 1: Design

### Request flow

1. Register task-owned repeatable `StringArray` flags and extend global-reset helpers used by tests.
2. Parse during argument validation. Include each flag in the existing search-selector detection for explicit and stdin keys. Keep parser support separate from the Cobra constructor.
3. Make task request construction return `(task.SearchRequest, error)` and propagate parse failures; do not ignore errors because validation ran earlier or cache parsed state globally. Update direct request-builder tests/callers.
4. Convert clauses mechanically into the additive domain query field, preserving optional false booleans with `toolx.CopyPtr` and clause order.
5. Each adapter validates/builds all clauses before HTTP and attaches `localVariables`; an empty set omits that field. Use existing native search and existing tenant configuration.
6. Existing traversal/counting carries the query across pages. Existing rendering/enrichment receives only selected tasks. Filter-only execution adds no task-variable reads.

Full input and output behavior is defined in [contracts/cli.md](contracts/cli.md); facade/native mapping is defined in [contracts/facade-service.md](contracts/facade-service.md) and [data-model.md](data-model.md).

### Validation and acceptance mapping

| Requirements | Implementation validation |
| --- | --- |
| FR-001–002 | Same fixtures through PI/task wrappers; all operators and alias; quoting, commas, arrays, wildcard escapes, invalid clauses, command-global isolation |
| FR-003–005 | Exact native localVariables requests across versions; ordinary/tenant predicates retained; zero filter-triggered variable/name discovery; optional live local/parent/shadowed checks |
| FR-006/012 | Explicit/stdin key conflicts, malformed CLI and direct-facade operands, no requests on validation failure, retained v87 unsupported errors |
| FR-007 | Same filters on cursor/offset/sparse pages, within-page limits, exact/capped counts, visitor stop, cancellation, errors |
| FR-008–009 | Execute human/JSON/keys/quiet/total and supported combinations; exact empty text/zero-byte keys/JSON EOF; real-terminal stdin and separate inherited/configured stderr |
| FR-010–011 | Filtered optional display with selected-task bounds, no duplicate reads, excluded modes, truncation/value-limit and failure preservation |
| FR-013–015 | Unfiltered and PI regressions, source help/capabilities, README/generated docs, no mutation/client-side filtering or unrelated changes |

Request fixtures prove mapping, not live server semantics. Do not document `$exists=false` as a client-defined missing-name complement. Optional live read-only checks distinguish absence, JSON null, and string-null and record backend behavior.

### Delivery sequence

1. Add reusable task filter input types and mechanical domain conversion, with direct-input regression coverage.
2. Add version-specific local filter mapping and request validation; confirm omission for unfiltered requests.
3. Wire flags and narrow shared parser orchestration; prove PI compatibility and key conflicts.
4. Exercise existing pagination/count/display integration and real-terminal output with filters; fix only feature-related integration gaps.
5. Update source help, metadata, README, and generated docs; complete focused validation from [quickstart.md](quickstart.md).

No runtime tests run during this plan. For implementation start with closest checks, then focused race runs. Use `make test` only if the actual diff broadens shared runtime behavior across packages or targeted failures leave unresolved regression risk; record the concrete reason. Do not repeat tests solely for documentation, commits, or phase completion.

## Complexity Tracking

No constitutional violations. Small adapter-local mapping duplication is deliberate: generated types differ by version and PI adapters remain unchanged. Shared grammar and domain validation prevent user-visible drift without a new abstraction layer.
