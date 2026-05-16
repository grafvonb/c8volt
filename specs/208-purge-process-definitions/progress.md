# Progress: Ops Purge All Process Definitions

## Mandatory Context

- GitHub issue: https://github.com/grafvonb/c8volt/issues/208
- Feature branch/directory: `208-purge-process-definitions`
- Ralph launch context: every Ralph iteration MUST include `--implementation-context specs/ralph-implementation-rules.md`.
- Commit rule: every commit subject MUST follow Conventional Commits and end with `#208`.

## Setup Notes

- The feature spec, plan, research, data model, quickstart, and CLI contract were generated from issue #208.
- The existing ops purge workflows for orphan process instances and process instances with incidents provide the closest command, report, JSON, automation, and docs patterns.
- The existing process-definition delete source of truth lives in the resource delete path used by `delete pd`; the purge workflow should delegate active-instance impact checks, `--force`, history cleanup, process-definition deletion, wait/no-wait, worker, fail-fast, and no-worker-limit behavior there.

## Codebase Patterns

- Before every implementation iteration, read and apply `specs/ralph-implementation-rules.md` in addition to the feature artifacts. Stop and surface any conflict between those rules and the feature plan.
- Keep command behavior in `cmd/`, public orchestration in `c8volt/ops`, version-neutral workflow behavior in `internal/services/ops`, process-definition discovery through existing process-definition services, and process-definition delete planning/deletion through the existing resource delete source of truth.
- Reuse existing ops purge/report/output patterns before adding new helpers. Do not shell out to `c8volt get pd` or `c8volt delete pd`.
- Preserve `get pd` and `delete pd` behavior while adding the high-level workflow.
- Model the command orchestration after `cmd/ops_purge_processinstances_with_incidents.go`: validate static flags before remote work, call `requireAutomationSupport`, use `shouldImplicitlyConfirm(cmd)`, preflight report-file paths before discovery, run a dry-run planning pass before interactive destructive confirmation, freeze discovered candidate keys into the execution request, then render/write reports through shared ops helpers.
- Model the internal workflow after `internal/services/ops/incident_purge.go`: create a result at start, validate service dependencies, perform discovery once unless frozen candidates were provided, skip planning/deletion for no-target discovery, build a delete plan from unique frozen candidates, and finish through a report-populating helper with workflow step statuses.
- Process-definition selection should reuse `cmd/get_processdefinition.go` / facade search semantics: supported filters are `--key`, `--bpmn-process-id`, `--pd-version`, `--pd-version-tag`, and `--latest`; display-only flags such as `--xml` and `--stat` stay out of the purge command.
- Process-definition deletion safety already lives in `internal/services/resource/workflow.go`: `PreviewDeleteProcessDefinitions` deduplicates keys and computes active-instance impact, while `DeleteProcessDefinitions` delegates force cancellation, active-instance drain waiting, process-instance history cleanup, process-definition delete submission, worker count, fail-fast, no-worker-limit, and no-wait behavior.
- Public ops facade changes should stay thin: add the API method, request/result models, conversions, and client delegation in `c8volt/ops`, converting service errors through `ferrors.FromDomain`.
- Internal ops service construction can remain backward-compatible through the existing `New(piAPI, incAPI, ...)` constructor while feature-specific service dependencies are introduced through a focused constructor that wires only the additional services needed by the new workflow.
- Command contract metadata should mirror existing state-changing ops commands: set mutation to state-changing, contract support to full, automation support to full with concrete notes, and output modes for JSON/machine support where the command renders shared envelopes.
- Generated CLI docs are refreshed via `make docs-content`; do not hand-edit `docs/cli/*` or `docs/index.md` when command metadata/help changes.
- Early command-registration iterations can keep the target command non-mutating by using Cobra `Args` for local validation and a help-only `RunE`; later discovery/execution stories should replace the run path with facade orchestration while preserving the validated flag surface and metadata.

## Validation Log

- Pending: Ralph implementation iterations will record targeted validation and final `make test` results here as work units complete.

---
## Iteration 1 - 2026-05-16 18:26:19 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Record mandatory Ralph context and issue traceability in `specs/208-purge-process-definitions/progress.md`
- [x] T002: Inspect existing #186/#187/#199 ops purge/report flows, `get pd` selection, `delete pd` preflight/deletion, command contract metadata, and docs generation patterns; record reusable discoveries in `specs/208-purge-process-definitions/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/208-purge-process-definitions/tasks.md
- specs/208-purge-process-definitions/progress.md
**Learnings**:
- The closest command/report template is the incident-based purge workflow because it already implements frozen candidate discovery, report preservation on local aborts, `shouldImplicitlyConfirm(cmd)`, and full automation metadata.
- The process-definition delete source of truth is the resource workflow; future tasks should call into that layer through facade/service boundaries rather than duplicating impact checks or deletion mechanics.
- `get pd` owns the exact process-definition filter names and latest/non-latest branching; future purge command tests should protect that display-only flags remain unsupported on the purge command.
- Documentation generation is Makefile-driven through `make docs-content`, which runs `go run ./docsgen -out ./docs/cli -format markdown`.
---
---
## Iteration 2 - 2026-05-16 18:34:39 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T003: Define internal all-process-definitions purge request/result domain models in `internal/domain/ops_all_process_definitions_purge.go`
- [x] T004: Define public ops all-process-definitions purge request/result models in `c8volt/ops/model.go`
- [x] T005: Extend public ops facade API for all-process-definitions purge in `c8volt/ops/api.go`
- [x] T006: Extend internal ops service interface for all-process-definitions purge in `internal/services/ops/api.go`
- [x] T007: Implement public/internal all-process-definitions purge model conversions in `c8volt/ops/convert.go`
- [x] T008: Implement thin public ops facade all-process-definitions purge method in `c8volt/ops/client.go`
- [x] T009: Add foundational ops facade wiring tests for all-process-definitions purge in `c8volt/ops/client_test.go`
- [x] T010: Add foundational internal ops service validation tests for all-process-definitions purge in `internal/services/ops/all_process_definitions_purge_test.go`
- [x] T011: Mark Phase 2 tasks complete and record validation notes in `specs/208-purge-process-definitions/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/client.go
- c8volt/ops/api.go
- c8volt/ops/client.go
- c8volt/ops/client_test.go
- c8volt/ops/convert.go
- c8volt/ops/model.go
- internal/domain/ops_all_process_definitions_purge.go
- internal/services/ops/api.go
- internal/services/ops/all_process_definitions_purge.go
- internal/services/ops/all_process_definitions_purge_test.go
- specs/208-purge-process-definitions/tasks.md
- specs/208-purge-process-definitions/progress.md
**Learnings**:
- All-process-definitions purge needs process-instance, process-definition, and resource services at the internal ops boundary; the existing incident service is not required for this workflow.
- The public process package is the repository's facade home for process-definition filter/result types, so the ops model uses a local selection struct to add `LatestOnly` without leaking internal domain fields.
- The Phase 2 service implementation intentionally supports the frozen-candidate shape and dependency validation only; remote discovery, delete planning, execution, and command wiring remain scoped to later work units.
- Validation passed: `go test ./c8volt/ops -run 'TestClientPurgeAllProcessDefinitionsMapsServiceBoundary|TestClientPurgeProcessInstancesWithIncidentsMapsServiceBoundary' -count=1`; `go test ./internal/services/ops -run 'TestPurgeAllProcessDefinitions' -count=1`; `go test ./c8volt ./c8volt/ops ./internal/services/ops -count=1`; `go test ./... -count=1`.
---
---
## Iteration 3 - 2026-05-16 18:42:13 CEST
**User Story**: User Story 1 - Register All Process Definitions Purge Command
**Tasks Completed**:
- [x] T012: Add command registration, help, and alias tests for all-process-definitions purge in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T013: Add unsupported display-only process-definition flag tests in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T014: Add command contract metadata tests for state-changing and automation support in `cmd/command_contract_test.go`
- [x] T015: Add `ops purge all-process-definitions` Cobra command, alias, summary, and safe examples in `cmd/ops_purge_all_processdefinitions.go`
- [x] T016: Register supported process-definition selection flags and delete workflow flags in `cmd/ops_purge_all_processdefinitions.go`
- [x] T017: Map static flag validation failures through existing invalid-input helpers in `cmd/ops_purge_all_processdefinitions.go`
- [x] T018: Set mutation, output-mode, required flag, and automation metadata in `cmd/ops_purge_all_processdefinitions.go`
- [x] T019: Mark US1 tasks complete and record validation notes in `specs/208-purge-process-definitions/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/ops_purge_all_processdefinitions.go
- cmd/ops_purge_all_processdefinitions_test.go
- cmd/command_contract_test.go
- specs/208-purge-process-definitions/tasks.md
- specs/208-purge-process-definitions/progress.md
**Learnings**:
- The all-process-definitions command now exposes only the supported process-definition selection flags (`--key`, `--bpmn-process-id`, `--pd-version`, `--pd-version-tag`, `--latest`) plus shared delete workflow flags; display-only `get pd` flags remain unknown on the purge surface.
- Static validation is currently local and remote-free through Cobra argument validation, which protects invalid keys, non-positive explicit `--pd-version`, non-positive explicit `--workers`, and report-format/report-file combinations before later stories add discovery.
- Validation passed with `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions|TestCommandCapabilityForCommand_OpsPurgeAllProcessDefinitionsContract' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions|TestCommandCapabilityForCommand_OpsPurge(AllProcessDefinitions|ProcessInstancesWithIncidents|OrphanProcessInstances)Contract|TestCapabilitiesCommand_Ops' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd -count=1`.
---
