# Tasks: Complete User Tasks by Explicit Key

**Input**: Design artifacts in `specs/311-complete-user-tasks/`.
**Branch / issue**: `311-complete-user-tasks` / #311.
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [facade/service contract](contracts/facade-service.md), [quickstart.md](quickstart.md).
**Tests**: Required by the specification. Each implementation task includes its fixtures, meaningful tests, formatting and targeted validation. Add relevant failing regression/contract cases before implementing the behavior within that same work unit; do not create separate test-only tasks.
**Organization**: Two shared foundation tasks and two US1-owned delivery tasks implement the common completion workflow once. US2–US5 retain separate acceptance phases with explicit coverage references, not duplicate implementation tasks. Their coverage is integral to these four work units, not deferred or optional. A task's primary story label does not authorize a Ralph iteration to start unrelated story work.

## Format and execution rules

Task rows use `- [ ] Tnnn [P?] [USn?] Description with path`. Paths are repository-relative. `[P]` means a genuinely independent file set once stated prerequisites pass; it is not an instruction to launch agents. All tasks are initially unchecked.

Every work unit reads current guidance and feature artifacts, stays compiling, adds required comments, formats touched Go files, runs the smallest relevant checks, reviews its diff, and records evidence in `specs/311-complete-user-tasks/progress.md` when the implementation workflow requires progress tracking. These responsibilities are not separate tasks. Use Conventional Commits ending in ` #311` if the implementation workflow reaches its commit step.

All `isolated` commands below use the clean-environment wrapper from [quickstart.md](quickstart.md). Inspect selected tests/targets before execution; fixtures must explicitly use test-owned configurations and servers. No live tags, Camunda installations, smoke/integration targets, cluster inspection, provisioning, deployments, or live cleanup. A missing live environment is never a completion blocker. Do not run runtime checks merely to generate or consolidate this document.

## Phase 1: Setup (Shared Infrastructure)

No initialization task is needed: the Go module, generated clients, pool, HTTP diagnostics, logger, subprocess/terminal runners, and documentation generator already exist. Reuse them. Context reading, branch checks, fixture inventory and dependency inspection belong to the first work unit that needs them; do not add setup scaffolding or dependencies.

## Phase 2: Foundational (Blocking Prerequisites)

**Goal**: Supply native version-compatible completion and one reusable collection coordinator with its thin public facade before exposing the command.
**Independent test**: Isolated adapter, service and public-facade calls prove request mapping, acceptance evidence, deduplication and worker behavior without a CLI or live backend.

- [x] T001 Implement native completion and its domain evidence in `internal/domain/usertask_completion.go`, `internal/services/usertask/api.go`, `internal/services/usertask/v88/contract.go`, `internal/services/usertask/v89/contract.go`, `internal/services/usertask/v810/contract.go`, and new `completion.go` / `completion_test.go` files in each version package; update all affected service/generated-client test doubles in the same compiling change.

  Add scalar `CompleteUserTask` and generated `CompleteUserTaskWithResponse` contracts exactly as specified in `contracts/facade-service.md`. Use the existing generated request with nil action, nullable nil variables, and pointer-to-empty-map for `{}`; accept documented 204 without requiring a success payload. No reads, assignment, scope writes, legacy fallback, polling, or generated-code edits. Preserve explicit-key backend authorization and existing factory/default-version behavior.

  The result fields must retain these verbatim model constraints: Key is “Selected task key; always present”; Status is “Local execution result, defined below”; Submission is “Backend acceptance evidence, independently defined below”; StatusCode is “Last observed HTTP status, omitted when none; not proof of an earlier attempt's outcome”; Error is “Existing machine-readable failure evidence; never interpolate into human output”. Implement statuses `preview`, `accepted`, `failed`, `timed_out`, `canceled`, `not_attempted` and submission values `not_submitted`, `accepted`, `rejected`, `unknown` with the exact meanings and transition rules in `data-model.md`: “An item with status accepted must have submission accepted. Preview and not-attempted must have submission not_submitted.” “Timed-out/canceled items must not imply rejection solely from their execution status.” “A final rejection after an earlier transport ambiguity retains submission unknown.” “A later 204 establishes acceptance even if an earlier attempt was ambiguous.”

  Reuse `RetryCamundaMutation` for the existing finite 429, RESOURCE_EXHAUSTED 503, and temporary/timeout network-error policy. Retain last status and ambiguous-attempt evidence across backoff cancellation; handle pre-submission cancellation, nil/malformed responses, later rejection after ambiguity, and eventual 204 accurately. Do not introduce 404/409/500/504 retries or reconciliation. Keep retry progress bounded and verbose-only using the configured logger and a constant operation label; existing transport DEBUG remains independent.

  Keep version-specific mock/fake cases for generated wire mapping, omitted/empty/nested data, no-content success, payload ownership, logging-option forwarding and response/body/error delegation. Cover 400/401/403/404/409, exhausted throttling, uncertain 504/transport errors, cancellation before request/during retry delay, and malformed responses once beside the shared implementation in `internal/services/common/usertask_completion_test.go`. Mocks must provide request URL/method required by existing HTTP error conversion. Assert no forbidden calls or raw normal-level logging. Format changes and run `isolated go test ./internal/services/usertask/... -run 'Complete|Completion' -count=1`; run targeted tests in any additional package whose interface doubles changed. Do not mark complete with consumers failing compilation. This internal-only increment requires no CLI documentation change.

- [x] T002 Deliver the collection completion API through its service and public facade: implement and test the coordinator in `internal/services/usertask/completion.go` and `internal/services/usertask/completion_test.go`, using the native operation from T001 and existing `toolx/pool` scheduling, and expose it through `c8volt/task/{api.go,model.go,convert.go,completion.go,completion_test.go}` in the same unit.

  Apply these exact input constraints from `data-model.md`: “Nonempty selection; exactly 16 decimal digits per key under existing user-task conventions; stable first-occurrence uniqueness; flags precede stdin”; “Nil means omitted; nonnil empty map means supplied `{}`; nested valid JSON values retain their types; one object shared by every completion”; “Zero means automatic at the service boundary; positive values bound concurrency; negative invalid. CLI explicitly supplied zero is invalid”; “Existing fail-fast, no-worker-limit, dry-run, verbose, and explicit-key authorization options; no-wait rejected”. Flags/stdin merging remains CLI-owned; the service validates the supplied key sequence and stable uniqueness independently.

  Validate all keys, options and JSON serializability and snapshot nested payloads before submission; guard missing API dependency. Build single and multiple key handling together, including optional variables. Direct-call dry-run returns ordered preview/not_submitted items without calls. Use `DetermineNoOfWorkers`, retain explicit-worker precedence over no-worker-limit, and initialize one ordered result slot per unique key. Follow existing job scheduling patterns without copying its discovery/reconciliation lifecycle: pool context stops scheduling, original caller context governs started requests, and a started flag distinguishes unscheduled slots. Keep already accepted work when another key fails; join real callback errors and original caller cancellation rather than synthetic fail-fast cancellation.

  Preserve the collection constraints: “is initialized to an empty array when no item evidence exists”; “For a valid executed or dry-run selection, it contains exactly one item per unique key in selection order. Do not sort by completion time or drop unattempted slots. Return partial results alongside a nonnil error.”

  Tests use synchronization gates/counters for sequential ordering, worker bounds/defaults/precedence, out-of-order responses, continue-on-error, fail-fast with in-flight acceptance and queued remainder, pre-start and mid-flight cancellation, unknown submission, snapshot ownership, duplicates, invalid late keys/payloads, empty selection and no-wait. Assert every unique key is accounted for and forbidden reads/writes/polls never occur. Format and run `isolated go test ./internal/services/usertask/... -run 'Complete|Completion' -count=1`, with targeted race coverage for the new concurrent coordinator. Keep the facade mapping checks below in this same work unit; this increment does not change CLI docs.

  Expose the collection completion facade in `c8volt/task/api.go`, `c8volt/task/model.go`, `c8volt/task/convert.go`, new `c8volt/task/completion.go`, and `c8volt/task/completion_test.go`, updating affected public-interface stubs together.

  Add the exact public `CompleteUserTasks` signature in `contracts/facade-service.md`. Map options and delegate inputs to this coordinator, which owns validation and snapshotting, map partial results before applying `ferrors.FromDomain`, and preserve constructors/root composition. Public models must match all T001 status/submission invariants and use `key`, `status`, `submission`, `statusCode,omitempty`, `error,omitempty`, and initialized `items` JSON fields. Do not expose internal/generated types, schedule workers, add backend reads, reinterpret tenant filtering, or discard outcome evidence on failure.

  Add tests for worker/option forwarding, nil-versus-empty variables, nested snapshots, each enum and JSON omission rule, empty-array representation, original classified error and partial-result preservation. Retain existing reads, process-instance resolution, variable-update constructors/schemas and no-wait behavior; update only required interface doubles. Format and start with `isolated go test ./c8volt/task -run 'Complete|Completion' -count=1`, then run affected task facade regressions. No CLI-visible behavior is exposed yet; document the new public method with Go comments rather than changing CLI docs prematurely.

**Checkpoint**: All supported native contracts compile; the public collection API provides tested ordered outcomes, optional payload support and preserved partial errors for both single and collection calls.

## Phase 3: User Story 1 — Complete Explicitly Selected Tasks (Priority: P1)

**Goal**: Expose one completion workflow through the existing task facade and CLI, including mandatory safety, output and compatibility behavior shared with the other stories.
**Independent test**: Run the actual command with isolated responses for every alias and key form; verify one logical completion per unique key, acceptance-only results, and zero mutation for invalid input or dry-run.

- [x] T003 [US1] Build the local completion request-to-report contract in new `cmd/update_usertask_complete_input.go`, `cmd/update_usertask_complete_input_test.go`, `cmd/cmd_views_usertask_complete.go`, and `cmd/cmd_views_usertask_complete_test.go`, without exposing a partially implemented completion flag.

  Implement a focused request parser for the future runner, using configured stdin only for positional `-`. Validate flag and stdin keys with “Nonempty selection; exactly 16 decimal digits per key under existing user-task conventions; stable first-occurrence uniqueness; flags precede stdin”. Ignore blank stdin lines, reject empty flag tokens and invalid late tokens, allow valid flag keys with empty stdin, and reject an empty combined selection. Do not alter shared variable-mode key parsing or add discovery.

  Support omitted, inline and file variables from the start, with “Nil means omitted; nonnil empty map means supplied `{}`; nested valid JSON values retain their types; one object shared by every completion”. Reuse `parseUpdateVariablesFromFlags` only when a source is present; reject conflicting sources, unreadable/empty files, malformed/trailing JSON, arrays/scalars/outer null. Enforce explicit positive workers, complete+no-wait rejection, existing JSON/verbose and confirmation guardrails. Snapshot the local request before preview/confirmation; validate everything before a facade call is possible.

  Add isolated parser/command-fixture cases for all source combinations, ordering and invalid controls; existing variable-mode parser behavior must remain unchanged. These are preparation tests, not a substitute for actual command-execution coverage in T004. Format and run `isolated go test ./cmd -run 'TestUpdateUserTaskCompletionInput' -count=1` and affected existing variable-parser tests. Keep the parser and views in their respective files, but develop them together against the same frozen request; this task neither registers `--complete` nor changes shared flags.

  Implement typed completion views and bounded failure formatting in `cmd/cmd_views_usertask_complete.go` and `cmd/cmd_views_usertask_complete_test.go`, consuming the public models from T002.

  Implement the complete preview/execution payload from `data-model.md`. Preserve field constraints verbatim: operation is “Always `complete`”; phase is “`preview` or `execution`”; requestedKeys is “Frozen unique selection, always an initialized array”; requestedCount is “Length of requestedKeys”; variablesProvided “Distinguishes omitted source from `{}`”; variableCount is “Count of top-level supplied variable names; no variable values echoed”; items is “Initialized array in requested order”; counts has “Integer fields `preview`, `accepted`, `failed`, `timedOut`, `canceled`, `notAttempted`, including zero values”. “For complete preview/execution reports, the six status counts sum to requestedCount and correspond exactly to items.” “Preview items are all preview/not_submitted; their count is not presented as accepted or attempted work.”

  Render completion human/verbose results to configured stdout, not `renderHumanLine`; use safe typed per-task facts and aggregate backend-confirmed completion wording with unchanged accepted machine evidence. Keys-only preview emits selected keys, execution accepted keys only, with zero bytes when none qualify. Preserve quiet machine output and JSON precedence. Use succeeded preview, explicit `renderAcceptedResult` for all-accepted execution, and one `resultEnvelopeForError` with partial payload on failures. Preserve original machine errors and variable-mode schemas.

  Format aggregate human errors from counts, typed statuses and known HTTP labels only, grouping at most three named reasons plus other failures; never interpolate raw Error strings, transport chains or unknown backend text. Keep formatting separate from the runner's configured logger emission. Unit/renderer tests cover exact human/keys output, report invariants, all outcome types, no variable echo, one-envelope EOF, and malicious/distinctive error text excluded from human output while machine evidence survives. Format and run `isolated go test ./cmd -run 'TestUpdateUserTaskCompletionView' -count=1`. Add a local parser-to-preview test that feeds real parsed inputs into the report builder and renders human/JSON/keys/quiet results: verify frozen key order, omitted versus empty variables, accurate counts, zero calls and no acceptance claim. Reuse this request/report setup in T004 instead of inventing a second shape or preview implementation. Real logger routing/exit/terminal acceptance remains mandatory in T004; renderer tests alone do not establish it.

- [x] T004 [US1] Expose the complete mode with execution-level safety, compatibility and operator documentation in `cmd/update_usertask.go`, new `cmd/update_usertask_complete.go`, `cmd/update.go`, `cmd/update_usertask_complete_test.go`, `cmd/update_usertask_complete_output_test.go`, `cmd/update_usertask_complete_terminal_test.go`, `cmd/update_usertask_complete_logging_test.go`, `cmd/command_contract_test.go`, `README.md`, and regenerated `docs/cli/`.

  After the local request/report contract passes, register `--complete` and route it before mandatory variable parsing; keep ordinary variable execution and `--complete=false` unchanged. Use one frozen request, explicit-key tenant context without new reads/resolved-tenant fabrication, local dry-run, one plain confirmation through `cmd.ErrOrStderr()`, existing unattended rules, and the thin facade. Wire worker/fail-fast options unchanged. No temporary flag guards, second single-key workflow, discovery, variable planning, task-state polling or assignment. Reset/register the new flag in existing test helpers to prevent cross-test leakage.

  Integrate result rendering, one configured-logger ERROR for non-JSON failure, and exit using the original converted error; preserve bootstrap/local-validation helpers and existing no-error-code override. Never invoke a second error renderer after the execution envelope. Classify every new output as result, prompt, or diagnostic and trace service → facade → rendering → logging → exit; audit direct prints and raw error formatting. Fix issues at their owner rather than weakening earlier contracts.

  Actual command/subprocess tests must cover all aliases, repeated/comma/stdin/combined keys, duplicate collapse, invalid late input and empty-selection failure, flags plus empty stdin, optional/empty/nested/file variables and all invalid-source cases. Count requests and fail on any read/scope-write/assignment/poll call; omitted/empty/matching values must still submit completion. Cover accepted/rejected/unknown/timeout/cancel/not-attempted results, sequential/bounded workers, fail-fast preserving in-flight outcomes, JSON guardrails, automation, normal/verbose/quiet, quiet+JSON/keys, JSON precedence, exact human output, accepted-only keys and zero bytes when none accepted, one envelope followed by EOF, partial evidence and original exit codes.

  Real-terminal tests must use `testx.NewCmdTerminalRunner`, separately capture stdout/stderr, and exercise configured/inherited stderr, accepted/declined/default/EOF confirmation, redirected stdout, preview, auto-confirm and automation. Assert one prompt for the whole selection, no logger prefix, and no mutation on abort/preview; pipe-only tests do not satisfy this gate.

  Diagnostics must execute repeated multi-key failures with distinctive URL/method/body/key/secret markers through the command. At normal/verbose/quiet, assert actual configured logger framing, ERROR severity/destination, one compact grouped diagnostic, and no raw HTTP details on either stream; plain and structured formats plus inherited/explicit destinations are required. JSON keeps original evidence without duplicate human emission. Test DEBUG by flag and configuration independently of verbose, existing quiet precedence, redaction, actual completion/retry counts and unchanged result stdout under workers one and bounded concurrency. Prove these assertions fail when the new logger route is temporarily replaced with a direct stderr print or raw aggregate leakage is introduced; restore immediately and never commit intentional violations.

  Preserve and exercise existing variable planning, scope/shared-target updates, no-op output, ordinary verification, no-wait, reads, resolver behavior and `get pi --has-user-tasks` across supported versions; retain generic unsupported-version handling. Add/update metadata tests and both-mode help/examples, worker/dry-run/fail-fast wording, README and generated CLI docs in this same unit. Run `isolated make docs-content` after source metadata changes; do not hand-edit generated docs.

  Format touched Go files. Start with `isolated go test ./cmd -run 'TestUpdateUserTaskCompletion' -count=1`, then run the affected regression groups and docs checks from `quickstart.md`; verify patterns select the intended tests. Finish the completed cross-package/interface/concurrency change with the documented untagged `isolated make test` once. Record actual coverage/limitations and failure-path review in `specs/311-complete-user-tasks/progress.md`; update `quickstart.md` only if actual test names need correction. Do not repeat passed broad checks merely for a commit or later documentation-only edits.

**Checkpoint**: US1 has an actual, documented, tested command. Shared US2–US5 obligations implemented here are required completion gates, not work left for later phases.

## Phase 4: User Story 2 — Supply Optional Completion Variables (Priority: P1)

**Goal**: Omitted variables, `{}`, inline objects and file objects all use native completion semantics, with one identical payload per unique task.
**Independent test**: Through command execution, compare omitted/empty/nested/inline/file requests; reject invalid sources before calls; fail fixtures on any effective-variable read, diff planning, scope write or verification request.
**Tasks**: No additional task. T001 supplies native wire mapping; T002 owns validation/snapshotting and public mapping; T003 parses optional sources and distinguishes variablesProvided/count; T004 proves actual command behavior. Creating another payload implementation or a stand-alone test task would duplicate the same workflow. This story is complete only when those cases pass, including matching existing values still causing completion.

## Phase 5: User Story 3 — Retain Existing User-Task Workflows (Priority: P1)

**Goal**: Preserve variable updates, task reads/resolution, supported versions and clear both-mode documentation.
**Independent test**: Existing variable dry-run/no-op/scope/shared-target/wait/no-wait cases, reads/variables/resolution and process-instance has-user-tasks tests retain schemas, stream behavior and exit semantics; help describes both modes.
**Tasks**: No additional task. T001 preserves factories/adapters; T002 preserves constructors/public contracts; T003 leaves shared parser callers unchanged; T004 preserves dispatch, resets flags, runs command regressions and updates all operator documentation. Compatibility failures block the changing task, rather than being postponed to a later repair phase.

## Phase 6: User Story 4 — Preview, Confirm, and Control Bulk Execution (Priority: P2)

**Goal**: Review and confirm once, bound execution, stop scheduling safely and report partial/uncertain outcomes truthfully.
**Independent test**: Synchronization-controlled service fakes prove bounds, ordering, fail-fast and cancellation; real-terminal command tests prove prompt routing/abort; preview and unattended output combinations verify request counts and no acceptance claims before submission.
**Tasks**: No additional task. T001 preserves retry evidence, T002 builds single/collection scheduling once, T003 freezes input and renders preview/partial facts and T004 exercises the actual runner/terminal. Rebuilding a single-key path for bulk control would violate the project work-unit rules.

## Phase 7: User Story 5 — Consume Reliable Results and Diagnostics (Priority: P2)

**Goal**: Provide stable selected-format results, one envelope on failures, compact configured diagnostics and independent redacted DEBUG exchanges.
**Independent test**: Actual command failures across logger formats/destinations, output modes, both DEBUG entry points and worker policies verify one ERROR, no normal/verbose transport leakage, EOF after one JSON envelope, partial evidence and original exit status. Direct-print/raw-error mutation checks must fail.
**Tasks**: No additional task. T001 retains adapter evidence and existing transport instrumentation; T002 preserves normalized errors; T003 supplies typed safe rendering; T004 owns configured emission, command-level assertions and diagnostic regression sensitivity. Unsafe output is never an accepted intermediate CLI implementation.

## Phase 8: Polish & Cross-Cutting Concerns

No separate cleanup, review, formatting, test-run or documentation checkbox. These are included in their owning task; T004 includes final source metadata, generated docs, affected regressions, race validation and recorded coverage. Do not launch Ralph, commit, publish, or run live validation as a consequence of generating this file.

## Dependencies & Execution Order

```text
T001 native domain/adapters
  -> T002 collection service + public facade
     -> T003 local input + preview/result views
        -> T004 actual command + documentation + final validation
```

| Task | Prerequisites | Independently verifiable result |
|---|---|---|
| T001 | Existing repository context | Supported native completion with correct payload/error evidence |
| T002 | T001 | Public collection API with ordered outcomes, bounded scheduling and preserved partial errors |
| T003 | T002 | Validated frozen input produces accurate preview/results in each selected format without backend calls |
| T004 | T003 | End-to-end isolated command, compatible workflows, accurate help and diagnostics |

US1–US5 are acceptance perspectives of the same feature. Their release checkpoint is T004. All four tasks are required for the MVP; a component checkpoint does not establish command-level acceptance. Follow the dependency chain sequentially. No task exists solely to enable parallel execution, and no `[P]` tasks or agent delegation are required.

## Requirement and Acceptance Coverage

| Requirements | Owning tasks | Acceptance stories / success criteria |
|---|---|---|
| FR-001, FR-002, FR-003 | T002, T003, T004 | US1, US3; SC-001, SC-002, SC-006 |
| FR-004, FR-005 | T001, T002, T003, T004 | US2; SC-003 |
| FR-006, FR-007 | T001, T002, T004 | US1; SC-001, SC-004 |
| FR-008, FR-009 | T003, T004 | US4; SC-002 |
| FR-010, FR-011 | T001, T002, T004 | US4; SC-004, SC-007 |
| FR-012, FR-013 | T001, T002, T003, T004 | US1, US3, US4; SC-004, SC-006 |
| FR-014, FR-015, FR-016 | T002, T003, T004, T009 | US5; SC-005 |
| FR-017 | T001, T003, T004 | US5; SC-007 |
| FR-018 | T001, T002, T004 | US3; SC-006 |
| FR-019 | T004 | US3; SC-006 |
| FR-021 | T008, T009 | US4, US5; scope/plan/warnings/confirmation/result rhythm |

Every data-model field/invariant is assigned to T001–T003 and exercised through T004 where externally visible. No input, nullable/empty-array, status, omission, count, stream, tenant, or failure constraint is left to implementation-time discretion.

## Implementation Strategy and Task-Boundary Review

**MVP**: Complete T001–T004 for explicit-key native completion on all three versions, including optional variables, compatibility, preview/confirmation, safe automation/output and diagnostic coverage. Do not add search, assignment, downstream waiting or diagnostic instrumentation.

**Direct review result — six tasks become four**: The earlier split treated independently testable code and disjoint files as sufficient reasons for separate iterations. That was too fine-grained here. Keep package/file boundaries, but group the work around complete outcomes:

| Previous draft | Revised owner | Reason |
|---|---|---|
| Native adapters | T001 | Keep interface changes, all three implementations and affected doubles compiling together |
| Collection coordinator + separate facade | T002 | Existing `c8volt/task/client.go` collection methods already delegate to area functions; `convert.go` owns mechanical mappings. The facade adds no constructor, scheduler or backend workflow. Implement and verify partial-result/error mapping with its producer |
| Separate input parser + result views | T003 | Both define the same frozen selection, payload-presence facts and preview counts. One locally executable parse-to-preview contract removes a handoff and duplicated request/report fixture design while retaining separate source files |
| Command integration | T004 | Keep actual dispatch, prompts, configured logging, exit handling, command tests and operator documentation together |

**Why stop at four**: T001 contains three generated-client mappings plus retry/ambiguity evidence; T002 contains concurrency, cancellation and public error mapping. Combining them would join two substantial failure domains. T003 has bounded local behavior: reuse payload source selection when a source exists and decode completion numbers exactly and existing envelope helpers, adding completion-specific key validation and typed reports. It has no command lifecycle or remote planning. T004 already carries the real-terminal, logger-format/destination, DEBUG, exit-code and compatibility matrices. Absorbing T003 into T004 would make that integration unit less practical. No separate docs, tests, setup, or cleanup task is needed.

**Keep verification focused**: Service gates/counters own exhaustive scheduling semantics; facade tests own conversions/options and partial-error preservation; local CLI tests own detailed parsing/report cases; actual command tests still prove wiring, zero-call validation, prompts, streams, configured logging and exits. Reuse fixtures and table cases within their existing package where helpful; do not introduce a generic cross-package test framework or multiply every input case by every output/worker/version combination. Retain the required combinations and every acceptance obligation above. Unit tests never replace the mandated command-execution or terminal checks.

**Progress and references**: All original checkboxes were unchecked. Original draft T001 maps to current T001; original T002/T003 to current T002; original T004/T005 to current T003; original T006 to current T004. No external task references were found in repository Markdown. This revision changes planning only; it does not claim implementation or runtime validation.

**Counts**: 4 total (down from 6): foundational 2, US1 primary ownership 2; setup, additional US2–US5 tasks and polish 0. All five stories, 19 functional requirements and seven success criteria retain mandatory coverage. No parallel-only work units.


## Phase 9: Convergence

- [x] T005 [US1] Preserve exact numeric completion variables from CLI input through native submission per FR-004, US2/AC1, SC-003 and T003–T004 (resolved; F1).

  Original evidence before the fix: `cmd/update_usertask_complete_input.go` calls `parseUpdateVariablesFromFlags` in `cmd/update_variables_payload.go`, whose `json.Unmarshal` into `map[string]any` converts numbers to float64 before `toolx.CopyJSONMap` can preserve them. Consequently, an inline or file payload containing `9007199254740993` is submitted as `9007199254740992`; nested numbers are affected too. The native adapter tests already check exact large numbers, but `cmd/update_usertask_complete_test.go` captures request bodies with a default JSON decoder and tests only small numeric values, masking this CLI gap.

  Provide a completion-specific number-preserving decoding path, or an opt-in shared helper, using `json.Decoder.UseNumber` before any float64 conversion. Preserve existing variable-update callers and semantics, source exclusivity, omitted versus empty objects, outer-object validation, trailing-input rejection, complete validation before submission, and snapshot ownership. Keep the fix scoped to the existing completion input workflow; do not alter the facade, generated clients or backend workflow unnecessarily.

  Include failing-then-passing parser and actual-command regressions for both `--vars` and `--vars-file`, with large integers and precise decimals nested in objects/arrays and sent identically to multiple unique keys. Assert exact outbound numeric values using raw request bodies or a number-preserving decoder; float64-based equality or `JSONEq` alone is insufficient. Retain invalid/trailing-payload zero-request cases and variable-mode compatibility checks. Format touched Go files and run targeted isolated completion input/payload tests, then affected variable-parser/update regressions under the clean-environment rules in `quickstart.md`. No live validation is required or permitted by this task.

  Completed: completion now reads the shared source and decodes with `UseNumber` plus EOF validation; variable mode retains its original decoder. Exact parser and two-key native-request regressions for inline/file payloads failed before the fix and passed afterward. Redundant CLI/facade JSON copies were removed, with coordinator/native ownership tests retained. Shared adapter outcome tests moved to `internal/services/common/usertask_completion_test.go`; version wire, ownership, invalid-payload, logging and delegation coverage remains. Full isolated `make test` passed with the race detector (8,049 tests/subtests; 86 tested packages; no test skips). See `progress.md` for validation details.

## Phase 10: Requested preflight checks

- [x] T006 [US1] Prepare completion with one native lookup per unique key before approval; skip observed completed tasks with existing failed/not_submitted conflict evidence and omitted statusCode, warn on eligible unassigned tasks, and warn on unavailable checks while leaving eligibility to the backend. Keep backend preparation/execution in the service and thin facade, with frozen inputs reused after approval and no reread/polling. Suppress transport retries only for the advisory read. Prompt for the eligible count only; fail-fast preflight conflicts prevent all submissions. Preserve local dry-run, output schemas, counts, keys-only selection and original exits. Update spec, contracts, source help, README and generated docs. Verify duplicates, snapshots, cancellation, read failures, stale observations, mixed outcomes, all-completed/no-prompt, warnings before real-terminal approval/decline, logger routing/redaction, quiet and machine modes, supported adapters, and ordinary-read retry compatibility using isolated fixtures; finish with targeted checks and the race suite.

T006 supersedes only the earlier execution zero-read constraints in T002/T004: preflight now makes one keyed native read per unique task. Existing no-read requirements still apply to local preview, validation and the scalar completion adapter. T001–T005 completion evidence remains historical; T006 extends FR-006, FR-008–FR-017 and FR-019 with FR-020 and US1/US4/US5 coverage. Dependency: T005. Current total: six tasks, one cohesive new work unit; no parallel-only or standalone test/documentation task.

T006 completed: targeted checks and the final isolated race suite passed; generated docs, Go formatting, feature links and whitespace checks passed. See `progress.md` for red/green and diagnostic-regression evidence. No live validation or polling was added.

## Phase 11: Human wording refinement

- [x] T007 [US5] Lead completed-task errors with state and no-submission consequence, using singular/plural grammar. Omit zero counters from human results, avoid repeating them in errors, and mention variables only when supplied (including `{}`). Keep JSON fields, zero counts, item error evidence, keys-only behavior, logger routing and conflict exit code unchanged. Update source help, spec/contracts, README, generated docs and exact execution tests for omitted/empty/nonempty variables, single/multiple conflicts, verbose/preview/machine modes and safe aggregate diagnostics. Run isolated targeted completion, metadata and docs tests and record results. Dependency: T006; no service change or polling.

Current total: seven cohesive tasks. T007 supersedes earlier human wording expectations only; historical implementation/test evidence and machine contracts remain intact.

T007 completed: exact human/verbose/preview wording and unchanged machine evidence/conflict exits passed targeted checks and command/docsgen regressions. Diagnostic mutation checks detected direct printing and raw-error exposure; both mutations were restored. See progress.md for results.

## Phase 12: Consistent update rhythm

- [x] T008 [US5] Align completion with scope → plan → warnings → confirmation → result. Use configured INFO/WARN diagnostics, plain stderr prompts and stdout results. Name a single key; use eligible bulk counts and correct singular/plural grammar. Explain mixed fail-fast without prompting; retain the short all-completed conflict path. Preserve one lookup per key, zero-request preview, no polling, machine schemas, keys output and exits. Update source help, feature contracts, README and generated docs, plus real-terminal ordering/mode/logger tests and existing output regressions. Validate isolated targeted tests, diagnostic mutation sensitivity and the race suite. Dependency: T007; covers FR-021.

Current total: eight cohesive tasks. T008 supersedes earlier human preview, prompt and result wording only; historical evidence and machine contracts remain intact.

T008 completed: targeted completion/metadata/docs tests, diagnostic mutation checks and race validation passed after correcting action-focused help wording. See progress.md for exact commands and results. No live tests or polling were added.

## Phase 13: INFO logging for human completion output

- [x] T009 [US5] Route all completion human previews, summaries and verbose rows through the configured INFO helper; preserve JSON/keys-only stdout, plain prompts, existing exits, one lookup per key and no polling. Update FR-015/FR-021, contracts, operator docs and generated references. Verify configured/inherited destinations, plain/structured formats, logger levels and quiet suppression, success/failure/preview/verbose paths, real-terminal ordering, machine compatibility, safe aggregate failures and DEBUG redaction. Prove direct printing and raw-error regressions fail; run isolated targeted and affected command/docs regression tests. Dependency: T008.

Current total: nine cohesive tasks. T009 supersedes all earlier human-stdout requirements in T003/T004/T008 and their historical evidence; machine-output and backend contracts are unchanged.

T009 completed: targeted checks, diagnostic mutation checks, documentation regeneration and the affected command/docs race suite passed. See progress.md for evidence. No backend changes, polling or live validation were added.

## Phase 14: Shared scalar job guard and compact errors

- [x] T010 [US5] Reuse the existing scalar job planning lookup to guard COMPLETED for `--complete`, `--fail` and `--throw-bpmn-error` before plan/prompt/submission (including dry-run), with conflict exit 6 and variables wording only when supplied. Keep other states backend-owned, no new reads/polling, and preserve backend exits/JSON, scalar success/prompts and bulk behavior. Bound lookup/submission human errors through the configured logger. Update spec/contracts, source help, README and generated docs. Verify three adapters, request counts, mode/payload matrix, real terminal behavior, logger destinations/formats/redaction, original JSON evidence/exits, repeated batch regressions, and diagnostic mutation checks; format and run targeted checks then command/docs race tests. Dependency: T009.

Current total: ten cohesive tasks. T010 is one narrowly scoped follow-up; implementation, fixtures, docs and validation stay together. It extends the original job non-goal only as specified by FR-022.

T010 completed: targeted regressions, real-terminal approval/decline checks, both diagnostic mutation checks, docs regeneration and the command/docs race suite passed (5,178 tests/subtests, zero skips). See `progress.md`. No new reads, polling, model changes or live validation.
