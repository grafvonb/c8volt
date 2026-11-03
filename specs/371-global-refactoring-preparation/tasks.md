# Tasks: Global compatible refactoring

**Status:** planned, not started. **Input:** [spec.md](spec.md), [plan.md](plan.md),
[research.md](research.md), [data-model.md](data-model.md), [compatibility contract](contracts/compatibility.md)
and [isolated validation guide](quickstart.md). All tests/paths named below are implementation
requirements, not files/tests created or executed by planning. The twelve finding IDs retain
the audit's order; task order follows dependencies and change risk instead.

## Execution requirements for every work unit

Read current feature/repository context, confirm working-tree safety and check the task's actual
owner before edits. Do not activate this feature or launch implementation from this document.
Future execution requires separate authorization. Preserve the scan prompt and completed 370
work. Do not stage, commit, push or launch Ralph unless separately authorized.

Each task includes necessary code, adapters/callers/doubles, fixtures, comments, targeted tests,
formatting and validation as one compiling outcome. Write meaningful regression assertions
before the fix; for a pure extraction retain/characterize existing behavior first, then prove
its preservation. Keep native protocol tests where they provide independent evidence. No
standalone setup, context-reading, formatting, review or commit checkbox. New Go declarations
and materially changed tests require comments. Never edit generated clients or run live tests.

Apply C01–C10 verbatim from spec.md. In particular, preserve public type identity and existing
interface method sets; use additive free functions rather than required methods. Preserve CLI
grammar/defaults/flags/aliases, output shapes, exact established bounded text, config precedence,
request counts/order, tenant/version restrictions, empty/sparse/partial/abort behavior and
submission-versus-confirmation facts. Keep legacy public exports even without local callers.

For command/output changes, execute the complete relevant contract matrix through isolated
command execution with configured/inherited loggers, separate streams, plain/structured formats,
INFO/WARN/quiet, verbose, redacted DEBUG via flag and configured level, original exits/partial
results and exactly one JSON envelope followed by EOF. Include repeated multi-resource fake
HTTP failures, real-terminal prompts/empty scopes and successful/dry-run summaries. These must
detect direct prints, logger bypass, raw normal-level errors and duplicate envelopes. C05 gates
any observable legacy diagnostic correction; do not weaken assertions or silently change text.

Before any validation, inspect flags/build tags/test dependencies and clear inherited GOFLAGS.
Use only test-owned fakes. Start with the task's command in quickstart.md after introducing its
TestRefactor* cases; a zero-test match does not pass. Broaden only for affected existing suites
or new evidence. Run gofmt on changed Go and git diff --check within each unit. No generated CLI
docs refresh unless a separately accepted metadata/behavior change actually requires it.

## Phase 1: Setup responsibilities

The repository and runtime already exist. Context, baseline/API capture and branch checks are
responsibilities of implementation units; no project initialization or standalone setup task.

## Phase 2: Shared foundations

US1 delivers working shared primitives with consumers and acceptance coverage. No interface-
only or registration-only task may leave consumers uncompilable between work units.

## Phase 3: US1 — Safe, correctly layered shared primitives (P1)

**Independent acceptance:** frozen JSON/payload ownership and existing public pool callers pass;
internal service worker dependencies are neutral. Initial bounded scope is T001, not a claim
that the whole global refactor is complete.





- [ ] T001 [US1] Deliver shared compatible JSON value operations and all concrete consumers in one unit: add mechanical normalization/comparison/validated snapshots in `toolx/json_value.go` alongside `toolx/json_copy.go`; migrate `internal/services/{element,processinstance,usertask}/update.go`, their `update_validation.go` helpers and `c8volt/{element,process,task}/convert.go`. Keep normalization separate from representation-preserving copying; retain accepted JSON, exact numeric values, established concrete types, nil/empty conventions and every owner's validation timing/error wrapper. Reuse CopyJSONMap only where its existing semantics match. Add meaningful `TestRefactorJSON` assertions before changes for typed/ordinary nested collections, arrays, RawMessage, custom marshalers, invalid/cyclic input and precision; prove caller/adapter/result mutation cannot change frozen approved payloads. Include explicit affected external-interface/API compile probes, required fakes, comments and formatting. Run the T001 command in quickstart.md and affected existing variable suites. All consumers must compile and pass before completion; do not ship an element-only policy increment or a shared helper with unconverted equivalent callers.

- [ ] T002 [US1] Deliver a neutral worker engine with every internal consumer and original public compatibility in one unit. Extract the existing algorithm from `toolx/pool/pool.go` into `toolx/pool/execution/pool.go`; keep the original ExecuteSlice/ExecuteNTimes/ExecuteBulkOperation/Reporter/Reports exports, exact signatures, option identity and legacy helper logs as delegating entry points in `toolx/pool`. Move/adapt private stagger coverage with the engine and retain public generic/FacadeOption probes. Enumerate and migrate every current production internal importer, including incident bulk/lookup/plan, ops smoke/analysis/repair/retention/cleanup, element execution, job/completion, processdefinition delete, processinstance bulk/variables/update/enrichment/business-ID/dry-run/lifecycle/frozen-delete, versioned bulk adapters/waiter and usertask bulk/assignment/completion/planning/update. Update necessary doubles atomically. Remove unused `internal/services/common/bulk.go` only after full reference audit. Add a transitive production-import guard, preserve ordering/slots, cancellation, fail-fast, worker normalization, staggering, joined errors and DEBUG records, and run the T002 race command plus affected existing consumer suites. No facade option type is moved/redefined and no exported legacy helper is deleted. The dependency closure, wrappers and first/all consumers must be complete together.

## Phase 4: US2 — Stable command results and diagnostics (P1)

**Independent acceptance:** six formatter transcripts remain exact; command tests establish
configured logging, partial/error/empty behavior, plain prompts, original exits and single
machine envelopes. A C05 mismatch is explicitly blocked pending correction acceptance.

- [ ] T003 [US2] Consolidate first-seen bounded failure aggregation into `cmd/cmd_failure_summary.go`, consuming allowlisted reasons from `cmd/cmd_views_job_logging.go`, `cmd_views_usertask_assignment.go`, `cmd_views_usertask_complete.go`, `cmd_views_element_update.go`, `cmd_views_processinstance_suspend.go` and `cmd_views_processinstance_resume.go`. Keep six local classifiers and their labels, count eligibility, punctuation, ordering, fallback/other-bucket text and mode policy; never parse raw errors. Characterize current transcripts, add `TestRefactorFailureSummary` table/command cases, format and run targeted checks plus the exact existing affected logging families.

- [ ] T004 [US2] Consolidate legacy command failure/exit plumbing around `cmd/cmd_views_contract.go`, bootstrap helpers and configured logger/context. Inventory direct ferrors exits in `cmd/embed_deploy.go`, `embed_export.go`, `get_processdefinition.go`, `update_usertask.go`, config commands and shared error helpers; keep the public `c8volt/ferrors` API and local/bootstrap/output-write distinctions. Introduce only necessary `cmd/cmd_failure_dispatch.go` support and `cmd/refactor_command_failure_test.go` fake command/subprocess/terminal coverage. Preserve original errors, partial payloads, emission-attempt state, no-err-codes and all stated CLI contracts. Exercise the full diagnostics matrix including success and dry-run. If raw legacy wording conflicts with mandatory logging, save the exact isolated reproduction/proposed bounded correction in future feature evidence and hold only that part pending explicit acceptance under C05; do not implement observable wording changes or call failing mandatory cases passed. Run `TestRefactorCommandFailure` then affected existing command/bootstrap families.

## Phase 5: US3 — Efficient task/scope fan-out (P2)

**Independent acceptance:** indexed and reference outputs/rejections are identical; backend
call counts are unchanged; pure local measurements show the actual work/allocation profile.

- [ ] T005 [US3] Build and reuse immutable invocation-local scope/task/association indexes in `internal/services/usertask/update_validation.go`, `update.go` and focused `scope_index.go`; use existing identity types and keep source slices as order owners. Replace repeated findScopeVariableTarget/targetAssociatesTask/associatedNamesForTask scans during validation and result assembly without relaxing coverage, duplicate, tenant or payload checks. Add `TestRefactorScopeIndex` equivalence, sharing, sparse/disordered/invalid-plan and request-count cases and `BenchmarkRefactorScopeIndex` pure local sparse/shared scaling with ReportAllocs. Preserve verification ordering, copies and worker semantics; run targeted race tests, the local benchmark and affected existing bulk/variable cases. No network benchmark, remote cache or wall-clock pass threshold.

## Phase 6: US4 — Shared process-instance lifecycle mechanics (P2)

**Independent acceptance:** separate suspend/resume scenarios retain frozen scope, eligibility,
acceptance/uncertainty and confirmation behavior through every native/CLI boundary.

- [ ] T006 [US4] Share descendant traversal, completeness checking, ordered deduplication and immutable plan facts from `internal/services/processinstance/suspend_plan.go`, `resume_plan.go` and `descendant_plan_validation.go` in a focused tree-plan file. Keep per-action capability, selected-key normalization, empty error, preview/timing preflight, eligibility and public plan/result types explicit. Reuse existing walker and copy helpers, including T001 where applicable; do not requery during rendering or widen scope. Add/retain `TestRefactorTreePlan` and current service/facade tree fixtures for overlapping roots, independent descendants, sparse/missing evidence and errors; run targeted plan tests and affected command tree/terminal cases.

- [ ] T007 [US4] Share compatible preview-before-dispatch, worker scheduling, ordered merge and skipped-slot/error handling in `internal/services/processinstance/suspend.go`, `resume.go` and focused `lifecycle_bulk.go` using the neutral execution core. Keep public signatures, action-specific adapter calls/statuses, automatic/explicit/unlimited workers, empty/duplicate/invalid keys, dry-run/no-wait, cancellation and fail-fast semantics. Include required service/facade/command double adaptations in this task; add `TestRefactorLifecycleBulk` and retain existing lifecycle bulk/logging/JSON/keys/terminal regressions. Run targeted race validation and exact affected CLI families, proving no new mutation/confirmation calls or fabricated acceptance.

- [ ] T008 [US4] Consolidate repeated v810 HTTP response/evidence mechanics in `internal/services/processinstance/v810/suspension_submission.go`, `resumption_submission.go` and a focused native lifecycle submission file. Keep native endpoint/body selection and operation labels explicit. Preserve valid metadata checks, only documented empty 204 acceptance, explicit versus ambiguous rejection, unknown/accepted evidence across retries, local request-rate failure restoration, ctx/network timeout class and independent redacted DEBUG. Reuse existing RetryCamundaMutation/httpc admission helpers; do not generalize endpoint-specific problem schemas. Add `TestRefactorLifecycleSubmission` paired fake-response/retry/admission cases and retain native pacing/protocol suites; run targeted race tests and relevant command failure evidence checks.

- [ ] T009 [US4] Share bounded observation/timer/backoff mechanics in `internal/services/processinstance/waiter/suspension.go`, `resumption.go` and focused `lifecycle_observation.go`. Retain separate reader/public result compatibility, allowed states, identity and authorization/404 behavior, timestamps, context-after-read check, attempt semantics and exact errors. Quote the policies in tests: suspension confirms SUSPENDED; resume confirms ACTIVE or COMPLETED; terminal conflicts differ. Keep matching reads independent from submission proof. Add `TestRefactorLifecycleWait`, retain existing waiter/facade/adapter cases, and run targeted tests plus affected lifecycle command timeout/no-wait/terminal regressions. No shared retry path may resubmit a mutation while observing.

## Phase 7: US5 — Consistent variable-page decoding (P2)

**Independent acceptance:** three adapters per area pass the same raw corpus while retaining
native wire assertions; effective-required and root/local-optional metadata remain distinct.

- [ ] T010 [US5] Extract version-independent effective-variable response structs/decoding from `internal/services/usertask/v88/variables.go`, `v89/variables.go`, `v810/variables.go` into `internal/services/usertask/variablepage/decode.go`. Avoid factory/adapter import cycles; preserve required field presence, empty values, totals/capped metadata, raw count, truncation, tenant/owner identities and offset continuation facts. Keep native request/query construction in each version and existing usertask traversal in its owner. Introduce `TestRefactorEffectiveVariablePage` shared raw corpus and per-version delegation/wire checks; migrate affected fixtures/assertions in the same compiling unit. Run targeted all-version decoder and existing effective-variable/page/sparse tests.

- [ ] T011 [US5] Extract compatible root/local raw response decoding from `internal/services/variable/v88/convert.go`, `v89/convert.go`, `v810/convert.go` into `internal/services/variable/variablepage/decode.go`, preserving generated-type converters/native requests in adapters. Retain distinct legacy-root versus strict-local filtering, optional totals/presence, truncation, owner/tenant checks, raw-before-filter counts, cursor/offset and malformed-response behavior; reuse existing variable/search traversal without merging its protocol into effective variables. Add `TestRefactorScopeVariablePage` cross-version raw corpus plus native root/local wire checks, adapt existing scope/legacy tests and run the targeted variable/... suites, including sparse/cursor/overflow/empty/error cases. Do not tighten accepted legacy responses as incidental consolidation.

## Phase 8: US6 — Service-owned command preparation (P2)

**Independent acceptance:** identical fake query/deployment transcripts, existing external
facade implementers compile, and command rendering/prompt stages do not change.

- [ ] T012 [US6] Move selector normalization, per-ID existence/latest/version/tag query strategy and near-match fallback from `cmd/process_definition_selector_validation.go` into `internal/services/processdefinition/selectors.go`; adapt public inputs/results in additive `c8volt/process/selectors.go` free functions using an adapter over existing page-search methods. Add necessary internal/domain selector facts and conversions; do not add methods to existing public process/root interfaces or import facades into services. Keep command-owned no-prompt/TTY recovery/listing/error formatting and exact option/tenant/request ordering. Include affected command consumers and doubles together. Add `TestRefactorSelectorPreparation` service/facade cases and explicit existing-interface implementation compile probes; run targeted tests and current selector/machine/terminal/error cases.

- [ ] T013 [US6] Move deployment-to-run identifier precedence and XML fallback from `cmd/cmd_deploy_run_data.go` into `internal/services/resource/deployment_run.go` with an additive mechanical public preparation function in `c8volt/resource/deployment_run.go`; adapt `cmd/deploy_processdefinition.go` and `embed_deploy.go` to use it. Keep file/embedded acquisition and visible deploy-render-then-start dispatch in cmd, reuse existing service-owned ordered creation, and leave public interfaces unchanged. Preserve key/definition-ID/BPMN fallback order, tenant/options, duplicates, malformed XML/no-ID errors and partial deployment evidence if later creation fails. Add `TestRefactorDeploymentRun`, migrate existing preparation fixtures, and run pure service/facade checks plus deployment/embedded/creation stage, no-wait, JSON/keys, logging and request-count cases with fakes. Do not introduce a combined envelope or new start mode.

## Phase 9: US7 — Maintainable cross-version contracts (P2)

**Independent acceptance:** every prior meaningful version/case/assertion has an owner in the
shared harness or an explicit native-version test; test dependencies remain isolated.

- [ ] T014 [US7] Consolidate remaining repeated service-factory selection and native usertask assignment/completion test mechanics using existing `testx` and narrowly named area-level external-test contract fixtures, covering `internal/services/*/factory_test.go` and `internal/services/usertask/{v88,v89,v810}/{assignment,completion}_test.go`; reuse existing job/coordinator and the T010/T011 shared corpora instead of rebuilding them. Keep supported-version/capability errors, request bodies, exact values, retry/logging/admission and negative evidence assertions. Record old-test -> retained-contract ownership in feature evidence, add `TestRefactorVersionContract` where needed, and run each affected factory/native contract and its existing wire suites. No production test hooks, reduced matrix, live helpers or catch-all fake server; tests still expose wrong endpoint/version payloads and copied-fixture drift.

## Phase 10: US8 — Cohesive files and maintained tooling (P3)

**Independent acceptance:** packages retain declarations/contracts after movement, only proven
internal/private dead paths disappear, and docs workflows behave identically under fake tools.

- [ ] T015 [US8] Regroup remaining variable planning/execution/validation concerns in `internal/services/usertask/update.go`, `internal/services/processinstance/update.go` and neighboring focused update files after shared JSON/index/decoder migrations. Move execution into named execution files and planning into owning plan files; keep semantics, service interfaces, helper ownership and comments unchanged. Do not repeat upstream helper extraction or add generic helper files. Compare moved declarations/diff, preserve required doubles, and run affected existing variable suites plus the relevant TestRefactor cases; reuse prior race evidence only when no executable change invalidates it.

- [ ] T016 [US8] Regroup public process and ops models/converters within their current packages: `c8volt/process/{model,convert}.go` into focused variable/lifecycle/selection/business-ID files, and `c8volt/ops/{model,convert}.go` into repair/cleanup/smoke/analysis files. Preserve all exported names/signatures/type identity, field/tag/JSON conventions, methods, private approval ownership and init behavior; retain legacy converter and constructor signatures. Include existing explicit consumer compile probes and model/conversion fixtures, compare declaration/API inventories before/after, and run facade suites plus any truly affected command contract tests. This is mechanical cohesion only, with no public renames, type aliases replacing established types, field cleanup or unrelated algorithm changes.

- [ ] T017 [US8] Remove only proven unused internal/private scaffolding in `internal/services/common/envs.go`, `cmd/get_variable.go` and other reference-audited private wrappers, retaining config's maintained binder owner and all current public API exports. Preserve `c8volt/variable` package/API/New and toolx/pool/ferrors compatibility entry points; document their deliberate retention. Confirm registration/help/capabilities/completion do not acquire or lose a live command and config/env precedence stays unchanged. Add `TestRefactorScaffolding` only for meaningful compatibility/dependency assertions, run targeted config/common/cmd checks, final public/transitive-import scans, and the single justified broad isolated race checkpoint described in quickstart.md. Preserve historical specs and scan prompt; no generated/source/runtime cleanup outside verified scope.

- [ ] T018 [US8] Share duplicate docs build/publishing steps from `.github/workflows/docs-release.yaml` and `release.yaml` using a local `.github/actions/publish-docs/action.yml`; route an obsolete `scripts/docs-serve.sh` wrapper through `scripts/docs-site.sh` only if its interface/behavior is retained. Preserve triggers, permissions, setup versions, manual VERSION validation/precedence, concurrency, secret forwarding, publish path and server flags. Add isolated `TestRefactorDocumentationWorkflow` YAML/shell checks in `testx` using fixture-owned fake make/sudo/apt/lftp and runner commands with dummy secrets, and required script syntax checks. Never publish, install dependencies, run actual Jekyll/SFTP/server or execute integration runners. Keep API generation/provenance/version pins and integration safety gates unchanged. Update only relevant tooling documentation if invocation details need clarification; run no application test merely for documentation completion.

## Phase 11: Cross-cutting completion

No standalone polish checkbox. Each unit owns its acceptance evidence and formatting; the last
runtime cleanup unit owns the broad isolated race/API/dependency checkpoint. Tooling validation
is isolated afterward. No current checkbox is complete, no runtime validation is claimed, and
an unresolved required diagnostic case must be recorded as blocked rather than hidden.

## Dependency graph

| Task | Prerequisites | Reason |
| --- | --- | --- |
| T001 | None | Shared JSON plus all concrete consumers form one consistent contract |
| T002 | None | Neutral engine, old API wrappers and complete internal dependency closure |
| T003 | None | Existing formatter facts are sufficient for exact transcript preservation |
| T004 | T003 | Stable bounded aggregation before routing convergence |
| T005 | T001, T002 | Snapshot/worker contracts settled before indexed worker fan-out |
| T006 | None | Pure tree-plan extraction using existing walker |
| T007 | T002 | Shared collection scheduling uses neutral core |
| T008 | None | Native response/evidence extraction remains endpoint-owned |
| T009 | None | Observation mechanics remain independent of mutation submission |
| T010 | None | Effective-variable raw contract is self-contained |
| T011 | None | Root/local raw contract is self-contained |
| T012 | T004 | Stable command-aware error mapping for moved selector failures |
| T013 | T004 | Stable staged failure handling for deploy/start |
| T014 | T001, T002, T005, T006, T007, T008, T009, T010, T011, T012, T013 | Runtime and affected test consumers settle before residual contract consolidation |
| T015 | T001, T005, T010, T011, T014 | Group surviving service concerns once upstream edits settle |
| T016 | T001, T006, T007, T008, T009, T012, T013, T014 | Group public declarations after runtime/facade edits settle |
| T017 | T002, T014, T015, T016 | Last runtime cleanup and cross-package acceptance |
| T018 | None | Independent tooling; scheduled last to preserve runtime evidence |

Default execution is numerical, serial order. No [P] markers: no agent launch or concurrency is
authorized, command suites may share globals, and adjacent edits overlap package/test ownership.
Read-only review of independent task evidence can be separated after explicit coordination;
hypothetical parallelism is not a reason for task boundaries.

## Story-level coordination examples

US1 coordinates all shared primitive consumers; US2 coordinates six reason formatters then
command result/exit paths; US3 pairs algorithm equivalence with local benchmark review; US4
checks action-specific protocols independently while one owner integrates shared mechanics;
US5 keeps effective versus root/local corpora separate; US6 retains CLI prompt/render policy;
US7 maps each old assertion to retained native/shared coverage; US8 groups settled files and
validates tooling with dummy tools. These are ownership notes, not instructions to spawn agents.

## Finding and compatibility ownership

| Finding | Final task owner(s) |
| --- | --- |
| F01 | T001 |
| F02 | T002 |
| F03 | T004 |
| F04 | T005 |
| F05 | T006, T007, T008, T009 |
| F06 | T003 |
| F07 | T010, T011 |
| F08 | T012 |
| F09 | T013 |
| F10 | T014; necessary implementation tests already belong to their units |
| F11 | T015, T016 |
| F12 | T017, T018; unused internal bulk removal also T002 |

C01 is owned by every signature/model-touching task; C02–C05 by all affected command consumers
and US2; C06 by US1/US3/variable conversions; C07 by all workflow units; C08–C10 by every unit.
Current evidence is planning-only. All proposed test names must exist and run before their
owner can be marked complete. First delivery increment is the shared primitive story; global
completion requires all twelve finding dispositions and all remaining work, not merely that
increment. The required consolidation review is saved in consolidation-review.md: 20 draft tasks became 18 cohesive outcomes, preserving all requirements.
