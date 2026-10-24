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

  Add version-specific mock/fake cases for omitted/empty/nested data, no-content acceptance, 400/401/403/404/409, exhausted throttling, uncertain 504/transport errors, cancellation before request/during retry delay, and malformed responses. Mocks must provide request URL/method required by existing HTTP error conversion. Assert no forbidden calls or raw normal-level logging. Format changes and run `isolated go test ./internal/services/usertask/... -run 'Complete|Completion' -count=1`; run targeted tests in any additional package whose interface doubles changed. Do not mark complete with consumers failing compilation. This internal-only increment requires no CLI documentation change.

- [x] T002 Deliver the collection completion API through its service and public facade: implement and test the coordinator in `internal/services/usertask/completion.go` and `internal/services/usertask/completion_test.go`, using the native operation from T001 and existing `toolx/pool` scheduling, and expose it through `c8volt/task/{api.go,model.go,convert.go,completion.go,completion_test.go}` in the same unit.

  Apply these exact input constraints from `data-model.md`: “Nonempty selection; exactly 16 decimal digits per key under existing user-task conventions; stable first-occurrence uniqueness; flags precede stdin”; “Nil means omitted; nonnil empty map means supplied `{}`; nested valid JSON values retain their types; one object shared by every completion”; “Zero means automatic at the service boundary; positive values bound concurrency; negative invalid. CLI explicitly supplied zero is invalid”; “Existing fail-fast, no-worker-limit, dry-run, verbose, and explicit-key authorization options; no-wait rejected”. Flags/stdin merging remains CLI-owned; the service validates the supplied key sequence and stable uniqueness independently.

  Validate all keys, options and JSON serializability and snapshot nested payloads before submission; guard missing API dependency. Build single and multiple key handling together, including optional variables. Direct-call dry-run returns ordered preview/not_submitted items without calls. Use `DetermineNoOfWorkers`, retain explicit-worker precedence over no-worker-limit, and initialize one ordered result slot per unique key. Follow existing job scheduling patterns without copying its discovery/reconciliation lifecycle: pool context stops scheduling, original caller context governs started requests, and a started flag distinguishes unscheduled slots. Keep already accepted work when another key fails; join real callback errors and original caller cancellation rather than synthetic fail-fast cancellation.

  Preserve the collection constraints: “is initialized to an empty array when no item evidence exists”; “For a valid executed or dry-run selection, it contains exactly one item per unique key in selection order. Do not sort by completion time or drop unattempted slots. Return partial results alongside a nonnil error.”

  Tests use synchronization gates/counters for sequential ordering, worker bounds/defaults/precedence, out-of-order responses, continue-on-error, fail-fast with in-flight acceptance and queued remainder, pre-start and mid-flight cancellation, unknown submission, snapshot ownership, duplicates, invalid late keys/payloads, empty selection and no-wait. Assert every unique key is accounted for and forbidden reads/writes/polls never occur. Format and run `isolated go test ./internal/services/usertask/... -run 'Complete|Completion' -count=1`, with targeted race coverage for the new concurrent coordinator. Keep the facade mapping checks below in this same work unit; this increment does not change CLI docs.

  Expose the collection completion facade in `c8volt/task/api.go`, `c8volt/task/model.go`, `c8volt/task/convert.go`, new `c8volt/task/completion.go`, and `c8volt/task/completion_test.go`, updating affected public-interface stubs together.

  Add the exact public `CompleteUserTasks` signature in `contracts/facade-service.md`. Copy/map inputs and options, delegate to this coordinator, map partial results before applying `ferrors.FromDomain`, and preserve constructors/root composition. Public models must match all T001 status/submission invariants and use `key`, `status`, `submission`, `statusCode,omitempty`, `error,omitempty`, and initialized `items` JSON fields. Do not expose internal/generated types, schedule workers, add backend reads, reinterpret tenant filtering, or discard outcome evidence on failure.

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

  Render completion human/verbose results to configured stdout, not `renderHumanLine`; use safe typed per-task facts and aggregate acceptance-without-verification wording. Keys-only preview emits selected keys, execution accepted keys only, with zero bytes when none qualify. Preserve quiet machine output and JSON precedence. Use succeeded preview, explicit `renderAcceptedResult` for all-accepted execution, and one `resultEnvelopeForError` with partial payload on failures. Preserve original machine errors and variable-mode schemas.

  Format aggregate human errors from counts, typed statuses and known HTTP labels only, grouping at most three named reasons plus other failures; never interpolate raw Error strings, transport chains or unknown backend text. Keep formatting separate from the runner's configured logger emission. Unit/renderer tests cover exact human/keys output, report invariants, all outcome types, no variable echo, one-envelope EOF, and malicious/distinctive error text excluded from human output while machine evidence survives. Format and run `isolated go test ./cmd -run 'TestUpdateUserTaskCompletionView' -count=1`. Add a local parser-to-preview test that feeds real parsed inputs into the report builder and renders human/JSON/keys/quiet results: verify frozen key order, omitted versus empty variables, accurate counts, zero calls and no acceptance claim. Reuse this request/report setup in T004 instead of inventing a second shape or preview implementation. Real logger routing/exit/terminal acceptance remains mandatory in T004; renderer tests alone do not establish it.

- [ ] T004 [US1] Expose the complete mode with execution-level safety, compatibility and operator documentation in `cmd/update_usertask.go`, new `cmd/update_usertask_complete.go`, `cmd/update.go`, `cmd/update_usertask_complete_test.go`, `cmd/update_usertask_complete_output_test.go`, `cmd/update_usertask_complete_terminal_test.go`, `cmd/update_usertask_complete_logging_test.go`, `cmd/command_contract_test.go`, `README.md`, and regenerated `docs/cli/`.

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
| FR-014, FR-015, FR-016 | T002, T003, T004 | US5; SC-005 |
| FR-017 | T001, T003, T004 | US5; SC-007 |
| FR-018 | T001, T002, T004 | US3; SC-006 |
| FR-019 | T004 | US3; SC-006 |

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

**Why stop at four**: T001 contains three generated-client mappings plus retry/ambiguity evidence; T002 contains concurrency, cancellation and public error mapping. Combining them would join two substantial failure domains. T003 has bounded local behavior: reuse `parseUpdateVariablesFromFlags` when a source exists and existing envelope helpers, adding completion-specific key validation and typed reports. It has no command lifecycle or remote planning. T004 already carries the real-terminal, logger-format/destination, DEBUG, exit-code and compatibility matrices. Absorbing T003 into T004 would make that integration unit less practical. No separate docs, tests, setup, or cleanup task is needed.

**Keep verification focused**: Service gates/counters own exhaustive scheduling semantics; facade tests own conversions/options and partial-error preservation; local CLI tests own detailed parsing/report cases; actual command tests still prove wiring, zero-call validation, prompts, streams, configured logging and exits. Reuse fixtures and table cases within their existing package where helpful; do not introduce a generic cross-package test framework or multiply every input case by every output/worker/version combination. Retain the required combinations and every acceptance obligation above. Unit tests never replace the mandated command-execution or terminal checks.

**Progress and references**: All original checkboxes were unchecked. Original draft T001 maps to current T001; original T002/T003 to current T002; original T004/T005 to current T003; original T006 to current T004. No external task references were found in repository Markdown. This revision changes planning only; it does not claim implementation or runtime validation.

**Counts**: 4 total (down from 6): foundational 2, US1 primary ownership 2; setup, additional US2–US5 tasks and polish 0. All five stories, 19 functional requirements and seven success criteria retain mandatory coverage. No parallel-only work units.
