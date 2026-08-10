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

## Gotchas

- `cmd/get_processdefinition.go` currently contains watch lifecycle declarations plus ordinary lookup execution; US1 should move watch execution/state/timing/retry/status/request construction to `cmd/get_processdefinition_watch.go` while keeping command construction and ordinary dispatch in the base file.
- `cmd/cmd_views_processinstance_dryrun.go` currently performs facade-backed dry-run planning. Later US3 work should move planning coordination out of renderer ownership before treating the renderer as presentation-only.

## Reusable Commands

- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `rg -n "^(func|type|const|var) " cmd/<file>.go`
- `go test ./cmd -run 'TestCommandContract' -count=1`
- `git diff --check`

## Do Not Repeat

- Do not redo the setup ownership audit from scratch; use `specs/270-cmd-mode-reorg/ownership-followups.md` and only refresh notes for files a later task actually touches.

## Current Handoff
- Next iteration should start with T008 in Foundational: add renderer ownership regression checks that fail if view files call public facades or internal services in `cmd/cmd_views_get_test.go`; T009-T010 also remain before any user story work starts.
