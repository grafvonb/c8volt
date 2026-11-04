# Task Consolidation Review: Issue #362

**Date**: 2026-11-02
**Status**: COMPLETED — reviewed 6 draft tasks and saved 4 cohesive implementation units.

## Evidence reviewed

Read the feature spec, plan, research, models/contracts/validation guide, AGENTS task-planning/output/testing rules, relevant Ralph rules, and implementation neighbors. Repository facts:

- `internal/domain/business_id.go:ValidateBusinessID` already supplies UTF-8 and UTF-16 counting; assignment adds only blank rejection and must preserve creation whitespace behavior. A validator-only prerequisite has no useful separate delivery boundary.
- `internal/services/processinstance/update.go:ExecuteProcessInstanceVariableUpdates` and `c8volt/process/variable_updates.go` provide copied plans, ordered results, pool and facade patterns. Direct assignment still needs a new exact-key native port, assignment-specific planner/executor, no-replay evidence, and bounded BID observation; variable planning is not its implementation.
- `cmd/update_processinstance.go` currently parses mandatory variables before dispatch and rejects no keys; assignment needs a distinct lifecycle, deliberate-empty-stdin behavior, mode validation, new views, and configured logger/real-terminal execution coverage.
- `internal/services/job/completion/execution_plan.go:snapshotExecutionBatch` already copies slices/maps, while its validation protects prepared membership. Extend these mechanisms rather than count them as new algorithms.
- `internal/services/job/completion/execute.go:executeCandidate` and `reconcile.go:reconcile` can satisfy plain completion from COMPLETED; assignment requires conditional attribution/satisfaction changes with retained errors.
- `internal/services/job/v810/service.go:submitJobCompletion` retries plain scalar completion. BID-bearing submission must be one-shot. `completion_request.go:completeJobWithResponse` already preserves optional-field omission; no regeneration task is needed.
- `cmd/update_job_outcome.go:executeUpdateJobWorkerOutcome` drops returned evidence on errors; `cmd/update_job_request.go` preserves input form and exact zero-retry selector behavior. CLI delivery must retain evidence through views/logger/exit and both scalar/bulk modes.
- AGENTS requires implementation, fixtures, tests, formatting and targeted validation in one task, and routine progress/context work is not a standalone task. Ralph permits only the current primary-story work unit per iteration.

## Applied merges and progress

| Draft IDs | Final ID | Consolidated outcome |
| --- | --- | --- |
| T001 + T002 | T001 | Assignment validator, tests, native policy, frozen process service/facade and observation are delivered together with the validator's first useful consumer |
| T003 | T002 | Direct CLI delivery retains its command/output/terminal/documentation validation |
| T004 | T003 | Scalar and bulk atomic completion retain all models, ports, adapters, copies, one-shot evidence and reconciliation together |
| T005 + T006 | T004 | Completion CLI delivery includes final shared-runtime race validation and actual coverage evidence, avoiding a validation-only handoff |

All draft tasks were unchecked; no completed or partial implementation evidence existed to preserve. No obligations were removed. The final tasks retain value constraints, all adapter/direct/prepared guards, HTTP evidence, pointer/nested-map isolation, concurrency/no replay, exact keys, stale snapshots, zero-retry predicates, complete discovery, mode precedence, output/logging/DEBUG/redaction, real-terminal tests, one JSON envelope/original exit, docs generation, and broad isolated verification.

## Retained boundaries and strongest rejected merge

Strongest combined candidate: deliver the entire direct assignment service, facade and CLI in T001+T002, including the shared validator, new exact-key native adapter, frozen collection planner/executor, bounded observation and typed partial evidence, then input-mode dispatch, empty-stdin handling, all output modes, configured logging, DEBUG/redaction, real-terminal confirmation/EOF/abort, and docs.

That combined implementation and validation is impractical as one short Ralph iteration: direct assignment adds a new prepared collection lifecycle and post-acceptance observer with wire/ordering/fail-fast/unknown-response tests, whereas its CLI adds a separate stateful mode and the full mandatory logger/terminal/output matrix. Existing variable service and CLI tests provide patterns but cannot implement or prove these new lifecycle contracts. Keep a compiling tested public service outcome followed by its CLI consumer, as explicitly allowed in the plan's work-unit guidance. This boundary costs one facade handoff; removing it would require both distinct new lifecycles and both full verification matrices before either outcome can finish.

Also considered T003+T004: existing bulk traversal/pools are reused, not new work. Nonetheless T003 changes the common submission port across all adapters, scalar approved-payload copying and retry policy, bulk attribution/reconciliation, and public partial evidence, with mandatory cross-layer wire/immutability/lost-response tests. T004 separately changes mode parsing and scalar error rendering, proves stdin/search/form/zero-retry behavior plus the complete command logger/terminal matrix, and integrates final full-race validation. Keep these two practical units; no intermediate unit may defer unsafe evidence or submission semantics.

The workflow boundary between process assignment and job completion remains required by Ralph's single-primary-story rule. Merging T002 with T003 would combine US1 command delivery and US2 service behavior in one iteration. US3 and US4 are covered within first consumers, so no duplicative cross-cutting implementation tasks exist to merge.

## Dependency and coverage review

Final graph is acyclic: T001 → T002 → T003 → T004 by default, with the essential shared-validator edge T001 → T003. T003 does not depend on the direct CLI at runtime; the default sequence avoids shared contract/doc edits. US1 MVP is T001+T002. US3/US4 acceptance is embedded across T001–T004. Counts are primary story US1=2, US2=2, US3=0 standalone, US4=0 standalone; all four stories have explicit acceptance owners. No artificial [P] markers remain.

## Checks and limitations

Read back the saved tasks and this review; checked four sequential unchecked IDs, checklist syntax and paths, references/dependency ranges, preservation of draft obligations, story/requirement coverage, links, whitespace and absence of live validation gates. Application tests were not run for planning-only changes. No implementation, live-engine verification, commit or launch was performed.
