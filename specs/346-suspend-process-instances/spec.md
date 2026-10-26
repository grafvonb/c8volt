# Feature Specification: Suspend Process Instances

**Feature Branch**: `346-suspend-process-instances`

**Created**: 2026-10-25

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/346 — feat(process): add process-instance suspension commands"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Suspend exactly the selected instance (Priority: P1)

As an operator, I want to pause an active process instance by its key, including a child instance, so I can control that instance without canceling it or changing its relatives.

**Why this priority**: Exact targeting and truthful confirmation are the core value of suspension and prevent unintended operational impact.

**Independent Test**: With isolated instance and submission responses, suspend an active target, confirm its suspended state, and verify that only the supplied target receives a suspension request. Exercise the existing inspection and expectation commands against the same simulated state.

**Acceptance Scenarios**:

1. **Given** an active instance on configured Camunda 8.10 and sufficient suspension/read permissions, **When** the operator runs `c8volt suspend process-instance --key <key>` and approves any required confirmation, **Then** the request targets that key and success is confirmed only after a successful submission and an observation of `SUSPENDED` for that instance.
2. **Given** a selected child with an active parent, root, siblings, and descendants, **When** suspension succeeds, **Then** only the selected child is targeted; there is no root escalation, relative suspension, or hierarchy discovery.
3. **Given** a selected parent with separately running children, **When** suspension succeeds, **Then** the command does not suspend those children or describe them as suspended.
4. **Given** configured Camunda 8.8 or 8.9, **When** suspension is requested through the CLI or public library, including dry-run, **Then** the established unsupported-capability error is returned before any unsupported request.
5. **Given** an already-suspended, terminal, terminating, missing, or unauthorized target, **When** its suspension request is rejected, **Then** the rejection remains a failure even if a subsequent observation reports `SUSPENDED`.
6. **Given** successful suspension, **When** the operator uses existing `get pi` and `expect pi` commands, **Then** they can observe or verify suspension using the existing state and suspension-date behavior.

---

### User Story 2 - Distinguish submission from observed suspension (Priority: P1)

As an operator or automation author, I want to distinguish accepted suspension from observed suspension and from an uncertain outcome, so I can make safe follow-up decisions when state changes or observation fails.

**Why this priority**: Reporting suspension without evidence could lead operators to assume work has stopped when it has not.

**Independent Test**: Supply controlled submission and observation sequences, including delayed suspension, terminal transitions, read errors, and interruption. Verify the reported acceptance evidence, last observation, confirmation status, and bounded completion.

**Acceptance Scenarios**:

1. **Given** an accepted submission followed by `ACTIVE` and then `SUSPENDED`, **When** default waiting completes within its configured bound, **Then** suspension is confirmed with no mutation resubmission caused by polling.
2. **Given** an accepted submission, **When** `c8volt suspend pi --key <key> --no-wait` is used, **Then** the result reports acceptance without claiming observed suspension and performs no post-submission confirmation reads.
3. **Given** a preflight observation of `ACTIVE`, **When** the state changes and submission is rejected, **Then** the command reports rejection rather than acceptance or confirmation, regardless of later state.
4. **Given** an accepted submission, **When** a terminal state is observed before `SUSPENDED`, **Then** waiting stops for that target and reports the observed terminal outcome without claiming confirmed suspension.
5. **Given** an accepted submission, **When** observations remain `ACTIVE`, `ABSENT`, or `UNKNOWN`, contain only a suspension timestamp, or fail to read the instance, **Then** none of those observations confirms suspension and waiting cannot exceed its configured bound.
6. **Given** an accepted submission, **When** waiting times out, is interrupted, or fails because read authorization is denied, **Then** the result retains acceptance evidence and the last observed state, or explicitly records that no state was observed, while reporting the unconfirmed failure.
7. **Given** a transport failure without a successful submission response, **When** an observation happens to show `SUSPENDED`, **Then** the command does not claim confirmed acceptance or successful suspension attributable to its request; ambiguity is preserved.

---

### User Story 3 - Suspend explicit collections safely in scripts (Priority: P2)

As an automation author, I want to provide several explicit keys, preview the operation, and retain individual outcomes even when some targets fail, so I can handle suspension without direct suspension requests outside c8volt.

**Why this priority**: Collection input and reliable output make the command usable in established operator pipelines and fixture setup.

**Independent Test**: Exercise repeated/comma-separated key flags and newline-separated stdin, mixed outcomes, duplicate keys, empty input, previews, and interactive/unattended execution with isolated dependencies and separately captured output streams.

**Acceptance Scenarios**:

1. **Given** a parent and child are both explicitly supplied, including duplicate appearances across flags and stdin, **When** suspension runs, **Then** each unique selected key remains a distinct target and has its own outcome; neither target is collapsed into a root and no hierarchy discovery occurs.
2. **Given** multiple selected keys with confirmed, rejected, and accepted-but-unconfirmed outcomes, **When** execution finishes with a failure, **Then** partial results and per-key evidence remain available and the established failure class and exit behavior are preserved.
3. **Given** valid explicit keys on a supported version, **When** `--dry-run` is used, including with `--no-wait`, **Then** the result is a preview of only those targets, with zero mutations, zero mutation-confirmation prompts, and no accepted-submission claim.
4. **Given** empty or whitespace-only stdin supplied through `-`, missing explicit selection, or malformed keys, **When** the command runs, **Then** it follows existing input-error conventions without mutation requests, confirmation prompts, search fallback, or accepted-submission claims.
5. **Given** interactive confirmation is eligible, **When** approval is requested or declined, **Then** the plain prompt uses configured stderr, selected-format results remain on stdout, and decline submits no mutations; supported auto-confirm and automation modes preserve existing unattended behavior.
6. **Given** a partially failing collection and JSON output, **When** the result is emitted, **Then** stdout contains exactly one shared envelope retaining partial results and error evidence, with no second envelope or diagnostic text appended.
7. **Given** repeated failures containing raw request details, **When** normal, verbose, or quiet output is selected, **Then** a compact aggregate error is emitted through the configured logger with counts and a bounded useful reason; raw request details do not appear on either stream. DEBUG diagnostics remain available independently of verbose and follow existing redaction rules.
8. **Given** keys-only or quiet output, including supported combinations with JSON, **When** the command produces results or an error, **Then** existing output precedence applies, keys-only results contain one qualifying key per line and zero bytes when none qualify, and quiet suppresses informational summaries without suppressing required failures or explicitly requested machine output.

---

### User Story 4 - Understand suspension before relying on it (Priority: P2)

As an operator, I want help and examples to explain eligibility, permissions, scope, and practical limitations, so I do not mistake suspension for cancellation or an immediate stop of external work.

**Why this priority**: Suspension can leave workers and external effects in flight; accurate guidance is necessary for safe operational use.

**Independent Test**: Review command help, README guidance, generated command documentation, and the existing Bash suspension fixture against the documented acceptance and scope rules, without executing live fixtures.

**Acceptance Scenarios**:

1. **Given** an operator consulting help or documentation, **When** they review suspension, **Then** they can identify the required version, active-state eligibility, suspension permission, additional read permission for default confirmation, explicit-key input forms, preview, and wait/no-wait behavior.
2. **Given** an operator considering suspension during an incident, **When** they review its limitations, **Then** guidance explains retained runtime state, independent parent/child behavior, already-running workers and external side effects, rejection of job completion/failure during suspension, possible repeated job execution after resumption, and backpressure-related delays or rejection.
3. **Given** the existing Bash suspension fixture, **When** its setup is reviewed, **Then** suspension setup uses the new c8volt command instead of direct suspension requests, without making execution of that fixture an implementation validation gate.

### Edge Cases

- A selected parent and selected child must both survive deduplication; only identical keys may be combined.
- Input validation failures and empty stdin are errors, not evidence that a suspension operation succeeded or had no work.
- A preflight state may become stale before submission; a prior `ACTIVE` or `SUSPENDED` observation cannot override the authoritative submission response.
- A rejection followed by `SUSPENDED` must remain a rejection; a transport failure followed by `SUSPENDED` must not erase submission uncertainty.
- A successful submission followed by a terminal observation is accepted but not confirmed as suspended; the terminal observation ends that target's wait.
- `ABSENT`, `UNKNOWN`, a timestamp alone, or a failed read must not be substituted for observed `SUSPENDED`.
- Suspension permission and read permission may differ; a read failure after acceptance must retain evidence that submission succeeded.
- Interruption may occur after some targets were accepted and before others were submitted. Preserve known per-key outcomes and distinguish unsubmitted work from accepted work.
- A dry-run combined with no-wait remains a preview. Aborts, rejected requests, and malformed input never become accepted results.
- Quiet, verbose, JSON, keys-only, and DEBUG combinations must preserve existing precedence and separate results, prompts, and diagnostics.
- Backpressure and ambiguous transport outcomes must follow established retry behavior without confirmation polling triggering additional suspension submissions.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provide `c8volt suspend process-instance` with alias `suspend pi`, accepting explicit `--key`/`-k` selections using established repeated/comma-separated forms and newline-separated stdin through `-`.
- **FR-002**: Merge and deduplicate keys using existing conventions, retaining first-input order for reporting. Distinct parent and child keys MUST remain distinct selected targets. Missing, malformed, and empty stdin input MUST retain established invalid-input behavior without search fallback.
- **FR-003**: Support suspension for configured Camunda 8.10. CLI and public-library entry points MUST reject configured 8.8/8.9 through the established capability error, including dry-run, before unsupported requests.
- **FR-004**: Target only the supplied instance keys. Do not escalate to a root, discover a hierarchy, collapse related selections, traverse a tree, or automatically suspend relatives.
- **FR-005**: Only `ACTIVE` instances are eligible. Preserve backend-consistent rejection outcomes for already-suspended, terminal, terminating, missing, and unauthorized instances; do not treat an already-suspended rejection as idempotent success.
- **FR-006**: Treat any preflight observation as advisory. Only a successful mutation response establishes accepted submission. Preserve uncertainty for transport failures without successful responses and do not rewrite a rejected mutation as success using later reads.
- **FR-007**: By default, use bounded observation after successful submission to confirm `SUSPENDED` for each selected target. `ACTIVE`, terminal states, `ABSENT`, `UNKNOWN`, suspension timestamps alone, and read failures MUST NOT establish confirmed suspension.
- **FR-008**: Stop waiting for a target when a terminal state is observed before suspension confirmation, reporting that terminal outcome without a suspension-success claim. Timeout, interruption, and read-authorization failure MUST retain accepted-submission evidence and the last observed state, distinguishing no observation from an observed state.
- **FR-009**: `--no-wait` MUST report accepted submission without claiming observed suspension or performing post-submission confirmation reads. Confirmation polling MUST NOT resubmit mutations; any submission retries MUST follow existing policy and preserve ambiguity when acceptance is unknown.
- **FR-010**: Support non-mutating dry-run previews of the exact selected keys, preserving capability validation. Dry-run MUST NOT prompt for mutation approval, submit mutations, or claim acceptance or confirmed suspension, including when combined with no-wait.
- **FR-011**: Support existing confirmation, auto-confirm, and automation conventions. Prompt eligibility, wording style, default answers, EOF handling, and abort behavior MUST follow established mutation-command patterns. No mutation may occur after declined confirmation.
- **FR-012**: Preserve per-key outcomes and partial results for collections, including accepted-but-unconfirmed, rejected, uncertain, and unsubmitted targets when applicable. A failure for one key MUST NOT erase evidence for others or turn the overall partial failure into unconditional success.
- **FR-013**: JSON output MUST use the shared envelope exactly once, including failure paths, retaining partial results, existing machine-readable error evidence, and established failure classes and exit behavior. Do not emit a second error envelope after rendering partial results.
- **FR-014**: Reserve stdout for selected-format results. Send plain interactive prompts to configured stderr and human mutation summaries, dry-run summaries, verbose outcome details, and diagnostics through the configured command-context logger. Informational summaries must obey logger level, format, and destination; they are not stdout result payloads. Keys-only output MUST contain one qualifying key per line and zero bytes when no keys qualify. Preserve established mode precedence, quiet suppression of informational summaries, and explicit machine output.
- **FR-015**: Aggregate repeated collection failures into one compact human diagnostic with counts, grouped causes, and a useful bounded reason. Normal, verbose-only, and quiet output MUST exclude raw transport chains, URLs, methods, bodies, and cursors on both streams. Low-level diagnostics MUST remain in the established redacted DEBUG path, enabled by either the debug flag or configured debug level independently of verbose.
- **FR-016**: Empty stdin MUST produce the existing no-keys input error with zero mutation requests, zero confirmation prompts, and no acceptance claims. Rendering any empty or failed result MUST NOT itself trigger discovery or mutation work.
- **FR-017**: Preserve existing process-instance state and `suspendedDate` observations. Do not manufacture state or timestamps to establish confirmation. Existing `get pi` and `expect pi` behavior MUST remain usable to inspect suspension.
- **FR-018**: Help, examples, README guidance, and generated command documentation MUST describe active-only eligibility, Camunda 8.10 support, `SUSPEND_PROCESS_INSTANCE` permission on `PROCESS_DEFINITION` when authorization is enabled, and the additional permission to read the instance for default confirmation.
- **FR-019**: Documentation MUST explain that suspension is independent for each selected instance, pauses engine execution while retaining runtime state and resources, does not cancel instances, does not interrupt already-running worker code or undo external effects, rejects job completion/failure attempts during suspension, and can result in affected jobs executing again after resumption. It MUST describe backpressure-related delay/rejection and avoid promising an instantaneous emergency stop.
- **FR-020**: Update the existing Bash suspension fixture to use this command for suspension setup instead of direct suspension requests. Preserve its intended scenarios; updating it MUST NOT authorize running live validation.
- **FR-021**: Preserve existing cancellation and forced-deletion behavior, including their distinct scope and outcome semantics. No resumption, search-based selection, native batch suspension, or definition-lifecycle changes are included.

### Key Entities

- **Selected instance**: An explicit unique process-instance key whose suspension is requested; its hierarchy does not expand or replace the selection.
- **Suspension preview**: A non-mutating description of the selected targets and any advisory observations, without acceptance or confirmation claims.
- **Submission outcome**: Per-target evidence that suspension was accepted, rejected, not submitted, or remains uncertain.
- **Suspension observation**: A read of the selected instance's state, with any existing suspension-date evidence; dates alone cannot prove suspension.
- **Suspension result collection**: Ordered per-key outcomes retaining submission evidence, last observed state when available, confirmation status, and failures or interruption information.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every successful confirmed suspension in acceptance coverage has both a successful submission response and an observed `SUSPENDED` state for the same selected key. There are zero false acceptance or confirmation claims across rejection, state-race, transport-failure, terminal, absent/unknown, timestamp-only, read-failure, timeout, and interruption cases.
- **SC-002**: Exactly the unique supplied keys remain suspension targets in all hierarchy scenarios, including simultaneous parent/child selection, with zero root escalation, hierarchy discovery, or requests to suspend unselected relatives.
- **SC-003**: Every dry-run, invalid-input, empty-input, and declined-confirmation case produces zero mutations. Dry-run and empty-input cases produce zero mutation prompts. Unsupported-version coverage produces zero unsupported requests through both CLI and public-library entry points.
- **SC-004**: All default confirmation scenarios complete within the configured waiting bound or terminate earlier on interruption or an observed terminal outcome. Every no-wait case produces zero post-submission confirmation reads, and observation alone causes zero mutation resubmissions.
- **SC-005**: Every unique selected key is accounted for in collection outcomes, and 100% of known acceptance evidence survives later per-key failures or unsuccessful confirmation. Partial failures preserve the established failure exit behavior.
- **SC-006**: Every supported output-mode acceptance case produces uncontaminated results: exactly one JSON envelope when JSON is effective, one qualifying key per line when keys-only is effective, and zero result bytes when no keys qualify. Prompts and diagnostics never appear on stdout.
- **SC-007**: All repeated-failure cases at normal, verbose-only, and quiet levels retain a compact aggregate error with zero raw HTTP-detail leakage. Configured/inherited logger routing, severity, DEBUG visibility, redaction, and absence of duplicate emission pass their isolated acceptance checks.
- **SC-008**: Operators can find all eligibility, permission, targeting, wait/no-wait, and operational limitations in command documentation. The existing suspension fixture has zero direct suspension requests, and cancellation, forced-deletion, inspection, and expectation regression coverage remains compatible.

## Assumptions

- The authoritative scope is [issue #346](https://github.com/grafvonb/c8volt/issues/346), retrieved on 2026-10-25. It provides sufficient detail to proceed without unresolved scope questions. [Camunda suspension requirements](https://docs.camunda.io/docs/apis-tools/orchestration-cluster-api-rest/specifications/suspend-process-instance/) and [operational semantics](https://docs.camunda.io/docs/components/concepts/process-instance-suspension/) support the eligibility, permission, and independent-instance rules.
- Existing explicit-key input, deduplication, configuration, timeout/backoff, retry, confirmation, output-mode precedence, and error conventions are dependencies. Empty stdin currently reports `stdin contained no keys`; this feature preserves that error instead of inventing a successful empty-selection workflow.
- Backend acceptance remains authoritative even if preflight reads are used. This specification does not require hierarchy discovery, all-or-nothing collection transactions, or an additional state-read permission merely to interpret a successful submission as accepted. The exact advisory-preflight policy must preserve the issue's submission and permission semantics.
- The issue's implementation constraints remain binding for planning: use its specified single-instance capability and existing generated support, keep public facades thin, and place mutation execution, collection handling, and waiting in internal services. Client regeneration is not a prerequisite. Detailed interfaces and component design belong in the plan.
- Validation uses isolated unit tests, generated-client mocks, test-owned HTTP fake servers, and isolated CLI/subprocess/terminal tests only. Do not run integration/smoke scenarios, discover live targets, or provision or mutate live services. Updating the Bash fixture does not authorize its execution; real-engine behavior remains unverified unless a separate target-specific workflow is explicitly requested.
- Isolated acceptance coverage must exercise version guards, exact-key request mapping, duplicate and parent/child selection, races, retry uncertainty, partial outcomes, wait/no-wait, dry-run, empty input, aborts, and supported human/JSON/keys-only/quiet/verbose/automation combinations. JSON checks must decode one envelope and require EOF while preserving failure exit behavior.
- Interactive checks use real terminal stdin with separately captured stdout and stderr, including redirected stdout and configured/inherited stderr destinations. Diagnostic checks use repeated errors carrying distinctive URL/method/body/key markers, plain and structured logger formats, and both DEBUG entry points with redaction checks. Assertions must detect a regression to direct stderr printing, raw aggregate errors, or duplicate emission.
- Native batch operations, search-based selection, resumption commands, definition lifecycle output, automatic relative suspension, and changes to cancellation/deletion are out of scope. Related issue #347 does not expand this feature's scope.
