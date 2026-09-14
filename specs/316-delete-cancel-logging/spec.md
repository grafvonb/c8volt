# Feature Specification: Delete and Cancel Logging Consolidation

**Feature Branch**: `codex/316-delete-cancel-logging`

**Created**: 2026-09-13

**Status**: Draft

**Input**: [GitHub issue #316 — complete delete/cancel logging consolidation across workflow, polling, and errors](https://github.com/grafvonb/c8volt/issues/316)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Understand an Unconfirmed Cancellation (Priority: P1)

As an operator deleting a process instance, I want a concise explanation when cancellation confirmation times out so I can distinguish submitted work from confirmed outcomes and decide what to investigate next.

**Why this priority**: The reported incident obscured a successful cancellation submission and continuing ACTIVE states behind repeated nested errors, making the operational outcome difficult to interpret.

**Independent Test**: Execute deletion of an active child whose deletion conflicts, whose root cancellation is accepted, and whose root and child remain ACTIVE until confirmation times out. Read the normal transcript without DEBUG.

**Acceptance Scenarios**:

1. **Given** child deletion conflicts and root cancellation is accepted, **When** confirmation times out while the root and child remain ACTIVE, **Then** normal output identifies the cancellation-confirmation phase, affected root/scope, timeout, available last observed states, cancellation submission, the attempted child deletion, and the subsequent deletion stage that was never reached.
2. **Given** a tree fails, **When** its warning and final command error are presented, **Then** the warning concisely identifies the failed tree and the final error summarizes the command failure without repeating a full nested causal chain in both.
3. **Given** cancellation was submitted but its outcome was not confirmed, **When** timeout is reported, **Then** output states that confirmation timed out without claiming cancellation was rolled back or definitively failed.
4. **Given** DEBUG is admitted for the failed workflow, **When** failure diagnostics are emitted, **Then** the complete causal chain is available once and the existing error classification is preserved.

---

### User Story 2 - Follow Cancellation and Deletion Progress (Priority: P1)

As an operator using verbose output, I want each meaningful transition explained once so I can follow why deletion requires cancellation and what the command is waiting for.

**Why this priority**: Operators need workflow context during a wait, independently of low-level troubleshooting records.

**Independent Test**: Execute conflict-driven deletion with verbose output, once with cancellation confirmation succeeding and once timing out; also exercise cancellation directly.

**Acceptance Scenarios**:

1. **Given** a deletion conflict requires cancellation and cancellation escalates from a child to its root, **When** verbose output is enabled, **Then** each applicable reason is explained once at the corresponding transition.
2. **Given** cancellation is submitted and confirmation begins, **When** verbose output is enabled, **Then** submission, awaited scope and target states, timeout, and backoff policy are each explained once for that phase.
3. **Given** cancellation is confirmed, **When** deletion resumes, **Then** verbose output explains that resumption once; a timed-out confirmation never claims resumption occurred.
4. **Given** verbose is enabled without DEBUG, **When** the workflow runs, **Then** the explanations appear without enabling HTTP diagnostics; DEBUG without verbose does not enable verbose-only narration.

---

### User Story 3 - Inspect Polling Without Repeated Chatter (Priority: P2)

As an operator collecting DEBUG logs, I want one completed state observation per polling check alongside the existing exchange diagnostic so I can inspect progress without redundant messages.

**Why this priority**: The reported 36 checks produced 252 DEBUG records; reducing repetition preserves useful evidence while making the transcript easier to scan.

**Independent Test**: Observe successful cached-authentication polling, failed lookups, terminal observations, and a direct lookup outside a wait, with DEBUG admitted.

**Acceptance Scenarios**:

1. **Given** a polling check completes, **When** its observation is logged, **Then** exactly one state-observation record contains the process-instance key, attempt number, observed state or lookup failure, elapsed wait time, and next delay only when another check will occur.
2. **Given** 36 successful cached-authentication checks each require one exchange, **When** DEBUG is admitted, **Then** polling emits exactly 72 records: 36 state observations and 36 unchanged HTTP diagnostics, excluding phase-start/end records.
3. **Given** a lookup occurs within polling, **When** DEBUG is admitted, **Then** redundant checking, fetching, state-result, and routine authentication cache lookup/hit records are absent.
4. **Given** a direct lookup or token acquisition, refresh, or failure occurs, **When** its diagnostics are eligible, **Then** useful troubleshooting evidence remains available and credentials remain absent.

---

### User Story 4 - Preserve Script and Interactive Contracts (Priority: P2)

As an automation author or interactive operator, I want logging changes to preserve result formats, prompts, and mutation behavior so existing usage remains reliable.

**Why this priority**: A clearer transcript must not change destructive-operation safeguards or machine-consumed results.

**Independent Test**: Compare supported human, JSON, keys-only, quiet, automation, verbose, and DEBUG combinations, including dry-run and no-wait where supported; capture stdout and stderr separately and verify requests and outcomes.

**Acceptance Scenarios**:

1. **Given** JSON or keys-only results, **When** delete or cancel completes or fails, **Then** existing result schemas, envelopes, omission rules, and output precedence remain unchanged and diagnostic text never contaminates stdout.
2. **Given** quiet or automation mode, **When** a workflow runs, **Then** existing visibility, warning policy, confirmation eligibility, and explicit machine-result behavior remain unchanged.
3. **Given** real terminal stdin with captured or redirected stdout, **When** confirmation is eligible, **Then** prompts retain plain wording, input/default/EOF behavior, and the configured or inherited stderr destination.
4. **Given** identical controlled outcomes before and after consolidation, **When** delete or cancel runs, **Then** request counts, mutation order, retry and wait policy, timeout behavior, and exit classification remain unchanged across supported Camunda versions.

### Edge Cases

- A failed state lookup produces one observation describing the failure rather than inventing a state; subsequent retries retain existing policy.
- A terminal observation, timeout, or interrupted wait does not advertise a next delay when no further check will occur.
- Last observed state information may be partial or unavailable; output identifies only available evidence and never presents stale observations as confirmed current state.
- Multiple roots or concurrent waits keep observations attributable to the correct key and attempt, with concise per-tree failure warnings and a compact aggregate final error.
- Token acquisition or refresh may add legitimate diagnostics; the two-record polling target applies to successful single-exchange checks using cached authentication.
- Empty completed discovery, dry-run, explicit keys, user aborts, and no-wait preserve their existing outcomes. No work must not be described as a submitted mutation, and dry-run remains a preview.
- Tenant context retains its established severity, ordering, formatting, and suppression. JSON log formatting remains independent of JSON command-result formatting.
- A cancellation submission failure is distinguishable from an accepted submission whose confirmation times out.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Delete/cancel polling MUST emit exactly one completed state observation per check when DEBUG is admitted, containing key, attempt, observed state or lookup failure, elapsed wait time, and next delay only if a further check will occur.
- **FR-002**: Polling MUST suppress redundant nested checking, fetching, and state-result diagnostics while preserving useful diagnostics for direct lookups outside a wait.
- **FR-003**: Routine authentication cache lookup/hit records MUST be absent from DEBUG. Eligible token acquisition, refresh, and failure diagnostics MUST remain available without exposing credentials.
- **FR-004**: Existing `api #...` exchange diagnostics MUST retain their formatting, fields, redaction, and exactly-one-record-per-observed-exchange behavior. A successful cached-authentication check with one exchange MUST produce one observation plus one exchange diagnostic.
- **FR-005**: Verbose output MUST explain each applicable cancellation prerequisite, child-to-root escalation, submitted cancellation, awaited scope and states, timeout and backoff policy, and post-confirmation deletion resumption exactly once per corresponding transition or wait phase.
- **FR-006**: Verbose and DEBUG MUST remain independent and obey existing log-level, format, quiet, and output-mode filtering. No new user flags or logging settings may be introduced.
- **FR-007**: Failure presentation MUST use a concise per-tree warning and compact final command error, without repeating the complete causal chain across those summaries. The complete causal chain MUST be available once at DEBUG when admitted, with underlying classification and inspection preserved.
- **FR-008**: Cancellation-confirmation timeout reporting MUST identify the failed phase, affected root/scope, timeout, available last observed states, and mutation stages already submitted or never reached. It MUST distinguish submission from confirmation and MUST NOT claim rollback or definitive cancellation failure solely from a confirmation timeout.
- **FR-009**: Result stdout, JSON envelopes and schemas, keys-only output, quiet warning policy, automation, dry-run, no-wait, empty-result behavior, and output precedence MUST remain compatible. Diagnostic records MUST use the established configured or inherited stderr destination.
- **FR-010**: Interactive prompts MUST remain plain text on configured or inherited stderr, preserving wording, eligibility, default answers, input and EOF handling, auto-confirm, and abort behavior even when stdout is redirected.
- **FR-011**: Logging consolidation MUST preserve discovery and mutation request counts and ordering, cancellation/deletion orchestration, retries, wait timing and backoff, timeout outcomes, and exit classification. Observability MUST NOT cause additional requests.
- **FR-012**: The same polling and workflow diagnostic contract MUST hold across all supported Camunda versions. Established tenant logging and HTTP diagnostic contracts from issues #303 and #305 MUST be preserved.
- **FR-013**: Acceptance validation MUST cover the complete reported child-conflict/root-cancellation/continued-ACTIVE/timeout transcript, success and lookup-failure paths, independent verbose/DEBUG settings, and output compatibility, including real-terminal confirmation with separately captured streams.
- **FR-014**: User documentation and command help MUST describe any changed documented logging behavior consistently with the delivered behavior.

### Key Entities

- **Workflow scope**: The affected root and related process instances participating in cancellation and deletion.
- **State observation**: The completed result of one polling check, associated with a key, attempt, elapsed time, outcome, and conditional next delay.
- **Workflow transition**: A meaningful event such as escalation, submission, waiting, or resumed deletion that warrants one operator explanation.
- **Failure summary**: Concise operational evidence identifying the failed phase, affected scope, and confirmed or unconfirmed mutation stages, with detailed causes available separately.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The equivalent 36-check cached-authentication scenario yields exactly 72 polling-related DEBUG records, reduced from 252, excluding phase-start/end records, with zero redundant nested polling or routine cache-hit records.
- **SC-002**: In every applicable verbose acceptance scenario, each required workflow explanation appears exactly once per transition or wait phase; verbose-only runs emit zero HTTP diagnostic records.
- **SC-003**: From normal output alone, an operator can identify the failed phase, root/scope, timeout, available last states, submitted work, and unreached stages in the reported timeout case; zero messages claim cancellation rollback or definitive failure based solely on unconfirmed completion.
- **SC-004**: Every failed-workflow acceptance case retains its exit classification, avoids full-chain repetition across normal warnings and errors, and exposes the complete causal chain exactly once when DEBUG is admitted.
- **SC-005**: All compatibility cases preserve result content and request behavior, with zero diagnostic contamination of stdout, zero extra requests, and unchanged prompt behavior and destinations.
- **SC-006**: All supported-version acceptance cases satisfy the same observation and narration rules, and all targeted regression checks and the full project validation suite pass before implementation is accepted.

## Assumptions

- Issue #316 is authoritative and is a continuation of the established tenant and exchange diagnostic behavior in #303 and #305.
- The reported reproduction is an active child deletion conflict (409), accepted root cancellation (204), repeated ACTIVE observations for the affected root/child scope, and cancellation-confirmation timeout. Acceptance tests use short deterministic wait settings while preserving production wait policy.
- Existing wording and result contracts remain authoritative wherever this specification does not explicitly change diagnostic presentation. One explanation per phase permits separate explanations for genuinely distinct roots or repeated phases.
- The issue's delivery constraints remain binding during planning: reuse existing logging, progress, and output facilities; preserve inspectable wrapped/joined causes; exercise the real service, waiter, progress, and error path against local HTTP fixtures; apply shared-waiter changes consistently across supported adapters.
- Implementation validation includes targeted tests and `make test`. Where documented behavior changes, update command source metadata and relevant user documentation and regenerate CLI docs with `make docs-content`.
- A general logging framework rewrite, new flags, quiet-warning policy redesign, mutation orchestration changes, retry timing changes, and result-schema changes are outside scope.
