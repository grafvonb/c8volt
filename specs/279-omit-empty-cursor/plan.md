# Implementation Plan: Omit Empty Initial Search Cursor

**Branch**: `279-omit-empty-cursor` | **Date**: 2026-08-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/279-omit-empty-cursor/spec.md`

## Summary

Correct latest process-definition request construction for Camunda 8.8, 8.9, and 8.10 so the initial page serializes as limit-only, while continuation pages use a non-empty returned cursor unchanged and ordinary searches retain offset pagination. The change stays inside the three version-specific process-definition adapters, uses each generated client's existing limit-only pagination variant, adds wire-shape regression tests, and is operationally verified by starting ten instances by BPMN process ID against Camunda 8 Run 8.9.17 with default H2/RDBMS storage.

## Technical Context

**Language/Version**: Go 1.26 with toolchain Go 1.26.2

**Primary Dependencies**: Go standard library JSON/HTTP support, generated Camunda v8.8/v8.9/v8.10 clients, oapi-codegen runtime 1.4.0, Testify 1.11.1; no new dependency

**Storage**: N/A for c8volt; validation targets Camunda 8 Run 8.9.17 default H2/RDBMS secondary storage

**Testing**: Go `testing` and Testify adapter tests, shared service regressions, operational CLI validation against a disposable Camunda cluster, and race-enabled `make test`

**Target Platform**: Cross-platform c8volt CLI and library for Linux, macOS, and Windows connecting to Camunda 8.8, 8.9, or 8.10

**Project Type**: Single Go CLI/library with version-neutral facades and services plus version-specific Camunda adapters

**Performance Goals**: No additional backend requests for established selector validation; paging retains existing result limits and stops when no non-empty continuation cursor is available

**Constraints**: Do not edit generated clients; do not change v8.7; preserve filters, tenant scope, result limits, latest stable sort, ordinary offset requests, exact-version/key selection, CLI flags, output, and exit behavior

**Scale/Scope**: Three version-specific request builders and their closest tests, one external request-wire contract, and one focused Camunda 8.9.17 operational proof

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

| Principle | Pre-Research Gate | Post-Design Check |
|---|---|---|
| I. Operational Proof Over Intent | PASS: selector validation must finish before mutation, and the previously failing environment must prove ten created instances. | PASS: [quickstart.md](quickstart.md) requires an 8.9.17/H2 readiness check followed by a ten-instance BPMN-ID run with exact key-count evidence. |
| II. CLI-First, Script-Safe Interfaces | PASS: this is an internal request correction with no new or changed command, flag, prompt, output field, wording, or exit contract. | PASS: [latest-process-definition-search.md](contracts/latest-process-definition-search.md) explicitly preserves CLI and facade contracts while changing only backend page JSON. |
| III. Tests and Validation Are Mandatory | PASS: each affected version requires initial, continuation, and ordinary paging assertions, followed by operational proof and `make test`. | PASS: the contract and quickstart define per-version wire tests, focused regressions, the external H2 proof, and the full race-enabled gate. |
| IV. Documentation Matches User Behavior | PASS: operator-visible behavior and syntax remain unchanged, so README and generated CLI docs should not change. | PASS: the design records the no-docs decision and requires a diff check to catch accidental user-facing changes. |
| V. Small, Compatible, Repository-Native Changes | PASS: use existing adapter-local page builders and generated `LimitPagination` variants; add no abstraction or dependency. | PASS: the design touches only established version adapters/tests and keeps shared domain, facade, command, and generated code intact. |

No constitution violation requires complexity tracking. There are no unresolved clarifications.

## Project Structure

### Documentation (this feature)

```text
specs/279-omit-empty-cursor/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── latest-process-definition-search.md
├── checklists/
│   └── requirements.md
└── tasks.md                              # created later by $speckit-tasks
```

### Source Code (repository root)

```text
internal/services/processdefinition/
├── search.go                             # existing shared continuation propagation; unchanged unless a regression test exposes a gap
├── search_test.go                        # existing shared end-cursor propagation coverage
├── v88/
│   ├── service.go                        # v8.8 three-way page encoding
│   └── service_test.go                   # v8.8 wire-shape regressions
├── v89/
│   ├── service.go                        # v8.9 three-way page encoding
│   └── service_test.go                   # v8.9 wire-shape regressions
└── v810/
    ├── service.go                        # v8.10 three-way page encoding
    └── service_test.go                   # v8.10 wire-shape regressions

internal/clients/camunda/{v88,v89,v810}/  # generated clients consumed but not edited
c8volt/process/                           # facade behavior preserved
cmd/                                      # selector and CLI behavior preserved
integration/cli/                          # existing focused regressions may be run; no harness expansion required
```

**Structure Decision**: Preserve the current single-project layering. Each version adapter owns conversion to its generated page union, because those generated types differ slightly by Camunda release. The shared domain request and traversal already carry a non-empty `EndCursor` forward unchanged, while the facade and commands correctly route latest BPMN-ID validation. The defect is therefore fixed at the smallest valid boundary: adapter request serialization.

## Design Sequence

1. Update each `newProcessDefinitionSearchPageRequest` implementation to select exactly one page encoding: a non-empty `After` uses cursor-forward pagination; latest mode without `After` uses generated limit-only pagination; ordinary mode without `After` retains offset pagination.
2. Keep filter construction and latest sorting (`processDefinitionId ASC`, then `tenantId ASC`) unchanged in all three adapters.
3. Replace the existing initial-latest tests that assert an empty cursor with serialized-wire assertions for `limit: 1000`, absent `after`, and absent `from`.
4. Add or focus one latest continuation assertion per adapter proving a non-empty opaque cursor is serialized exactly and no offset is introduced.
5. Add or retain one ordinary-search assertion per adapter proving `from` and `limit` remain present and `after` remains absent.
6. Run focused adapter and shared-service tests, then facade/command selector regressions and `make test`.
7. Build a temporary binary and execute the ten-instance BPMN-ID workflow against a disposable Camunda 8 Run 8.9.17 default-H2 profile. Record exit status, ten returned keys, and absence of the previous process-definition search 500 response.

## Documentation Impact

README and generated CLI documentation remain unchanged because commands, flags, examples, output schemas, prompts, defaults, and success wording do not change. The feature-local contract and quickstart document the backend interoperability correction and its proof. If implementation changes any user-facing metadata unexpectedly, that change must stop for scope review rather than silently trigger docs regeneration.

## Risk Controls

- **Union accessor ambiguity**: assert serialized JSON field presence/absence, not only conversion through generated union accessors.
- **Required empty cursor fields in v8.8/v8.9**: use `LimitPagination`; do not attempt to encode a nil value into cursor-forward variants whose `After` field is required.
- **Accidental offset regression**: keep an ordinary non-latest request test for every affected version.
- **Cursor corruption**: use the exact non-empty `After` string without trimming, decoding, or normalization.
- **Generated-client drift**: consume existing generated types and require no diff under `internal/clients/camunda`.
- **Scope expansion**: keep v8.7, facades, command wiring, selectors, output renderers, and creation workflows unchanged.
- **External environment gap**: the repository does not provision Camunda 8 Run 8.9.17; [quickstart.md](quickstart.md) makes the disposable H2 cluster and working profile explicit prerequisites.
- **Destructive validation**: perform live proof only against a disposable profile and run readiness/version checks before deployment or process creation.

## Complexity Tracking

No constitution violations, new architectural layers, or justified complexity exceptions are required.
