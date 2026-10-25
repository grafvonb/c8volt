# Tasks: Suspend Process Instances

**Input**: Design documents from `specs/346-suspend-process-instances/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [facade/service contract](contracts/facade-service.md), and [quickstart.md](quickstart.md).

**Branch / issue**: `346-suspend-process-instances` / #346

**Tests**: Required by the specification. Tests, fixtures, formatting, documentation appropriate to the change, and targeted validation are part of each implementation task; they are not deferred test-only tasks. Write the relevant behavioral assertions first, demonstrate the missing behavior with isolated fakes once the test scaffolding compiles, then implement and pass them before marking the task complete.

**Organization**: Three cohesive work units. T001 is shared protocol groundwork; T002 is owned by US1 and includes inseparable US2 safety and US3 collection behavior; T003 is owned by US3 and includes US4 documentation/fixture adoption. The US2 and US4 phases retain explicit acceptance ownership without inventing separate tasks that repeat the same code. Story labels identify the primary owner, not exclusive coverage. This implements the plan's three-unit sequence and respects Ralph's rule against starting an unrelated story within an iteration.

## Format and paths

Task lines use `- [ ] Tnnn [USn] Description with paths`, omitting story labels for foundations. Supporting scope, constraints, and completion criteria immediately below each task are part of its task description. All source paths are repository-relative. Planned new files are distinguished by the action to add them. No task has a `[P]` marker because the executable work units depend on each other.

## Responsibilities within every work unit

Read the active feature artifacts, `AGENTS.md`, `.specify/memory/constitution.md`, and `specs/ralph-implementation-rules.md`; inspect nearby implementation/helpers before editing. Preserve the CLI → thin facade → internal service → version adapter boundaries. Add required Go comments for new/materially changed declarations and tests, run `gofmt` on touched Go files, and use relevant lightweight diff checks. Update feature progress/task evidence within the implementation unit, without claiming checks that were skipped or unavailable.

Before any validation command, inspect its entry points/build constraints and isolate inherited flags, profile selectors, credentials, config directories, and endpoints. Use `GOENV=off GOFLAGS=` for the Go commands listed below; subprocess tests must supply test-owned config and fake endpoints. No integration build tags, environment-backed helpers, real Camunda, Docker/Kind/Kubernetes, provisioning, live target discovery, deployment, smoke tests, live resource mutation, or cleanup are permitted. Do not run or source the Bash integration fixture. Test examples in this file are future implementation checks; no application tests are required for generating this task list.

Keep existing error-class/exit precedence, mode precedence, cancellation/deletion behavior, generated clients, and shared envelope schemas unchanged. Do not add general workflow infrastructure, dependencies, root traversal, search selection, resumption, native batch suspension, or definition lifecycle changes. Each issue-associated implementation commit must use Conventional Commits with its subject ending in ` #346`.

## Phase 1: Setup (Existing Infrastructure)

The repository, Go toolchain, generated client, configuration, logger, and isolated test helpers already exist. No initialization, branch-check, context-reading, or dependency-installation task is needed. Those are work-unit responsibilities, not deliverables.

## Phase 2: Foundational — Authoritative Submission Evidence

**Goal**: Provide a tested internal primitive that submits an exact key and retains acceptance/rejection/uncertainty without exposing an incomplete public suspension workflow.

- [x] T001 Add suspension domain evidence models in `internal/domain/processinstance_suspension.go` and the private v810 submission primitive in `internal/services/processinstance/v810/suspension_submission.go`, extending `internal/services/processinstance/v810/contract.go` and its generated-client test double in `internal/services/processinstance/v810/service_test.go`; include protocol/retry/pacing tests in `internal/services/processinstance/v810/suspension_submission_test.go`, required comments, formatting, and targeted validation.

**T001 implementation and completion criteria**:

- Add `ProcessInstanceSuspensionResult(s)` and typed workflow/submission/confirmation enums according to the exact field constraints below. Existing Reporter/state/date types remain unchanged. Public facade models are added with the complete operation in T002.
- Use the already generated `SuspendProcessInstanceWithResponse` with `{}` as its request body and exactly the supplied key. Add its exact signature only to the v810 narrow client interface and matching mocks/assertions. Do not regenerate or hand-edit generated clients.
- Require a valid HTTP 204 response without a call error to establish acceptance. Reject nil/malformed HTTP metadata, another 2xx, and call-error-plus-partial-response as acceptance evidence. Preserve explicit already-suspended/terminal/terminating/missing/unauthorized rejections; do not read or reconcile state after rejection or uncertain submission.
- Reuse `services.RetryCamundaMutation` with the original call error/response/body for retry classification; retain cumulative submission evidence outside the callback. Cover 429 and 503 RESOURCE_EXHAUSTED, temporary transport failure, ambiguity followed by rejection, ambiguity followed by valid 204, retry exhaustion, backoff cancellation, unsent request-rate admission, earlier uncertainty followed by an unsent attempt, and HTTP-timeout wrapping of local pacing failure. Only actual valid acceptance may replace uncertainty; a later rejection cannot.
- Keep submission responses, typed error causes, and machine evidence separate from safe human facts. Suppress functional retry chatter where appropriate without disabling configured HTTP DEBUG diagnostics. No raw normal/verbose logging or new logger is introduced.
- Tests use generated-client mocks/test-owned HTTP transport and deterministic short retry controls or immediate Retry-After values. Assert method/body/path and attempt counts as well as returned evidence. Include context cancellation before sending and unexpected server/proxy outcomes. This unit performs no confirmation and does not register CLI/public methods, so it cannot claim a default workflow is complete.
- Run `env GOENV=off GOFLAGS= go test ./internal/services/processinstance/v810 -run 'ProcessInstanceSuspensionSubmission' -count=1` after verifying that the new tests match. Include focused domain checks only where they test actual validation behavior, not duplicate struct declarations. Internal-only groundwork needs no user-facing docs regeneration.

**Verbatim field constraints from data-model.md, binding in T001 and its T002 public mapping**:

| Field / JSON | Constraint |
| --- | --- |
| `Key` / `key` | “Requested instance key, populated even for skipped or failed work.” |
| `Status` / `status` | “`preview`, `accepted`, `confirmed`, `failed`, `timed_out`, `interrupted`, or `not_attempted`; workflow disposition, not engine state.” |
| `Submission` / `submission` | “`not_submitted`, `accepted`, `rejected`, or `unknown`; retained independently of confirmation.” |
| `Confirmation` / `confirmation` | “`not_requested`, `unconfirmed`, or `confirmed`. Preview, rejected/unknown submission, skipped work, and no-wait use `not_requested`; accepted default waiting starts as `unconfirmed`.” |
| `StatusCode` / `statusCode,omitempty` | “Last valid submission HTTP status, including 204 after acceptance; never overwritten with a confirmation-read status. Zero means no valid response status.” |
| `HasObservation` / `hasObservation` | “True only when a matching instance was read or a typed not-found observation established ABSENT. False differs from an actual UNKNOWN observation.” |
| `LastObservedState` / `lastObservedState,omitempty` | “Last observed state, empty when `hasObservation=false`. SUSPENDED proves confirmation only after accepted submission.” |
| `SuspendedDate` / `suspendedDate,omitempty` | “Existing mapped date from the latest successful matching-instance observation; never synthesized. Omit after an ABSENT observation; preserve along with the prior state on a later read failure.” |
| `Error` / `error,omitempty` | “Per-key machine error evidence. Never use as a human logger message or verbose detail.” |
| Collection `Items` / `items` | “Initialize the collection as an empty array, not null. Return one item per normalized selected key in first-input order. No asynchronous batch identifier is created.” |

**Checkpoint**: The protocol/evidence foundation compiles and its isolated tests pass. No new default public operation is exposed yet.

## Phase 3: User Story 1 — Suspend Exactly the Selected Instance (P1)

**Goal**: Deliver the complete public-library single/collection suspension workflow, including the US2 confirmation guarantees needed to make US1 truthful.

**Independent test**: Through factory-composed facades backed by isolated fake servers, suspend an ACTIVE target and confirm only after SUSPENDED; test parent-only, child-only, and simultaneous parent/child selection with zero hierarchy calls. On 8.8/8.9, all entry points reject capability before unsupported I/O. CLI realization of this story is completed by T003.

- [x] T002 [US1] Deliver complete single/collection suspension through `internal/services/processinstance/api.go`, `internal/services/processinstance/suspend.go`, `internal/services/processinstance/waiter/suspension.go`, the `v88/contract.go`, `v89/contract.go`, `v810/contract.go` and corresponding `suspend.go` adapters under `internal/services/processinstance/`, and `c8volt/process/api.go`, `model.go`, `convert.go`, and `suspend.go`; include all affected interface/test-double updates and service/waiter/facade tests in the same compiling unit, with bounded confirmation, no-wait, preview, partial-result evidence, formatting, and targeted validation.

**T002 implementation and completion criteria**:

- Add `ValidateProcessInstanceSuspension() error` and the singular suspension method to the existing service API/all adapter contracts. Return wrapped `domain.ErrUnsupported` from v88/v89 without I/O; v810 uses T001. Guard singular, collection, dry-run, and empty-collection entry paths before unsupported requests or pool dispatch. Factory wiring remains the existing version selection.
- Add public `SuspendProcessInstance(ctx,key,opts...)(SuspensionResult,error)` and `SuspendProcessInstances(ctx,keys,wantedWorkers,opts...)(SuspensionResults,error)` with the exact signatures in `contracts/facade-service.md`. Keep mapping/delegation thin and return converted results even when `ferrors.FromDomain` returns a nonnil error. Copy slices, preserve all T001 field/omission constraints, and keep internal/generated types out of public signatures. Document exported semantics and the public interface addition; command-facing README/generated docs remain T003 because no command exists yet.
- Service-owned normalization follows the verbatim rule “Trim and validate using the established 16-digit key convention, then deduplicate by first occurrence.” Validate all inputs before mutations. Follow “Distinct parent and child keys never collapse.” Preview is a local capability/input check, not an eligibility read. v810 empty collections return `items: []` with no requests; v88/v89 empty/preview calls still fail capability validation. No-wait never converts preview/empty results into acceptance.
- Build collection behavior once for single/multiple input using `toolx.DetermineNoOfWorkers` and `pool.ExecuteSlice`, with existing worker/fail-fast/no-worker-limit controls. Initialize keyed not-attempted slots, merge actual results by index, retain joined causes and partial evidence, and explicitly retain context cancellation even if no callback ran. Fail-fast may stop scheduling but cannot erase in-flight acceptance or claim unstarted work succeeded.
- Implement the focused suspension waiter using a narrow keyed-reader dependency, existing positive backoff configuration/`NextDelay`, and the earlier configured/caller deadline, including in-flight reads and transport read retries. Validate timing before mutation. Accept only a matching instance's SUSPENDED state; preserve the existing date without requiring/manufacturing it. Stop unsuccessfully on COMPLETED/CANCELED/TERMINATED with conflict-class cause. Typed not-found records ABSENT and continues; ACTIVE/UNKNOWN/unrecognized nonterminal states/timestamp alone do not confirm. Never use `State.IsTerminal()` or raw `404`/`not found` substring matching to decide suspension success or absence.
- Read denial/other read errors stop observation with accepted evidence and prior state intact. Timeout/attempt exhaustion retain timeout-class cause; interruption retains cancellation cause. Explicitly test no-observation versus observed UNKNOWN, ABSENT date omission, failed-read preservation of prior date, and mismatched identity. Polling never has a mutation callback. No-wait performs zero confirmation reads; rejected or ambiguous submissions also perform zero confirmation reads. There is no mandatory preflight read or hierarchy discovery.
- Suppress nested human error emission; retain configured DEBUG diagnostics and safe verbose facts. The command added in T003 will own the aggregate failure record. Do not repurpose cancellation's root-specific failure annotations or change generic waiter/cancel/delete semantics.
- Add tests under `internal/services/processinstance/suspend_test.go`, `waiter/suspension_test.go`, `v88/suspend_test.go`, `v89/suspend_test.go`, `v810/suspend_test.go`, and `c8volt/process/suspend_test.go` using a `TestProcessInstanceSuspension` prefix. Cover all data-model transitions, state races, terminal stop counts, no-op library collections, guard ordering, malformed keys, parent+child/duplicate selection, worker limits, fail-fast and cancellation before/during scheduling, option/context propagation, response-code preservation, JSON field mapping/omission, and returned partial reports with original error classification. No fake may hide an omitted production version guard: include factory-composed facade cases.
- Update concrete stubs in `c8volt/process/client_test.go`, `c8volt/resource/client_test.go`, `internal/services/processinstance/dryrun_test.go`, `internal/services/processdefinition/delete_test.go`, `internal/services/ops/orphan_purge_test.go`, and any additional compiler-identified implementations with the same change. Include explicit suspension callbacks in new tests even where interface embedding would compile automatically.
- Run `env GOENV=off GOFLAGS= go test ./internal/services/processinstance/... -run 'ProcessInstanceSuspension' -count=1` and `env GOENV=off GOFLAGS= go test ./c8volt/process -run 'ProcessInstanceSuspension' -count=1`. Confirm patterns match; run closest affected cancellation/forced-deletion/expectation/date regressions. Compile affected test-double consumer packages with isolated checks before declaring the unit compiling; do not leave their repair to T003. If shared runtime helpers are changed, run their tests plus consumers now rather than relying only on new suspension tests.

**Checkpoint**: The complete public-library operation can prove acceptance versus confirmation for one or many keys. US1 and US2 backend acceptance, plus US3 collection mechanics, are implemented together. Full command acceptance remains dependent on T003.

## Phase 4: User Story 2 — Distinguish Submission from Observed Suspension (P1)

**Goal**: Preserve truthful accepted, rejected, uncertain, unconfirmed, and confirmed outcomes under state changes and failures.

**Task ownership**: No separate task. T001 owns authoritative/retry submission evidence; T002 owns confirmation and facade preservation; T003 owns CLI rendering/exit assertions. Deferring this behavior to a later story would make the US1 default operation unsafe, so it is explicitly included in those units rather than reimplemented here.

**Independent test criteria**: Script accepted → ACTIVE → SUSPENDED, every true terminal transition, typed ABSENT/UNKNOWN, date-only evidence, first-read failure versus observed UNKNOWN, read denial, timeout/attempt bound, in-flight deadline, interruption, malformed identity, rejected/ambiguous submission with a possible later SUSPENDED read, and no-wait. Assert last-observation/submission facts and exact request counts; no failed submission reconciliation or polling resubmission. CLI tests decode the same evidence and preserve exit classes. Acceptance requires T001–T003, not a test-only follow-up.

## Phase 5: User Story 3 — Suspend Explicit Collections Safely in Scripts (P2)

**Goal**: Expose the already-complete collection operation as one safe CLI lifecycle for all input sizes, with operator guidance and fixture adoption in the same delivery.

**Independent test**: Execute canonical/alias commands against isolated fakes with explicit/duplicate/stdin keys, preview, mixed outcomes, worker controls, and real terminal input. Assert exact outputs, one JSON envelope with EOF, accepted keys-only semantics, configured logger records and real failure exits, unchanged relative targets, and zero mutations/prompts for invalid or preview paths.

- [x] T003 [US3] Deliver the complete command and operator contract in new `cmd/suspend.go`, `cmd/suspend_processinstance.go`, `cmd/suspend_processinstance_batch.go`, and `cmd/cmd_views_processinstance_suspend.go`, with execution/output/logger/terminal tests in `cmd/suspend_processinstance_test.go`, `cmd/suspend_processinstance_output_test.go`, `cmd/suspend_processinstance_logging_test.go`, and `cmd/suspend_processinstance_terminal_test.go`; update metadata tests in `cmd/command_contract_test.go`, source help, `README.md`, generated `docs/cli/` and `docs/index.md`, and suspension setup in `integration/scripts/run-c810-suspended.sh`, including formatting, isolated targeted/regression/race checks, docs generation, and static Bash syntax validation.

**T003 implementation and completion criteria**:

- Build one collection path for both one key and many keys. Support `suspend process-instance` / `suspend pi`, repeated/comma-separated `--key`/`-k`, newline stdin through `-`, dry-run, no-wait, workers/no-worker-limit/fail-fast, inherited confirmation/automation/output/logging/request-rate flags, and existing backoff bindings. Use `collectExplicitPIAdminInputOptions()`; do not filter explicit keys by configured discovery tenant. Add no force/no-state-check/search/limit/batch-size flags.
- Validate all local input/mode restrictions first. Empty/whitespace stdin keeps `stdin contained no keys`; missing selection and terminal stdin with `-` remain invalid. JSON requires dry-run, auto-confirm, or automation; JSON+verbose is invalid. Preserve effective JSON > keys-only > human precedence and quiet/debug flag behavior. Advertise state-changing, full shared-contract and automation support, with explicit supported output modes.
- Ask the service for a no-I/O preview, copy its ordered unique keys, return a mode-aware preview for dry-run, otherwise apply the existing prompt eligibility and execute exactly that frozen scope. No state eligibility claim, hierarchy request, renderer-discovery request, or premature machine-key output is allowed. Keep lifecycle in the focused batch file, formatting in the view file, and all backend mechanics in services.
- Prompt with `confirmCmdOrAbortFn` on `cmd.ErrOrStderr()` using “You are about to suspend N process instance(s). Do you want to proceed?” and preserve shared formatting/default-no/EOF/abort behavior. Terminal stdin with redirected stdout still permits confirmation. No prompt on invalid/empty input, unsupported capability, dry-run, auto-confirm, or automation. Decline submits no mutation.
- Implement the exact view contract in `contracts/cli.md`: human summary on configured stdout; explicit succeeded preview even with no-wait; accepted envelope only for actual no-wait acceptance; error-envelope selection before no-wait with complete partial payload. Keys-only preview prints selected keys; execution prints only accepted keys, including accepted-but-unconfirmed, with failure exit retained. Rejected/unknown/unsubmitted keys are omitted and zero qualifying keys means zero bytes. Quiet suppresses human informational summaries but not required errors or explicit machine output. Preserve `items: []` for direct empty-result view checks without creating a successful empty-input CLI branch.
- Trace service error → facade conversion → rendering → logger → exit. Before rendering, use established bootstrap/command helpers with safe human text. After partial rendering, non-JSON failure emits exactly one compact grouped aggregate ERROR through the configured logger and calls `exitAfterRenderedResult` with the original operation error. JSON receives no duplicate diagnostic/envelope. Writer failure never invokes a second renderer and preserves an existing operation error's exit. Never log raw per-key errors, raw joined causes, transport strings, URLs/methods/bodies/cursors at normal or verbose levels. Keep machine error evidence and existing schemas intact.
- Use `TestSuspendProcessInstance` names and execution-path tests with `testx` fake HTTP/subprocess helpers. Cover normal/default, preview, no-wait, human/JSON/keys/quiet, quiet+machine output, JSON+keys precedence, invalid combinations, auto-confirm/automation, workers/fail-fast, parent+child/duplicates, alias/stdin input, capability rejection including dry-run, empty input, mixed/partial failures, state races and unconfirmed results. Decode exactly one JSON envelope and require EOF; assert subprocess exit and request counts, not just renderer outputs. Reuse T002 state fixtures rather than duplicate the entire Cartesian product across every mode.
- Diagnostic tests use repeated multi-key errors with distinctive URL/method/body/key/cursor/secret markers; capture both streams. Assert one ERROR's severity, bounded wording, grouping/counts, configured/inherited destination, and absence of duplicates and raw details in normal/verbose/quiet human and keys modes. Preserve intended machine evidence under JSON. Test DEBUG via flag and configured level independently of verbose, actual quiet-over-debug precedence, plain/time and structured formats, and redacted HTTP diagnostics with stdout uncontaminated. Assertions must fail for a direct-stderr logger bypass or raw aggregate logging. Inspect every changed direct print and raw error formatting boundary.
- Real-terminal tests use actual PTY stdin and separately captured stdout/stderr, configured and inherited stderr destinations, redirected stdout, approval/decline/default/EOF, human/keys/quiet-keys/no-wait, and prompt suppression for dry-run/auto-confirm/automation. Pipe-only empty stdin tests are complemented by prompt-free missing-selection tests with terminal stdin. Assert no writes before approval or after abort. Include activity/progress behavior under the established modes where relevant; no progress markers may contaminate result stdout.
- Ship US4 guidance with the command: ACTIVE-only eligibility; Camunda 8.10; `SUSPEND_PROCESS_INSTANCE` on `PROCESS_DEFINITION` and extra read permission for default confirmation; no mandatory preflight read; keys-only acceptance meaning; preview/no-wait/default wait; independent relatives; retained runtime state/resources; in-flight workers and external effects; rejected job completion/failure during suspension; possible repeated execution after resumption; backpressure delay/rejection and no instantaneous emergency-stop claim. Update help/examples, README, metadata tests, and run `make docs-content`; do not hand-edit generated docs.
- Replace only direct suspension setup in `integration/scripts/run-c810-suspended.sh` with the contracted `c8 suspend pi --key "$key" --auto-confirm --backoff-timeout 60s --json` invocation and successful confirmed-envelope checks. Preserve the subsequent `expect pi` scenario, configured target behavior, owned-root cleanup, and other scenarios. Remove obsolete suspension-only curl/auth setup only if unused. Run `bash -n integration/scripts/run-c810-suspended.sh`; never source or execute it.
- Run `env GOENV=off GOFLAGS= go test ./cmd -run 'SuspendProcessInstance' -count=1` after confirming matching tests, then affected existing command/metadata/cancel/delete/get/expect regressions. Run `env GOENV=off GOFLAGS= make docs-content`, static Bash syntax and diff checks, and the ordinary `env GOENV=off GOFLAGS= make test` once after targeted checks. Verify Make overrides/targets and absence of live tags first. The full race suite is justified by the integrated concurrent collection and multi-package interface changes; do not rerun it merely for a commit or subsequent prose-only edits. Record actual coverage/platform skips and the fact that real-engine behavior remains unverified.

**Checkpoint**: All command paths, complete failure diagnostics, real-terminal interactions, documentation, and fixture adaptation ship together. No later hardening task is required to satisfy output or permission semantics.

## Phase 6: User Story 4 — Understand Suspension Before Relying on It (P2)

**Goal**: Give operators accurate limits and replace direct suspension fixture setup with the supported command.

**Task ownership**: No separate task. T003 owns help, metadata, README, generated docs, fixture adoption and static checks because they must stay aligned with the command's final behavior.

**Independent test criteria**: Review generated help/README for every permission/scope/wait/operational limitation in FR-018–FR-019; assert metadata and usage examples through command tests; inspect the fixture diff for zero direct suspension requests, preserved existing scenarios, and confirmed-result validation; run `bash -n` only. Updating the fixture is not permission to execute it. Acceptance is part of T003.

## Phase 7: Polish and Cross-Cutting Completion

No standalone polish, test-only, documentation-only, or bookkeeping task is added. Required formatting, compatibility checks, docs, fixture syntax, diagnostic review, progress evidence, and the justified integrated race check are already assigned to the implementation unit that changes the behavior. A known failing required check prevents that task from being marked complete.

## Dependencies and Story Completion

```text
T001 internal submission evidence
  └── T002 complete service/facade workflow (US1 + US2, shared US3 mechanics)
        └── T003 command/output/interaction/docs/fixture (US3 + US4, CLI US1 + US2)
```

T002 cannot expose suspension until its waiter/default behavior is complete. T003 must reuse T002's collections rather than add a CLI pool or a second single-key path. T001 and T002 are internal/library increments; the smallest command MVP is completion of T001–T003. A US1-only CLI release still needs shared US2 truthfulness and US3 output/confirmation safety, so it is not honest to claim US1 is command-complete at T002.

| Story | Primary-labeled tasks | Shared acceptance owners | Completion criterion |
| --- | --- | --- | --- |
| US1 (P1) | 1: T002 | T001, T003 | Exact target, supported capability, authoritative rejection, observed suspension and get/expect compatibility. |
| US2 (P1) | 0; deliberately co-delivered | T001, T002, T003 | No false acceptance/confirmation; retained state/evidence and bounded waiting under all failure cases. |
| US3 (P2) | 1: T003 | T002 | Ordered explicit collections, safe preview/prompt/automation, partial results, one envelope, correct streams/logging/exits. |
| US4 (P2) | 0; deliberately co-delivered | T003 | Complete operational guidance and c8volt-based fixture setup with static verification. |
| Foundation | 1: T001 | All stories | Tested protocol/submission evidence primitive without premature public exposure. |

**Total**: 3 tasks; 3 completed; 0 in progress; no pre-existing task IDs/evidence were replaced.

## Parallel Opportunities and Examples by Story

No implementation task is marked `[P]`: each depends on the preceding operation contract, and related tasks share adapter/contracts/result code. Do not manufacture tasks to enable hypothetical parallel work.

- **US1 example**: After T002 implementation is stable, its independent fake-server version/target cases can execute in parallel only when they do not mutate shared flags/config/test globals. This is test execution within T002, not an independently schedulable task.
- **US2 example**: Deterministic waiter subtests with independent fake readers/clocks may run independently of facade-mapping subtests after both exist. Keep submission evidence and waiter composition changes in the same T002 unit.
- **US3 example**: Review stdout envelope assertions and PTY assertions independently after the CLI contract is fixed, but run shared Cobra/global-state cases without `t.Parallel()` unless isolated by a subprocess. T003 remains one owner of runner, views, diagnostics, and docs.
- **US4 example**: Static review of the fixture diff and generated-doc review can occur independently after T003 generates them. Neither review permits live execution or a separate incomplete fixture task.

## Requirement and Acceptance Traceability

| Requirement | Implementation and validation owners |
| --- | --- |
| FR-001 | T003 command grammar/alias/stdin execution tests. |
| FR-002 | T002 service normalization and T003 flag/stdin/invalid-input order tests. |
| FR-003 | T002 all-version service/facade guards; T003 CLI dry-run/version checks. |
| FR-004 | T001 exact wire target; T002/T003 parent-only, child-only, both, no hierarchy requests. |
| FR-005 | T001 backend rejection classification; T002/T003 preserve rejection without state reconciliation. |
| FR-006 | T001 response/retry/pacing evidence; T002/T003 uncertainty preservation. |
| FR-007 | T002 exact suspension waiter; T003 confirmed/unconfirmed command results. |
| FR-008 | T002 terminal/read/deadline/interruption state machine; T003 retained partial payload and exit. |
| FR-009 | T001 established submission retry policy; T002 no-wait/no resubmission; T003 accepted rendering. |
| FR-010 | T002 no-I/O preview; T003 all preview modes/no-wait/no-prompt checks. |
| FR-011 | T003 real-terminal confirmation/abort/EOF and unattended-mode coverage. |
| FR-012 | T002 worker/fail-fast/unscheduled outcomes and facade preservation; T003 partial-result rendering. |
| FR-013 | T003 one envelope with EOF, machine evidence, no duplicate error renderer, original exits. |
| FR-014 | T003 exact human/keys/JSON streams, quiet precedence, configured logger and plain prompts. |
| FR-015 | T001/T002 safe service diagnostics; T003 repeated-error/logger/DEBUG/redaction regression matrix. |
| FR-016 | T003 empty stdin and missing input, no mutation/prompt/acceptance/rendering-side requests. |
| FR-017 | T002 state/date and get/expect regressions; T003 isolated CLI inspection/expectation compatibility. |
| FR-018 | T003 help/README/metadata/generated-doc permission/version/eligibility coverage. |
| FR-019 | T003 documented scope/resources/workers/effects/reexecution/backpressure limitations. |
| FR-020 | T003 fixture suspension replacement, result assertion, static syntax check only. |
| FR-021 | T002/T003 cancellation/forced-deletion regression coverage and explicit scope exclusions. |
| SC-001 | T001–T003 false-acceptance/false-confirmation matrices. |
| SC-002 | T001–T003 exact-target/request-count/hierarchy invariants. |
| SC-003 | T002/T003 capability/input/preview/abort no-mutation and no-prompt evidence. |
| SC-004 | T002 bounded waiter/no-wait/no-resubmission; T003 observed outcomes. |
| SC-005 | T002 collection/facade evidence ordering; T003 partial payload/exit preservation. |
| SC-006 | T003 exact result streams/envelope/keys and prompt separation. |
| SC-007 | T003 configured logger severity/destination/normal-verbose-quiet/DEBUG/redaction checks. |
| SC-008 | T002/T003 compatibility regressions; T003 docs/fixture checks. |

## Implementation Strategy

First prove the private submission protocol without prematurely exposing a public operation. Then deliver the complete service/facade behavior, including all single/collection/confirmation safety in one unit. Finally expose it once through the CLI, with all output, prompt, logging, docs, and fixture obligations in the same delivery. Tests are introduced with their behavior and must pass before the corresponding checkbox is marked complete. Use [quickstart.md](quickstart.md) for detailed isolated validation and evidence requirements.
