# Implementation Plan: Runtime Element Instance Command

**Branch**: `234-get-element-command` (Spec Kit), `240-get-element-command` (git) | **Date**: 2026-07-16 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/234-get-element-command/spec.md`

**Note**: Spec Kit reports the active feature branch label as `234-get-element-command` from `.specify/feature.json`; the checked-out git branch created by the specify hook is `240-get-element-command`.

## Summary

Add a read-only `c8volt get element` command that fetches or searches Camunda runtime element instances for Camunda 8.8 and 8.9, returns a clear unsupported-version error for Camunda 8.7, and follows existing `get` command contracts for paging, totals, JSON, keys-only, compact human rows, and documentation. The implementation should mirror the repository's `get job` and `get incident` slices: command validation/rendering in `cmd`, thin public facade in `c8volt/element`, version-neutral domain and service contracts in `internal/domain` and `internal/services/element`, and version-specific adapters under `internal/services/element/v87`, `v88`, and `v89`.

## Technical Context

**Language/Version**: Go, using the repository's current Go module configuration

**Primary Dependencies**: Cobra CLI, existing c8volt facade/service layering, generated Camunda v8.8/v8.9 clients, existing `toolx` timestamp helpers and command rendering helpers

**Storage**: N/A; read-only runtime inspection with no persisted local state

**Testing**: Go tests close to changed packages; targeted `go test ./cmd ./c8volt/element ./internal/services/element/... -count=1`, then `make test` before merge

**Target Platform**: Cross-platform CLI for operators running c8volt

**Project Type**: CLI application with public Go facade and internal Camunda service adapters

**Performance Goals**: Preserve existing `get` paging behavior; `--limit` caps returned rows across pages and JSON aggregates only bounded results

**Constraints**: No mutation; no custom `--all`; `--key` is mutually exclusive with search filters; multiple search filters combine with AND semantics; Camunda 8.7 must fail clearly

**Scale/Scope**: One new standalone command plus reusable service/facade capability for a future `get pi --with-elements` story; no listener/job enrichment, summary output, metrics, or loop aggregation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature is read-only, but command success still depends on observable lookup/search results or explicit unsupported-version errors. Totals, keys-only, JSON, and human output contracts are verifiable through command tests.
- **CLI-First, Script-Safe Interfaces**: PASS. The plan exposes behavior through a Cobra subcommand with stable flags, exit behavior, JSON payloads, keys-only output, and total-only output.
- **Tests and Validation Are Mandatory**: PASS. The plan requires version adapter tests, facade tests, command validation/rendering tests, paging/total tests, and documentation checks.
- **Documentation Matches User Behavior**: PASS. The new command changes user-visible CLI behavior and therefore requires README and generated CLI documentation updates via command metadata and `make docs-content`.
- **Small, Compatible, Repository-Native Changes**: PASS. The design reuses existing package ownership, generated clients, paging helpers, render helpers, and facade/service patterns. No new architectural layer is introduced.

## Project Structure

### Documentation (this feature)

```text
specs/234-get-element-command/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── cli.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output from /speckit-tasks
```

### Source Code (repository root)

```text
c8volt/
├── client.go            # Wire ElementAPI into the public aggregate client
└── element/
    ├── api.go           # Public element facade contract
    ├── client.go        # Thin facade over internal service
    ├── convert.go       # Public/domain mapping
    ├── model.go         # Public element request/result types
    └── client_test.go

internal/
├── domain/
│   └── element.go       # Version-neutral element domain types
└── services/
    └── element/
        ├── api.go       # Internal service contract and version assertions
        ├── factory.go   # Version-specific factory
        ├── v87/
        │   └── service.go
        ├── v88/
        │   ├── contract.go
        │   ├── convert.go
        │   ├── service.go
        │   └── service_test.go
        └── v89/
            ├── contract.go
            ├── convert.go
            ├── service.go
            └── service_test.go

cmd/
├── get_element.go
├── get_element_search.go
├── get_element_test.go
├── cmd_views_element.go
├── cmd_views_element_test.go
└── command_contract_test.go

README.md
docsgen/
└── main_test.go         # If generated docs expectations need coverage
```

**Structure Decision**: Use the established single-project CLI layout. The new `element` area mirrors `job` more closely than `incident`: direct lookup and paged search are first-class, 8.7 returns unsupported errors, 8.8/8.9 use generated runtime endpoints, and command rendering remains in `cmd`.

## Complexity Tracking

No constitution violations or justified extra complexity.

## Phase 0: Research

See [research.md](research.md). All planning unknowns are resolved.

## Phase 1: Design & Contracts

See [data-model.md](data-model.md), [contracts/cli.md](contracts/cli.md), and [quickstart.md](quickstart.md).

## Post-Design Constitution Check

- **Operational Proof Over Intent**: PASS. Quickstart and contracts define observable command outcomes for lookup, search, totals, keys-only, JSON, incidents, paging, and unsupported version behavior.
- **CLI-First, Script-Safe Interfaces**: PASS. The CLI contract specifies flags, selector composition, output modes, and machine-readable payload shape.
- **Tests and Validation Are Mandatory**: PASS. Quickstart names targeted command, facade, service, docs, and full validation commands.
- **Documentation Matches User Behavior**: PASS. Documentation updates and `make docs-content` are part of the validation path.
- **Small, Compatible, Repository-Native Changes**: PASS. Design artifacts keep the work scoped to existing `cmd`, `c8volt`, `internal/domain`, and `internal/services` ownership boundaries.
