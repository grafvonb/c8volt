# Ralph Progress Log

Feature: 181-resolve-incident-commands
Started: 2026-05-08 21:06:02

## Codebase Patterns

- Generated Camunda v8.8 and v8.9 clients expose `ResolveIncidentWithResponse`, `GetIncidentWithResponse`, `SearchProcessInstanceIncidentsWithResponse`, `ResolveProcessInstanceIncidentsWithResponse`, and batch incident resolution response helpers. Direct incident resolution responses carry status/body/problem fields but no success JSON payload.
- Existing incident service boundary exposes only `SearchProcessInstanceIncidents`; v8.7 rejects lookup as unsupported before mutation, while v8.8/v8.9 use `/v2/process-instances/{key}/incidents/search` with tenant filtering and map `IncidentResult` values to domain incident details.
- Mutation commands are Cobra leaves that validate optional stdin `-`, call `readKeysIfDash`, merge repeated flags and stdin with `mergeAndValidateKeys(...).Unique()`, gate automation through `requireAutomationSupport`, and declare metadata with `setCommandMutation`, `setContractSupport`, and `setAutomationSupport`.
- Existing process facade incident lookup tests verify option propagation, exact detail mapping, per-key association, and filtering out incident rows whose `ProcessInstanceKey` does not match the owning process instance.
- Existing command incident enrichment tests assert request ordering, direct incident rendering under each process-instance row, limit-aware lookup scheduling, and shared JSON envelope helpers for `get process-instance`.
- Incident resolution should use single-incident generated endpoints for per-incident mutation results, while process-instance resolution should discover active incident keys through `internal/services/incident` and then resolve those keys directly; no incident methods belong in `internal/services/processinstance`.
- Incident confirmation polling now follows the service waiter pattern: direct incident waits accept `404/not found` or non-active incident state as resolved, and process-instance waits poll the scoped incident lookup until the initially discovered active incident keys disappear.
- Expanding the exported process facade API requires updating test doubles in command/resource packages even before CLI wiring exists; keep those stubs panic-only unless a test explicitly exercises resolution.
- Resolve mutation command rendering should use shared JSON envelopes in machine mode and compact per-target human lines plus a totals line in one-line mode, matching update process-instance result rendering.
- `resolve process-instance` mirrors direct incident command input handling, but schedules facade `ResolveProcessInstancesIncidents` so commands never call incident lookup/resolution services directly and `internal/services/processinstance` remains free of incident resolution methods.
- Resolve dry-run support is already driven by facade `WithDryRun`; command leaves only need to expose `--dry-run`, pass `collectOptions()`, reject `--json --verbose`, and render planned/skipped results without submitting mutation.

## Iteration 1 - 2026-05-08 21:07:28 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Inspect generated incident resolution methods in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [x] T002: Inspect current mutation command patterns in `cmd/update_job.go`, `cmd/cancel_processinstance.go`, and `cmd/update_processinstance.go`
- [x] T003: Inspect existing incident lookup tests in `internal/services/incident/v87/incidents.go`, `internal/services/incident/v88/incidents.go`, `internal/services/incident/v89/incidents.go`, `c8volt/process/client_test.go`, and `cmd/get_processinstance_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - blocked by `.git` write restriction in this sandbox
**Files Changed**:
- specs/181-resolve-incident-commands/tasks.md
- specs/181-resolve-incident-commands/progress.md
**Learnings**:
- v8.8 and v8.9 generated incident resolution surfaces are effectively parallel, so future version implementations should stay symmetric unless tests prove a response contract difference.
- Direct incident resolution should likely use the generated single-incident endpoint for per-target reporting; the process-instance and batch resolution endpoints return batch-operation style responses and do not satisfy the planned per-incident reporting contract by themselves.
- Resolution implementation should extend `internal/services/incident` and facade stubs before command work, because current process-facing tests already rely on `incsvc.API` as the incident lookup boundary.
- `git add` and `git commit` currently fail because the sandbox cannot create `.git/index.lock`; the next iteration should commit this completed setup work once Git metadata writes are available.
---
---
## Iteration 2 - 2026-05-08 21:21:54 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T004: Add incident resolution domain result fields or reuse existing domain models in `internal/domain/processinstance.go`
- [x] T005: Extend the incident service API with incident resolution and state lookup methods in `internal/services/incident/api.go`
- [x] T006: Add v8.7 unsupported resolution tests in `internal/services/incident/v87/incidents_test.go`
- [x] T007: Add v8.8 resolution service tests in `internal/services/incident/v88/incidents_test.go`
- [x] T008: Add v8.9 resolution service tests in `internal/services/incident/v89/incidents_test.go`
- [x] T009: Implement unsupported v8.7 incident resolution behavior in `internal/services/incident/v87/incidents.go` and `internal/services/incident/v87/contract.go`
- [x] T010: Implement v8.8 incident resolution calls in `internal/services/incident/v88/incidents.go` and `internal/services/incident/v88/contract.go`
- [x] T011: Implement v8.9 incident resolution calls in `internal/services/incident/v89/incidents.go` and `internal/services/incident/v89/contract.go`
- [x] T012: Add incident service factory/API compile checks and version tests in `internal/services/incident/factory_test.go`
- [x] T013: Add process facade resolution plan/result models, dry-run status, mutation-submission status, and totals helpers in `c8volt/process/model.go`
- [x] T014: Extend the process facade API with incident and process-instance resolution methods in `c8volt/process/api.go`
- [x] T015: Implement versioned-service lookup polling for post-mutation confirmation, following the `update pi --vars` waiter pattern, in `internal/services/incident`
- [x] T016: Implement facade resolution orchestration and wait result mapping in `c8volt/process/client.go` and `c8volt/process/bulk.go`
- [x] T017: Add facade tests for direct incident resolution, process-instance discovery, dry-run plans, lookup-polling confirmation, partial failures, `--no-wait`, and unsupported errors in `c8volt/process/client_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/api.go
- c8volt/process/client_test.go
- c8volt/process/model.go
- c8volt/process/resolve.go
- c8volt/resource/client_test.go
- cmd/process_api_stub_test.go
- internal/domain/processinstance.go
- internal/services/incident/api.go
- internal/services/incident/factory_test.go
- internal/services/incident/v87/contract.go
- internal/services/incident/v87/incidents.go
- internal/services/incident/v87/incidents_test.go
- internal/services/incident/v88/contract.go
- internal/services/incident/v88/incidents.go
- internal/services/incident/v88/incidents_test.go
- internal/services/incident/v89/contract.go
- internal/services/incident/v89/incidents.go
- internal/services/incident/v89/incidents_test.go
- internal/services/incident/waiter/waiter.go
- specs/181-resolve-incident-commands/progress.md
- specs/181-resolve-incident-commands/tasks.md
**Learnings**:
- Generated v8.8/v8.9 single incident resolution endpoints return status/body/problem fields only, so facade-level result models must carry submission and confirmation metadata rather than relying on a success JSON payload.
- Process-instance resolution can stay entirely inside the incident service boundary by combining scoped incident lookup, direct incident mutation calls, and scoped lookup polling from the process facade.
- `--no-wait` should be modeled as accepted/submitted output in facade results, while dry-run should call lookup paths only and keep `mutationSubmitted` false.
- Targeted validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/incident/... -count=1`, `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process -count=1`, plus compile-only `go test` checks for `./cmd`, `./c8volt/resource`, and `./...`.
---
---
## Iteration 3 - 2026-05-08 21:29:19 CEST
**User Story**: User Story 1 - Resolve Known Incidents
**Tasks Completed**:
- [x] T018: Add command tests for incident key flags, stdin `-`, duplicate keys, and invalid keys in `cmd/resolve_incident_test.go`
- [x] T019: Add human and JSON view tests for incident resolution results and non-dry-run payloads in `cmd/cmd_views_resolve_test.go`
- [x] T020: Add command contract expectations for `resolve incident`, alias `inc`, and mutation metadata in `cmd/command_contract_test.go`
- [x] T021: Add the `resolve` root command with shared backoff and state-changing metadata in `cmd/resolve.go`
- [x] T022: Add `resolve incident` command parsing, aliases, flags, stdin key merge, validation, automation support, and worker flags in `cmd/resolve_incident.go`
- [x] T023: Add incident resolution human and JSON rendering in `cmd/cmd_views_resolve.go`
- [x] T024: Wire `resolve incident` to the process facade and preserve per-target failures in `cmd/resolve_incident.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_resolve.go
- cmd/cmd_views_resolve_test.go
- cmd/command_contract_test.go
- cmd/get_processinstance_test.go
- cmd/resolve.go
- cmd/resolve_incident.go
- cmd/resolve_incident_test.go
- specs/181-resolve-incident-commands/progress.md
- specs/181-resolve-incident-commands/tasks.md
**Learnings**:
- Direct `resolve incident` can wire straight to the process facade `ResolveIncidents` bulk helper; command-local responsibility is target collection, validation, automation metadata, and rendering.
- The command test server can confirm default wait behavior by returning a resolved state from `GET /v2/incidents/{key}` immediately after `POST /v2/incidents/{key}/resolution`.
- Targeted validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestResolveIncident|TestRenderIncidentResolution|TestCommandCapabilityForCommand_ResolveIncidentContract|TestCapabilityDocumentForRoot_ResolveCommandFamily' -count=1`, the broader command slice, and `GOCACHE=/tmp/c8volt-gocache go test ./cmd -count=1`.
---
---
## Iteration 4 - 2026-05-08 21:35:59 CEST
**User Story**: User Story 2 - Resolve Process Instance Incidents
**Tasks Completed**:
- [x] T025: Add command tests for process-instance key flags, stdin `-`, duplicate keys, no active incidents, and lookup failures in `cmd/resolve_processinstance_test.go`
- [x] T026: Add facade tests proving process-instance resolution uses incident lookup and never adds incident methods to `internal/services/processinstance` in `c8volt/process/client_test.go`
- [x] T027: Add command contract expectations for `resolve process-instance` and alias `pi` in `cmd/command_contract_test.go`
- [x] T028: Add `resolve process-instance` command parsing, aliases, flags, stdin key merge, validation, automation support, and worker flags in `cmd/resolve_processinstance.go`
- [x] T029: Implement process-instance resolution command orchestration in `cmd/resolve_processinstance.go`
- [x] T030: Complete process-instance resolution result rendering for no-op, success, partial failure, and JSON output in `cmd/cmd_views_resolve.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - blocked by `.git` write restriction in this sandbox
**Files Changed**:
- c8volt/process/client_test.go
- cmd/cmd_views_resolve.go
- cmd/cmd_views_resolve_test.go
- cmd/command_contract_test.go
- cmd/get_processinstance_test.go
- cmd/resolve_processinstance.go
- cmd/resolve_processinstance_test.go
- specs/181-resolve-incident-commands/progress.md
- specs/181-resolve-incident-commands/tasks.md
**Learnings**:
- Process-instance resolution command tests need two scoped incident-search calls for successful wait-confirmed paths: one discovery lookup before mutation and one confirmation lookup after mutation.
- Human output can treat no active incidents as a successful skipped target while JSON output remains the shared process facade result envelope.
- Targeted validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestResolveProcessInstance|TestRenderProcessInstanceResolution|TestCommandCapabilityForCommand_ResolveProcessInstanceContract|TestCapabilityDocumentForRoot_ResolveCommandFamily' -count=1`, `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process -run 'TestResolveProcessInstanceIncidents|TestResolveProcessInstanceIncidentsKeepsIncidentBoundary' -count=1`, plus full `go test ./cmd -count=1` and `go test ./c8volt/process -count=1`.
---
---
## Iteration 5 - 2026-05-08 21:43:59 CEST
**User Story**: User Story 3 - Preview Resolution Plan
**Tasks Completed**:
- [x] T031: Add command dry-run tests for direct incident keys, stdin keys, duplicate keys, and no submitted mutation in `cmd/resolve_incident_test.go`
- [x] T032: Add command dry-run tests for process-instance incident discovery, no active incidents, and no submitted mutation in `cmd/resolve_processinstance_test.go`
- [x] T033: Add JSON dry-run and `--json --verbose` rejection tests in `cmd/resolve_incident_test.go`, `cmd/resolve_processinstance_test.go`, and `cmd/cmd_views_resolve_test.go`
- [x] T034: Add command contract expectations proving `resolve incident` and `resolve process-instance` expose `--dry-run` in `cmd/command_contract_test.go`
- [x] T035: Add `--dry-run` parsing and request propagation in `cmd/resolve_incident.go` and `cmd/resolve_processinstance.go`
- [x] T036: Implement lookup-backed dry-run plan construction in `c8volt/process/client.go` and `c8volt/process/bulk.go`
- [x] T037: Implement compact human dry-run rendering and stable JSON dry-run payloads in `cmd/cmd_views_resolve.go`
- [x] T038: Reject `--json --verbose` for resolve commands before lookup or mutation in `cmd/resolve_incident.go` and `cmd/resolve_processinstance.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/model.go
- c8volt/process/resolve.go
- cmd/cmd_views_resolve.go
- cmd/cmd_views_resolve_test.go
- cmd/command_contract_test.go
- cmd/resolve.go
- cmd/resolve_incident.go
- cmd/resolve_incident_test.go
- cmd/resolve_processinstance.go
- cmd/resolve_processinstance_test.go
- specs/181-resolve-incident-commands/progress.md
- specs/181-resolve-incident-commands/tasks.md
**Learnings**:
- Direct incident dry-run loads current incident state with `GET /v2/incidents/{key}` and classifies active incidents as planned without calling `/resolution`.
- Process-instance dry-run uses the same scoped incident search as mutation mode, but stops after discovery and renders planned/skipped results without confirmation polling.
- Stable JSON dry-run output stays on the shared envelope and now carries the resolve operation name, while `--json --verbose` is rejected before lookup or mutation.
- Targeted validation passed with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestResolveIncidentCommand_DryRun|TestResolveIncidentCommand_JSON|TestResolveProcessInstanceCommand_DryRun|TestResolveProcessInstanceCommand_JSON|TestRenderIncidentResolutionResults|TestRenderProcessInstanceResolutionResults|TestCommandCapabilityForCommand_Resolve' -count=1`, `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process -run 'TestResolveIncidentDryRun|TestResolveProcessInstanceIncidents' -count=1`, plus full `go test ./cmd -count=1` and `go test ./c8volt/process -count=1`.
---
