# Progress: Element Terminology Standardization

## Codebase Patterns

- Public incident facade/domain fields are now `ElementId` and `ElementInstanceKey`; public JSON tags are `elementId` and `elementInstanceKey`.
- Public process facade/domain parent context is now `ParentElementInstanceKey`; the v8.7 process-instance adapter still reads generated `ParentFlowNodeInstanceKey` at the adapter boundary.
- Command code must reference canonical facade fields after the foundational rename, even where later user stories still own public flag/help and compact-label behavior changes.
- `cmd/get_incident.go` currently owns incident search flag grammar, including legacy `--flow-node-id` and `--fni-key`.
- `cmd/cmd_views_processinstance_incidents.go` currently renders incident context labels with `fn` and `fni`.
- Public incident fields currently live in `c8volt/incident/model.go` and map through `c8volt/incident/convert.go`.
- Public process parent context currently appears in `c8volt/process/model.go`, `c8volt/process/convert.go`, `c8volt/ops/convert.go`, and `c8volt/resource/convert.go`.
- v8.8/v8.9 generated Camunda clients already expose `elementId`, `elementInstanceKey`, and `parentElementInstanceKey` for many v2 paths; older generated Operate clients still contain `FlowNode*` fields.
- README and generated docs currently contain `--flow-node-id` and flow-node wording for incident filters.
- `cmd/get_incident.go` registers legacy incident search flags as `--flow-node-id` and `--fni-key`; validation treats `--fni-key` as a key-like value and search-mode detection includes both legacy flag names.
- Incident human output flows through `flatRowProcessInstanceIncidentWithTimezone` and `flatRowIncidentWithTimezone` in `cmd/cmd_views_processinstance_incidents.go`, which currently emit `fn:` and `fni:` labels.
- Ops repair and purge workflows each define their own incident filter flag variables and map them into `incident.Filter`, so canonical filter changes must cover both command files instead of assuming `get incident` owns all registrations.
- Command contract help assertions already use canonical job element flags, but incident help expectations still require `--flow-node-id` and `--fni-key`.
- Public docs expose legacy terms in README, generated `docs/cli/c8volt_get_incident.md`, generated ops command pages, `docs/index.md`, and non-generated `docs/ops/repair-incident.md`.

## Planning Notes

- Clarification gate completed with no formal questions; issue #233 explicitly defines canonical names, forbidden aliases, and adapter boundary rules.
- Architecture memory was reused because the existing command/facade/domain/service/generated-client and docs-generation boundaries already cover this feature.
- Ralph launch must include `--implementation-context specs/ralph-implementation-rules.md`.

## Implementation Status

- Speckit specification: complete.
- Clarification: complete; no questions asked.
- Architecture grounding: complete; existing memory reused.
- Planning artifacts: complete.
- Tasks: complete.

---
## Iteration 1 - 2026-05-25 18:53:51 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Inspect current incident command flags and validation in `cmd/get_incident.go`
- [x] T002: Inspect current incident and process human renderers in `cmd/cmd_views_processinstance_incidents.go` and nearby `cmd/cmd_views_*.go`
- [x] T003: Inspect public incident and process models/converters in `c8volt/incident/`, `c8volt/process/`, `c8volt/ops/`, and `c8volt/resource/`
- [x] T004: Inspect internal domain and service mappings in `internal/domain/`, `internal/services/incident/`, and `internal/services/processinstance/`
- [x] T005: Inspect ops incident filter reuse in `cmd/ops_repair_incident*.go` and `cmd/ops_purge_processinstances_with_incidents*.go`
- [x] T006: Inspect command contract expectations in `cmd/command_contract_test.go`
- [x] T007: Inspect documentation surfaces in `README.md`, `docs/cli/`, `docs/ops/`, and `docs/index.md`
- [x] T008: Record discovered ownership notes in `specs/233-element-terminology/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/233-element-terminology/tasks.md
- specs/233-element-terminology/progress.md
**Learnings**:
- Phase 1 was discovery-only; no source behavior changed in this iteration.
- Issue traceability is persisted in `spec.md` and `contracts/cli-element-terminology.md`, so the work-unit commit subject should end with `#233`.
- Existing `progress.md` status says tasks are complete, but `tasks.md` is the authoritative task checklist for Ralph iteration state.
---
---
## Iteration 2 - 2026-05-25 19:00:12 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T009: Add failing model/converter tests for canonical incident fields in `c8volt/incident/client_test.go`
- [x] T010: Add failing model/converter tests for canonical process parent fields in `c8volt/process/client_test.go`
- [x] T011: Add failing domain/service conversion tests for canonical incident fields in `internal/services/incident/v87/`, `internal/services/incident/v88/`, and `internal/services/incident/v89/`
- [x] T012: Rename public incident filter/result fields from flow-node terms to element terms in `c8volt/incident/model.go`, `c8volt/incident/convert.go`, and `internal/domain/incident.go`
- [x] T013: Rename public process parent fields from flow-node terms to element terms in `c8volt/process/model.go`, `c8volt/process/convert.go`, `c8volt/ops/convert.go`, and `c8volt/resource/convert.go`
- [x] T014: Update incident service adapter conversions while keeping generated legacy names adapter-only in `internal/services/incident/v87/`, `internal/services/incident/v88/`, and `internal/services/incident/v89/`
- [x] T015: Run targeted compile validation for shared model changes in `c8volt/incident`, `c8volt/process`, `c8volt/ops`, `c8volt/resource`, and `internal/services/incident`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/incident/model.go
- c8volt/incident/convert.go
- c8volt/incident/client_test.go
- c8volt/process/model.go
- c8volt/process/convert.go
- c8volt/process/client_test.go
- c8volt/ops/convert.go
- c8volt/resource/convert.go
- internal/domain/incident.go
- internal/domain/processinstance.go
- internal/services/incident/
- internal/services/processinstance/
- cmd/
- specs/233-element-terminology/tasks.md
- specs/233-element-terminology/progress.md
**Learnings**:
- The public facade rename forces command code to reference canonical struct fields immediately, even before later stories rename flags and compact human labels.
- The v8.7 process-instance generated client still uses `ParentFlowNodeInstanceKey`, so only the adapter input side keeps that legacy name while domain output is canonical.
- Validation passed with `go test ./c8volt/incident ./c8volt/process ./c8volt/ops ./c8volt/resource ./internal/services/incident/... ./internal/services/processinstance/... ./cmd`.
---
