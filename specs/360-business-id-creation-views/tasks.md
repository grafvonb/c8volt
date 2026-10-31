# Tasks: Business-ID Creation and Complete Resource Views

**Branch**: `360-business-id-creation-views`
**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [creation contract](contracts/creation.md), [resource-view contract](contracts/resource-views.md), [quickstart.md](quickstart.md)

**Organization**: Six cohesive work units, each with one primary story label and its own implementation, fixtures, tests, formatting, documentation, and targeted validation. US3 acceptance is embedded in these units as required by the plan. No setup, context-reading, commit, or final test-only task is needed. All tasks are initially unstarted.

## Format and execution responsibilities

Checklist entries use `- [ ] Tnnn [USn] description with paths`. No `[P]` markers are assigned: shared command models, report views, and generated documentation make concurrent editing unsafe without additional coordination. Paths are repository-relative; files explicitly called new are implementation outputs.

Before each work unit, read the linked design and `AGENTS.md`; Ralph execution also reads `specs/ralph-implementation-rules.md`. Keep the project compiling, include affected test doubles and converters together, and honor one primary Ralph story per iteration. Context checks and progress tracking are responsibilities, not extra tasks. Issue-associated commits, when the execution workflow reaches its commit step, use Conventional Commits and end with ` #360`.

Write or extend the relevant regression tests before changing behavior where practical, observe meaningful failure, then implement and pass them within the same task. Do not commit an intentionally broken intermediate test/model stage. Name new acceptance tests with `BusinessID` and require a nonzero test inventory; existing regression patterns must also run where affected.

Every unit must use only mocks, test-owned HTTP servers, and isolated subprocess/terminal fixtures. Follow the sanitization and fake-ownership checks in [quickstart.md](quickstart.md); all commands below assume its `isolated` helper. Never run integration tags/targets, discover/provision live environments, deploy fixtures, or perform live cleanup. Runtime checks below are implementation obligations and have not been executed during task generation.

### Shared completion contract

These requirements belong to each affected task and are not deferred to US3 or final polish:

- Keep CLI logic in `cmd`, mapping/error conversion in facades, backend scheduling/waiting in internal services, and native differences in version adapters. Reuse `toolx.CopyPtr`, existing pool/wait helpers, and configured logging. Do not edit generated clients or introduce a generic capability framework.
- Preserve the data-model constraints verbatim: "Nil means omitted"; "non-nil empty is invalid" for creation; "Never fill an absent value from an input, parent, sibling, or related process instance." Observed empty strings are retained and must not be validated as creation inputs. Use `BusinessId *string` with `json:"businessId,omitempty"` on public/output types and copy mutable pointer values.
- Preserve envelopes, enrichment sections, omitted/null/empty-array conventions, totals, output-mode precedence, and keys-only output (one actual key per line, zero bytes when empty). Render exactly one shared JSON envelope and require EOF in tests. No extra read solely to populate IDs in key/status-only results or local exact-key previews; rendering issues no requests.
- Human mutation, dry-run, and verbose result lines use configured INFO logging; ordinary query rows stay on stdout; prompts stay plain configured stderr. Test configured/inherited destinations, plain/structured formats, WARN and quiet suppression, and actual command execution with separate streams. Renderer-only or substring-only checks are insufficient.
- For changed failure paths, trace service → facade → renderer → logger → exit. Test repeated backend failures carrying distinctive method/URL/body/key data: one bounded aggregate ERROR, no raw HTTP details on either stream under normal or verbose-only output, quiet retains ERROR, both DEBUG entry points independently expose only appropriately redacted diagnostics, and stdout remains diagnostic-free. Preserve partial machine evidence and original error/exit classification; assertions must fail for logger bypass or duplicate emission.
- For changed empty scopes, cover normal/dry-run and supported human/JSON/keys/quiet, quiet+machine, automation/auto-confirm/no-wait combinations; authoritative discovery only, no confirmation/mutation, correct request counts, real terminal stdin when confirmation would otherwise be eligible. Keep sparse pages, nonempty/explicit-key, abort, EOF, and error regressions.
- Use safely quoted human IDs and escaped Markdown while preserving exact JSON/model values. Add Go doc/intent comments to new or materially changed declarations/tests, run `gofmt` on touched Go files, and run the closest checks before marking complete.
- Update affected command metadata/examples and README in the behavior's own task; run `isolated make docs-content` when command behavior/output changes and review generated differences. Do not hand-edit `docs/cli/`. Documentation-only follow-up does not invalidate prior runtime test evidence.

## Phase 1: Setup

The repository, toolchain, dependencies, generated clients, and test infrastructure already exist. No initialization task or new dependency is required. The first implementation unit starts directly after normal context and branch checks.

## Phase 2: Foundational

No separate infrastructure work blocks the feature. Shared process observation fields and the narrow validator are introduced with the first independently usable public creation outcome, T001. Later tasks reuse those fields instead of reimplementing them.

## Phase 3: User Story 1 — Create Instances with a Business Reference (P1, MVP)

**Goal**: Supply an exact optional reference through public creation and `run pi`, using either selector and either wait mode, while retaining successful results after a later failure.

**Independent test**: Use owned HTTP fixtures for both native request variants, observed/native response differences, old-version rejection, and counted mixed outcomes; execute the command with separate streams and assert exact payloads, partial evidence, one envelope, original exits, and configured logging.

- [x] T001 [US1] Deliver business-ID-aware public creation for single, sequential, and counted calls in `internal/domain/business_id.go` (new), `internal/domain/processinstance.go`, `internal/services/processinstance/v810/convert.go`, `internal/services/processinstance/v810/service.go`, `internal/services/processinstance/v88/service.go`, `internal/services/processinstance/v89/service.go`, `internal/services/processinstance/bulk.go`, and `c8volt/process/model.go`, `c8volt/process/convert.go`, `c8volt/process/client.go`, `c8volt/process/bulk.go`, with fixtures and tests in those packages; enforce "Maximum: 256 UTF-16 code units. Each Unicode code point at or below U+FFFF counts once; a supplementary code point counts twice." and "Nil is valid and omitted. Supplied empty strings and malformed UTF-8 are invalid input."; preserve successful partial slices plus original errors through the thin facade, and validate before completing the unit.

  **Implementation and acceptance**: Add optional input, native creation-result, and process-observation fields together. Map both by-ID/by-key payload variants, creation results, observed PI results, public input, creation output, and both directions of PI conversion. Copy values at mutable boundaries. On 8.8/8.9, reject any non-nil input as unsupported at adapter entry before value validation or mutation; do not allow bulk validation to change that precedence. v810 validates before each mutation. Snapshot counted input before scheduling, reuse existing workers/order/fail-fast/retry rules, retain accumulated sequential successes on error, and compact counted results using successful-return evidence so failed/unattempted slots never become zero-key resources; do not change shared pool semantics. Ordinary waiting adopts the observed ID even when nil; no-wait retains native response ID. A failed wait remains failure; do not invent per-attempt acceptance evidence or rollback.

  **Tests and validation**: Add `internal/domain/business_id_test.go` and extend `internal/services/processinstance/v810/convert_test.go`, versioned `service_test.go`, `internal/services/processinstance/bulk_test.go`, `c8volt/process/convert_test.go`, `c8volt/process/client_test.go`, and a focused public bulk test file if needed. Verify generated serialized bodies, omission, empty/whitespace/comma/mixed-case values, malformed UTF-8, 256/257 BMP, 128/129 supplementary, and 254/255 BMP plus one supplementary character. Test both selectors, single/list/count public APIs, v88/v89 unsupported input including invalid values with zero mutation calls, absence preserving old behavior, normal/no-wait source selection, pointer snapshots, sequential success then conflict, bounded out-of-order success/failure, fail-fast/cancellation, stable partial order, and retained error classes. Run `isolated go test ./internal/domain -run 'Test.*BusinessID' -count=1`, `isolated go test ./internal/services/processinstance/... -run 'Test.*BusinessID|Test.*Create.*ProcessInstance' -count=1`, and `isolated go test ./c8volt/process -run 'Test.*BusinessID|Test.*Create.*ProcessInstance' -count=1`; add the same targeted service/public race checks for changed counted execution. Document public input/partial-result behavior and unkeyed-literal compatibility in `README.md`; update doubles atomically if affected. Covers FR-001–FR-010 as applicable to creation and base PI data, SC-001/SC-003/SC-004, US3 scenarios 1–2.

- [x] T002 [US1] Deliver `run pi --bid` with complete creation output and partial-failure handling in `cmd/run_processinstance.go` and new `cmd/cmd_views_processinstance_run.go`, reuse `cmd/cmd_views_contract.go` and existing process/job logging helpers, and include execution/logger/terminal tests in `cmd/run_test.go` and new focused `cmd/run_processinstance_business_id_test.go` / `cmd/run_processinstance_logging_test.go`, plus command metadata, `README.md`, and regenerated `docs/cli/`; preserve explicit flag presence and render exactly one envelope with successful items and the original error class on partial failure.

  **Dependency**: T001.

  **Implementation and acceptance**: Use a scalar string flag with `Flags().Changed("bid")`, copying the optional value into every existing selector input, including supported multiple selectors and `--count`; never trim/split/change IDs or turn them into variables. Let adapters apply version/value precedence. Move the touched dedicated creation renderer/flat-row builder out of the command. Preserve result sorting, tenant evidence, accepted-vs-observed semantics and existing flag precedence. On partial failure render the existing `ProcessInstances` payload before exiting, with actual returned-item total, no zero slots, original error classification/`NoErrCodes`, and no second error renderer. A writer failure must not trigger another envelope or override an existing operation-error exit. Keys-only prints only successful keys; human rows and summaries use configured INFO logging and compact quoted IDs. Emit one bounded human aggregate ERROR using only supported outcome counts; unattempted requests are not backend rejections. Suppress duplicate service detail where needed using existing options, retaining functional verbose detail and independent DEBUG diagnostics.

  **Tests and validation**: Through actual command execution, cover both selectors/aliases, omission/explicit empty, supported and unsupported versions, exact comma/whitespace IDs, count with sequential and concurrent mixed outcomes, wait/no-wait including absent observations, failed validation, all-success/zero-success/partial failures, quiet+machine, keys-only zero bytes, actual original exits and renderer write failure. Exercise the full shared diagnostic contract with repeated raw transport failures, configured/inherited plain and structured loggers, normal/verbose/quiet/WARN, `--debug` and configured debug, redaction, single-envelope EOF, and no direct-print escape. Use existing real-terminal helpers for affected eligible controls and redirected stdout; introduce no new prompt. Retain selector/worker, activity/progress and no-wait regression tests. Run `isolated go test ./cmd -run 'TestRunProcessInstance|TestRunHelp|Test.*BusinessID' -count=1` plus focused logging/race cases as named; regenerate docs. Help/README must explain both selectors, 8.10-only input, opaque values, UTF-16 limit, uniqueness conflicts with repeated count, no-wait, partial outcomes, and the human stream correction. Covers FR-002–FR-007/FR-011–FR-017 for creation, SC-001/SC-003–SC-006, US3 scenarios 1–8 as applicable.

**Checkpoint / MVP**: T001–T002 deliver fully tested public and CLI creation. No resource-view task is required to demonstrate this story against isolated fixtures; complete-view acceptance remains US2 work.

## Phase 4: User Story 2 — Inspect Accurate Business References Everywhere (P1)

**Goal**: Preserve each resource's observed ID or absence in ordinary, enriched, historical/derived, preview, and saved report representations without extra reads.

**Independent test**: Fixtures supply parent `new`, child/job/task/listener `old`, nil, and observed-empty values simultaneously. Verify all eleven specification surface groups, round trips, pointer isolation, existing output shapes, request counts, and command-level stream behavior.

- [x] T003 [US2] Complete process-instance inspection and observed mutation-impact views in `c8volt/process/convert.go`, the duplicated process mapper in `c8volt/ops/convert.go`, `cmd/cmd_views_processinstance.go`, `cmd/cmd_views_processinstance_vars.go`, `cmd/cmd_views_processinstance_incidents.go`, `cmd/cmd_views_processinstance_activity.go`, `cmd/cmd_views_walk.go`, `cmd/cmd_views_walk_incidents.go`, `cmd/cmd_views_processinstance_dryrun.go`, `cmd/cmd_views_processinstance_update.go`, and `cmd/delete_processinstance_selector.go`, including process-definition impact and suspension/resumption view paths; add tests alongside these files and `internal/services/processdefinition/delete_test.go`, and deliver matching docs with the views.

  **Dependency**: T001; use T002 as the scheduled predecessor to finish the MVP first.

  **Implementation and acceptance**: Reuse the PI fields/mappers already delivered; finish duplicated operations mapping and all inward/outward/frozen plan copies. Cover single/list with age metadata, variable/incident/element/combined activity wrappers, family parent/child flat/tree enriched walks, observed mutation previews/plans, and deletion impact instances. Extend bespoke `processInstanceDryRunSelectedFinalState` and `processInstanceDryRunRequiresCancelBeforeDelete` entries and `aggregateDeleteSearchPlan` reconstruction only where the source is observed data. Preserve "Never fill an absent value from an input, parent, sibling, or related process instance." Key-only fallback/local preview values remain absent and read-free. Do not alter filters, traversal, sparse-page continuation, or enrichment section shapes. Listener ID additions are T005, but existing listener presence/absence behavior must already remain intact.

  **Tests and validation**: Extend `c8volt/process/convert_test.go`, `c8volt/ops/convert_test.go`, `cmd/cmd_views_processinstance_test.go`, activity/dry-run/variable/incident view tests, `cmd/walk_test.go`, command mutation-selector/definition-delete tests, and isolated `internal/services/processdefinition/delete_test.go`. Test present/nil/observed-empty and divergent parent/child values, round trips, one-envelope EOF, request counts, no extra discovery, human quoting, machine sections, totals and keys. Exercise affected mutation/dry-run summaries through command loggers including INFO/WARN, quiet, configured/inherited destinations and both formats. Cover authoritative empty discovery under real terminal stdin with no confirmation/mutation; preserve supported automation/no-wait combinations, nonempty, sparse pages, explicit keys, abort/EOF, and errors. Run targeted `BusinessID` and named changed regression patterns separately in `./c8volt/process`, `./c8volt/ops`, `./internal/services/processdefinition/...`, and `./cmd`; inspect inventory to avoid no-test matches. Update affected get/walk/mutation source metadata, `README.md`, and generated CLI docs. Covers FR-008–FR-017 for PI views/impacts and SC-002/SC-004–SC-006.

- [x] T004 [US2] Deliver job and user-task business IDs from native observations through ordinary/enriched views and job mutation candidates in `internal/domain/job.go`, `internal/domain/usertask.go`, `internal/domain/job_completion.go`, `internal/services/job/v810/convert.go`, `internal/services/usertask/v810/convert.go`, `internal/services/job/completion/prepare.go`, `internal/services/job/completion/execution_plan.go`, `c8volt/job/model.go`, `c8volt/job/client.go`, `c8volt/job/convert.go`, `c8volt/task/model.go`, `c8volt/task/convert.go`, `cmd/cmd_views_job.go`, `cmd/cmd_views_usertask.go`, and `cmd/cmd_views_usertask_vars.go`, with native/facade/plan/command tests and documentation in the same unit.

  **Dependency**: T003 in the scheduled sequence; only T001 is a data prerequisite, but shared README/generated docs are not safe parallel-edit targets.

  **Implementation and acceptance**: Add `BusinessId *string` with `json:"businessId,omitempty"` to public observations and completion candidates; preserve nil and backend-provided empty. Native mappings are v810-only; older reads remain unchanged. Update matching domain/public candidate structs together so direct casts remain compilable, then ensure explicit pointer copies in both directions and frozen execution batches. Preserve IDs when preparation and `jobCompletionCandidateRow` reconstruct job data. Cover ordinary single/list jobs, job-update previews, bulk-completion candidates, user tasks and variable-enriched tasks. Do not expose completion-time business-ID assignment or add related-instance reads. Reuse existing safely quoted row formatting rather than adding a second style.

  **Tests and validation**: Extend v810 job service and user-task native/search/variable tests, `internal/services/job/completion/prepare_test.go` and `execute_test.go`, `c8volt/job/client_test.go`, candidate conversion tests, `c8volt/task/convert_test.go` / `model_test.go`, `cmd/get_job_test.go`, `cmd/update_job_plan_test.go`, bulk-completion acceptance tests, `cmd/get_usertask_output_test.go`, and variable-enriched view tests. Prove pointer isolation by mutating the source after plan creation; cover exact observed IDs, absent/empty values, older-version baseline, no parent fallback/extra reads, command output and logger invariants including affected empty/partial-error paths. Run targeted `BusinessID` plus named affected regression cases in `./internal/services/job/...`, `./internal/services/usertask/...`, `./c8volt/job`, `./c8volt/task`, and `./cmd`; use race checks for changed frozen/counted candidate fixtures. Update command output metadata/README and regenerate docs. Covers FR-008–FR-017 for jobs/tasks/candidates, SC-002/SC-004–SC-006.

- [x] T005 [US2] Preserve independent listener and slow-analysis observations in `internal/domain/job.go` (`RuntimeListenerJobFromJob`), `internal/domain/ops_slow_process_analysis.go`, `internal/services/ops/slow_process_analysis.go`, public models/converters in `c8volt/process`, `c8volt/element`, and `c8volt/ops`, and `cmd/cmd_views_listener.go`, `cmd/cmd_views_element.go`, `cmd/cmd_views_processinstance_activity.go`, `cmd/cmd_views_ops_slow_process_analysis.go`, `cmd/cmd_views_ops_slow_process_analysis_machine.go`, with complete mapping/command tests and updated analysis/activity documentation.

  **Dependency**: T003 and T004 (PI and job observations).

  **Implementation and acceptance**: Add optional listener IDs to all three public listener models and copied pointers to their converters; preserve the data-model rule "Preserve `*[]RuntimeListenerJob` collection semantics." Map the slow-analysis root's own ID in `slowProcessAnalysisRootFromProcessInstance`, the public slow-analysis converter, and the human renderer that reconstructs `process.ProcessInstance`. Carry listener IDs through the activity/element/slow listener adapters and shared listener rows, default hotspot summaries and full timelines. Existing query results retain stdout routing and mutation contexts retain logger routing; do not inherit a newer root ID into older listeners or synthesize values for missing roots.

  **Tests and validation**: Extend `internal/domain/job_test.go`, `internal/services/ops/slow_process_analysis_test.go`, `c8volt/process/convert_test.go`, `c8volt/element/client_test.go`, `c8volt/ops/convert_test.go` / `model_test.go`, listener/element/activity/slow-analysis view tests and corresponding command tests. Use one fixture with root `new`, listener/job `old`, nil and observed-empty, verify all conversions, requested-empty versus unrequested listener sections, no extra requests, JSON EOF, normal/full-timeline/keys/quiet behavior where supported, and safely escaped human IDs. Run targeted `BusinessID` and named listener/activity/slow-analysis regressions in `./internal/domain`, `./internal/services/ops/...`, each changed facade, and `./cmd`. Update affected source metadata, `README.md`, and generated docs. Covers FR-008–FR-017 for listener/slow-analysis surfaces and SC-002/SC-004–SC-006.

- [x] T006 [US2] Complete observed operations previews and saved report output in `c8volt/ops/convert.go`, `cmd/cmd_views_ops_execute_retention_policy.go`, `cmd/cmd_views_ops_purge_orphan_processinstances.go`, `cmd/cmd_views_ops_purge_processinstances_with_incidents.go`, `cmd/cmd_views_ops_purge_all_processdefinitions.go`, `cmd/cmd_views_ops_repair.go`, and `cmd/ops_report_markdown.go` plus applicable `cmd/ops_report*.go` writers; include execution/report tests in `cmd/ops_report_json_test.go`, `cmd/ops_report_markdown_test.go` and affected operation command tests, and deliver documentation plus the final cross-package validation evidence in `specs/360-business-id-creation-views/progress.md` when implementation runs.

  **Dependency**: T003–T005; final validation also requires T001–T002.

  **Implementation and acceptance**: Reuse completed mappings and add IDs to custom scalar lists such as `retentionProcessInstanceItems`, shared incident-purge report paths, repair/orphan/retention/definition-purge previews, and all report sections containing observed process-instance or job data. Preserve key/status-only placeholders and local exact-key plans without reads. Escape Markdown delimiters/line breaks/markup while keeping underlying values exact; retain report schema, counts, compact aggregate human output, logger routing, and report-file generation semantics. Do not add an ID lookup or propagate the root's value to another resource. Finish any actual missing operation copy site discovered against the explicit surface inventory, without unrelated refactoring.

  **Tests and validation**: Execute affected operations against owned HTTP fixtures and compare actual saved JSON/Markdown files as well as result streams, including divergent/absent/observed-empty IDs, quoting/markup, partial reports, empty/no-op discovery and request counts. Apply the shared logger/terminal/error contract to affected execution paths, not just writer functions. Run targeted `BusinessID` and named operations/report regressions in `./c8volt/ops`, affected internal service packages, and `./cmd`; update metadata/README and regenerate docs. After these checks pass, run `isolated make test PKG=./...` once for final cross-package model/pointer/creation-error propagation and record actual results plus any platform skips. This is part of delivering complete reports, not a separate test-only task; do not rerun for later prose-only edits. Covers FR-008–FR-017 for operations reports and the remaining resource-view acceptance inventory, SC-002/SC-004–SC-006.

**Checkpoint**: Every row of the resource-view contract has command-level or report-file proof in addition to required model/native tests. Related resources retain their own IDs and no extra population requests occur.

## Phase 5: User Story 3 — Keep Existing Versions and Automation Reliable (P2)

**Goal**: Existing version behavior, machine output, logging, documentation, and automation remain dependable.

**Independent test**: Run isolated 8.8/8.9/8.10 creation/read scenarios, single-envelope/EOF and keys/count assertions, inherited/configured plain and structured logger capture, and real-terminal no-op/prompt scenarios. Required checks are implemented with the behavior in T001–T006; this phase is an acceptance checkpoint, not a duplicate testing task or permission to defer compliance.

| US3 acceptance | Owning work units |
| --- | --- |
| 1. Unsupported input before mutation, including public calls | T001 native/public; T002 command |
| 2. Existing no-ID behavior and default version | T001–T006 in each affected version/resource family |
| 3. One envelope, partial evidence, quiet+JSON | T002 creation; T003–T006 affected view/operation paths |
| 4. Keys-only and totals | T002–T006 |
| 5. Configured INFO routing and suppression | T002 creation; T003/T004/T006 mutation/preview paths; T005 mixed rendering contexts |
| 6. Bounded ERROR, independent DEBUG and redaction | T002 failure path; T003–T006 wherever changed output reaches execution errors |
| 7. Plain prompts, no extra prompts/reads | T002–T006, with real terminal tests for eligible affected paths |
| 8. Accurate help, examples, README and generated docs | Each behavior-delivery unit; full review in T006 |

**Task count**: US3 has zero standalone tasks because its eight acceptance scenarios are inseparable completion criteria for the six implementation tasks. This preserves story traceability without repeating work across Ralph story scopes.

## Phase 6: Polish and Cross-Cutting Completion

No standalone polish task. Formatting, source/docs alignment, closest tests, output review, and progress evidence belong to their owning implementation task. T006 performs the once-per-feature race suite and complete surface acceptance review after the preceding implementation is available. No runtime check is a task-generation completion gate and no live check is permitted.

## Dependencies and execution order

| Task | Technical prerequisites | Scheduled order |
| --- | --- | --- |
| T001 | Existing repository only | First |
| T002 | T001 | After T001 |
| T003 | T001 core PI field | After T002 to deliver creation MVP first |
| T004 | T001 pointer/row conventions; no runtime dependence on T003 | After T003 to avoid shared docs/output editing |
| T005 | T003 PI views and T004 jobs | After T004 |
| T006 | T003–T005; T001–T002 for final validation | Last |

```text
US1: T001 → T002 → creation MVP
                  ↓
US2:             T003 → T004 → T005 → T006 → complete resource views/reports
US3: enforced within every owning unit; acceptance closes with T006
```

The scheduled sequence is deterministic for Ralph. No unfinished model/interface change is split from the necessary implementation or test-double repair. Public creation and observed PI fields are built once in T001 and reused by US2.

## Parallel execution examples

No parallel implementation tasks are marked; existing file overlap and documentation generation justify serial edits, not artificial task splitting.

- **US1**: After T001's edits stabilize, domain-validator tests and facade/native tests may run in independent test processes against separate fixtures. T002 waits for T001; do not edit shared creation files concurrently.
- **US2**: Within a stable task, independent resource mapping tests and renderer/command tests can run concurrently if each owns its fixtures and no shared output directories are written. T005 cannot start until T004's job observations exist; report/doc generation stays serial.
- **US3**: Plain and structured logger cases may run in isolated subprocesses after the owning edit. Do not use `t.Parallel` on cases mutating shared Cobra globals, process environment, or command trees. This is validation concurrency, not a new task or authorization to launch additional agents.

## Implementation strategy and coverage

1. Deliver US1 with T001–T002 and demonstrate it only through isolated acceptance fixtures. This is the MVP: exact ID creation and truthful partial results.
2. Deliver complete PI inspections, job/task observations, listener/slow views, and operations reports incrementally in T003–T006. Each task leaves a compilable, independently verifiable result.
3. Enforce US3 from the first unit. Complete final broad validation with T006; report actual coverage and unverified real-engine behavior accurately.

| Requirement group | Owner |
| --- | --- |
| FR-001 version/default compatibility | T001–T002; older-read regressions T003–T006 |
| FR-002–FR-007 creation, input, count, partial outcomes | T001–T002 |
| FR-008–FR-010 mappings, copied observations, no synthesis | T001 base PI, T003 PI/operations, T004 jobs/tasks/candidates, T005 listeners/slow, T006 report copies |
| FR-011 complete surfaces | T002 creation, T003 PI/walk/impacts, T004 job/task, T005 listener/slow, T006 operations/files |
| FR-012–FR-016 machine/stream/no-read/error contracts | Owning output task T002–T006; all share the completion contract |
| FR-017 documentation | T001 public behavior; T002–T006 user-visible behavior in the same unit |

**Success criteria**: SC-001 and SC-003 are proven by T001–T002; SC-002 by T001/T003–T006; SC-004–SC-006 by the per-unit compatibility/output/documentation checks and T006's final coverage review. All eleven specified surface groups, both native creation variants, all three version policies, and every specified isolated validation category have an owner.
