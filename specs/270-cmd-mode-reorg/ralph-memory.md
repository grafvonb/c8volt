# Ralph Memory

Feature: 270-cmd-mode-reorg
Started: 2026-08-10T11:35:19Z

## Codebase Patterns

- For #270, `cmd` owns command construction, flags, validation, prompts, render-mode selection, stdout/stderr rendering, command metadata, and help; public facades map public inputs/errors; internal services own backend paging, traversal, frozen discovery, mutation planning, polling, retries, and worker execution.
- #254 assessment is the baseline for command ownership risk. Preserve its explicit `delete process-instance` frozen aggregate delete semantics and avoid generic ops workflow extraction unless identical safety/reporting semantics are proven.

## Decisions

- T001 found no conflict between `specs/ralph-implementation-rules.md` and `specs/270-cmd-mode-reorg/spec.md`.
- Created `specs/270-cmd-mode-reorg/ownership-followups.md` as the durable tracking artifact for included moves, deferred ownership corrections, helper removals, and validation evidence.
- T007 added `TestCommandContractFocusedModeFilesOwnLifecycleDeclarations` in `cmd/command_contract_test.go`; it parses top-level Go declarations and tracks process-definition watch lifecycle declarations in the current base-file baseline until `cmd/get_processdefinition_watch.go` exists, then requires those declarations to move there.
- T008 added `TestGetViewFilesAvoidBackendOwnership` in `cmd/cmd_views_get_test.go`; it parses `cmd_views_*.go`, fails on internal-service imports or public facade calls from renderer files, and allowlists only the known `cmd_views_processinstance_dryrun.go` planning exception for US3 T041.
- T009 recorded helper caller audit notes in `ownership-followups.md`; no candidate helper is removal-ready before its planned ownership split.
- T011 added focused watch snapshot request behavior tests in `cmd/get_processdefinition_watch_test.go` without creating `cmd/get_processdefinition_watch.go`; creating the production mode file must wait for T015/T016 because the existing contract test will then require all watch lifecycle declarations to move.
- T012 added `TestGetProcessDefinitionBaseDispatchSkipsWatchLifecycle` in `cmd/get_processdefinition_test.go`; it keeps `flagGetPDWatchInterval` intentionally invalid and verifies ordinary list, key, and XML process-definition paths still bypass watch lifecycle validation and output.
- T013 extended `TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata` to pin process-definition watch discovery metadata, unsupported automation status, summary text, and the help text documenting JSON/keys-only/XML/quiet/automation rejection before lookup.
- T014 added `TestGetProcessDefinitionWatchOutputParityAssertions` in `cmd/get_processdefinition_watch_test.go`; it pins human/verbose refresh stdout parity with normal rows and local rejection of JSON, keys-only, quiet, and automation modes before watch refresh work.
- T015/T016 created `cmd/get_processdefinition_watch.go` and moved the guarded process-definition watch lifecycle declarations there. `cmd/get_processdefinition.go` now keeps process-definition command construction, flags, validation, dispatch, XML/key/search execution, and shared ordinary lookup logic.
- T017 moved process-definition watch scenarios, subprocess rejection helper, and watch harness helpers from `cmd/get_processdefinition_test.go` to `cmd/get_processdefinition_watch_test.go`. The base test file now keeps selector/filter, non-watch machine modes, base dispatch, XML/search, paging activity, and shared ordinary helpers.
- T018-T020 completed the US1 audit and checkpoints: watch lifecycle declarations remain in `cmd/get_processdefinition_watch.go`, base validation still owns incompatible watch output-mode rejection, and both watch plus non-watch process-definition targeted commands passed.
- T021 created `cmd/cmd_views_processinstance_test.go` and moved process-instance row, list, age metadata, variable enrichment, incident enrichment, activity enrichment, and process-instance incident-line tests out of `cmd/cmd_views_get_test.go`; shared flat-row, process-definition, and plain incident renderer tests remain in the old mixed file until their US2 tasks.
- T022 created `cmd/cmd_views_processdefinition_test.go`, moved the process-definition human list alignment test there, and added single-item human/JSON/keys-only, list JSON/keys-only, and watch-list parity renderer tests.

## Gotchas

- `cmd/get_processdefinition.go` still contains watch flag registration and watch-specific incompatible-output validation because T016 keeps command flags and validation in the base command owner; watch timing resolution and lifecycle execution live in `cmd/get_processdefinition_watch.go`.
- `cmd/cmd_views_processinstance_dryrun.go` currently performs facade-backed dry-run planning. Later US3 work should move planning coordination out of renderer ownership before treating the renderer as presentation-only.
- When US3 moves dry-run planning out of `cmd_views_processinstance_dryrun.go`, remove the matching `allowedViewFacadeCalls` entry from `cmd/cmd_views_get_test.go`; the renderer guard should then reject all view-file facade calls.

## Reusable Commands

- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `rg -n "^(func|type|const|var) " cmd/<file>.go`
- `go test ./cmd -run 'TestCommandContract' -count=1`
- `go test ./cmd -run 'TestCommandContract|Test.*View' -count=1`
- `go test ./cmd -run 'TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch|TestValidateGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata' -count=1`
- `go test ./cmd -run '^TestGetProcessDefinitionBaseDispatchSkipsWatchLifecycle$' -count=1`
- `go test ./cmd -run 'TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestGetProcessDefinitionWatchOutputParityAssertions|TestValidateGetProcessDefinitionWatch|TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch' -count=1`
- `go test ./cmd -run 'TestCommandContractFocusedModeFilesOwnLifecycleDeclarations|TestGetProcessDefinition.*Watch|TestProcessDefinition.*Watch|TestValidateGetProcessDefinitionWatch|TestCommandCapabilityForCommand_ProcessDefinitionWatchMetadata|TestGetProcessDefinitionBaseDispatchSkipsWatchLifecycle' -count=1`
- `go test ./cmd -run 'TestGetProcessDefinition|TestProcessDefinitionSelectorValidationHelpContract' -count=1`
- `go test ./cmd -run 'Test(ProcessInstance|OneLinePI|ListProcessInstances|IncidentEnrichedProcessInstances|VariableEnrichedProcessInstances|ProcessInstanceVariableHumanLine|IncidentHumanLine|GetViewFilesAvoidBackendOwnership|ListProcessDefinitionsView|ListIncidentsView|FormatFlatRows|TruncateIncident)' -count=1`
- `go test ./cmd -run 'Test(ListProcessDefinitionsView|ProcessDefinitionView|ProcessDefinitionWatchView)' -count=1`
- `go test ./cmd -run 'Test.*View|Test.*JSON|Test.*KeysOnly' -count=1`
- `git diff --check`

## Do Not Repeat

- Do not redo the setup ownership audit from scratch; use `specs/270-cmd-mode-reorg/ownership-followups.md` and only refresh notes for files a later task actually touches.

## Current Handoff
- Next iteration should continue US2 with T023 by adding incident renderer tests for human, JSON, keys-only, and process-instance-key output in `cmd/cmd_views_incident_test.go`; do not start US3.
