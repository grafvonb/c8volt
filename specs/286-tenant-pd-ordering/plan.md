# Implementation Plan: Stable Tenant-Aware Process-Definition Ordering

**Branch**: `286-tenant-pd-ordering` | **Date**: 2026-08-31 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/286-tenant-pd-ordering/spec.md`

## Summary

Make every process-definition collection use one deterministic order: exact tenant ID ascending, exact BPMN process ID ascending, numeric version descending, then exact process-definition key ascending. Backend queries will request the strongest compatible sort for stable paging, while the version-neutral service will own complete traversal, tenant-aware latest selection, and a final canonical sort. The CLI, facade, statistics enrichment, watch snapshots, and all output renderers will consume that same ordered collection without changing direct-key or XML retrieval.

## Technical Context

**Language/Version**: Go 1.26 with toolchain Go 1.26.2

**Primary Dependencies**: Go standard library (`slices`), Cobra 1.10.2, the existing public `c8volt/process` facade, version-neutral process-definition service, and generated Camunda 8.7-8.10 clients

**Storage**: N/A; this feature reads Camunda APIs and holds a bounded result collection in memory for normalization

**Testing**: Go `testing`, Testify 1.11.1, focused package tests, command contract tests, and repository-wide `make test` race validation

**Target Platform**: Cross-platform Go CLI against Camunda 8.7, 8.8, 8.9, and 8.10 clusters

**Project Type**: Go CLI plus public facade and internal service/adapters

**Performance Goals**: Preserve existing request/page sizes and produce deterministic output in O(n log n) time over collected definitions; page sizes 1, 2, and 1000 must produce the same ordered keys

**Constraints**: Keep backend mechanics out of `cmd` and the facade; do not hand-edit generated clients; preserve exact case-sensitive identifiers; keep statistics out of the comparator; retain compact output and existing schemas; preserve the Camunda 8.7 compatibility cap of 1000 visible definitions

**Scale/Scope**: One domain comparator, one version-neutral collection path, four version adapters, public process facade, process-definition CLI/watch render paths, README/help/generated CLI docs, and their focused tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Pre-design gate | Post-design gate | Evidence |
|-----------|-----------------|------------------|----------|
| I. Operational Proof Over Intent | PASS | PASS | This read-only feature makes no state-transition claim. Acceptance validates complete, exact-once ordered results across paging and latest selection. |
| II. CLI-First, Script-Safe Interfaces | PASS | PASS | Existing commands, aliases, flags, exit behavior, envelopes, and output schemas are preserved; only collection order becomes deterministic. |
| III. Tests and Validation Are Mandatory | PASS | PASS | The plan covers the domain comparator, shared service, all adapters, facade, command modes, page-size invariance, watch behavior, docs, and full `make test`. |
| IV. Documentation Matches User Behavior | PASS | PASS | Command metadata and README will document the order, followed by `make docs-content` to regenerate CLI documentation. |
| V. Small, Compatible, Repository-Native Changes | PASS | PASS | The design extends existing request/service paths and sort helpers, adds no dependency or parallel hierarchy, and keeps direct retrieval unchanged. |

No constitution violation or unresolved gate remains.

## Project Structure

### Documentation (this feature)

```text
specs/286-tenant-pd-ordering/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── process-definition-ordering.md
└── tasks.md                         # Generated later by $speckit-tasks
```

### Source Code (repository root)

```text
internal/domain/
└── processdefinition.go             # Canonical comparator and latest-group identity

internal/services/processdefinition/
├── search.go                        # Complete traversal, latest reduction, final ordering
├── v87/service.go                   # Camunda 8.7 compatible backend sort/traversal
├── v88/service.go                   # Camunda 8.8 native latest and sort fields
├── v89/service.go                   # Camunda 8.9 native latest and sort fields
└── v810/service.go                  # Camunda 8.10 native latest and sort fields

c8volt/process/
├── client.go                        # Thin ordered collection facade
└── convert.go                       # Order-preserving public conversion

cmd/
├── get_processdefinition.go         # Listing dispatch, flags, help metadata
├── get_processdefinition_watch.go   # Watch snapshot collection
└── cmd_views_processdefinition.go   # Order-preserving human/JSON/keys rendering

README.md
docs/cli/                            # Regenerated from command metadata
```

**Structure Decision**: Keep ownership in the existing layers. The domain package defines ordering semantics; the version-neutral service owns collection mechanics; adapters express API-version differences; the facade maps types without reordering; and commands only select the mode and render the returned sequence.

## Phase 0: Research

Research decisions and rejected alternatives are recorded in [research.md](research.md). All technical unknowns were resolved and no clarification remains open.

## Phase 1: Design and Contracts

- [data-model.md](data-model.md) defines the canonical sort key, exact tenant/process group, latest selection, collection, statistics, and watch snapshot invariants.
- [contracts/process-definition-ordering.md](contracts/process-definition-ordering.md) defines the user-visible ordering, paging, latest, output, statistics, compatibility, and regression contract.
- [quickstart.md](quickstart.md) provides focused, full-suite, documentation, and manual validation steps.

## Implementation Strategy

1. Replace the current partial process-definition sort with a shared domain comparator over tenant ID, BPMN process ID, numeric version, and opaque key.
2. Request the strongest compatible backend order in every adapter: tenant/process/version/key for ordinary searches, and tenant/process for native latest queries where the API restricts sort fields.
3. Extend the existing search request with an additive `Latest` intent. Traverse all available pages through the shared service, use native latest filtering on Camunda 8.8-8.10, and reduce all visible versions locally on 8.7.
4. After page accumulation, select at most one definition per exact `(tenantID, BPMNProcessID)` group when latest is requested, apply the canonical comparator, and then apply any latest-result limit. Ordinary searches keep existing limit semantics and do not deduplicate.
5. Route public latest search, broad CLI `--latest`, selector validation, and watch snapshots through the shared collection path. Keep direct-key and XML paths separate and unchanged.
6. Update source help and README descriptions, regenerate CLI docs with `make docs-content`, and preserve all existing human, JSON, keys-only, and watch schemas.
7. Add a reusable fixture matrix covering default/named tenants, case distinctions, multiple processes and versions, text-key ties, shuffled pages, statistics, native/local latest behavior, and all output modes.

## Compatibility and Risk Controls

- Exact values drive identity and ordering; there is no case folding, locale collation, numeric key parsing, or special placement for `<default>`.
- Backend sorting stabilizes page boundaries; the final local sort makes the result independent of backend and page arrival order.
- Native `isLatestVersion` is retained for Camunda 8.8-8.10, but complete paging and final normalization are service-owned. Camunda 8.7 reduces locally within its existing 1000-definition compatibility window.
- Statistics enrichment stays in its existing version-capable path and cannot affect the comparator or row position. Unsupported Camunda 8.7 statistics behavior remains unchanged.
- Each ordinary matching definition is emitted once as received across the traversal; only explicit latest reduction removes older group members.
- Direct-key and XML retrieval, filters, authorization, errors, envelopes, and output field schemas remain unchanged.
- Ordinary `Limit` semantics remain unchanged. When `Latest` and a limit are combined internally, the limit applies after latest reduction and canonical sorting so group selection is complete and deterministic.

## Validation Plan

1. Run focused domain and shared-service tests for comparator transitivity, exact-text behavior, latest grouping, complete paging, and page-size invariance.
2. Run focused adapter tests for emitted backend sort requests, native/latest behavior, and Camunda 8.7 local compatibility.
3. Run facade and command tests for broad listing, selector validation, statistics parity, human/JSON/keys parity, watch stability, and unchanged direct-key/XML behavior.
4. Regenerate documentation with `make docs-content` and inspect the generated diff for a single consistent contract.
5. Run `make vet`, `make test`, and `git diff --check`; optionally perform the cluster-backed commands in [quickstart.md](quickstart.md).

## Complexity Tracking

No constitution violations require justification. The only additive request field and shared comparator reuse existing structures and avoid new dependencies, packages, or generated-client changes.
