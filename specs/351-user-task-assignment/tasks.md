# Tasks: User-Task Assignment and Unassignment

**Input**: Design artifacts in `specs/351-user-task-assignment/`.
**Branch / issue**: `351-user-task-assignment` / #351.
**Prerequisites**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli-user-task-assignment.md), [facade contract](contracts/facade-user-task-assignment.md), [quickstart.md](quickstart.md).
**Tests**: Required by the specification. Each task includes its fixtures, meaningful tests, formatting and relevant checks; add failing contract/regression cases before implementing the corresponding behavior within the same work unit. Do not create separate test-only, documentation-only or routine-review tasks.
**Organization**: Two foundation tasks and two US1-owned delivery tasks implement the shared assignment family once. US2–US6 retain individual acceptance phases with explicit ownership references. Their common capabilities are mandatory parts of the listed work units, not deferred story tasks. A Ralph iteration completes its current work unit only; it does not start a separate later story or polish section.

## Format and execution rules

Task rows use `- [ ] Tnnn [P?] [USn?] Description with file path`. Paths are repository-relative. `[P]` requires a genuinely independent file set and completed prerequisites; no tasks here qualify. All implementation tasks remain unchecked until their actual implementation and required checks pass.

Every work unit reads `AGENTS.md`, the feature artifacts and, for Ralph, `specs/ralph-implementation-rules.md`; inspects nearby code; keeps all consumers compiling; adds necessary comments; formats touched Go files; and records actual evidence in `specs/351-user-task-assignment/progress.md` when required by the implementation workflow. These are responsibilities, not extra tasks. Commit only when the implementation workflow reaches that step; an issue-associated subject must end with ` #351`.

All commands prefixed `isolated_check` below use the clean-environment helper in [quickstart.md](quickstart.md). Inspect selected tests/targets and inherited configuration before execution. Use test-owned HTTP servers, mocks, temporary configurations and isolated subprocess/terminal fixtures only. No live tags, saved-profile executions, Camunda/cluster discovery, provisioning, deployment, integration/smoke targets, live mutations or cleanup. Missing real-engine coverage is a reported limitation, never a blocked implementation gate. No application test is required merely to generate or review this document.

## Phase 1: Setup (Shared Infrastructure)

No initialization checkbox: the module, generated clients, version factories, pool, retry policy, configured logging, HTTP diagnostics, subprocess/terminal helpers and docs generator already exist. Reuse them; do not add dependencies, scaffolding or a generic mutation framework. Setup/context reading is included in the first task that needs it.

## Phase 2: Foundational (Blocking Prerequisites)

**Goal**: Provide protected native assignment/unassignment, truthful response evidence and one collection service with a thin public facade before exposing new command modes.
**Independent test**: Isolated adapter/service/facade tests verify both actions without command registration or real Camunda.

- [x] T001 Implement native assignment/unassignment and shared response evidence in `internal/domain/usertask_assignment.go`, `internal/services/calloption.go`, `internal/services/common/usertask_submission.go`, `internal/services/common/usertask_assignment.go`, `internal/services/common/usertask_completion.go`, `internal/services/usertask/api.go`, and `internal/services/usertask/v88/assignment.go`, `internal/services/usertask/v89/assignment.go`, `internal/services/usertask/v810/assignment.go`; update all three `contract.go` files and affected test doubles, and include native/common tests in the same compiling unit.

  Implement the per-key `AssignUserTask` and `UnassignUserTask` signatures from the facade contract, assignment-specific result/status/submission types, and the internal `WithUserTaskAssignmentOverride()` option with default false. Compare v810 first, then v89/v88. Extend each `GenUserTaskClientCamunda` with existing generated assignment/unassignment methods and implement every affected API consumer/double together. Relevant doubles include each version's `service_test.go` and `completion_test.go`, `internal/services/usertask/{completion,workflow,variables,bulk,update,update_wait,search}_test.go`, and `c8volt/task/{client,variables,completion,search}_test.go`; discover any additional implementations instead of leaving compilation fixes for later tasks. Unrelated read/completion fixtures must fail on unexpected assignment calls.

  Use POST `/user-tasks/{key}/assignment` with a nonnil assignee pointer and nonnil `allowOverride` pointer set false unless explicitly enabled; leave optional custom action unset. Use bodyless DELETE `/user-tasks/{key}/assignee` for unassignment. Preserve backend-authorized explicit-key tenant behavior and existing factory/default-version handling. Do not read tasks, resolve Tasklist IDs, assign before completion, unassign before assignment, poll, compensate, or edit generated clients. Same-assignee and already-unassigned native errors remain errors.

  Preserve these exact domain field constraints from `data-model.md`: Key is “Selected unique task key, always present”; Status is “Local operation outcome”; Submission is “Evidence about backend acceptance”; StatusCode is “Last observed submission response code; zero when none”; Error is “Existing machine-readable error evidence; not human display text”; Items is “One item per unique validated key; initialized empty for validation failures”. Status values are `preview`, `accepted`, `failed`, `timed_out`, `canceled`, `not_attempted`; submission values are `not_submitted`, `accepted`, `rejected`, `unknown`. Do not reuse completion-named public/domain result types for assignment.

  Extract only private native user-task response/evidence handling, retaining `CompleteUserTaskSubmission` as a compatibility wrapper with identical existing completion types, messages and behavior. Require documented 204 with valid HTTP metadata and no overriding call error. Reuse `RetryCamundaMutation` and `httpc.HttpStatusErr`; pass original transport errors to retry handling, keep last response and sticky ambiguity outside retry return values, and do not add generic 504/409 retries. New 504/transport timeout makes submission unknown even after an earlier rejection; deadline/cancellation during backoff after a definite rejection may retain rejected only when no attempt was ambiguous. Later rejection cannot erase ambiguity; later valid 204 can establish accepted. Pre-start cancellation remains not_attempted/not_submitted. Preserve existing timeout/error classes and configured detail-suppression/DEBUG routing.

  Add `assignment_test.go` beside each adapter and `internal/services/common/usertask_assignment_test.go`; retain `internal/services/common/usertask_completion_test.go` regressions. Test real generated request serialization with test-owned HTTP servers, false/true override, opaque identity fidelity, exact routes/body shape, delayed 204, forbidden extra requests, backend same-assignee/already-unassigned success/rejection, 400/401/403/404/409, malformed/nil/undocumented success, partial 204 plus call error, generic 500/503/504 uncertainty, 429/resource-exhausted 503/temporary-network retry evidence, exhaustion, cancellation and backoff. Cover response-state transitions once in the common helper and mapping in each adapter without a redundant cross-product. Keep human error logging out of these per-key boundaries.

  Format touched Go files; run `isolated_check go test ./internal/services/usertask/... -run 'Test(Assign|Unassign|Complete)UserTask' -count=1` and `isolated_check go test ./internal/services/common -run 'Test(Assign|Unassign|Complete)UserTask' -count=1`, then affected facade tests whose doubles changed. Assert test names match and all consumers compile; “no tests to run” is not a pass. This internal increment does not change CLI documentation.

- [x] T002 Deliver one collection assignment/unassignment coordinator and its thin facade in `internal/services/usertask/assignment.go`, `internal/services/usertask/assignment_test.go`, `c8volt/task/assignment.go`, `c8volt/task/assignment_test.go`, `c8volt/task/api.go`, `c8volt/task/model.go`, `c8volt/task/convert.go`, and `c8volt/foptions/options.go` with mapping tests in `c8volt/foptions/options_test.go`; update public-interface doubles including `cmd/process_api_stub_test.go` in the same unit (depends on T001).

  Add the public `AssignUserTasks(ctx, keys, assignee, wantedWorkers, opts...)` and `UnassignUserTasks(ctx, keys, wantedWorkers, opts...)` signatures exactly as in `contracts/facade-user-task-assignment.md`, both returning `UserTaskAssignmentResults`. Add public `WithUserTaskAssignmentOverride()` and `UserTaskAssignmentOverride bool` plus explicit mapping to T001's internal option. Do not infer override from force/auto-confirm/automation. Reject no-wait for both actions and enabled assignment override for unassignment. Reuse existing dry-run, worker, fail-fast, explicit-key and logging-suppression controls. Public `wantedWorkers == 0` requests automatic selection and negative values are invalid.

  Build single-key and collection behavior together in one private assignment-family coordinator with explicit action wrappers. Validate every key, identity, option and service dependency before any mutation. Preserve exact model constraints: keys are “Nonempty; exactly 16 ASCII digits each; unique first-input order”; assignee is “Nonblank for assign; absent for unassign; preserve nonblank supplied bytes”; override is “False by default; true only from explicit permission; false for unassign”. Domain/facade keys are exact values; command whitespace normalization remains command-owned. Snapshot the keys and evaluated options before scheduling so mutable input/option closures cannot change submitted intent across keys or retries.

  Reuse `toolx.DetermineNoOfWorkers` and `toolx/pool.ExecuteSlice`, including explicit-worker precedence over no-worker-limit. Preinitialize ordered not_attempted/not_submitted results, track started slots, use the caller context for in-flight submissions and separate scheduling-stop evidence for fail-fast, retain all observed successes/errors, and join real operation errors with applicable caller cancellation. No reads, completion preflight, discovery, variable operations or polling. Direct-call dry-run returns ordered preview/not_submitted items with zero requests. Preserve T001's uncertainty transitions and do not manufacture outcomes from zero pool slots.

  Facades only map/delegate/convert: map every partial result before `ferrors.FromDomain`, preserve constructor/root composition, and expose no internal/generated types. Public JSON constraints must remain verbatim: `key`, `status`, `submission` are “Required”; `statusCode` is “Omit when zero”; `error` is “Omit when empty; retain machine failure evidence”; `items` is “Initialized array, never null”. Existing completion types and schemas stay unchanged.

  Tests must cover both actions: empty and late-invalid keys, blank and opaque identities, option forwarding/default false/true, rejected no-wait/override, nil dependency, duplicates, snapshot ownership, preview/no-call, single and multiple keys, sequential/automatic/bounded workers, no-worker-limit precedence, out-of-order response completion, continue-on-error, fail-fast with accepted in-flight work and queued remainder, pre-start/mid-flight cancellation, caller error after success, uncertainty and ordered partial conversion. Use synchronization gates/counters rather than timing guesses. Preserve facade read/resolve/variable/completion regressions; new doubles reject forbidden reads.

  Format and run targeted `isolated_check go test ./c8volt/task -run 'Test(Assign|Unassign|Complete)UserTasks' -count=1`, `isolated_check go test ./c8volt/foptions -run 'TestUserTaskAssignmentOverride' -count=1`, and `isolated_check go test ./internal/services/usertask/... -run 'Test(Assign|Unassign)UserTasks' -race -count=1`; run affected task facade regressions. Add API comments in this unit; no CLI flags/help are exposed yet.

**Checkpoint**: Both native operations and public collection APIs compile and have isolated validation of protection, ordering, concurrency and evidence. The command remains unchanged.

## Phase 3: User Story 1 — Assign Explicitly Selected Tasks Safely (Priority: P1, MVP owner)

**Goal**: Deliver the shared assignment-family command once, with the approval, bulk output and compatibility guarantees required by the other stories.
**Independent test**: Actual command execution against mutation-only fakes proves selected keys/identity, explicit non-overriding policy, opt-in override, aliases, local preview and no unsafe mutation on invalid input.

- [x] T003 [US1] Build the shared local input-to-report contract in `cmd/update_usertask_input.go`, `cmd/update_usertask_assignment_input.go`, `cmd/update_usertask_assignment_input_test.go`, `cmd/cmd_views_usertask_assignment.go`, and `cmd/cmd_views_usertask_assignment_test.go`, narrowly adapting `cmd/update_usertask_complete_input.go` and its existing tests without registering new command flags (depends on T002).

  Extract strict key collection and pflag empty-occurrence tracking only, preserving completion-specific payload decoding and legacy variable input behavior. Test the new parser with isolated command-local flag registration. Resolve assignment by flag presence, including invalid empty value; unassignment/completion by boolean value. Reject multiple active modes, any variable-payload flag presence, enabled no-wait, and override flag presence outside assignment even false. Preserve existing JSON/verbose rejection and JSON execution confirmation guard. Assignment with explicit override false is valid. False selectors do not change existing variable mode.

  Quote and enforce all frozen-input model invariants: operation “Exactly one active new mode”; keys “Nonempty; exactly 16 ASCII digits each; unique first-input order”; assignee “Nonblank for assign; absent for unassign; preserve nonblank supplied bytes”; override “False by default; true only from explicit permission; false for unassign”; workers “CLI omitted = automatic; explicit CLI values positive”. Flag keys precede stdin lines, blank stdin lines are ignored, explicit empty flag occurrences/components are rejected, valid flag keys plus empty stdin remain valid, scanner failure is bounded local invalid-input text, and late invalid input aborts before any facade call. Freeze input before preview/approval, without task reads or identity resolution.

  Implement typed assignment preview/execution reports and safe human/failure formatters together against this frozen input. Preserve the report constraints: operation is “`assign` or `unassign`”; phase is “`preview` or `execution`”; requestedKeys is “Frozen unique selection”; requestedCount is “Number of requested unique keys”; assignee is “Assign only”; allowOverride is “Assign only, including false”; items is “Ordered public outcomes”; counts is “Counts for every known local status”. Required array/object/scalar fields remain required as defined in `data-model.md`; assign includes assignee and false/true override, unassign omits both rather than serializing null. “Count fields always present: `preview`, `accepted`, `failed`, `timedOut`, `canceled`, `notAttempted`.” “Their sum equals requestedCount for complete execution/preview reports.” Never include completion-variable fields or infer already-completed conflicts from failed/not_submitted.

  Preview items are preview/not_submitted with no response code/error; execution copies service facts without fabricating missing items. Implement the unchanged shared envelope: succeeded preview, accepted successful execution, classified error envelope with partial payload once, and no invented report for local validation errors. Keys-only prints ordered unique preview keys or accepted execution keys, zero bytes if none. Check machine modes before quiet. Human preview/results/verbose lines use `renderHumanLine` only with a configured command logger, never direct stdout/stderr; safe quoted identity text must not insert control characters. Use bounded known reason groups (maximum three plus remainder), preserve unknown submission evidence and raw machine error data, and never interpolate backend errors into human text.

  Parser/report tests cover both operations, invalid controls, immutable preview, field presence/omission, all enums/counts, one-envelope decoding plus EOF, partial evidence, keys selection/zero bytes, quiet precedence and safe human wording. Retain `TestUpdateUserTaskCompletionInputKeyTrackingPreservesLegacy` and completion/variable parser regressions. Tests at this boundary do not establish the actual command/logger/prompt contract; that gate remains in T004. Format and run `isolated_check go test ./cmd -run 'TestUpdateUserTask(Assignment|Unassignment|Completion)Input|Test.*UserTaskAssignment.*(Report|Render|View)' -count=1`, ensuring implemented names match; run affected legacy input and view tests. Do not expose partial flags or change user-facing docs before T004.

- [ ] T004 [US1] Expose both protected assignment modes through `cmd/update_usertask.go` and `cmd/update_usertask_assignment.go`, add actual-command coverage in `cmd/update_usertask_assignment_execution_test.go`, `cmd/update_usertask_assignment_logging_test.go`, `cmd/update_usertask_assignment_terminal_test.go`, and `cmd/update_usertask_assignment_output_test.go`, and update `cmd/update.go`, `README.md` and generated `docs/cli/` in the same delivery unit (depends on T003).

  Register assign/unassign/override flags, preserve aliases and existing mode metadata, resolve conflicts before completion/variable dispatch, and call the shared local parser/report from T003. Base command retains construction/validation/dispatch only; runner/state/approval behavior stays in the dedicated mode file and final formatting in the view file. Do not put scheduling, retry, polling or generated client calls in command/facade code.

  Freeze intent, attach known explicit-key tenant context without discovery, preview locally with zero facade/backend requests, or approve once then call the corresponding T002 public method. Reuse `attachExplicitKeysTenantContext`, `collectExplicitAdminInputOptions`, `shouldImplicitlyConfirm`, `confirmCmdOrAbortFn(cmd.ErrOrStderr(), ...)` and existing configured logger setup. Assignment previews/approval identify action, selected keys/scope, quoted opaque assignee and whether replacement is allowed; unassignment names its action and scope. Preserve real-terminal eligibility with redirected stdout, default-no/yes parsing, decline/EOF, nonterminal behavior, auto-confirm/automation and mode precedence. No completion preparation, reads, warnings, state skipping, extra mutations or additional polling; native responses remain authoritative.

  Trace the entire failure path from versioned submission through service evidence, facade conversion, rendering, configured logging and exit. Render exactly one JSON error envelope with partial report, then use `exitAfterRenderedResult` without a second error renderer. Non-JSON execution preserves selected results, emits one bounded aggregate ERROR and exits with the original class. If rendering fails, use one compact configured diagnostic and retain original operation-error exit precedence; do not render again or expose raw error chains. Accepted means documented successful response including listener processing; never merely dispatch, independent read-back or permanent state. Respect configured no-error-code behavior.

  Build a shared assign/unassign command fixture whose backend rejects every read, completion, variable write and unrelated route; do not reuse the completion server that expects preflight reads. Exercise all aliases, repeated/comma/stdin/combined keys and empty occurrences, blank identities, invalid late inputs/workers, active/false/present flags, JSON guardrails, same-assignee/already-unassigned success/rejection, explicit-key tenant behavior, delayed success, fail-fast/continue-on-error, cancellation, partial and all-failed reports. Require exact request counts and zero mutations for validation/abort/preview. Cover human/JSON/keys-only/quiet, quiet+machine, auto-confirm/automation and both actions; no-wait true is rejected and false is inactive. Require JSON EOF after one envelope, retained machine error evidence and original exits, exact successful-key order and zero-byte keys-only output when none qualify.

  Through actual command execution, assert empty human stdout; INFO records for success, dry-run and verbose outcomes at configured/inherited destinations with active plain/structured formats and activity routing; suppression at WARN and under quiet; plain prompt text independently of log records. Repeated multi-resource failures must contain distinctive URL/method/body/key evidence in the fake backend while normal/verbose-only output on both streams remains compact, grouped and bounded with exactly one ERROR. Quiet keeps ERROR. Exercise `--debug` and configured DEBUG independently of verbose, existing flag precedence and redaction without contaminating stdout. Assertions must fail if logger emission is replaced with a direct stderr print or raw aggregates are exposed; substring-only and renderer-only checks are insufficient.

  Use `testx.NewCmdTerminalRunner` and actual terminal stdin with separate stdout/stderr/logger captures to prove single approval, configured/inherited stderr, safe identity/override wording, redirected stdout eligibility, approve/default-no/decline/EOF, and prompt-free invalid/dry-run/auto-confirm/automation paths. Preserve completion's advisory preflight, warnings, result wording/schema, optional variables and no-wait rejection; preserve variable planning/scope/verification/no-ops/no-wait, user-task reads/resolution and `get pi --has-user-tasks`. Record any genuinely skipped terminal coverage and real-engine listener limitation accurately.

  Update command and parent-group help/capability metadata, README guidance and examples for the four separate operations, explicit override, local preview, workers, partial/uncertain outcomes and native-response waiting. Format touched Go files and regenerate CLI documentation with `isolated_check make docs-content`, never by hand. Start with `isolated_check go test ./cmd -run 'TestUpdateUserTask(Assignment|Unassignment)' -count=1`, then affected `TestUpdateUserTaskCompletion`, variable/read/resolution and command-contract regressions per `quickstart.md`. Once targeted checks pass, run `isolated_check make test` once for the shared options, evidence extraction, interfaces and concurrency changes, after verifying live-test guards/configuration. Run `git diff --check`; review every changed direct print and error interpolation at output boundaries. Do not repeat runtime suites solely for later prose edits. Record actual coverage in the implementation progress record; all required output and terminal checks belong to this task before marking the CLI complete.

**Checkpoint**: The command exposes both modes with tested approval, diagnostics, output, docs and regression behavior; it is the first releasable CLI increment. US1's isolated assignment scenarios can be assessed independently while the shared implementation already covers US2–US6.

## Phase 4: User Story 2 — Remove Explicit Task Assignments (Priority: P1)

**Goal**: Unassign one or multiple explicit tasks while preserving native success/rejection and avoiding assignment or completion side effects.
**Independent test**: Invoke unassignment through each alias/key form using mutation-only fakes, including already-unassigned, missing, denied, mixed-success and incompatible flags; assert one logical removal per unique key, no reads/other mutations, and ordered outcomes.
**Task ownership**: T001 owns native removal; T002 owns the shared collection/facade; T003 owns unassignment intent/field omission; T004 owns dispatch, actual command/terminal/output tests and docs. No extra checkbox or second implementation of the shared workflow.

## Phase 5: User Story 3 — Preview and Approve the Exact Change (Priority: P2)

**Goal**: Preview frozen intent locally and approve the action once without changing replacement permission.
**Independent test**: Dry-run performs zero backend requests; real-terminal approve/decline/EOF and redirect scenarios prove configured plain stderr, action/key/identity/override wording, bypass semantics and zero mutation when approval fails.
**Task ownership**: T002 covers direct facade preview; T003 covers immutable local input/report; T004 covers actual prompts, configured logging, no-call preview, automation and terminal behavior. No separate approval feature or post-delivery safety task.

## Phase 6: User Story 4 — Understand Bulk Progress and Uncertain Outcomes (Priority: P2)

**Goal**: Account for all selected keys, bound work and retain truthful evidence after failures, timeouts and cancellation.
**Independent test**: Isolated synchronized responses prove worker bounds, ordered out-of-order results, fail-fast in-flight preservation, pre/post-start cancellation, ambiguous-then-rejected and later-success evidence without rollback claims.
**Task ownership**: T001 owns native evidence and retries; T002 owns collection scheduling/partial facade conversion; T003 owns typed report/counts; T004 owns command partial results/exits. No separate single-key workflow or later bulk rewrite.

## Phase 7: User Story 5 — Use Reliable Automation and Diagnostics (Priority: P2)

**Goal**: Produce script-safe machine results and compact configured diagnostics for both new modes.
**Independent test**: Actual command success/failure under both logger formats and inherited/configured destinations proves INFO/ERROR routing/suppression, one JSON envelope plus EOF, keys-only ordering/zero bytes, raw-detail exclusion, both DEBUG entry points/redaction and original exits. Direct-print/raw-error regressions must fail assertions.
**Task ownership**: T001 retains common HTTP evidence/instrumentation, T002 preserves classified errors, T003 supplies safe typed rendering, and T004 owns real command emissions and all diagnostic/exit tests. No diagnostic infrastructure or late output-compliance task.

## Phase 8: User Story 6 — Preserve Existing Task Workflows (Priority: P2)

**Goal**: Keep completion, variables, reads/resolution and supported-version contracts unchanged while documenting additive modes.
**Independent test**: Existing completion/variable/read regressions run against isolated dependencies; source/generated help shows accurate new flags without changing old semantics; unsupported versions retain generic rejection.
**Task ownership**: T001 preserves completion evidence and version factories, T002 preserves existing facade behavior, T003 preserves completion/legacy input behavior, T004 owns existing command regression coverage and source/generated docs. Compatibility is part of each changing unit, not a delayed repair story.

## Phase 9: Polish & Cross-Cutting Concerns

No separate routine review, formatting, commit, test-run or docs task. Each belongs to its implementation owner; T004 includes final source metadata/docs, proportional broad validation and actual evidence recording. Generating this list does not launch implementation, Ralph, a commit, publishing or live validation.

## Dependencies & Execution Order

```text
T001 native operations + evidence + internal interfaces/doubles
  -> T002 shared collection execution + thin facade/options + public doubles
     -> T003 frozen input + local preview/result contract + parser regressions
        -> T004 complete CLI execution + output/terminal tests + docs + final checks
```

| Task | Prerequisite | Independently verifiable result |
| --- | --- | --- |
| T001 | Existing repository | Three native adapters and common evidence, including unchanged completion behavior |
| T002 | T001 | Usable public collection APIs with validated ordering, worker bounds and partial evidence |
| T003 | T002 | Locally testable frozen request-to-report contract; no partially exposed command mode |
| T004 | T003 | Complete user-facing assignment family with all command, output, terminal and documentation gates |

US1 owns the command delivery. US2–US6 acceptance obligations are included in T001–T004 at the layer where each behavior first appears. These phases do not authorize crossing into an unrelated story in a Ralph iteration; finish the one selected work unit and its explicit shared acceptance obligations.

## Parallel Opportunities and Per-Story Examples

There are no `[P]` tasks: protocol/result interfaces, options, key parsing, command metadata and fixtures overlap or depend on preceding work. Do not create artificial tasks or spawn implementation agents to manufacture parallelism. Independent test cases may share table-driven fixtures within a unit without weakening boundary assertions.

| Story | Safe execution example | Parallel-task decision |
| --- | --- | --- |
| US1 | T001 → T002 → T003 → T004, then assess explicit assignment cases | Sequential; command relies on all previous contracts |
| US2 | Assess unassignment cases within the same owner tasks | No separate parallel removal runner; shared APIs and fixtures |
| US3 | Local preview checks in T003, real-terminal approval checks in T004 | No parallel implementation; prompt depends on frozen report |
| US4 | Evidence cases in T001, synchronized scheduling cases in T002 | No parallel implementation; scheduler consumes native evidence |
| US5 | Typed views in T003, command/logger/exit matrix in T004 | No parallel implementation; configured routing needs the runner |
| US6 | Run the affected existing regressions in each owner unit | No independent compatibility rewrite; validate the changing owner |

## Requirement and Acceptance Coverage

| Requirements | Owning work | Success criteria / stories |
| --- | --- | --- |
| FR-001, FR-002, FR-003, FR-004 | T002 validation; T003 input; T004 wiring/execution | SC-001, SC-003; US1, US2, US6 |
| FR-005, FR-006, FR-007, FR-008, FR-009 | T001 native mappings/options; T002 intent; T003/T004 approval and policy | SC-001, SC-002; US1, US2 |
| FR-010, FR-011, FR-012, FR-013 | T002 preview; T003 frozen local report; T004 real approval | SC-002, SC-003; US3 |
| FR-014, FR-015 | T001 retry; T002 scheduling; T004 command outcomes | SC-007; US4 |
| FR-016, FR-017, FR-018, FR-019 | T001 evidence; T002 partial results; T003/T004 reporting/exits | SC-004, SC-006; US4 |
| FR-020, FR-021 | T003 schema/keys; T004 full execution and exits | SC-005; US5 |
| FR-022, FR-023, FR-024 | T001 shared diagnostics; T003 safe views; T004 logger/DEBUG/terminal matrix | SC-005, SC-008; US5 |
| FR-025, FR-026, FR-027 | T001 adapters/compatibility; T002/T003 regressions; T004 docs/final coverage | SC-009; US6 |

Model field constraints are quoted in their owning task, including status/submission transitions, opaque identity, default-false override, null/empty/omitted representations, required fields, enum values and count invariants. Contract and spec obligations remain authoritative if a task's wording is abbreviated.

## Implementation Strategy

Complete the native and collection layers first, then the local contract, then expose both command modes with their full execution/output safety. Assignment is the MVP acceptance journey; because unassignment shares the required underlying workflow, the first releasable CLI increment includes both after T004. Do not claim T001–T003 deliver a working new CLI or defer mandatory approval/diagnostics to a later release.

Keep fixtures shared where the same behavior is being proved, retain per-version wire mapping and actual command/terminal assertions where boundaries differ, and record only checks actually performed. Do not seek a live target. If implementation size justifies another split, preserve a compiling independently verifiable outcome and keep its tests/docs attached, rather than splitting interfaces from implementations or moving safety validation after CLI exposure.
