# Tasks: Late Business-ID Assignment and Atomic Job Completion

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [public contract](contracts/public-api.md), [quickstart.md](quickstart.md).

**Tests**: Required by the specification. Each implementation checkbox includes its fixtures, tests, formatting, relevant documentation and targeted validation. Write the relevant behavioral assertions first and observe their expected failure before implementing; keep the final unit compiling. Use isolated fakes only.

**Format**: `- [ ] Tnnn [P?] [USn] Outcome with exact file paths`. Paths are repository-relative. No [P] marker is assigned because the retained implementation units share contracts or depend on prior units.

## Phase 1: Setup

The existing Go project, generated clients, test tooling and dependencies are sufficient. No initialization, live-target discovery, provisioning or standalone context-reading task is required. Every work unit reads active artifacts and repository rules, checks inherited test settings, updates its evidence, and preserves unrelated work.

## Phase 2: Foundational

Shared value validation is implemented and tested with its first assignment consumer in T001; no standalone prerequisite checkbox remains. Existing infrastructure needs no separate work unit.

## Phase 3: User Story 1 — Assign a Business Reference After Process Start (P1, MVP)

**Goal**: Preview and assign to exact explicit process keys, with service-owned copied plans, bounded execution and truthful observation.

**Independent test**: Owned fakes prove exact key/body, distinct-key submission, child rejection without parent writes, normal confirmation/no-wait, dry-run, partial results and lost-response safety. Command execution then proves the operator workflow.

- [x] T001 [US1] Deliver the process assignment service and public plan/execute API in `internal/domain/process_instance_business_id_assignment.go`, `internal/services/processinstance/business_id_assignment.go`, `internal/services/processinstance/api.go`, `internal/services/processinstance/v810/contract.go`, `internal/services/processinstance/v88/contract.go`, `internal/services/processinstance/v89/contract.go`, and `c8volt/process/business_id_assignment.go`, with exact-key observation and isolated service/facade tests.
  - In `internal/domain/business_id.go` and `internal/domain/business_id_test.go`, implement and test `ValidateBusinessIDAssignment(*string)` using the existing validator plus whitespace-only rejection. Preserve these model constraints verbatim: “Nil means absent in existing job operations. Direct assignment requires a value.” “Reject empty, whitespace-only, invalid UTF-8, and more than 256 UTF-16 code units. Preserve accepted bytes, including surrounding whitespace, case and commas.”
  - Test nil, empty/blank, invalid UTF-8, padded/comma/case/numeric-looking values, BMP and supplementary-character boundaries at 256/257 UTF-16 units; leave creation's whitespace-accepted regression intact. Run `env GOFLAGS= go test ./internal/domain -run 'TestValidateBusinessID' -count=1` and gofmt touched Go files.
  - Add models/converters in `c8volt/process/model.go` and `c8volt/process/convert.go`, public methods in `c8volt/process/api.go`, constructor wiring in `c8volt/process/client.go` and aggregate `c8volt/api.go` where needed, and all affected interface implementations/doubles in this same compiling unit. Facades only copy/map/delegate/convert through ferrors.
  - Preserve model constraints verbatim: “Required exact frozen assignment”; “Complete validated keys, normalized and deduplicated in stable order”; “One item per distinct target; never expanded to parent/family”; “Optional prior fact; absence is not synthesized”; “Authoritative complete preparation or incomplete diagnostic evidence.” Validate the entire input and plan readiness before writes; copy key/value storage; reject inconsistent/forged plans. Explicit empty prepared scope is successful, while incomplete scope is not executable.
  - Use existing key normalization, `toolx/pool` and call-option/backoff patterns. Add exact native v810 assignment under `internal/services/processinstance/v810/business_id_assignment.go`, widening its narrow generated-client interface and mocks as required; generated client files stay untouched. Guard unsupported v88/v89 planning and execution, including caller-created plans, before I/O. Never traverse or redirect to a parent, look up variables or probe live uniqueness configuration.
  - Submit exactly once per distinct key. Backend rejection remains rejection for child/inactive/existing BID including identical BID/uniqueness-enabled fixtures. Classify accepted, rejected and ambiguous evidence without RetryCamundaMutation or fallback. Retain original errors and ordered partial/not-started outcomes under concurrency, fail-fast and cancellation.
  - Add bounded exact-key observation under `internal/services/processinstance/waiter/business_id_assignment.go` if needed. After known acceptance, normal execution observes exact BID or returns accepted-unconfirmed failure; no-wait skips confirmation reads. Lost responses remain unknown even if later state matches. Never replay to resolve uncertainty. Keep requested, prior and later observations distinct and never backfill job/task/listener snapshots.
  - Add tests close to these packages, including `internal/services/processinstance/business_id_assignment_test.go`, v810 wire/response tests, old-adapter rejection and `c8volt/process/business_id_assignment_test.go`: exact body/key, duplicate/child behavior, empty/incomplete/forged scope, pointer isolation, worker bounds/order/fail-fast, HTTP success without JSON, conflict, malformed/lost response, confirmation failure and no-wait. Preserve original error class with partial facade results.
  - Run targeted `env GOFLAGS= go test ./internal/services/processinstance/... -run 'BusinessIDAssignment' -count=1` and corresponding `./c8volt/process` tests, plus affected aggregate/double compilation checks. Verify regexes select tests, gofmt touched files, and document new public API/source compatibility in `README.md`. This unit exposes library behavior only; CLI docs change with T002.

- [ ] T002 [US1] Deliver `update pi --set-bid` with its complete output/error/prompt contract in `cmd/update_processinstance.go`, new `cmd/update_processinstance_assignment.go`, new `cmd/cmd_views_processinstance_assignment.go`, and isolated command/terminal tests, then update `README.md` and generated CLI docs.
  - Depends on T001. Dispatch assignment before mandatory variable parsing. Support scalar/repeated keys and full newline stdin; deduplicate by process key through the service, validate all input before mutation, reject invalid trailing input, and freeze targets/value before one confirmation. Reject `--bid`, `--vars`, `--vars-file`, and existing disallowed search-control combinations. Preserve explicit authorization, worker/fail-fast controls and ordinary variable-update behavior, including a variable named businessId.
  - A deliberate empty stdin scope completes without prompt, mutation, confirmation reads or accepted/pending claims; no-key invocation without stdin stays invalid. Keep dry-run non-mutating, previews non-guaranteeing, normal accepted/confirmed and no-wait accepted/skipped evidence distinct. Rendering never performs discovery. Use mode-aware empty helpers with established empty/null shape and keys meaning.
  - Implement the mandatory output matrix below in `cmd/update_processinstance_assignment_test.go`, `cmd/update_processinstance_assignment_logging_test.go`, and existing/new terminal tests. Cover success, dry-run, rejection, unknown response, partial failure, completed empty, planning failure, auto-confirm/automation/no-wait and abort. Trace service → facade → one view → configured logger → original exit classification. Do not use the cancellation-specific process error formatter.
  - Update help, examples, metadata/capabilities and README for exact-key assignment, selection versus assignment, root/active/no-BID/uniqueness-disabled eligibility, identical conflicts, validation, snapshot independence, no replay, no-wait and dry-run races. Run targeted `env GOFLAGS= go test ./cmd -run 'Test.*(BusinessIDAssignment|UpdateProcessInstance)' -count=1`, gofmt, `env GOFLAGS= make docs-content`, and `git diff --check`.

## Phase 4: User Story 2 — Assign the Reference While Completing Work (P1)

**Goal**: Extend existing scalar and bulk completion to assign atomically with copied approval payloads and one-shot evidence.

**Independent test**: Public/service paths and owned HTTP captures prove one combined request, nil omission, stable preparation, independent job targets, no fallback/replay and correct unknown results; command execution proves input shapes, selection and output.

- [ ] T003 [US2] Deliver atomic BID-bearing scalar and bulk completion through `c8volt/job/model.go`, `c8volt/job/client.go`, `c8volt/job/convert.go`, `internal/domain/job.go`, `internal/domain/job_completion.go`, `internal/services/job/completion/prepare.go`, `internal/services/job/completion/execution_plan.go`, `internal/services/job/completion/execute.go`, `internal/services/job/completion/reconcile.go`, and all versioned completion ports, with isolated tests and public API documentation.
  - Depends on the shared validator delivered by T001; execute after T002 in the default sequence to avoid shared-contract edits. Add optional `SetBusinessID *string` and requested/assignment evidence without reusing observed BusinessId. Preserve these model constraints verbatim: “Nil means absent in existing job operations. Direct assignment requires a value.” “Copy pointers at request conversion and plan preparation boundaries. Do not share mutable caller storage.” “Distinct job keys remain distinct even if ProcessInstanceKey is identical. Scalar/collection input provenance persists before deduplication.” “No assignment field is emitted when intent is absent.”
  - Freeze optional BID and nested variables in public/domain conversions, scalar approved plans, bulk preparation and execution snapshots with `toolx.CopyPtr`/`toolx.CopyJSONMap`. Prevent a changed request/plan attachment from replacing the approved key/mode/payload. Extend the private Submit port with a typed frozen payload and update v88/v89/v810 adapters, interfaces and all doubles together. Apply assignment validation and unsupported-adapter guards before discovery/submission and on caller-prepared execution; reject non-completion worker modes at direct public/service boundaries.
  - In `internal/services/job/v810/service.go` populate generated BusinessId with variables and retain other supported fields. Reuse `internal/services/job/v810/completion_request.go` omission behavior. Preserve the constraint: “Without assignment, omit businessId entirely, rather than null or empty.” Keep listener-variable restrictions and supported lease-token serialization unchanged; add no new token feature.
  - Bypass RetryCamundaMutation only for BID-bearing scalar completion and use one-shot evidence classification; retain existing plain completion behavior. Bulk stays one-shot. Atomic rejection never splits into assignment+completion, retries without BID, or falls back to plain completion. Each distinct selected job carries the BID even when another job in the same process succeeds first.
  - Make executeCandidate/reconcile assignment-aware: preexisting COMPLETED cannot satisfy assignment; unknown/rejected submission followed only by COMPLETED preserves independent observed completion, unknown/rejected assignment, unsatisfied result and original error classification. Accepted acknowledgement is acceptance, not refreshed snapshots or independent confirmation. Preserve separate requested/observed/evidence fields and ordered partial results through facade mappings.
  - Extend `internal/services/job/v810/completion_payload_test.go`, `internal/services/job/v810/completion_test.go`, shared prepare/execute/reconcile tests and `c8volt/job/completion_test.go` to prove exact atomic wire/omission, all public guards, Unicode values, pointer/nested-map isolation after preparation, duplicates versus two jobs/one process, already-completed jobs, child/uniqueness/identical conflicts, lost scalar/bulk response then COMPLETED with exactly one submission, and original failure evidence. Keep non-BID retry/reconciliation regressions intact.
  - Run targeted tests from `quickstart.md` for `./internal/services/job/...`, `./c8volt/job` and affected domain models, verify real selections, gofmt, and document optional field/public source compatibility in `README.md`. Complete safety/evidence with submission in this unit; do not defer it to CLI integration.

- [ ] T004 [US2] Deliver `update job --complete --set-bid` across scalar, explicit collection, stdin and selector command execution in `cmd/update_job.go`, `cmd/update_job_request.go`, `cmd/update_job_outcome.go`, `cmd/update_job_batch.go`, `cmd/update_job_plan.go`, `cmd/cmd_views_job.go`, and `cmd/cmd_views_job_logging.go`, with acceptance/output/terminal tests and regenerated docs.
  - Depends on T003. Preserve scalar/bulk form before deduplication, including a one-item collection. Carry copied BID through parsing/plans/previews/approved payload/submission. Reject assignment with retry-update, timeout-update, failure or BPMN error. Assignment alone is not a selector. Preserve exact keyless `--retries 0`, keyed retry rejection, tenant-scoped AND discovery, limit/batch rules, full-input validation, sparse continuation, frozen targets, job-key deduplication and confirmation.
  - Preserve listener restrictions, prior observed job/task/listener BID facts, and both submission routes. Fix scalar BID-bearing error handling to retain returned partial/unknown evidence instead of discarding it. Render exactly one result envelope and use exit-after-render with the original error; never invoke a second error renderer. Preserve bulk keys-only rejection and existing JSON+verbose guardrails rather than adding modes.
  - Prove completed empty stdin/search succeeds with no prompt/mutation/pending claim in normal/dry-run/no-wait and supported machine/quiet/automation combinations. Failed or incomplete discovery after earlier candidates submits nothing. Dry-run never mutates; rendering never adds requests or replaces frozen membership. Preserve existing nonempty, explicit-key, abort/EOF, already-completed guards and non-assignment outcomes.
  - Extend `cmd/update_job_request_test.go`, `cmd/update_job_outcome_guard_test.go`, `cmd/update_job_batch_acceptance_test.go`, `cmd/update_job_batch_logging_test.go`, `cmd/update_job_batch_terminal_test.go` and scalar equivalents to execute the full mandatory output matrix below, including lost-response → COMPLETED and two jobs/one process. Assert exact request counts, one envelope+EOF, original exit codes, requested-versus-observed evidence and scalar-versus-bulk shapes.
  - Update source help/examples/metadata and `README.md` for atomicity, first-assignment alternatives, root eligibility/uniqueness disabled, identical conflicts, validation, no replay, no-wait and stale snapshots. Run targeted `env GOFLAGS= go test ./cmd -run 'Test.*(UpdateJob|BulkCompletion)' -count=1`, gofmt, `env GOFLAGS= make docs-content`, and `git diff --check`.
  - After both workflow deliveries and targeted checks pass, finish shared-runtime verification and record actual commands/results in `specs/362-late-business-id-assignment/quickstart.md`. Verify Makefile test target and inherited settings cannot enable live tests; run one `env GOFLAGS= make test` because shared completion coordination, adapter interfaces, payload copying and concurrent execution changed. Do not run integration targets/tags or live scripts. Keep real-engine eligibility/timing explicitly unverified, never label it passed. Record actual commands/test counts/results and material gaps in feature evidence; run link/whitespace checks without repeating runtime tests for later prose-only edits.

## Phase 5: User Story 3 — Trust Evidence and Automation Results (P1)

**Goal**: Keep intent, acceptance, observation, rejection and uncertainty distinguishable in both workflows.

**Independent test**: Execute each workflow against lost-response, rejection, partial/empty/incomplete and success fakes with separately captured result/prompt/log destinations. Require one JSON envelope plus EOF and original exit classification.

**Implementation ownership**: T001/T003 implement evidence and no-replay; T002/T004 implement configured output, prompts, docs and all mode coverage. This phase has no duplicate checkbox: these requirements are mandatory within their first useful consumer, not a later polish story. It completes when all four mapped units pass the following matrix.

### Mandatory output matrix for T002 and T004

- Test actual command execution with owned dependencies, configured and inherited logger destinations, plain and structured handlers, and stdout/stderr captured separately. Human success summaries, dry-run summaries and verbose outcomes require empty stdout and INFO records in the configured destination, suppressed at WARN and quiet. Prompts stay plain configured stderr; selected machine output stays stdout.
- Repeated backend failures must contain a distinctive URL, method, body and resource key. Assert one bounded grouped ERROR at normal/verbose/quiet and absence of raw HTTP diagnostics on both streams. Enable redacted DEBUG independently via flag and configured level without verbose. Keep diagnostic stdout empty and test secret redaction. Assertions must fail if logger routing is replaced with a direct print or raw joined errors.
- Decode exactly one shared JSON envelope and EOF; preserve partial evidence, established empty/null shapes, quiet+machine precedence and original failure exit classification. Preserve supported keys meanings and zero bytes for empty results; bulk job keys-only remains rejected.
- Use real terminal stdin to test configured/inherited prompt destination, redirected stdout, confirmation/decline/EOF and prompt-free empty completion. Pipe-only input or mocked terminal checks do not establish routing. Exercise supported dry-run/no-wait/auto-confirm/automation combinations plus nonempty and error regressions.

## Phase 6: User Story 4 — Validate Assignment Consistently (P1)

**Goal**: Enforce identical exact-value/version rules through commands and direct public/prepared paths without changing omitted-input behavior.

**Independent test**: Invalid/unsupported calls produce zero requests; accepted Unicode boundary values remain byte-exact; plain creation, search, variable updates and completion retain their contracts.

**Implementation ownership**: T001 owns shared value rules; T001/T003 wire and test every public/service/prepared entry point; T002/T004 own command validation and non-assignment regressions. No separate implementation checkbox is added because delayed validation would leave earlier assignment delivery unsafe.

Preserve evidence status meanings verbatim from the model/contracts: “not_submitted”, “accepted”, “confirmed”, “rejected”, “unknown”. “For accepted but unconfirmed direct assignment, status remains accepted and separate confirmation evidence records failure/skipped.” “A pre-submit already-completed job has assignment not_submitted/unsatisfied, even where ordinary completion would consider it satisfied.” Test all relevant transitions in the owning units, including absent assignment fields and no snapshot backfill. Native BID stays separate from variables; other mutation types gain no BID assignment.

## Phase 7: Polish & Cross-Cutting Verification

Final cross-package verification is included in T004 after its targeted checks. No standalone verification or documentation checkbox remains.

## Dependencies & Execution Order

```text
T001 → T002 (direct-assignment MVP)
T001 → T003 → T004 (atomic completion; reuse assignment validator)
Default execution: T001 → T002 → T003 → T004.
US3 and US4 complete through T001–T004 acceptance obligations.
```

Default order is numerical. US2 reuses validation from T001, not a second planner or generic framework. All API/interface/adapter/double changes remain inside their owning task. US3/US4 have overlapping acceptance ownership rather than postponed implementation or duplicate test-only tasks.

## Parallel Examples

- **US1**: No parallel implementation checkbox; T001 owns mutually dependent contracts/execution/tests, then T002 consumes it. Independent owned-fake test cases can run concurrently inside their existing test runner.
- **US2**: No parallel implementation checkbox; T003 changes shared ports with all implementations, then T004 consumes them. Scalar/bulk payload cases may run independently only after shared changes compile.
- **US3**: Logger handler and terminal fixtures are isolated cases inside T002/T004, not extra implementation handoffs.
- **US4**: Value-boundary cases and old-version rejection cases can be independently exercised within their owner task; do not split validator, guards and consumers for hypothetical staffing.

## Coverage Map

| Requirement / acceptance scope | Owning units |
| --- | --- |
| FR-001–FR-002, US4 scenarios 1–4 | T001–T004 |
| FR-003–FR-007, US1 scenarios 1–6 | T001–T002 |
| FR-008–FR-014, US2 scenarios 1–9 | T003–T004 |
| FR-015–FR-016, US3 scenarios 1–2 | T001–T004 |
| FR-017–FR-021, US3 scenarios 3–6 | T001–T004, full command matrix T002/T004 |
| FR-022, US3 scenario 7 | T002/T004 |
| FR-023, US4 scenario 5 | T001–T004 omitted-input and variable-separation regressions |
| SC-001–SC-002 | T001/T002, T003/T004 for job distinctness |
| SC-003 | T003/T004 |
| SC-004–SC-005 | T001–T004 |
| SC-006–SC-008 | T002/T004 |

## Implementation Strategy

MVP is T001–T002: complete direct assignment with public validation, observation, output/terminal checks and docs. Next deliver T003–T004 atomic completion with all safety/evidence requirements. T004 includes the final shared-runtime check. No live deployment, target selection or engine validation is required.

Every unit follows repository/Ralph ownership rules and has one primary story. US3/US4 are cross-cutting acceptance responsibilities within US1/US2 units; they do not authorize starting another independent workflow in a Ralph iteration. Keep interface additions with implementations and doubles, and backend safety with submission; never claim a task complete before its tests, docs and output/error path pass. Conventional issue commits, if the implementation workflow reaches its commit step, end their subject with ` #362`.

## Task Counts and Consolidation

Four unchecked implementation units: US1 has two primary tasks (T001/T002); US2 has two (T003/T004). US3 and US4 each have acceptance coverage in all four units and zero additional checkboxes; their phase sections and coverage map identify owners. This implements the plan's requirement to deliver evidence/output and validation within each behavior unit instead of rebuilding or deferring them. Setup/foundational/final phases add no routine-work tasks.

The six-task draft was consolidated to four: draft T001+T002 → T001 (validator with first useful consumer); draft T003 → T002; draft T004 → T003; draft T005+T006 → T004 (final broad validation with last behavior consumer). See [consolidation-review.md](consolidation-review.md) for evidence and retained boundaries. All four units include tests and required documentation; none is marked parallelizable.
