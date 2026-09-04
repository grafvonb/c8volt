# Ralph Memory

Feature: 291-force-cleanup-progress
Started: 2026-09-04T10:01:49Z

## Codebase Patterns
- APD command flow: `cmd/ops_purge_all_processdefinitions.go` builds `ops.AllProcessDefinitionsPurgeRequest`, installs a progress callback through `configureOpsPurgeAllProcessDefinitionsProgress`, runs a dry-run planning call for interactive confirmation, freezes discovered keys, then executes through `cli.PurgeAllProcessDefinitions`.
- Current APD progress callback handles preflight/page facts locally and forwards only completion facts into `processDefinitionDeleteSemanticProgress`; that reporter is fixed to `delete process definitions` and starts eagerly after confirmation when frozen keys exist.
- Service callback chain for real deletion is command `request.Progress` -> public facade conversion (`c8volt/ops/convert.go` and `c8volt/foptions/options.go`) -> `internal/services/ops.PurgeAllProcessDefinitions` -> `pdsvc.DeleteProcessDefinitions` -> PI cancel/delete completion callbacks and PD resource delete completion callbacks.
- Existing force cleanup path in `internal/services/processdefinition/delete.go`: preview with workers, `cleanupProcessDefinitionDeletePlanForceScope`, `pisvc.CancelProcessInstances`, `waitForProcessDefinitionDeletePlanActiveInstancesDrained`, `pisvc.DeleteProcessInstances`, then `DeleteProcessDefinitionResources`.

## Decisions
- Iteration 1 was setup/evidence only. No production code or generated docs changed.
- Actual checkout is `develop`; `.specify/feature.json` still points at `specs/291-force-cleanup-progress`; `AGENTS.md` still points at `specs/291-force-cleanup-progress/plan.md`.

## Gotchas
- `ralph-memory.md` and `progress.md` were already untracked at iteration start; they are now part of the coordinated setup commit.
- Commit policy supplied by the orchestrator is conventional with scope `ralph` and `issue: auto`; branch `develop` has no leading numeric prefix.
- The feature tasks mention appending `#291`, but the resolved orchestrator policy says infer issue suffix only from a leading numeric branch prefix. Use the resolved policy unless a later orchestrator defect says otherwise.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd -run 'TestOpsPurgeAllProcessDefinitions|TestOpsSemanticProgressReporter|TestNewOpsSemanticProgressReporter' -count=1`
- `go test ./internal/domain -run 'Test.*Progress' -count=1`
- `go test ./c8volt/ops -run 'TestProgressConversions|TestClientPurgeAllProcessDefinitions' -count=1`
- `go test ./c8volt/foptions -run 'Test.*Progress' -count=1`
- `go test ./internal/services/processdefinition/... -run 'Test.*(DeleteProcessDefinition|CleanupProcessDefinition)' -count=1`
- `go test ./internal/services/ops/... -run 'TestPurgeAllProcessDefinitions' -count=1`
- `go test ./internal/services/processinstance/... -run 'Test.*(Progress|Completion|CancelProcessInstances|DeleteProcessInstances)' -count=1`

## Do Not Repeat
- Do not treat `FrozenScope` as a stage entry or completion source for #291; stage visibility requires a new typed stage event.
- Do not reroute the ordinary non-force worker path through `DeleteProcessDefinitionResources`; the plan requires a private once-only entry hook in the existing `deleteProcessDefinition` path.

## Current Handoff
- Next iteration starts Phase 2 with T003: add stage-envelope contract tests in `internal/domain/ops_progress_test.go` before implementing the additive stage event.
