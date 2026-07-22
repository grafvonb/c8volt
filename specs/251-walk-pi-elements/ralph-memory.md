# Ralph Memory

Feature: 251-walk-pi-elements
Started: 2026-07-22T21:14:46Z

## Codebase Patterns
- `get pi --with-elements` already owns the runtime element grammar through `processInstanceActivityItem.Elements`, `formatProcessInstanceActivityLinesWithElementsWithTimezone`, `formatProcessInstanceElementRows`, and `elementEnrichedProcessInstancesView`.
- Walk activity rendering uses `activityItemsFromTraversal`, `walkActivityView`, `activityPathView`, and `renderActivityFamilyTree` for combined variable/incident detail sections.
- Command capability metadata is generated from live Cobra flags via `commandCapabilityForCommand`; US1 must register the flag before adding capability assertions.

## Decisions
- Foundation keeps actual walk element enrichment disabled until US1; `flagWalkPIWithElements` is resettable state only, and existing walk paths pass a zero-value element enrichment result.
- Traversal activity items now distinguish unrequested element enrichment (`nil`) from requested-but-empty element enrichment (empty slice), preserving future JSON array behavior.

## Gotchas
- `resetProcessInstanceCommandGlobals` lives in `cmd/get_processinstance_test.go` but is shared by walk tests; new walk command globals must be reset there.
- Tree renderers must pass `followingChildren` into the shared activity formatter so detail sections and child process-instance rows remain siblings.

## Reusable Commands
- `go test ./cmd -run 'TestActivityItemsFromTraversal|TestWalkActivityView|TestWalkProcessInstanceCommand_RegressionPreservesReadOnlyTraversalContract|TestProcessInstanceActivityInstancesView_HumanRowsGroupVarsIncidentsAndElements|TestProcessInstanceActivityInstancesView_HumanRowsRenderElements' -count=1`
- `go test ./cmd -run 'TestWalkProcessInstanceCommand|TestWalkHelp|TestProcessInstanceActivityInstancesView|TestCommandCapabilityForCommand' -count=1`
- `go test ./cmd -count=1`

## Do Not Repeat
- Do not add a walk-specific element row grammar; reuse `formatProcessInstanceActivityLinesWithElementsWithTimezone`.
- Do not call generated Camunda clients or versioned services from `cmd`; use `process.API.EnrichProcessInstancesWithElements`.

## Current Handoff
- Setup and foundational plumbing T001-T008 are complete and committed; next iteration starts US1 tasks T009-T017 by adding failing command/help/output tests, registering `--with-elements`, and wiring `flagWalkPIWithElements` through traversal enrichment.
