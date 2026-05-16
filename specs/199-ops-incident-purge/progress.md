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
- `cmd/cmd_views_ops_purge_processinstances_with_incidents.go` owns compact human discovery and plan output while `--json` uses the shared command envelope with the complete `ops.IncidentPurgeResult`.
- `c8volt/incident.Filter.String()` now provides compact selection filter rendering for incident-based ops output.
- Incident purge delete planning now reuses `internal/services/processinstance.DryRunCancelOrDeletePlan` from frozen candidate process-instance keys; dry-run renders the plan, while destructive runs without `--force` block on non-final affected instances before mutation.
- Incident purge plan output preserves candidate duplicates separately from duplicate resolved roots through `DuplicateCandidateProcessInstanceKeys` and `DuplicateResolvedRootKeys`.
- Confirmed incident purge execution now delegates to `internal/services/processinstance.DeleteProcessInstances`, preserving submitted root keys, no-wait status, delete reports, confirmation state, and deleted/partial/failed outcomes.
- Unconfirmed destructive incident purge runs first execute a dry-run preplan, use `shouldImplicitlyConfirm(cmd)` for the prompt decision, and pass `DiscoveredCandidateProcessInstanceKeys` into the confirmed service call so incident discovery is not rerun after confirmation.
- Incident purge report handling now mirrors the orphan/retention ops workflow lifecycle: validate report paths before remote work, write reports before human rendering, preserve existing files until deletion is submitted, and record local aborts in the report shape when possible.
- Incident purge Markdown and JSON reports are rendered from `ops.IncidentPurgeReport` in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`; compact human output stays count-oriented while verbose mode owns full key lists.
- US6 delete-pi regression coverage belongs in existing `cmd/delete_test.go`; this repository does not currently have a separate `cmd/delete_processinstance_test.go` despite the task label.
- `make docs-content` generates the incident-purge command page at `docs/cli/c8volt_ops_purge_process-instances-with-incidents.md` and refreshes the purge index and docs build metadata from Cobra command source.

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

---
## Iteration 5 - 2026-05-16 11:09:26 CEST
**User Story**: User Story 3 - Build Incident Purge Delete Plan
**Tasks Completed**:
- [x] T029: Add ops service delete-plan tests for candidate keys, resolved roots, affected keys, and duplicate handling in `internal/services/ops/incident_purge_test.go`
- [x] T030: Add non-final affected instance blocking test in `internal/services/ops/incident_purge_test.go`
- [x] T031: Add command dry-run plan rendering tests in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T032: Reuse existing process-instance delete planning from frozen candidate keys in `internal/services/ops/incident_purge.go`
- [x] T033: Preserve missing ancestor, traversal warning, duplicate, final-state, non-final, and semantic notice details in `internal/domain/ops_incident_purge.go`
- [x] T034: Map delete-plan details through `c8volt/ops/convert.go`
- [x] T035: Render compact delete-plan human output and complete JSON output in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T036: Mark US3 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/ops/client_test.go
- c8volt/ops/convert.go
- c8volt/ops/model.go
- cmd/cmd_views_ops_purge_processinstances_with_incidents.go
- cmd/ops_purge_processinstances_with_incidents_test.go
- internal/domain/ops_incident_purge.go
- internal/services/ops/incident_purge.go
- internal/services/ops/incident_purge_test.go
- specs/199-ops-incident-purge/progress.md
- specs/199-ops-incident-purge/tasks.md
**Learnings**:
- `DryRunCancelOrDeletePlan` is the right US3 reuse point because it already preserves root dedupe, affected scope expansion, selected final-state items, non-final affected items, missing ancestors, and traversal warnings.
- Existing US2 command discovery tests needed plan-aware expected output because dry-run now includes a planned delete-plan step whenever candidates are found.
- Validation run: `go test ./internal/services/ops -run 'TestPurgeProcessInstancesWithIncidents' -count=1`; `go test ./c8volt/ops -run 'TestClientPurgeProcessInstancesWithIncidents' -count=1`; `go test ./cmd -run 'TestOpsPurgeProcessInstancesWithIncidentsDryRunPlanRendering|TestOpsPurgeProcessInstancesWithIncidentsDryRunDiscoveryOutput|TestOpsPurgeProcessInstancesWithIncidentsDryRunJSONDiscoveryData' -count=1`; `go test ./cmd -run 'TestOpsPurgeProcessInstancesWithIncidents' -count=1`; `go test ./internal/services/ops -count=1`; `go test ./c8volt/ops -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./internal/services/ops -run 'TestPurgeProcessInstancesWithIncidents' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance -count=1`.
---

---
## Iteration 6 - 2026-05-16 11:20:03 CEST
**User Story**: User Story 4 - Execute Confirmed Purge Through Delete Plan
**Tasks Completed**:
- [x] T037: Add confirmed deletion command test for exact frozen-plan root submission in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T038: Add execution-control mapping tests for workers, fail-fast, no-worker-limit, no-wait, and force in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T039: Add `--automation --json` without `--auto-confirm` success test for supported state-changing incident purge in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T040: Add local-precondition failure subprocess tests for post-planning blockers and exit code in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T041: Execute deletion through existing process-instance deletion service from `internal/services/ops/incident_purge.go`
- [x] T042: Use `shouldImplicitlyConfirm(cmd)` for destructive confirmation decisions in `cmd/ops_purge_processinstances_with_incidents.go`
- [x] T043: Preserve no-wait, confirmation, per-key or per-batch status, and final outcome in `internal/domain/ops_incident_purge.go`
- [x] T044: Render deletion execution and final outcome in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T045: Mark US4 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_purge_processinstances_with_incidents.go
- cmd/ops_purge_processinstances_with_incidents.go
- cmd/ops_purge_processinstances_with_incidents_test.go
- internal/services/ops/incident_purge.go
- internal/services/ops/incident_purge_test.go
- specs/199-ops-incident-purge/progress.md
- specs/199-ops-incident-purge/tasks.md
**Learnings**:
- `internal/services/processinstance.DeleteProcessInstances` is the correct execution reuse point because it already owns root dedupe, worker selection, fail-fast scheduling, no-wait behavior, force/cancel-before-delete semantics, deterministic report order, and activity/log output.
- For unconfirmed destructive runs, the command must dry-run plan first, block local non-final affected scope before prompting, then execute with frozen candidate process-instance keys to avoid a second incident search expanding the confirmed scope.
- Validation run: `GOCACHE=/private/tmp/codex-go-build go test ./internal/services/ops -run 'TestPurgeProcessInstancesWithIncidents' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./cmd -run 'TestOpsPurgeProcessInstancesWithIncidents(ConfirmedDeletionUsesFrozenPlanRoots|AutomationJSONExecutesWithoutAutoConfirm|BlocksNonFinalScopeBeforeMutation|DryRun|Invalid|Help)' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./cmd -run 'TestOpsPurgeProcessInstancesWithIncidents' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./internal/services/ops -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./c8volt/ops -run 'TestClientPurgeProcessInstancesWithIncidents' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./... -count=1`.
---

---
## Iteration 7 - 2026-05-16 11:27:38 CEST
**User Story**: User Story 5 - Produce Compact Output, Complete Reports, And Automation-Safe JSON
**Tasks Completed**:
- [x] T046: Add verbose key-list output tests for incident, candidate, root, affected, and skipped keys in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T047: Add deterministic `--dry-run --json` and `--automation --json` output tests in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T048: Add Markdown incident purge report rendering test in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T049: Add JSON incident purge report rendering test in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T050: Add existing report-file preservation tests for dry-run, unconfirmed, and locally blocked runs in `cmd/ops_purge_processinstances_with_incidents_test.go`
- [x] T051: Reuse shared ops report-file validation, format inference, overwrite safety, and file writing in `cmd/ops_purge_processinstances_with_incidents.go`
- [x] T052: Extend report model/rendering for incident discovery, candidate set, plan, deletion, notices, errors, and outcome fields in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T053: Keep normal human output compact and gate detailed key lists behind verbose output in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T054: Print compact `report: written <path>` human output after report writes in `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`
- [x] T055: Mark US5 tasks complete and record validation notes in `specs/199-ops-incident-purge/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_purge_processinstances_with_incidents.go
- cmd/ops_purge_processinstances_with_incidents.go
- cmd/ops_purge_processinstances_with_incidents_test.go
- specs/199-ops-incident-purge/progress.md
- specs/199-ops-incident-purge/tasks.md
**Learnings**:
- Incident purge can reuse the existing shared report helpers directly while using command-specific rendering for the complete incident discovery, delete-plan, deletion, notice, and error report fields.
- Existing report-file preservation depends on write mode as well as preflight validation: dry-run and unconfirmed runs fail before remote work when a report exists, while a locally blocked auto-confirmed run still preserves the file because no deletion was submitted.
- Validation run: `GOCACHE=/private/tmp/codex-go-build go test ./cmd -run 'TestOpsPurgeProcessInstancesWithIncidents(Verbose|JSONOutputs|WritesMarkdownReport|WritesJSONReport|ExistingReportPreservation)' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./cmd -run 'TestOpsPurgeProcessInstancesWithIncidents' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./... -count=1`.
---

---
## Iteration 8 - 2026-05-16 11:34:54 CEST
**User Story**: User Story 6 - Preserve Documentation And Regression Contracts
**Tasks Completed**:
- [x] T056: Add regression tests for unchanged `get incident` selection and display-only behavior in `cmd/get_incident_test.go`
- [x] T057: Add regression tests for unchanged `delete pi` hierarchy planning and force/no-wait behavior in `cmd/delete_test.go`
- [x] T058: Add docs/contract assertions for incident purge command metadata in `cmd/command_contract_test.go`
- [x] T059: Update user-facing help examples for incident purge in `cmd/ops_purge_processinstances_with_incidents.go`
- [x] T060: Run `make docs-content` and review generated files under `docs/cli/` and `docs/index.md`
- [x] T061: Run targeted command tests with `go test ./cmd -run 'TestOpsPurge|TestCommandContract|TestDeleteProcessInstance|TestGetIncident' -count=1`
- [x] T062: Run facade and service tests with `go test ./c8volt/ops ./c8volt/incident ./c8volt/process ./internal/services/ops ./internal/services/incident ./internal/services/processinstance -count=1`
- [x] T063: Run repository validation with `make test`
- [x] T064: Mark US6 tasks complete and record final validation notes in `specs/199-ops-incident-purge/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/command_contract_test.go
- cmd/delete_test.go
- cmd/get_incident_test.go
- cmd/ops_purge_processinstances_with_incidents.go
- docs/cli/c8volt_ops_purge.md
- docs/cli/c8volt_ops_purge_process-instances-with-incidents.md
- docs/index.md
- specs/199-ops-incident-purge/progress.md
- specs/199-ops-incident-purge/tasks.md
**Learnings**:
- `get incident` retains display-only behavior through its own command surface; incident purge only reuses the selection subset and keeps display-only flags unknown there.
- `delete pi` regression coverage should stay in `cmd/delete_test.go`, where hierarchy planning, resolved root submission, force, no-wait, worker, fail-fast, and no-worker-limit behavior already live.
- Validation run: `GOCACHE=/private/tmp/codex-go-build go test ./cmd -run 'TestGetIncidentCommand_RegressionSelectionAndDisplayFlagsRemainDistinct' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./cmd -run 'TestDeleteProcessInstancesWithPlan_RegressionForceNoWaitAndWorkerControls' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./cmd -run 'TestCommandCapabilityForCommand_OpsPurgeProcessInstancesWithIncidentsContract' -count=1`; `make docs-content`; `GOCACHE=/private/tmp/codex-go-build go test ./cmd -run 'TestOpsPurge|TestCommandContract|TestDeleteProcessInstance|TestGetIncident' -count=1`; `GOCACHE=/private/tmp/codex-go-build go test ./c8volt/ops ./c8volt/incident ./c8volt/process ./internal/services/ops ./internal/services/incident ./internal/services/processinstance -count=1`; `GOCACHE=/private/tmp/codex-go-build make test`.
---
