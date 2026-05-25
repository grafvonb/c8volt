# Progress: Element Terminology Standardization

## Codebase Patterns

- Process-instance parent context is now canonical end-to-end as `ParentElementInstanceKey` / `parentElementInstanceKey`; only generated clients or v8.7 adapter reads may keep `ParentFlowNodeInstanceKey`.
- `cmd/get_processinstance_test.go` and `cmd/walk_test.go` are suitable focused command contract surfaces for asserting `parentFlowNodeInstanceKey` is absent from public JSON output.
- Shared incident human rows are rendered only through `flatRowProcessInstanceIncidentWithTimezone` and `flatRowIncidentWithTimezone`; canonical compact labels are now `e:` and `ei:`.
- Public incident facade/domain fields are now `ElementId` and `ElementInstanceKey`; public JSON tags are `elementId` and `elementInstanceKey`.
- Public process facade/domain parent context is now `ParentElementInstanceKey`; the v8.7 process-instance adapter still reads generated `ParentFlowNodeInstanceKey` at the adapter boundary.
- Command code must reference canonical facade fields after the foundational rename, even where later user stories still own public flag/help and compact-label behavior changes.
- `cmd/get_incident.go` owns incident search flag grammar; `get incident` now exposes canonical `--element-id` and `--element-instance-key` flags and leaves legacy `--flow-node-id`/`--fni-key` unregistered so Cobra rejects them as unknown.
- `cmd/cmd_views_processinstance_incidents.go` currently renders incident context labels with `fn` and `fni`.
- Public incident fields currently live in `c8volt/incident/model.go` and map through `c8volt/incident/convert.go`.
- Public process parent context currently appears in `c8volt/process/model.go`, `c8volt/process/convert.go`, `c8volt/ops/convert.go`, and `c8volt/resource/convert.go`.
- v8.8/v8.9 generated Camunda clients already expose `elementId`, `elementInstanceKey`, and `parentElementInstanceKey` for many v2 paths; older generated Operate clients still contain `FlowNode*` fields.
- README and generated docs currently contain `--flow-node-id` and flow-node wording for incident filters.
- `cmd/get_incident.go` validates `--element-instance-key` as a key-like value and includes canonical element flags in search-mode detection; human output still uses `fn:`/`fni:` until US2 owns renderer changes.
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
---
## Iteration 3 - 2026-05-25 19:03:56 CEST
**User Story**: User Story 1 - Filter Incidents With Element Terminology
**Tasks Completed**:
- [x] T016: Add command tests for `get incident --element-id` and `--element-instance-key` in `cmd/get_incident_test.go`
- [x] T017: Add command tests proving `--flow-node-id` and `--fni-key` are unknown flags in `cmd/get_incident_test.go`
- [x] T018: Add command contract tests for canonical incident flags and absence of old flags in `cmd/command_contract_test.go`
- [x] T019: Add facade mapping tests for canonical incident filter fields in `c8volt/incident/client_test.go`
- [x] T020: Add v8.8/v8.9 incident service tests for element filter construction or compatibility filtering in `internal/services/incident/v88/incidents_test.go` and `internal/services/incident/v89/incidents_test.go`
- [x] T021: Replace legacy incident filter flag variables and registrations with canonical flags in `cmd/get_incident.go`
- [x] T022: Update incident command validation, filter assembly, examples, and reset logic in `cmd/get_incident.go`
- [x] T023: Update command metadata expectations for canonical incident filter flags in `cmd/command_contract_test.go`
- [x] T024: Update incident facade and service filter field names used by `get incident` in `c8volt/incident/` and `internal/domain/incident.go`
- [x] T025: Update v8.8/v8.9 incident filter mapping and v8.7 compatibility mapping in `internal/services/incident/`
- [x] T026: Verify US1 with targeted tests for `cmd/get_incident_test.go`, `cmd/command_contract_test.go`, `c8volt/incident/client_test.go`, and `internal/services/incident/...`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_incident.go
- cmd/get_incident_test.go
- cmd/command_contract_test.go
- specs/233-element-terminology/tasks.md
- specs/233-element-terminology/progress.md
**Learnings**:
- `get incident` can drop legacy aliases entirely by not registering them; Cobra reports `unknown flag` before config loading or remote work.
- US1 validation passed with targeted command, facade, and incident service packages plus broader `go test ./cmd ./c8volt/incident ./internal/services/incident/...`.
- The foundational facade/domain/service element-field tests already cover T019/T020 behavior, so this iteration only needed command-surface changes.
---
---
## Iteration 4 - 2026-05-25 19:10:35 CEST
**User Story**: User Story 2 - Show Canonical Incident Context
**Tasks Completed**:
- [x] T027: Add JSON output tests for canonical incident fields in `cmd/get_incident_test.go`
- [x] T028: Add human row rendering tests for `e:` and `ei:` labels in `cmd/cmd_views_get_test.go`
- [x] T029: Add `get pi --with-incidents` output tests for canonical incident context in `cmd/get_processinstance_test.go`
- [x] T030: Add `walk pi --with-incidents` output tests for canonical incident context in `cmd/walk_test.go`
- [x] T031: Add ops repair/purge output regression tests for canonical incident context in `cmd/ops_repair_incident_test.go` and `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T032: Rename incident JSON fields and converter outputs in `c8volt/incident/model.go`, `c8volt/incident/convert.go`, and `internal/domain/incident.go`
- [x] T033: Update process-instance incident detail conversions in `c8volt/process/convert.go`, `c8volt/ops/convert.go`, and `internal/domain/`
- [x] T034: Replace human labels `fn` and `fni` with `e` and `ei` in `cmd/cmd_views_processinstance_incidents.go`
- [x] T035: Update command output assertions and fixtures that consume incident JSON in `cmd/get_incident_test.go`, `cmd/get_processinstance_test.go`, `cmd/walk_test.go`, and ops command tests
- [x] T036: Verify US2 with targeted tests for incident rendering, process-instance incident output, walk output, and ops output in `cmd/`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_processinstance_incidents.go
- cmd/cmd_views_get_test.go
- cmd/get_incident_test.go
- cmd/get_processinstance_test.go
- cmd/walk_test.go
- cmd/ops_repair_incident_test.go
- cmd/ops_purge_processinstances_with_incidents_test.go
- specs/233-element-terminology/tasks.md
- specs/233-element-terminology/progress.md
**Learnings**:
- Incident JSON fields and facade/domain conversions were already canonical from the foundational work; US2 focused on command output coverage and shared human render labels.
- Ops repair and purge machine/report payloads include incident details through shared `ProcessInstanceIncidentDetail`, so canonical JSON regression coverage belongs in those command tests even before US4 renames ops filter flags.
- Validation passed with `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd ./c8volt/incident ./c8volt/process ./c8volt/ops`.
---
---
## Iteration 5 - 2026-05-25 19:16:52 CEST
**User Story**: User Story 3 - Standardize Process Context Fields
**Tasks Completed**:
- [x] T037: Add process facade tests for `parentElementInstanceKey` in `c8volt/process/client_test.go`
- [x] T038: Add resource facade regression tests for renamed parent context in `c8volt/resource/client_test.go`
- [x] T039: Add process-instance command JSON tests for `parentElementInstanceKey` in `cmd/get_processinstance_test.go`
- [x] T040: Add walk command tests for canonical parent context in `cmd/walk_test.go`
- [x] T041: Add focused command-output checks proving `parentFlowNodeInstanceKey` is absent from public JSON output
- [x] T042: Rename public process parent fields in `c8volt/process/model.go` and `c8volt/process/convert.go`
- [x] T043: Rename parent context mappings in `c8volt/resource/convert.go` and `c8volt/ops/convert.go`
- [x] T044: Update internal domain process-instance parent context names in `internal/domain/` and affected service conversions in `internal/services/processinstance/`
- [x] T045: Update command views and JSON fixtures that render parent process context in `cmd/`
- [x] T046: Verify US3 with targeted tests for `c8volt/process`, `c8volt/resource`, `internal/services/processinstance/...`, and `cmd`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/resource/client_test.go
- cmd/get_processinstance_test.go
- cmd/walk_test.go
- specs/233-element-terminology/tasks.md
- specs/233-element-terminology/progress.md
**Learnings**:
- The model, converter, domain, and process-instance service layers were already canonical from the foundational work; US3 added missing public JSON regression coverage for process parent context.
- Resource delete-plan conversion preserves canonical parent context through `process.DryRunPIKeyExpansion`, and JSON output omits `parentFlowNodeInstanceKey`.
- Validation passed with `GOCACHE=/private/tmp/c8volt-go-build go test ./c8volt/process ./c8volt/resource ./internal/services/processinstance/... ./cmd`.
---
