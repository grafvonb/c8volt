# Tasks: Advisory Camunda Version Check

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli-version-check.md), and [quickstart.md](quickstart.md).

**Branch**: `222-camunda-version-check` · **Issue**: #222

**Organization**: Three sequential work units, owned by US1, US2, and US3 respectively. Story labels identify the primary acceptance concern, not separate implementations of the check. Shared comparison and failure policy are built once in T001, connected to every operational client in T002, and activated with the complete CLI contract in T003. Each unit compiles and includes its own relevant tests; full user-facing acceptance for all three stories requires T003.

**Tests**: Required by the specification. Write meaningful acceptance/regression assertions before the behavior they verify, observe the relevant failure when practical, and complete fixtures, implementation, formatting, and targeted checks within the same unit. No separate test-only task or deferred failing test is permitted.

## Format and execution rules

- Task rows use `- [ ] Tnnn [USn]` and concrete repository-relative paths. Detail and acceptance clauses beneath a row are part of that task.
- No task is marked `[P]`: dependencies and shared request/command state require sequential implementation.
- Read applicable project context and `specs/ralph-implementation-rules.md`, preserve user changes, review the output boundary, and update task/progress evidence as responsibilities of each work unit, not separate tasks. Do not cross into the next story phase within a Ralph iteration.
- Use only isolated fakes, mocks, and test-owned HTTP/subprocess/terminal fixtures. Before compiling tests, sanitize inherited configuration, helper-process variables, GOFLAGS/build tags, and make overrides as described in [quickstart.md](quickstart.md). Never run integration targets/tags, inspect clusters, select a live target, or perform live smoke checks.
- Proposed test names below become real tests during implementation. Use `go test <package> -list '<pattern>'` to verify selectors; zero matched tests do not count as passing coverage. Preserve existing test-name conventions where appropriate and update recorded commands to the actual names.
- Run `gofmt` on touched Go files and document checks actually performed. New or materially modified functions/types/tests require the comments prescribed by the Ralph rules. Use Conventional Commits ending in ` #222` when the implementation workflow reaches an authorized commit step.

## Phase 1: Setup (Shared Infrastructure)

The repository, dependencies, branch, and feature artifacts already exist. No setup task is needed. Context reading, branch checks, environment isolation, and progress updates remain work-unit responsibilities.

## Phase 2: Foundational (Blocking Prerequisites)

No standalone foundation task is needed. T001 owns its model, parser, gate contract, bypass, wrapper, checker, and required doubles together so it does not depend on an unimplemented interface from a later unit. Existing cluster topology, HTTP controls, and test helpers are reused.

## Phase 3: User Story 1 — Notice a Version Mismatch Without Interrupting Work (Priority: P1)

**Goal**: Deliver one complete, independently testable internal advisory checker that compares gateway release lines, completes once per invocation, and retains configured behavior on mismatch or failure.

**Independent test**: Invoke the checker and pre-Do wrapper with test-owned topology responses and recording callbacks. Prove a silent match, one mismatch outcome, one bounded attempt across concurrent/repeated callers, and safe continuation after every failure. The CLI is deliberately not activated until T003.

- [x] T001 [US1] Implement the complete bounded version-check primitive in `toolx/camunda_release.go`, `internal/domain/version_check.go`, `internal/services/httpc/before_request.go`, and `internal/services/cluster/version_check.go`, with `toolx/camunda_release_test.go`, `internal/services/httpc/before_request_test.go`, and `internal/services/cluster/version_check_test.go`; include all model/gate dependencies, failure policy shared with US2, fixtures, comments, formatting, and the focused validation below in this unit.

### T001 model constraints and implementation

Use the following field constraints from the data model verbatim:

| Field | Required constraint |
| --- | --- |
| Major | "Nonnegative decimal integer; reject parse overflow." |
| Minor | "Nonnegative decimal integer; numeric equality defines compatibility with Major." |
| Display | "Trimmed validated observed release, at most 128 bytes; safe ASCII only, no control characters." |
| Status | "`matched`, `mismatched`, or `failed`. Disabled/not-needed is represented by absence of an attempted outcome." |
| ConfiguredVersion | "Effective normalized configured release line; immutable for the invocation." |
| ObservedVersion | "Safe validated display value for a successful comparison; absent on invalid data." |
| FailureReason | "Fixed enum: `access_denied`, `timeout`, `invalid_version`, `unavailable`, or `canceled`; present only for failed checks." |
| WarningNeeded | "Derived from mismatched/failed status, independent of logger visibility. Used for explicit connection-test deduplication." |

1. Accept optional `v`/`V`, numeric major/minor, optional numeric patch, conventional nonempty dot-separated ASCII alphanumeric/hyphen prerelease/build identifiers, and surrounding spaces. Reject controls, negative or overflowing numbers, empty components, extra numeric components, malformed/overlong values. Compare numeric major/minor only: patch, prerelease, build metadata, and broker versions cannot cause a mismatch. Do not replace `cmd/config_diagnostics.go`'s legacy machine-warning parser/classification.
2. Define the minimal context gate and bypass contract in `httpc` with the pre-Do wrapper in this unit. Keep `httpc` independent of cluster imports. With no gate or a bypass marker, preserve direct delegation. Close an unconsumed body on a canceled early return; do not change the delegated client's redirect, body, or timeout behavior.
3. Own `idle → checking → complete(outcome)` in the cluster checker using a short lock plus completion channel, without holding locks across network I/O or the callback. One winner probes synchronously; waiters can cancel independently. Publish immutable outcome after the callback and before completion notification; preserve failed outcomes for later calls, and never serialize subsequent operational requests. Pre-canceled callers make no request; winner cancellation consumes the attempt. New checker instances have fresh state.
4. Construct no probe during initialization. Lazily use the existing neutral cluster factory/topology API and effective configured adapter. Turn probe-construction errors into failure outcomes, not bootstrap errors. Probe with a shallow HTTP-client copy sharing transport/jar/auth/pacing/diagnostics, refusing redirects only on the copy, and with a child context capped at two seconds, any shorter positive configured HTTP timeout, and the caller's earlier deadline. Set bypass and `httpc.WithoutReadRetry`, create a fresh `httpc.BeginRequestRateAttempt` observation, and pass no verbose workflow options. Do not alter shared client settings or cancel/reuse the original request context.
5. Derive reasons from typed domain/context/network causes: unauthorized/forbidden → access denied; timeout/deadline → timeout; malformed response or missing/unsafe version → invalid gateway version; caller cancellation → canceled; remaining failures/redirects → unavailable. Outcome/callback data must contain no raw errors, URLs, bodies, keys, or credentials. Only original caller cancellation may prevent operational delegation; mismatch/probe failure cannot become an operational error.

### T001 acceptance and validation

- Cover numeric 8.9 versus 8.10, major mismatch, 8.10.0-alpha4, patch/build matches, missing/malformed/nil response, denied access, timeout, generic transport/backend error, and unsafe/128-byte boundary values. Assert exact typed outcomes and callback counts, not just strings.
- Use deterministic synchronization to prove concurrent one-attempt behavior, cancellation while waiting, repeated calls after failure, bypass without recursion, completion ordering, local no-request construction, and original body/context handling.
- Exercise no retry for transient/429/503 responses, no redirect following, no global-state leak, and a shorter request budget with an otherwise active parent. Check the two-second cap through controlled deadlines rather than routinely waiting two seconds per case. No public facade or generated client modification is needed.
- Validate with `go test ./toolx -run 'TestCamundaRelease' -count=1`, `go test ./internal/services/cluster/... -run 'TestVersionCheck' -count=1`, and `go test ./internal/services/httpc/... -run 'TestVersionCheck' -race -count=1`. Internal APIs are not CLI-visible yet, so no README/generated CLI change is required for this unit.

**Checkpoint**: The complete advisory primitive, including failure behavior needed by US2, compiles and passes its isolated checks. This is an internal increment, not a released CLI feature.

## Phase 4: User Story 2 — Continue When the Check Is Unavailable (Priority: P1)

**Goal**: Make every generated operational client use the completed checker at the correct boundary, proving that probe failures cannot consume or corrupt the operation's timeout, context, retry policy, or submission evidence.

**Independent test**: Construct real versioned services/facades with a gate-bearing context and test-owned HTTP transport/server. Let a short-budget probe fail, then require the ordinary operation to receive its own timeout and retain its response/exit-class evidence. Run the same service without a gate and verify unchanged behavior.

- [x] T002 [US2] Connect the completed pre-Do wrapper through an additive `APIClient` field in `internal/services/common/deps.go`, migrate all generated-client arguments in the 33 exact `service.go` files listed below, and add `internal/services/common/deps_test.go` plus `internal/services/cluster/version_check_wiring_test.go`; retain concrete client/facade signatures and nested constructors, update affected local doubles, and include focused adapter/timeout/auth/pacing/recursion tests, formatting, and validation in this unit (depends on T001).

### T002 constructor scope

| Area | Exact files |
| --- | --- |
| Batch operation | `internal/services/batchoperation/v88/service.go`, `internal/services/batchoperation/v89/service.go`, `internal/services/batchoperation/v810/service.go` |
| Cluster | `internal/services/cluster/v88/service.go`, `internal/services/cluster/v89/service.go`, `internal/services/cluster/v810/service.go` |
| Element | `internal/services/element/v88/service.go`, `internal/services/element/v89/service.go`, `internal/services/element/v810/service.go` |
| Incident | `internal/services/incident/v88/service.go`, `internal/services/incident/v89/service.go`, `internal/services/incident/v810/service.go` |
| Job | `internal/services/job/v88/service.go`, `internal/services/job/v89/service.go`, `internal/services/job/v810/service.go` |
| Process definition | `internal/services/processdefinition/v88/service.go`, `internal/services/processdefinition/v89/service.go`, `internal/services/processdefinition/v810/service.go` |
| Process instance | `internal/services/processinstance/v88/service.go`, `internal/services/processinstance/v89/service.go`, `internal/services/processinstance/v810/service.go` |
| Resource | `internal/services/resource/v88/service.go`, `internal/services/resource/v89/service.go`, `internal/services/resource/v810/service.go` |
| Tenant | `internal/services/tenant/v88/service.go`, `internal/services/tenant/v89/service.go`, `internal/services/tenant/v810/service.go` |
| User task | `internal/services/usertask/v88/service.go`, `internal/services/usertask/v89/service.go`, `internal/services/usertask/v810/service.go` |
| Variable | `internal/services/variable/v88/service.go`, `internal/services/variable/v89/service.go`, `internal/services/variable/v810/service.go` |

### T002 implementation and acceptance

1. Preserve these model constraints: APIClient is an "Additive internal Do interface in ServiceDeps; consults gate before delegating to HTTPClient." The operational request "Retains its original context and original concrete HTTP client." The probe client is a "Shallow copy of HTTPClient sharing transport and cookie jar; advisory redirect refusal only. Never changes shared client settings."
2. Retain `ServiceDeps.HTTPClient` as `*http.Client`; add the wrapper capability rather than changing public `c8volt.WithHTTPClient` or factory signatures. Change the 36 generated `WithHTTPClient(deps.HTTPClient)` uses to `deps.APIClient`, including v88 Operate and v88/v89 Tasklist. Keep the three nested process-instance→variable service arguments as the original concrete client. Include interface/consumer/double changes together; no generated code editing.
3. Verify all service areas use the wrapper, including legacy and nested-client routes; a table-driven constructor/wiring fixture may cover mechanical cases without duplicating full behavioral matrices. Reuse existing adapter tests and add behavioral coverage for each configured version. Keep invalid/unsupported configuration rejection unchanged and instantiate zero extra version requests without an invocation gate.
4. With the actual shared HTTP service, assert auth/bootstrap/refresh calls are outside the operational wrapper; the probe shares auth/cookies, pacing, activity, and DEBUG diagnostics. An internal test may install an outcome recorder instead of a CLI renderer; production CLI activation belongs to T003.
5. Regression-test a delayed probe with a configured HTTP timeout shorter than two seconds followed by a fast successful operation: the operation receives its own full normal request timeout, bounded only by the existing parent. Verify an expired/canceled parent prevents dispatch, concurrent waiters cancel promptly, request bodies are not consumed by preflight, and existing operational body/redirect/read-retry semantics remain intact.
6. Prove recursive topology uses bypass; only the probe has retry disabled and redirect refusal. Fresh probe pacing observation must not mark an otherwise sent mutation `not_submitted` or erase earlier ambiguous submission evidence. Count actual shared limiter/diagnostic starts independently of logical attempts; a locally unadmitted probe may have no network start. Later requests, pages, batches, nested services, and polling reuse completed state.
7. Validate `go test ./internal/services/common/... -run 'TestPrepareServiceDeps' -count=1`, `go test ./internal/services/cluster/... -run 'TestVersionCheck' -race -count=1`, and `go test ./internal/services/httpc/... -run 'Test(VersionCheck|ReadRetry|RequestRate)' -race -count=1`, then `go test ./internal/services/... ./c8volt/... -count=1` for the constructor-wide compile/regression impact. CLI defaults are not changed yet; documentation remains with activation in T003.

**Checkpoint**: All operational adapters participate when given a gate, and ordinary standalone callers remain unchanged. US2's failure isolation is proven through actual service/client wiring; command-level evidence is completed in T003.

## Phase 5: User Story 3 — Control the Check and Preserve Script Output (Priority: P2)

**Goal**: Activate the single check for real CLI invocations with opt-out, correct configured WARN diagnostics, unchanged machine results/exit behavior, and complete operator guidance.

**Independent test**: Execute the real command paths against isolated fakes with separate output streams and configured/inherited diagnostic writers. Exercise match/failure, opt-out/local execution, repeated work, short timeout, partial failure, and explicit connection tests; verify results, request order/counts, warnings, and exits under the specified output/logging modes.

- [x] T003 [US3] Activate the checker and complete the operator contract in `cmd/root.go`, `cmd/cmd_flags.go`, `cmd/root_services.go`, new `cmd/root_version_check.go` and `cmd/cmd_views_version_check.go`, and `cmd/config_test_connection.go`/`cmd/config_diagnostics.go`; add command execution/logging tests in `cmd/root_version_check_test.go` and `cmd/root_version_check_logging_test.go`, update affected existing fixtures and metadata tests, update `README.md` and command help, regenerate CLI docs, and complete the full acceptance matrix and final validation below in the same unit (depends on T002).

### T003 implementation and compatibility

1. Add persistent `--no-version-check` default false, with normal placement and explicit true/false behavior. No new persisted configuration key. Keep focused flag translation/checker installation/outcome access outside root.go; install a fresh gate only after existing HTTP/auth initialization, using the configured command-context logger. Clear/reset the new flag in existing test cleanup without globally disabling default checking. Local validation, help, config-only execution, local preview, and pre-operation abort must not cause a version request.
2. Render only typed bounded outcome facts through the existing configured WARN logger; do not use direct print, a separate/global logger, extra severity/timestamp text, raw error formatting, or result renderers. Preserve match silence. Exact source examples: `version mismatch: configured 8.9, gateway 8.10.3; using 8.9`, `version check: access denied; using configured 8.9`, and `version check: timeout; using configured 8.9`. Other reasons follow the fixed [contract table](contracts/cli-version-check.md). Respect quiet/ERROR suppression, WARN visibility, format, destination, inheritance, activity routing, and existing quiet/debug precedence. Neither verbosity nor output mode independently disables the check.
3. Preserve `config test-connection`'s real topology operation and legacy actual-result JSON warnings/parser behavior, including historically permissive `8.9.garbage`. Suppress its extra human compatibility warning only when the automatic checker produced a warning outcome, independent of log filtering. A matching advisory followed by a changed actual topology may still warn. With opt-out, explicit connection compatibility evaluation remains available. Safely bound touched human fallback messages without changing machine warning evidence. Explicit `get cluster version`/`get cluster topology` still perform their requested read separately.
4. Keep results, prompts, existing failure conversion, partial submission evidence, and exits unchanged. Trace failed probe → typed advisory → WARN, then ordinary service failure → facade conversion → rendering/logger → original exit. Advisory failure must never trigger `ferrors`, replace an operation error, or produce an extra envelope. Do not change command schemas to solve human formatting.
5. Update strict handlers/counters in `cmd/root_api_diagnostics_test.go`, `cmd/root_request_rate_test.go`, `cmd/config_test.go`, `cmd/request_rate_mutation_test.go`, and affected batch/terminal fixtures to account for a matching topology probe and its rate/DEBUG start. Explicit opt-out is allowed only in narrowly unrelated fixtures; never hide default-on behavior via a shared reset/helper switch. Reuse test-owned environment isolation and `testx` helpers.
6. Update root and connection-test help/metadata where applicable, `cmd/command_contract_test.go`, `cmd/command_help_focus_test.go`, and README guidance. Explain default-on, major/minor comparison, two-second/shorter bound, shared pacing consequence, warning-only failures, opt-out, explicit connection-test distinction, and no automatic adapter selection. Run `make docs-content`; do not hand-edit `docs/cli/*` or `docs/index.md`.

### T003 acceptance matrix

| Execution group | Required assertions |
| --- | --- |
| Matching/mismatch/failure | Match is silent; major/minor mismatch, 401/403, timeout, malformed/nil/unsafe/overlong version, redirect/unavailable/transport failure yield one exact bounded automatic WARN when enabled. Compare enabled/disabled executions against the same operational responses and selected adapter. Cover 8.8/8.9/8.10 without repeating every logging combination for every version. |
| Invocation lifetime | First operational request follows the completed probe/callback; multiple facade constructions, pages, concurrent resources, retries and polling share one check, including after failure; separate in-process and subprocess invocations start fresh. No check during auth bootstrap/refresh; caller cancellation never dispatches new work. |
| Disabled/local/no-op | Omitted/true/false opt-out and flag placement; zero checks for help, capabilities, local version/config inspection, local invalid input, local dry-run/empty scopes, and declined local-only mutations. Discovery previews and completed remote discovery returning empty results check exactly once. Preserve sparse-page continuation and explicit-key, abort, and error behavior. |
| Output combinations | Human, JSON, keys-only, quiet, quiet+JSON, quiet+keys, dry-run, and supported auto-confirm/automation/no-wait combinations preserve existing precedence and results. JSON has exactly one existing envelope with EOF, including empty/partial failures; keys are exact or zero bytes. No added stdout summary. |
| Logger boundary | Execute with configured and inherited destinations, active plain and structured formats, normal/verbose-only/WARN/ERROR/quiet settings. Assert complete record severity and message, counts, destination, and no diagnostic stdout; a direct stderr substitution must fail. No raw URL/method/body/key/credential/chain on either stream at normal or verbose-only level. |
| DEBUG/redaction | Exercise `--debug` and configured DEBUG separately, with and without verbose, respecting quiet precedence; reuse existing redacted transport observation and assert sensitive markers absent. Do not replace existing DEBUG machinery or expose raw error strings as an additional diagnostic. |
| Multi-resource failure | Failed advisory followed by repeated distinctive backend failures and one success: one advisory WARN plus one existing compact aggregate ERROR; original nonzero exit and partial/submission evidence; one JSON envelope+EOF; exact successful keys or empty keys. Quiet suppresses WARN and INFO but retains compact ERROR. Existing successful mutation/dry-run/verbose INFO stays on its configured logger and respects WARN/quiet suppression. |
| Timeout/pacing | Short probe timeout leaves operation's own request budget usable; earlier parent deadline prevents dispatch. Probe's fresh admission observation never changes real mutation evidence; auth and probe starts count in shared pacing/reporting correctly. |
| Explicit connection test | Actual JSON warning evidence unchanged, duplicate human warning suppressed, changed second observation still diagnosed, opt-out preserves requested version/topology/compatibility results, and unsafe human fallback remains bounded. |
| Interactive regression | Real terminal stdin with separately captured stdout/stderr verifies eligible local abort and empty-scope completion, configured/inherited prompt destinations, redirected stdout, exact existing prompt wording/eligibility, no new confirmation, and zero accidental mutations. Pipe-only tests are insufficient. |

### T003 validation and completion

Run `gofmt` on touched Go files, then the narrowest relevant checks in the sanitized environment from [quickstart.md](quickstart.md):

```sh
go test ./cmd -run 'TestVersionCheck' -count=1
go test ./cmd -run 'Test(APIDiagnostics|RequestRate|ConfigTestConnection|BulkCompletionDiagnosticLogging|CapabilityDocument|AllCommandHelp)' -count=1
go test ./internal/services/httpc/... -run 'Test(VersionCheck|ReadRetry|APIDiagnostics|RequestRate)' -race -count=1
make docs-content
make test PKG=./...
git diff --check
```

Select the actual relevant terminal/mutation regression test names when the existing names differ from these families; record those selections. The full race suite is justified by shared HTTP wiring and concurrent command/service consumers, not by documentation edits. Fix failures within this unit; do not label unrun tests passed or rerun already successful checks solely to commit. Record real-engine coverage as unverified; no live-environment gate is added.

**Checkpoint**: US1, US2, and US3 all satisfy their command-level acceptance scenarios, and documentation reflects the shipped default/opt-out behavior.

## Phase 6: Polish & Cross-Cutting Concerns

No separate polish task. Formatting, comments, fixtures, regression checks, help/README updates, generated docs, and output-boundary review are part of their owning units above. T003 owns the final combined race/contract evidence; routine progress bookkeeping is not a separate outcome.

## Dependencies & Execution Order

```text
T001 [US1: complete checker] → T002 [US2: safe shared wiring] → T003 [US3: CLI contract]
```

The earlier phases provide independently verifiable internal increments. A story's internal mechanism passing does not establish its CLI acceptance: T003 is the completion owner for all command-execution scenarios. No temporary CLI enablement or partial default is introduced before that point.

| Story | Primary task count | Implementation ownership | Full acceptance prerequisites |
| --- | --- | --- | --- |
| US1 (P1) | 1 | T001 comparison/once-only state; T002 connects all adapters; T003 warnings/default | T001–T003 |
| US2 (P1) | 1 | T001 complete failure policy; T002 proves operational isolation; T003 failure/result routing | T001–T003 |
| US3 (P2) | 1 | T003 opt-out, local eligibility, diagnostics, scripts, explicit connection compatibility, docs | T001–T003 |

## Parallel Opportunities and Per-Story Examples

There are no independent implementation tasks to run concurrently. Do not create adapter-per-version tasks or duplicate checker/flag implementations to manufacture parallelism.

- **US1 example**: After T001's code is complete, pure parser cases with private data may run concurrently with independent checker fixtures; run the documented toolx and cluster checks independently. Gate concurrency is tested within a test-owned fixture, not by parallel edits to its state machine.
- **US2 example**: After wiring compiles, independent versioned-service cases with separate servers may use parallel subtests where existing helpers allow it. Migration remains one unit because `PrepareServiceDeps` and all consumers must agree; no adapter is an independently shippable feature.
- **US3 example**: Do not parallelize in-process Cobra tests that mutate global flags, command trees, or environment. Independent child-process fixtures with private configs/writers may run concurrently if the harness guarantees isolation. This is validation concurrency, not a `[P]` implementation task.

## Implementation Strategy and Requirement Coverage

**MVP**: Complete T001–T003. A mismatch-only CLI MVP would omit the required fail-open, opt-out, and script-safety guarantees; do not ship an intermediate default-on command. Earlier internal checkpoints remain useful for reviewing behavior and local regressions without changing CLI defaults.

| Requirement / success criterion | Primary implementation and acceptance owners |
| --- | --- |
| FR-001, FR-007; SC-001 | T001 single-attempt state; T002 all generated clients; T003 invocation/pagination/batch/polling tests. |
| FR-002, FR-013, FR-014; SC-002 | T001 bounded numeric parsing/classification; T002 supported adapters; T003 exact warnings and configuration rejection regression. |
| FR-003, FR-004, FR-010, FR-012; SC-006 | T001 typed safe outcome; T003 exact WARN routing, filtering, both DEBUG paths, redaction and duplicate prevention. |
| FR-005, FR-006; SC-003, SC-004 | T001 bounded no-retry/no-redirect probe; T002 timeout/auth/pacing/submission isolation; T003 operation result/exit comparison and cancellation. |
| FR-008, FR-009; SC-001, SC-005 | T003 flag/local/discovery/empty/interactive execution; T002 absent-gate pass-through. |
| FR-011; SC-004, SC-005 | T003 one-envelope/EOF, exact keys, partial outcomes/original exits, quiet combinations and existing prompt/INFO routing. |
| FR-015 | T003 source metadata/help, README and `make docs-content` output. |

All task markers start unchecked. Completing this task-generation workflow does not mark implementation or test execution complete.
