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
- All-process-definitions discovery should mirror `get pd` branching: `--key` performs a single `GetProcessDefinition` lookup, `--latest` calls `SearchProcessDefinitionsLatest`, and default/filter discovery calls `SearchProcessDefinitions` with `processdefinition.MaxResultSize`.
- All-process-definitions delete planning should call `internal/services/resource.PreviewDeleteProcessDefinitions` with the frozen unique candidate keys. Zero-candidate discovery keeps delete planning skipped, while non-empty dry-runs build the plan and expose active-instance `RequiresForce` blockers without mutating.
- Confirmed all-process-definitions mutation should call `internal/services/resource.DeleteProcessDefinitions` with the frozen delete-plan keys and preserve its no-wait, worker, fail-fast, no-worker-limit, force, history cleanup, wait, and per-key response behavior instead of adding an ops-specific delete path.
- Interactive all-process-definitions runs should perform a dry-run planning pass, reject active-instance blockers locally before prompting, freeze `planned.Discovery.CandidateProcessDefinitionKeys`, and execute the final request with that frozen scope so no second discovery can expand deletion.
- All-process-definitions report handling should mirror incident purge: validate report-file overwrite safety before remote planning for dry-run/unconfirmed paths, allow overwrite only for already confirmed/submitted mutations, write failure reports when audit data exists, and print `report: written <path>` only after a successful write.
- All-process-definitions compact human output should suppress detailed key lists by default; verbose output is the place for candidate metadata, duplicate candidate keys, affected process-instance keys, and active blocked keys.
- Generated CLI docs may create new per-command Markdown files and update `docs/index.md` build metadata; refresh them with `make docs-content` after help/example changes and do not hand-edit generated output.
- Tests observing process-definition delete submissions from worker-based paths should not assume request order when multiple workers can submit deletes concurrently; assert the submitted set instead.

## Validation Log

- Final validation passed in Iteration 8 with targeted command/facade/service tests and repository-wide `make test`.

---

---
## Iteration 8 - 2026-05-16 19:28:22 CEST
**User Story**: User Story 6 - Preserve Documentation And Regression Contracts
**Tasks Completed**:
- [x] T056: Add regression tests for unchanged `get pd` selection and display-only behavior in `cmd/get_processdefinition_test.go`
- [x] T057: Add regression tests for unchanged `delete pd` preflight, force, no-wait, and selector behavior in `cmd/delete_test.go`
- [x] T058: Add docs/contract assertions for all-process-definitions purge command metadata in `cmd/command_contract_test.go`
- [x] T059: Update user-facing help examples for all-process-definitions purge in `cmd/ops_purge_all_processdefinitions.go`
- [x] T060: Run `make docs-content` and review generated files under `docs/cli/` and `docs/index.md`
- [x] T061: Run targeted command tests with `go test ./cmd -run 'TestOpsPurge|TestCommandContract|TestDeleteProcessDefinition|TestGetProcessDefinition' -count=1`
- [x] T062: Run facade and service tests with `go test ./c8volt/ops ./c8volt/processdefinition ./c8volt/resource ./internal/services/ops ./internal/services/processdefinition ./internal/services/resource -count=1`
- [x] T063: Run repository validation with `make test`
- [x] T064: Mark US6 tasks complete and record final validation notes in `specs/208-purge-process-definitions/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/get_processdefinition_test.go
- cmd/delete_test.go
- cmd/command_contract_test.go
- cmd/ops_purge_all_processdefinitions.go
- cmd/ops_purge_all_processdefinitions_test.go
- docs/cli/c8volt_ops_purge.md
- docs/cli/c8volt_ops_purge_all-process-definitions.md
- docs/index.md
- specs/208-purge-process-definitions/tasks.md
- specs/208-purge-process-definitions/progress.md
**Learnings**:
- `get pd` keeps selection filters in `ProcessDefinitionFilter`, while XML remains a key-only display mode incompatible with search modifiers, `--stat`, JSON, and keys-only output.
- `delete pd --no-wait` skips batch-operation read/poll checks and returns after the resource delete request is accepted, so tests should assert accepted submission rather than final done logs.
- The task-listed facade path `./c8volt/processdefinition` does not exist in this repository; the actual public process facade package is `./c8volt/process`, and the corrected package set passed.
- Validation passed: `make docs-content`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd -run 'TestOpsPurge|TestCommandContract|TestDeleteProcessDefinition|TestGetProcessDefinition' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./c8volt/ops ./c8volt/process ./c8volt/resource ./internal/services/ops ./internal/services/processdefinition ./internal/services/resource -count=1`; `make test`.
---

---
## Iteration 7 - 2026-05-16 19:16:19 CEST
**User Story**: User Story 5 - Produce Compact Output, Complete Reports, And Automation-Safe JSON
**Tasks Completed**:
- [x] T046: Add verbose key-list output tests for candidate, duplicate, affected, and blocked keys in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T047: Add deterministic `--dry-run --json` and `--automation --json` output tests in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T048: Add Markdown all-process-definitions purge report rendering test in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T049: Add JSON all-process-definitions purge report rendering test in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T050: Add existing report-file preservation tests for dry-run, unconfirmed, and locally blocked runs in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T051: Reuse shared ops report-file validation, format inference, overwrite safety, and file writing in `cmd/ops_purge_all_processdefinitions.go`
- [x] T052: Extend report model/rendering for process-definition discovery, candidate set, plan, deletion, notices, errors, and outcome fields in `cmd/cmd_views_ops_purge_all_processdefinitions.go`
- [x] T053: Keep normal human output compact and gate detailed key lists behind verbose output in `cmd/cmd_views_ops_purge_all_processdefinitions.go`
- [x] T054: Print compact `report: written <path>` human output after report writes in `cmd/cmd_views_ops_purge_all_processdefinitions.go`
- [x] T055: Mark US5 tasks complete and record validation notes in `specs/208-purge-process-definitions/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_purge_all_processdefinitions.go
- cmd/ops_purge_all_processdefinitions.go
- cmd/ops_purge_all_processdefinitions_test.go
- specs/208-purge-process-definitions/tasks.md
- specs/208-purge-process-definitions/progress.md
**Learnings**:
- Report-file overwrite behavior now follows the shared ops workflow contract: dry-run and unconfirmed runs fail before discovery when a report already exists, while submitted destructive runs may overwrite and blocked destructive runs preserve existing reports.
- Markdown and JSON reports now include discovery filters and candidate metadata, duplicate candidates, delete-plan impact, blocked/affected process-instance keys, deletion results, notices, errors, runtime config metadata, and final outcome.
- Compact human output remains count-oriented, while verbose output exposes candidate process-definition details, candidate and duplicate keys, affected process-instance keys, and blocked process-instance keys.
- Validation passed: `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/resource ./c8volt/resource -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./... -count=1`.
---
## Iteration 6 - 2026-05-16 19:08:26 CEST
**User Story**: User Story 4 - Execute Confirmed Purge Through Delete PD
**Tasks Completed**:
- [x] T037: Add confirmed deletion command test for exact frozen candidate submission in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T038: Add execution-control mapping tests for workers, fail-fast, no-worker-limit, no-wait, and force in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T039: Add `--automation --json` without `--auto-confirm` success test for supported state-changing all-process-definitions purge in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T040: Add local-precondition failure subprocess tests for post-planning blockers and exit code in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T041: Execute deletion through existing process-definition resource delete service from `internal/services/ops/all_process_definitions_purge.go`
- [x] T042: Use `shouldImplicitlyConfirm(cmd)` for destructive confirmation decisions in `cmd/ops_purge_all_processdefinitions.go`
- [x] T043: Preserve no-wait, confirmation, per-key status, and final outcome in `internal/domain/ops_all_process_definitions_purge.go`
- [x] T044: Render deletion execution and final outcome in `cmd/cmd_views_ops_purge_all_processdefinitions.go`
- [x] T045: Mark US4 tasks complete and record validation notes in `specs/208-purge-process-definitions/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/ops/convert.go
- cmd/ops_purge_all_processdefinitions.go
- cmd/ops_purge_all_processdefinitions_test.go
- internal/domain/resource.go
- internal/services/ops/all_process_definitions_purge.go
- internal/services/ops/all_process_definitions_purge_test.go
- internal/services/resource/workflow.go
- specs/208-purge-process-definitions/tasks.md
- specs/208-purge-process-definitions/progress.md
**Learnings**:
- Confirmed all-process-definitions purge now submits exactly the frozen delete-plan candidate keys through the existing resource delete workflow, preserving `delete pd` behavior for force/no-wait/workers/fail-fast/no-worker-limit.
- Interactive destructive runs now plan once for confirmation, reject active-instance blockers as a local precondition before prompting, and execute using the frozen candidate key set rather than re-running discovery.
- Per-key deletion status is carried through the resource delete response so ops JSON/report models can expose submitted process-definition delete results.
- Validation passed: `GOCACHE=/private/tmp/c8volt-go-build go test ./internal/services/ops -run 'TestPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./c8volt/ops -run 'TestClientPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/resource ./c8volt/resource -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./... -count=1`.
---

---
---
## Iteration 5 - 2026-05-16 18:58:15 CEST
**User Story**: User Story 3 - Build Delete Plan From Frozen Candidates
**Tasks Completed**:
- [x] T029: Add ops service delete-plan tests for candidate keys, affected process-instance counts, active-instance blockers, and duplicate handling in `internal/services/ops/all_process_definitions_purge_test.go`
- [x] T030: Add unsafe active-instance blocking test without `--force` in `internal/services/ops/all_process_definitions_purge_test.go`
- [x] T031: Add command dry-run plan rendering tests in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T032: Reuse existing process-definition delete preflight from frozen candidate keys in `internal/services/ops/all_process_definitions_purge.go`
- [x] T033: Preserve process-definition delete-plan items, active impact, duplicate candidates, force readiness, and semantic notice details in `internal/domain/ops_all_process_definitions_purge.go`
- [x] T034: Map delete-plan details through `c8volt/ops/convert.go`
- [x] T035: Render compact delete-plan human output and complete JSON output in `cmd/cmd_views_ops_purge_all_processdefinitions.go`
- [x] T036: Mark US3 tasks complete and record validation notes in `specs/208-purge-process-definitions/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_purge_all_processdefinitions.go
- cmd/ops_purge_all_processdefinitions_test.go
- internal/services/ops/all_process_definitions_purge.go
- internal/services/ops/all_process_definitions_purge_test.go
- specs/208-purge-process-definitions/tasks.md
- specs/208-purge-process-definitions/progress.md
**Learnings**:
- Delete planning now reuses the existing process-definition delete preflight with the frozen candidate key set, preserving active-instance impact checks and duplicate candidate reporting before any mutation story starts.
- No-target discovery remains a successful planned no-op with the delete plan skipped, avoiding unnecessary preflight calls when there are no candidate keys.
- Unsafe destructive runs without `--force` now fail as a local precondition with deletion status `blocked`, while dry-run still reports the same active-instance blocker as plan data.
- Validation passed: `GOCACHE=/private/tmp/c8volt-go-build go test ./internal/services/ops -run 'TestPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./c8volt/ops -run 'TestClientPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./... -count=1`.
---
---
## Iteration 4 - 2026-05-16 18:52:11 CEST
**User Story**: User Story 2 - Discover And Freeze Candidate Process Definitions
**Tasks Completed**:
- [x] T020: Add process-definition discovery service tests for default all-version discovery, BPMN process ID filtering, version filtering, version-tag filtering, latest-only narrowing, duplicate detection, and no-target behavior in `internal/services/ops/all_process_definitions_purge_test.go`
- [x] T021: Add command dry-run discovery output tests in `cmd/ops_purge_all_processdefinitions_test.go`
- [x] T022: Add facade conversion tests for process-definition discovery result fields in `c8volt/ops/client_test.go`
- [x] T023: Reuse existing process-definition search semantics to discover candidate process definitions in `internal/services/ops/all_process_definitions_purge.go`
- [x] T024: Extract, deduplicate, and freeze candidate process-definition keys in `internal/services/ops/all_process_definitions_purge.go`
- [x] T025: Preserve candidate process-definition metadata, duplicate candidate keys, latest-only scope, notices, and errors in `internal/domain/ops_all_process_definitions_purge.go`
- [x] T026: Map discovery request/result through `c8volt/ops/client.go` and `c8volt/ops/convert.go`
- [x] T027: Render compact discovery output and complete JSON discovery data in `cmd/cmd_views_ops_purge_all_processdefinitions.go`
- [x] T028: Mark US2 tasks complete and record validation notes in `specs/208-purge-process-definitions/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/ops/client_test.go
- c8volt/ops/model.go
- cmd/cmd_views_ops_purge_all_processdefinitions.go
- cmd/ops_purge_all_processdefinitions.go
- cmd/ops_purge_all_processdefinitions_test.go
- internal/services/ops/all_process_definitions_purge.go
- internal/services/ops/all_process_definitions_purge_test.go
- specs/208-purge-process-definitions/tasks.md
- specs/208-purge-process-definitions/progress.md
**Learnings**:
- Discovery-only execution now performs exactly one process-definition lookup pass and freezes unique candidate keys before later delete-plan stories can consume them.
- Duplicate process-definition keys are recorded as semantic info notices and excluded from the frozen delete target list.
- Latest-only discovery is preserved as both structured discovery data and compact human output so narrowed scope is visible without verbose key lists.
- Validation passed: `GOCACHE=/private/tmp/c8volt-go-build go test ./internal/services/ops -run 'TestPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./c8volt/ops -run 'TestClientPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions' -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`; `GOCACHE=/private/tmp/c8volt-go-build go test ./... -count=1`.
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
---
## Correction - 2026-05-16 19:36 CEST
**User Feedback**: All-process-definitions purge human output should list candidate BPMN process IDs with bracketed version-level affected process-instance counts, similar in density to `get pd --stat`. Verbose affected process-instance key lists are too noisy for this workflow.
**Implementation Direction**:
- Normal human output should include compact grouped lines such as `invoice [v1: 240, v2: 180, v3: 0]` after the candidate count when delete-plan impact is available.
- Full process-definition keys and affected process-instance keys should remain available in JSON/report output for audit and automation.
- Terminal output should no longer suggest `--verbose` for process-instance key lists on this command.
---
## Correction - 2026-05-16 20:05 CEST
**User Feedback**: Confirmed `ops purge apd --force` should not print per-process-instance wait/cancel/delete lines while deleting process definitions. Output should stay focused on selected process definitions and aggregate progress.
**Implementation Direction**:
- Preserve process-definition-level progress from the reused `delete pd` path.
- Suppress nested process-instance detail logs only for the all-process-definitions purge delegation into process-instance cancel/delete internals.
- Leave direct `delete pd`, `cancel pi`, and `delete pi` logging behavior unchanged unless they explicitly opt into the suppression option.
---
## Correction - 2026-05-16 20:19 CEST
**User Feedback**: Destructive `ops purge apd --force` progress logs list raw process-definition keys but should also show the BPMN process ID so operators can correlate each deletion with the compact dry-run summary.
**Implementation Direction**:
- Add BPMN process ID metadata to process-definition delete impact plan items when the impact lookup already fetches the definition.
- Format process-definition delete progress subjects as `pd <key> <bpmn-process-id> v<version>/<tag> <tenant-id>` whenever the BPMN ID is available.
- Preserve key-only output as a fallback when the metadata is not available, such as no-state-check paths.
