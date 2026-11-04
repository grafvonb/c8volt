# Task Consolidation Review: User-Task Assignment and Unassignment

**Reviewed**: 2026-10-28
**Feature**: [spec.md](spec.md) · [plan.md](plan.md)
**Task list**: [tasks.md](tasks.md)
**Before / after**: 4 / 4 tasks
**Outcome**: No task changes. The initial decomposition already keeps thin facade mapping, interface/double changes, shared actions, tests and documentation with their implementation owner. The strongest remaining merge would combine a substantive local input/report contract with full command/process/terminal integration and is not a manageable improvement.

## Scope and evidence

Read the specification, plan, research, data model, both contracts, quickstart, task draft, project instructions and constitution. No progress or Ralph memory files exist for this feature; all four implementation tasks are unchecked. No existing task-ID references were found in the other feature artifacts before this review.

Inspected concrete reuse and validation surfaces:

- `internal/services/common/usertask_completion.go`: `CompleteUserTaskSubmission`, response validation, retry callback/error conversion, sticky ambiguity and backoff cancellation.
- `internal/services/usertask/completion.go`: complete-input validation, snapshots, `DetermineNoOfWorkers`, `pool.ExecuteSlice`, started-result markers, parent/scheduling contexts and ordered error/result aggregation. Its preflight lookup is deliberately excluded from assignment.
- `internal/services/usertask/v810/contract.go` and `completion.go`: narrow generated-client contracts, domain conversion and HTTP response/body/error delegation; v88/v89 counterparts and the verified assignment contracts remain required in T001.
- `c8volt/task/completion.go` and `c8volt/foptions/options.go`: thin facade delegation, partial-result conversion and explicit option mapping; no standalone facade task is needed.
- `internal/services/usertask/*_test.go`, `c8volt/task/*_test.go`, `cmd/process_api_stub_test.go`: interface ripple and fail-closed fixtures that must change with their contracts.
- `cmd/update_usertask_complete_input.go`: pflag empty-occurrence tracking, command-local stdin, empty component handling, strict key validation and completion-only variable parsing.
- `cmd/cmd_views_usertask_complete.go`: frozen preview/report representation, per-status counters, envelope routing, keys-only selection and configured human logging.
- `cmd/update_usertask_complete.go`: approval, configured command logger, service/facade boundary, render-failure handling and original exit precedence.
- `cmd/update_usertask_complete_logging_test.go` and `cmd/update_usertask_complete_terminal_test.go`: actual command diagnostics, both DEBUG entry points, configured/inherited destinations, plain/structured logs, PTY stdin, redirected stdout, EOF and prompt behavior. These are not replaceable by view/parser tests.

Historical #311 task organization was consulted for context only. Its obsolete stdout wording is not carried forward: current project instructions, code and #351 require human mutation results through the configured logger. Existing decomposition was not treated as evidence that a boundary must remain.

## Adjacent-boundary evaluation

| Boundary | Candidate merge and repeated work removed | Decision and concrete reason |
| --- | --- | --- |
| T001 → T002 | Combine native protocol/evidence with collection/facade; avoid reopening domain/call-option context | Retain. T001 changes an existing response-evidence state machine used by completion and verifies three generated wire mappings, including ambiguity across retries/backoff. T002 adds a separate scheduling/snapshot state machine with fail-fast and caller-cancellation races, plus public partial-result conversion. Combining both implementation and synchronized validation would join two substantial failure domains. Thin facade and public mapping are already consolidated into T002. |
| T002 → T003 | Combine public results with command input and views; remove the public-model handoff | Retain. T002 proves concurrent submitted/not-started evidence and immutable service inputs. T003 extracts pflag empty-occurrence/stdin behavior without changing legacy variable input, and defines a two-action report with distinct omission/presence and output precedence rules. Neither is thin wiring. Their focused concurrency versus local parse/representation validation produces useful complete increments without exposing partial CLI modes. |
| T003 → T004 | Combine frozen input/report with actual command execution; eliminate the prepared-but-unregistered command boundary | Strongest rejected merge. T003 includes strict parser extraction with legacy-mode regressions and complete local report/envelope/keys semantics. T004 must prove real command bootstrap/logger attachment, independent INFO/ERROR routing, raw-error exclusion, both DEBUG paths, PTY prompt/EOF/redirect behavior and render-failure exit precedence. Merging introduces all parser compatibility and report-state debugging into the process/terminal delivery unit. Keep the fully tested local contract first, then one complete CLI release with all mandatory command tests and docs; no new flags are exposed until T004. |

The retained boundaries are justified by substantive behavioral and validation complexity, not merely files, architectural layers or opportunities for parallel work. Four is an observed result, not a target count. T004 is still the largest unit, but it delegates protocol, scheduling, parsing and report mechanics already proven in T001–T003 and reuses existing subprocess/PTY/log decoding infrastructure. Its tests should use shared action tables rather than duplicate implementations or unnecessary cross-products.

## Shared and nonadjacent behavior

- **Assign versus unassign**: Already grouped in each owner. T001 uses concrete adapter methods with one assignment-family evidence path; T002 supplies one collection coordinator; T003 one typed local report family; T004 one mode runner/fixture family. A separate removal task would repeat interface, flag, fixture and documentation work without an independent backend workflow.
- **Single versus collection input**: T002 implements both together. No single-item implementation followed by a bulk rewrite exists.
- **Service plus facade**: Already consolidated in T002; `c8volt/task/completion.go` shows that delegation and conversion are thin. Splitting would add a handoff and public-interface/double repair step.
- **Interfaces plus implementations/doubles**: Internal/generated contracts and all required consumers are in T001; public API and command doubles are in T002. No incomplete-interface task exists.
- **Input plus frozen request and preview/report**: Already consolidated in T003. The output model is derived from the same immutable intent rather than built in a disconnected task.
- **Evidence across T001/T002/T003/T004**: Each owns a distinct boundary: native fact classification, indexed aggregation, display representation, and actual emission/exit. Merge none into a generic mutation framework; carry the same enums and invariants across boundaries.
- **Implementation plus fixtures/tests/formatting/docs**: All are attached to the changing owner. The externally visible command/doc change is T004, which includes README, source metadata and generated CLI docs. No unsafe command release is accepted before diagnostic/terminal coverage.
- **Completion/variables/read compatibility**: Existing tests run alongside the specific extraction/interface change, with whole-command regression at T004. This is not a deferred repair task or a second implementation of the same behavior.
- **Setup, review, commits and final test runs**: Responsibilities are already embedded. No separate checkbox solely for reading context, branch inspection, test execution or progress logging remains.

## Coverage and execution checks

- All 27 functional requirements and nine success criteria have explicit owners in the tasks coverage table.
- Six user-story acceptance phases remain present. Primary task ownership is two foundation tasks plus two US1-owned shared delivery tasks; US2–US6 have coverage references rather than duplicate tasks. The current unit's explicit shared obligations do not authorize starting another story/polish section during a Ralph iteration.
- Model constraints are quoted at the owning task: identity/key/worker/override rules, status/submission enums, required fields, initialized arrays, omitted zero/empty evidence, assignment-only fields and count invariants. Retry ambiguity/backoff distinctions from the final data model are retained.
- Command execution, configured logger routing/severity/destinations, success INFO suppression, repeated-backend failure exclusion, JSON EOF/partial exits, both DEBUG entry points/redaction and actual terminal input are still mandatory. Parser or renderer tests do not substitute for T004.
- The dependency chain is T001 → T002 → T003 → T004. No `[P]` markers or conflicting parallel examples exist. Each task states its independently verifiable result and relevant checks.
- All four rows follow checkbox/ID/story/path syntax and remain unchecked. IDs are unique and sequential. No renumbering or superseded-ID mapping is needed.
- Local document links, whitespace, required phases, prerequisite references and coverage identifiers were checked. No standalone review section existed in tasks.md, so no review relocation was needed.
- No previously completed work or evidence was changed. The task list was not rewritten merely to create this review; only this review artifact was added by the consolidation hook.

## Limitations and checks not performed

This was a document and code-grounded review, not implementation. Proposed assignment-specific files/test families do not yet exist and must be implemented and name-checked before their commands count as validation. No application tests, documentation generator, live service discovery, deployment, runtime mutation or integration/smoke checks were run. Actual engine listener behavior remains unverified as recorded in the plan; it is not an implementation completion gate. No unresolved planning inconsistency was identified.
