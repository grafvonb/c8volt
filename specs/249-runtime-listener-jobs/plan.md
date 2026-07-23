# Implementation Plan: Runtime Listener Jobs Under Elements

**Branch**: `249-runtime-listener-jobs` | **Date**: 2026-07-23 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/249-runtime-listener-jobs/spec.md`

## Summary

Add `--with-listeners` to element-oriented process investigation views so runtime `EXECUTION_LISTENER` and `TASK_LISTENER` jobs appear below the matching runtime element rows. The implementation will extend the existing process-instance element enrichment model with listener-job data from the job service, preserve opt-in output behavior, reject incompatible flag combinations before remote lookup, and keep existing output unchanged when listener enrichment is absent.

## Technical Context

**Language/Version**: Go 1.26 with toolchain go1.26.2

**Primary Dependencies**: Cobra for CLI commands; existing public facades under `c8volt/process`, `c8volt/element`, `c8volt/job`, and `c8volt/ops`; internal services under `internal/services/processinstance`, `internal/services/job`, `internal/services/element`, and `internal/services/ops`; shared helpers under `toolx`, `toolx/pool`, and `testx`

**Storage**: N/A; read-only CLI enrichment over existing Camunda runtime process, element, and job data

**Testing**: Go unit and command tests with `go test`; full repository validation through `make test`

**Target Platform**: Cross-platform command-line binary for c8volt-supported operator environments

**Project Type**: CLI application with public facade APIs and internal Camunda service adapters

**Performance Goals**: Listener enrichment performs no job lookup unless `--with-listeners` is set; when set, fetch listener jobs per selected process instance and attach them to already loaded elements by element-instance key instead of issuing one job request per element

**Constraints**: Preserve existing output when `--with-listeners` is absent; reject listener enrichment without element context and with keys-only output before remote enrichment; fail the whole command on requested listener lookup errors instead of rendering partial success; omit listener jobs that cannot be matched to an element instance; keep command-layer code out of generated Camunda clients and versioned service implementations

**Scale/Scope**: Keyed and search/list element views, keyed process-instance views with elements, walk process-instance family/children/parent/flat modes with elements, and keyed slow-process analysis; supports Camunda 8.8 and 8.9 job search; Camunda 8.7 returns existing unsupported-version style errors

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: PASS. The feature is read-only and renders listener-enriched output only after all requested element and listener lookups complete successfully; invalid flag combinations fail before remote enrichment.
- **CLI-First, Script-Safe Interfaces**: PASS. The feature is exposed through an explicit opt-in flag, stable validation rules, stable human nesting, and requested-only JSON listener arrays under element objects.
- **Tests and Validation Are Mandatory**: PASS. Planning requires targeted command, facade, service, rendering, unsupported-version, validation, JSON, and unchanged-output tests, plus `make test` before commit/merge.
- **Documentation Matches User Behavior**: PASS. User-visible flags, examples, JSON shape, and output contracts require command metadata updates, README alignment, and regenerated CLI docs with `make docs-content`.
- **Small, Compatible, Repository-Native Changes**: PASS. The plan extends existing enrichment and activity rendering patterns, reuses the job service search surface, and avoids new generated-client edits.

## Project Structure

### Documentation (this feature)

```text
specs/249-runtime-listener-jobs/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── runtime-listener-jobs-cli.md
│   └── runtime-listener-jobs-json.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
├── get_element.go                         # element flag registration and validation for --with-listeners
├── get_processinstance.go                 # get pi flag registration and validation
├── get_processinstance_enrichment.go      # shared activity enrichment orchestration
├── walk_processinstance.go                # walk flag registration, validation, and post-traversal enrichment
├── ops_analyse_slow_process_instances.go  # ops flag registration and request mapping
├── cmd_views_processinstance_activity.go  # shared element/listener human and JSON activity rendering
├── cmd_views_element.go                   # get element/list rendering output integration
├── cmd_views_ops_slow_process_analysis.go # slow-analysis listener rendering
├── command_contract.go                    # command capability metadata source
└── *_test.go                              # focused command, renderer, and contract tests

c8volt/process/
├── api.go                                 # public process enrichment contract
├── client.go                              # facade delegation to internal listener enrichment
└── model.go                               # public JSON models for element listener arrays

c8volt/ops/
├── model.go                               # public slow-analysis listener output model
├── convert.go                             # domain/public mapping
└── client.go                              # facade delegation remains thin

internal/domain/
├── processinstance.go                     # version-neutral process-instance models
├── processinstance_enrichment.go          # listener-enriched activity/element models
├── element.go                             # version-neutral element models
├── job.go                                 # existing job identity, kind, event, and ownership fields
└── ops_slow_process_analysis.go           # slow-analysis listener output fields

internal/services/processinstance/
├── enrichment.go                          # attach listener jobs to elements by element-instance key
└── enrichment_test.go                     # ownership, omission, ordering, and empty-array behavior

internal/services/job/
├── api.go                                 # existing job search service surface
├── v87/                                  # unsupported job search behavior
├── v88/                                  # listener job search through generated Camunda API
└── v89/                                  # listener job search through generated Camunda API

internal/services/ops/
├── slow_process_analysis.go               # include listeners in timeline/element rows when requested
└── slow_process_analysis_test.go

README.md                                  # user-facing examples and behavior notes
docs/cli/                                  # regenerated CLI reference from command metadata
```

**Structure Decision**: Use the existing single Go CLI project layout. Command wiring and rendering remain in `cmd`; public data contracts remain in `c8volt/process` and `c8volt/ops`; listener enrichment mechanics live in internal services; version-specific job lookup stays behind `internal/services/job/v88` and `v89` with `v87` unsupported behavior.

## Complexity Tracking

No constitution violations or justified complexity exceptions.

## Phase 0: Research

Research decisions are recorded in [research.md](./research.md). All technical context unknowns are resolved.

## Phase 1: Design & Contracts

Design artifacts:

- [data-model.md](./data-model.md)
- [contracts/runtime-listener-jobs-cli.md](./contracts/runtime-listener-jobs-cli.md)
- [contracts/runtime-listener-jobs-json.md](./contracts/runtime-listener-jobs-json.md)
- [quickstart.md](./quickstart.md)

Post-design Constitution Check:

- **Operational Proof Over Intent**: PASS. Contracts require validation before lookup and full command failure on requested listener lookup errors.
- **CLI-First, Script-Safe Interfaces**: PASS. CLI flag behavior, human nesting, JSON listener arrays, and invalid combinations are specified.
- **Tests and Validation Are Mandatory**: PASS. Quickstart defines targeted and full validation, including unchanged-output regressions.
- **Documentation Matches User Behavior**: PASS. Quickstart includes README and generated docs validation.
- **Small, Compatible, Repository-Native Changes**: PASS. Data model and contracts extend current enrichment structures and reuse the existing job search surface.
