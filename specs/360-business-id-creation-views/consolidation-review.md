# Task Consolidation Review: Business-ID Creation and Resource Views

**Date**: 2026-10-31
**Feature**: [spec.md](spec.md)
**Reviewed task list**: [tasks.md](tasks.md)
**Before / after**: 6 / 6 tasks
**Outcome**: No task changes. The fresh draft already groups implementation, thin facade mappings, required fixtures/tests, formatting, documentation, and validation by cohesive outcomes. No historical review relocation was needed.

## Inputs and progress

Reviewed specification, plan, data model, both contracts, research, quickstart, project guidance, constitution, and relevant Ralph scope rules. All six tasks are unchecked. There is no feature progress or Ralph-memory file and no external feature-artifact reference to a task ID. No implementation evidence or completed task was altered; no renumbering or superseded-ID mapping was needed.

The plan requires US3 compatibility and diagnostics inside behavior-delivery units. Its eight scenarios have explicit ownership in the tasks document; zero standalone US3 tasks does not remove any acceptance obligation. Setup and foundational phases require no new infrastructure and therefore have no artificial work items.

## Code-grounded review

Inspected these implementations and local validation patterns rather than relying on the proposed decomposition alone:

- `internal/services/processinstance/bulk.go`: `CreateProcessInstances` discards a successful prefix on later failure; `CreateNProcessInstances` returns fixed-slot pool results. Creation-specific compaction, input snapshots, and tests belong with the creation operation.
- `c8volt/process/client.go` and `c8volt/process/bulk.go`: thin creation methods discard partial results on error. Fixing them is part of T001, not a separate facade task.
- `cmd/run_processinstance.go`: both ordinary and counted paths call the exiting error helper before result rendering; dedicated renderer functions also currently write mutation rows directly to result stdout. T002 therefore includes real error-flow and logger changes, not just flag wiring.
- `cmd/cmd_views_contract.go`, `cmd/cmd_views_rendermode.go`, and `cmd/update_job_batch_logging_test.go`: existing one-envelope/exit helpers and extensive fake-backed severity/destination/debug/partial-failure coverage establish the required command validation boundary.
- `cmd/cmd_views_processinstance_dryrun.go`: bespoke selected-final-state and requires-cancel structs rebuild observed data; this is separate from merely embedding the public PI model.
- `c8volt/ops/convert.go`: independent `fromDomainProcessInstance` conversion is grouped with T003's PI copy coverage to avoid dropping IDs before derived report work.
- `c8volt/job/convert.go`: completion candidate converters cast between domain/public structs. T004 keeps matching shape changes, pointer copies, and caller updates together.
- `internal/services/job/completion/execution_plan.go`: `snapshotExecutionBatch` clones the candidate slice but needs to detach new pointer fields; immutable-plan tests stay in T004.
- `internal/domain/job.go:RuntimeListenerJobFromJob`, `internal/services/ops/slow_process_analysis.go:slowProcessAnalysisRootFromProcessInstance`, and `cmd/cmd_views_ops_slow_process_analysis.go:formatOpsSlowProcessAnalysisRootRow`: listener and root observations are reconstructed separately and need independent-ID/section-presence proof in T005.
- `cmd/cmd_views_ops_execute_retention_policy.go:retentionProcessInstanceItems` feeds both human preview details and custom Markdown lists; incident-purge report writers reuse it. T006 retains end-to-end file-output and escaping tests rather than relying on T003's JSON/model coverage.

## Evaluated adjacent boundaries

| Boundary | Plausible consolidation | Decision and concrete reason |
| --- | --- | --- |
| T001 / T002 | Full native/public/CLI creation in one task | Strongest merge candidate, rejected. T001 covers two request variants, three-version capability/value precedence, UTF-16 boundaries, observation/no-wait semantics, sequential and concurrent partial-result retention. T002 independently repairs an exiting command error flow and moves existing mutation output into configured logging, requiring actual subprocess/terminal, duplicate-envelope, writer-error, redaction and exit tests. Combining both implementation and these validation matrices would exceed a focused iteration. The intermediate public creation API is usable and verified without the CLI flag. No thin facade task remains. |
| T002 / T003 | Creation and all PI views together | Rejected. Creation result/error rendering has one dedicated contract; T003 covers discovery/enrichment/family/preview shape preservation, sparse and empty discovery, and deletion impact. Shared `ProcessInstance` is already delivered in T001 and reused, so merging removes no duplicate implementation while adding a second substantial acceptance surface. |
| T003 / T004 | All ordinary resources in one task | Rejected. T003's real-terminal empty-scope and custom PI-preview/family matrix is substantial. T004 adds native job/task observations and mutable completion candidate/frozen-plan behavior, not just more row fields. Combining their boundary and execution tests would be oversized. T004 already folds straightforward user-task mapping into the job-observation delivery rather than creating a thin task. |
| T004 / T005 | Jobs and derived listener/slow observations together | Rejected. T004 includes both-direction candidate casts, input isolation and immutable execution-plan verification. T005 has three listener facade representations plus requested-empty/unrequested section semantics and slow summary/full-timeline root reconstruction. Job mapping is built once in T004; T005 consumes it without rebuilding it. Their combined validation is too broad for a focused iteration. |
| T005 / T006 | All derived views and saved reports together | Rejected. T005 proves runtime observation provenance across listener/analysis representations. T006 produces actual operations preview/report files with escaping, partial/empty outcomes and logger behavior across retention/purge/repair. These are distinct usable outputs with materially different failure and file-output validation, not just different files. |

## Nonadjacent shared behavior and pre-consolidated work

- **T001 / T003**: PI model and native mapping are added once in T001. T003 only completes remaining inspection/impact copy sites. Repeating base fields or creating another model task would add work.
- **T003 / T006**: Both use operations PI conversion. T003 owns the duplicated converter; T006 owns actual saved report/scalar-list formatting. Merging the whole set would combine broad interactive discovery/empty-scope validation with file-output escaping and failure tests; neither is replaced by model tests.
- **T004 / T006**: Job candidate snapshots are produced once in T004. T006 consumes observed reports; no new assignment, eligibility, scheduling, or ID lookup is added.
- **Single / collection creation**: Already grouped in T001/T002; no temporary single-only API or later bulk rebuild.
- **Service / thin facade / doubles**: Already together in each resource task. Candidate struct casts and required tests cannot be split into broken intermediate stages.
- **Validation / frozen request / result output**: Capability/value validation and count snapshots belong in T001; CLI flag presence, result rendering, failure exit and diagnostic checks belong together in T002. No standalone validator or flag task exists.
- **Implementation / tests / docs**: Every checkbox includes or inherits the mandatory test/format/documentation checks. T006's final race run accompanies substantive report implementation; no duplicate test-only or documentation-only task is introduced.
- **Parallelism**: No `[P]` edits are claimed. Shared docs and source overlaps require coordination. Examples only allow independent fixture-owned validation after edits stabilize and explicitly reject parallel tests that mutate Cobra globals or shared environment.

## Coverage and consistency checks

- Verified all 17 functional requirements, six success criteria, eleven view/report surface groups, and eight US3 scenarios have named ownership.
- Both creation selectors, nil/empty/UTF-16/invalid-UTF-8 input, version precedence, normal/no-wait source selection, uniqueness conflicts, sequential/count partial errors, worker behavior and zero-slot exclusion remain required.
- Mapping round trips, absent/observed-empty preservation, independent resource IDs, immutable candidate pointers, bespoke previews, listener section presence, slow timelines, saved Markdown, and no extra reads remain required.
- Command execution, configured/inherited logging, both formats, INFO/WARN/quiet, repeated raw backend failures, both DEBUG entry points/redaction, one envelope plus EOF, original exits, actual terminal stdin, empty/sparse/abort behavior and request counts remain in the owning units.
- Checked sequential IDs T001–T006, unchecked progress, primary story labels, exact source paths, technical prerequisites, serial execution order, story ownership, and no dangling task IDs. The technical dependency graph is acyclic; scheduled order intentionally completes the creation MVP first.
- Local Markdown links, source-path references, whitespace, and fenced blocks were checked. No required task or progress was removed; no task file rewrite was necessary after review.

## Limitations

Application tests, generated documentation, and live checks were not run for this planning-only change. Proposed new test files and `BusinessID` test names are implementation outputs, clearly identified as such; implementation must inventory nonzero matches before running the selectors. Real-engine behavior remains unverified. No unresolved coverage or planning limitation was identified.
