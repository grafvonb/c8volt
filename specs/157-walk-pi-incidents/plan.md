# Implementation Plan: Walk Process-Instance Incident Details

**Branch**: `157-walk-pi-incidents` | **Date**: 2026-05-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/157-walk-pi-incidents/spec.md`

## Summary

Add `--with-incidents` to `c8volt walk process-instance` / `walk pi` for keyed traversal output. The implementation should keep traversal selection unchanged, reuse the process facade incident enrichment introduced for issue #154, enrich only the walked process instances returned by the selected mode, render human incident messages below matching rows, include per-item incident details in JSON traversal payloads, fail the command if any requested incident lookup fails, and preserve tenant/version safeguards.

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, existing process facade/service packages, generated Camunda clients under `internal/clients/camunda/v88/camunda` and `internal/clients/camunda/v89/camunda`, existing rendering helpers in `cmd/`  
**Storage**: No persistent storage changes; reads existing config/profile/root flag tenant through command bootstrap  
**Testing**: `go test`, `make test`, command tests under `cmd/`, facade tests under `c8volt/process/`, traversal tests under `internal/services/processinstance/walker` and `internal/services/processinstance/traversal`, versioned service tests under `internal/services/processinstance/v87`, `v88`, and `v89`  
**Target Platform**: Cross-platform CLI for local and CI use against supported Camunda versions; incident enrichment supported through existing v8.8 and v8.9 incident search paths, unsupported explicitly for v8.7 where tenant-safe keyed enrichment is unavailable  
**Project Type**: CLI  
**Performance Goals**: Walk command performs no incident searches when `--with-incidents` is omitted; when enabled, it performs incident searches only for process instances returned by the selected traversal mode  
**Constraints**: Preserve current walk traversal behavior, preserve default human/key-only/tree/JSON output when the flag is omitted, fail instead of rendering partially enriched output when incident lookup fails, keep incident lookups tenant-aware, avoid tenant-unsafe direct incident fallback, update generated CLI docs, finish with targeted tests and `make test`  
**Scale/Scope**: Command flag validation and render selection in `cmd/walk_processinstance.go`, human/JSON traversal rendering in `cmd/cmd_views_walk.go`, public traversal enrichment models/helpers in `c8volt/process/`, optional traversal result model expansion in `internal/services/processinstance/traversal`, command tests in `cmd/walk_test.go`, facade tests in `c8volt/process/client_test.go`, documentation under `docs/cli/c8volt_walk_process-instance.md` and README review

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature is read-only but must prove actual traversal output, incident association, incident lookup failure behavior, request construction, and unsupported-version outcomes through tests.
- **CLI-First, Script-Safe Interfaces**: Pass. The behavior is exposed through one Cobra flag, preserves default output, and provides JSON details for automation.
- **Tests and Validation Are Mandatory**: Pass. The issue requires validation, output, tenant-aware request construction, incident lookup failure, default-output preservation, and version-specific coverage plus final repository validation.
- **Documentation Matches User Behavior**: Pass. The walk command help and generated CLI docs must describe the keyed-only incident flag.
- **Small, Compatible, Repository-Native Changes**: Pass. The design extends existing walk rendering and process facade incident methods rather than adding command-level generated-client calls.

## Project Structure

### Documentation (this feature)

```text
specs/157-walk-pi-incidents/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── walk-pi-with-incidents.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── walk_processinstance.go
├── cmd_views_walk.go
├── cmd_views_get.go
├── walk_test.go
└── cmd_views_get_test.go

c8volt/process/
├── api.go
├── client.go
├── convert.go
├── model.go
├── walker.go
└── client_test.go

internal/services/processinstance/
├── api.go
├── traversal/
│   └── result.go
├── v87/
│   ├── incidents.go
│   └── service_test.go
├── v88/
│   ├── incidents.go
│   └── service_test.go
└── v89/
    ├── incidents.go
    └── service_test.go

README.md
docs/cli/c8volt_walk_process-instance.md
```

**Structure Decision**: Keep traversal ownership where it already lives. `cmd/walk_processinstance.go` owns flag registration, validation, mode selection, and choosing enriched versus default views. `cmd/cmd_views_walk.go` owns human and JSON walk rendering. `c8volt/process` owns reusable public enrichment shapes that combine traversal results with issue #154 incident detail models. Versioned incident lookup stays in `internal/services/processinstance`.

## Phase 0: Research

Research findings are captured in [research.md](research.md).

- Confirmed issue #154 already added public and domain incident detail models plus facade incident lookup/enrichment helpers.
- Confirmed `walk process-instance` already returns `process.TraversalResult` for parent, children, and family modes.
- Confirmed existing walk JSON payload includes traversal metadata and item lists; enriched JSON should preserve those fields and add incidents per item.
- Confirmed human walk output uses `oneLinePI` rows joined by mode-specific separators or rendered through ASCII tree output.
- Confirmed v8.8 and v8.9 incident search paths already avoid redundant process-instance-key filter values and include configured tenant filters where supported.
- Confirmed v8.7 incident lookup is already explicit unsupported, which should surface through walk enrichment when requested.
- Confirmed incident lookup failure is all-or-nothing for requested enrichment: traversal must not render partially enriched output after a failed incident lookup.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](data-model.md)
- [quickstart.md](quickstart.md)
- [contracts/walk-pi-with-incidents.md](contracts/walk-pi-with-incidents.md)

- Add an incident-enriched traversal payload that preserves mode, outcome, root key, keys, edges, missing ancestors, warning, and per-item process-instance data.
- Reuse `process.ProcessInstanceIncidentDetail` and the facade `SearchProcessInstanceIncidents` behavior from issue #154.
- Add a facade helper that enriches a `process.TraversalResult` by looking up incidents for `result.Keys` and attaching them to the corresponding process instance.
- Propagate any incident lookup error from traversal enrichment so the command fails instead of rendering partial incident details.
- Add `--with-incidents` flag registration and validation to `walk process-instance`.
- Render human incident messages directly below matching process-instance rows. Tree mode should preserve branch layout and indent incident lines under the tree node that owns them.
- Render JSON using the same shared envelope path as existing walk JSON while replacing plain `items` with incident-enriched items only when the flag is requested.
- Reject `--keys-only --with-incidents` with a clear validation error because key-only output cannot carry incident details.
- Regenerate CLI docs through `make docs-content` after help text changes.

### Version Support Matrix

| Version | Incident enrichment | Planned behavior |
|---------|---------------------|------------------|
| v8.7 | Unsupported | Return existing unsupported-capability style for `--with-incidents` because tenant-safe keyed enrichment is unavailable |
| v8.8 | Supported | Reuse generated `SearchProcessInstanceIncidentsWithResponse` path and configured tenant filtering from issue #154 |
| v8.9 | Supported | Match v8.8 behavior with v89 generated client types |

## Phase 2: Task Planning Approach

Task generation should keep the work in independently verifiable user-story slices:

1. Prepare shared enriched traversal models and command flag validation.
2. Deliver User Story 1 as the MVP: human-readable incident messages for walked process instances.
3. Add User Story 2 JSON enrichment with stable traversal metadata and per-item incidents.
4. Add User Story 3 regression coverage for default traversal behavior, key-only rejection, tree semantics, and incident lookup failure behavior.
5. Add User Story 4 tenant/version safeguards, docs generation, and final validation.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design requires tests for actual request wiring, traversal preservation, failure propagation, and rendered output.
- **CLI-First, Script-Safe Interfaces**: Still passes. The flag is deterministic, keyed-only, and JSON-capable.
- **Tests and Validation Are Mandatory**: Still passes. Tasks include focused package tests and `make test`.
- **Documentation Matches User Behavior**: Still passes. The command help/docs tasks are explicit.
- **Small, Compatible, Repository-Native Changes**: Still passes. The change extends current walk and process-instance boundaries.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
