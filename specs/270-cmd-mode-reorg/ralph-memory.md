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

## Gotchas

- `cmd/get_processdefinition.go` currently contains watch lifecycle declarations plus ordinary lookup execution; US1 should move watch execution/state/timing/retry/status/request construction to `cmd/get_processdefinition_watch.go` while keeping command construction and ordinary dispatch in the base file.
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
- `git diff --check`

## Do Not Repeat

- Do not redo the setup ownership audit from scratch; use `specs/270-cmd-mode-reorg/ownership-followups.md` and only refresh notes for files a later task actually touches.

## Current Handoff
- Next iteration should continue US1 with T015/T016 by creating `cmd/get_processdefinition_watch.go` and moving the process-definition watch lifecycle declarations there while keeping base command construction, flags, validation, metadata, dispatch, XML/key/search execution, and shared ordinary lookup logic in `cmd/get_processdefinition.go`.
