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
- US1 wires `walk pi --with-elements` through `enrichProcessInstancesWithElementActivityOptions` after traversal using `collectExplicitPIAdminInputOptions`, then passes the result into `activityItemsFromTraversal`.
- US2 confirms children, parent, and flat modes already reuse traversal key order for element enrichment; regression coverage lives in `cmd/walk_test.go`.

## Gotchas
- `resetProcessInstanceCommandGlobals` lives in `cmd/get_processinstance_test.go` but is shared by walk tests; new walk command globals must be reset there.
- Tree renderers must pass `followingChildren` into the shared activity formatter so detail sections and child process-instance rows remain siblings.
- Default family traversal can fetch the starting process instance twice before descendant search; request-order assertions for walk command tests must account for that existing behavior.
- Parent traversal element enrichment follows selected-to-root order, matching `TraversalResult.Keys` from `BuildAncestryResult`.

## Reusable Commands
- `go test ./cmd -run 'TestActivityItemsFromTraversal|TestWalkActivityView|TestWalkProcessInstanceCommand_RegressionPreservesReadOnlyTraversalContract|TestProcessInstanceActivityInstancesView_HumanRowsGroupVarsIncidentsAndElements|TestProcessInstanceActivityInstancesView_HumanRowsRenderElements' -count=1`
- `go test ./cmd -run 'TestWalkProcessInstanceCommand_.*WithElements|TestWalkProcessInstanceCommand_Default.*Without' -count=1`
- `go test ./cmd -run 'TestWalkProcessInstanceCommand|TestWalkHelp|TestProcessInstanceActivityInstancesView|TestCommandCapabilityForCommand' -count=1`
- `go test ./cmd -count=1`

## Do Not Repeat
- Do not add a walk-specific element row grammar; reuse `formatProcessInstanceActivityLinesWithElementsWithTimezone`.
- Do not call generated Camunda clients or versioned services from `cmd`; use `process.API.EnrichProcessInstancesWithElements`.

## Current Handoff
- US2 tasks T018-T025 are complete and committed once this work-unit commit lands; next iteration starts US3 tasks T026-T035 for JSON payloads, combined enrichment sections, invalid `--keys-only --with-elements`, v8.7 unsupported propagation, and element lookup failure behavior.
