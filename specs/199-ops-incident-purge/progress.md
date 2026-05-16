# Progress: Ops Purge Process Instances With Incidents

**Issue**: [#199](https://github.com/grafvonb/c8volt/issues/199)
**Feature**: `199-ops-incident-purge`
**Plan**: [plan.md](plan.md)
**Tasks**: [tasks.md](tasks.md)
**Mandatory Ralph Context**: `specs/ralph-implementation-rules.md`
**Commit Rule**: Conventional Commit subject ending in `#199`

## Codebase Patterns

- Mandatory Ralph implementation context is `specs/ralph-implementation-rules.md`; commit subjects for this feature must use Conventional Commits and end with `#199`.
- Existing #186 ops purge command wiring lives in `cmd/ops_purge_orphan_processinstances.go`: command files own Cobra flags, validation, `NewCli`, automation support checks, report-path planning validation, confirmation, facade calls, report writing, and final rendering.
- Existing #187 retention workflow in `cmd/ops_execute_retention_policy.go`, `internal/services/ops/retention_policy.go`, and `cmd/cmd_views_ops_execute_retention_policy.go` is the closest pattern for freeze-then-delete-plan behavior.
- Ops workflow command metadata uses `setCommandMutation`, `setContractSupport`, `setAutomationSupport`, and `setFlagContractRequired` from `cmd/command_contract.go`; capability regression tests live in `cmd/command_contract_test.go` and related command tests.
- Shared ops report helpers in `cmd/ops_contract.go` own report format inference, validation, overwrite mode, and secure file writes; workflow-specific Markdown and JSON rendering lives beside each workflow view.
- Incident command behavior and filters are centered in `cmd/get_incident.go`, `cmd/get_incident_search.go`, `c8volt/incident`, and `internal/services/incident`; #199 should reuse those semantics for candidate discovery.
- `get incident` keeps search selection and display-only rendering flags separate; #199 should register only the search subset and must not inherit `--pi-keys-only`, `--total`, `--error-message-limit`, or `--with-no-error-message`.
- Existing delete planning is `processinstance.DryRunCancelOrDeletePlan`, which deduplicates selected keys, resolves roots through ancestry traversal, expands descendants, preserves missing ancestor/traversal warnings, records selected final-state items and non-final affected items, and validates unresolved plans.
- Existing delete execution uses `processinstance.DeleteProcessInstances`, `toolx.DetermineNoOfWorkers`, `toolx/pool.ExecuteSlice`, `services.ApplyCallOptions`, `FailFast`, `NoWorkerLimit`, `NoWait`, and logger/activity helpers; command code should pass controls rather than owning worker logic.
- `internal/services/ops.New` now explicitly accepts process-instance and incident service dependencies; `c8volt/client.go` wires the existing incident service into ops so future discovery can stay behind the ops service boundary.
- Human ops output follows a compact sequence of workflow status lines; detailed key lists are gated by verbose output while JSON and reports keep complete structured data.
- Generated CLI docs are protected by `docsgen/main_test.go` and produced via `make docs-content`; generated files under `docs/cli/` and `docs/index.md` should not be hand-edited after command metadata changes.
- Incident purge uses dedicated command flag globals in `cmd/ops_purge_processinstances_with_incidents.go`; `--key` is carried as `incident.Filter.Keys`/`domain.IncidentFilter.Keys` so US2 can distinguish incident-key discovery from process-instance-key filters.
- Incident purge discovery now runs through `internal/services/incident.SearchIncidents`, with `--limit` used as the candidate incident cap before process-instance dedupe and `--batch-size` used as the default search size when no limit is set.
- US2 intentionally leaves delete planning and deletion skipped after discovery; `cmd/cmd_views_ops_purge_processinstances_with_incidents.go` owns compact human discovery output while `--json` uses the shared command envelope with the complete `ops.IncidentPurgeResult`.
- `c8volt/incident.Filter.String()` now provides compact selection filter rendering for incident-based ops output.

## Status

- Speckit specification created from GitHub issue #199.
- Clarification gate completed with no critical ambiguities worth formal questioning.
- Planning artifacts generated for Ralph-sized implementation work.

## Ralph Notes

- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, `spec.md`, `plan.md`, `tasks.md`, and this progress file before code changes.
- Each iteration should complete only the first incomplete work unit and update this file with validation results.
- Existing #186/#187 ops purge/report/delete-plan code is expected to be the closest implementation pattern.

---

---
## Iteration 3 - 2026-05-16 10:48:41 CEST
**User Story**: User Story 1 - Register Incident Purge Command
**Tasks Completed**:
- [x] T012: Add command registration, help, and alias tests for incident purge in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T013: Add unsupported display-only incident flag tests in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T014: Add command contract metadata tests for state-changing and automation support in `cmd/command_contract_test.go`
- [x] T015: Add `ops purge process-instances-with-incidents` Cobra command, alias, summary, and safe examples in `cmd/ops_purge_processinstances_with_incidents.go`
- [x] T016: Register supported incident selection flags and delete workflow flags in `cmd/ops_purge_processinstances_with_incidents.go`
- [x] T017: Map static flag validation failures through existing invalid-input helpers in `cmd/ops_purge_processinstances_with_incidents.go`
- [x] T018: Set mutation, output-mode, required flag, and automation metadata in `cmd/ops_purge_processinstances_with_incidents.go`
- [x] T019: Mark US1 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/incident/model.go
- c8volt/ops/convert.go
- cmd/command_contract_test.go
- cmd/ops_purge_processinstances_with_incidents.go
- cmd/ops_purge_processinstances_with_incidents_test.go
- internal/domain/incident.go
- specs/199-ops-incident-purge/progress.md
- specs/199-ops-incident-purge/tasks.md
**Learnings**:
- `ops purge process-instances-with-incidents` can reuse the existing ops command contract helpers directly, with explicit one-line and JSON output metadata while full human rendering remains scheduled for later stories.
- The incident purge command must not register display-only `get incident` flags; unsupported flag tests assert `--pi-keys-only`, `--total`, `--error-message-limit`, and `--with-no-error-message` stay outside the command surface.
- Validation run: `go test ./cmd -run 'TestOpsPurgeProcessInstancesWithIncidents|TestCommandCapabilityForCommand_OpsPurgeProcessInstancesWithIncidentsContract' -count=1`; `go test ./c8volt/ops -run 'TestClientPurgeProcessInstancesWithIncidents' -count=1`; `go test ./internal/services/ops -run 'TestPurgeProcessInstancesWithIncidents' -count=1`; `go test ./c8volt/incident -count=1`; `go test ./cmd -run 'TestOpsPurge|TestCommandContract' -count=1`; `go test ./cmd ./c8volt/ops ./c8volt/incident ./internal/services/ops -count=1`; `go test ./... -count=1`.
---
## Iteration 1 - 2026-05-16 10:30:28 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Record mandatory Ralph context and issue traceability in `specs/199-ops-incident-purge/progress.md`
- [x] T002: Inspect existing #186 ops purge, #187 retention delete-plan flow, incident search, process-instance delete planning, command contract metadata, and docs generation patterns; record reusable discoveries in `specs/199-ops-incident-purge/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/199-ops-incident-purge/progress.md
- specs/199-ops-incident-purge/tasks.md
**Learnings**:
- #199 has persisted GitHub issue traceability in `spec.md`, `plan.md`, and `progress.md`; completed work-unit commit subjects must end with `#199`.
- `cmd/ops_purge_orphan_processinstances.go` and `cmd/ops_execute_retention_policy.go` both validate local/report inputs before remote work, plan with `DryRun`, confirm through `shouldImplicitlyConfirm(cmd)`, then reuse frozen discovered keys for the confirmed service call.
- `internal/services/processinstance.DryRunCancelOrDeletePlan` is the delete-plan source of truth for dedupe, root resolution, descendant expansion, missing ancestor warnings, final-state selected items, and non-final blockers; incident purge should adapt candidate keys into this path.
- `internal/services/processinstance.DeleteProcessInstances` is the delete execution source of truth for worker count, fail-fast, no-worker-limit, no-wait logging, deterministic order, and activity reporting.
- Incident discovery should follow `get incident` search flag validation and paging semantics while excluding display-only output flags from the purge command surface.
- Command metadata must be set with `setCommandMutation`, `setContractSupport`, `setAutomationSupport`, and `setFlagContractRequired` where applicable; docs are regenerated from Cobra metadata with `make docs-content`.
---

---
## Iteration 2 - 2026-05-16 10:39:17 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T003: Define internal incident purge request/result domain models in `internal/domain/ops_incident_purge.go`
- [x] T004: Define public ops incident purge request/result models in `c8volt/ops/model.go`
- [x] T005: Extend public ops facade API for incident purge in `c8volt/ops/api.go`
- [x] T006: Extend internal ops service interface for incident purge in `internal/services/ops/api.go`
- [x] T007: Implement public/internal incident purge model conversions in `c8volt/ops/convert.go`
- [x] T008: Implement thin public ops facade incident purge method in `c8volt/ops/client.go`
- [x] T009: Add foundational ops facade wiring tests for incident purge in `c8volt/ops/client_test.go`
- [x] T010: Add foundational internal ops service validation tests for incident purge in `internal/services/ops/incident_purge_test.go`
- [x] T011: Mark Phase 2 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/client.go
- c8volt/ops/api.go
- c8volt/ops/client.go
- c8volt/ops/client_test.go
- c8volt/ops/convert.go
- c8volt/ops/model.go
- internal/domain/ops_incident_purge.go
- internal/services/ops/api.go
- internal/services/ops/incident_purge.go
- internal/services/ops/incident_purge_test.go
- internal/services/ops/orphan_purge_test.go
- internal/services/ops/retention_policy_test.go
- specs/199-ops-incident-purge/progress.md
- specs/199-ops-incident-purge/tasks.md
**Learnings**:
- Incident purge can reuse the existing ops facade pattern: public requests map mechanically to `internal/domain`, facade options map through `options.MapFacadeOptionsToCallOptions`, and domain errors normalize through `ferrors.FromDomain`.
- The foundational service method currently records request controls and initializes the report/result shape without discovery or mutation; US2 should replace the skipped no-target placeholder with `internal/services/incident.SearchIncidents`-backed discovery.
- Because `internal/services/ops.New` now requires both process-instance and incident services, existing ops service tests pass `nil` for the incident service unless they exercise incident purge.
- Validation run: `go test ./c8volt/ops -run 'TestClient.*Incident|TestClientPurgeOrphan|TestClientExecuteRetentionPolicy' -count=1`; `go test ./internal/services/ops -run 'TestPurgeProcessInstancesWithIncidents|TestNewCreatesOrphan|TestPurgeOrphan|TestExecuteRetentionPolicy' -count=1`; `go test ./c8volt/ops ./internal/services/ops -count=1`; `go test ./c8volt -run Test -count=1`; `go test ./... -count=1`.
---

---
## Iteration 4 - 2026-05-16 10:58:57 CEST
**User Story**: User Story 2 - Discover And Freeze Candidate Process Instances
**Tasks Completed**:
- [x] T020: Add incident discovery service or ops service tests for candidate extraction, duplicate detection, skipped incidents, and limit handling in `internal/services/ops/incident_purge_test.go`
- [x] T021: Add command dry-run discovery output tests in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T022: Add facade conversion tests for incident discovery result fields in `c8volt/ops/client_test.go`
- [x] T023: Reuse existing incident search semantics to discover candidate incidents in `internal/services/ops/incident_purge.go`
- [x] T024: Extract, deduplicate, and freeze candidate process-instance keys in `internal/services/ops/incident_purge.go`
- [x] T025: Preserve candidate incidents, duplicate candidate process instances, skipped incidents, notices, and errors in `internal/domain/ops_incident_purge.go`
- [x] T026: Map discovery request/result through `c8volt/ops/client.go` and `c8volt/ops/convert.go`
- [x] T027: Render compact discovery output and complete JSON discovery data in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T028: Mark US2 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/incident/model.go
- c8volt/ops/client_test.go
- cmd/cmd_views_ops_purge_processinstances_with_incidents.go
- cmd/ops_purge_processinstances_with_incidents.go
- cmd/ops_purge_processinstances_with_incidents_test.go
- internal/services/ops/incident_purge.go
- internal/services/ops/incident_purge_test.go
- specs/199-ops-incident-purge/progress.md
- specs/199-ops-incident-purge/tasks.md
**Learnings**:
- `internal/services/incident.SearchIncidents` is the correct reuse point for US2 because it preserves the existing version-specific incident search and local filtering behavior.
- Candidate extraction keeps all candidate incidents for JSON/report data, freezes unique process-instance keys for later planning, records duplicate process-instance keys once, and skips incidents missing process-instance keys with a structured reason.
- Validation run: `go test ./internal/services/ops -run 'TestPurgeProcessInstancesWithIncidents' -count=1`; `go test ./c8volt/ops -run 'TestClientPurgeProcessInstancesWithIncidents' -count=1`; `go test ./cmd -run 'TestOpsPurgeProcessInstancesWithIncidentsDryRun|TestOpsPurgeProcessInstancesWithIncidentsInvalid|TestOpsPurgeProcessInstancesWithIncidentsHelp' -count=1`; `go test ./cmd ./c8volt/ops ./c8volt/incident ./internal/services/ops ./internal/services/incident -count=1`; `go test ./... -count=1` with `GOCACHE=/private/tmp/codex-go-build`.
---
