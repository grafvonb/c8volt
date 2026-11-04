# Feature Specification: Late Business-ID Assignment and Atomic Job Completion

**Feature Branch**: `362-late-business-id-assignment`

**Created**: 2026-11-02

**Status**: Draft

**Input**: [GitHub issue #362](https://github.com/grafvonb/c8volt/issues/362) — Support late assignment and atomic job completion on Camunda 8.10 and later supported versions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Assign a Business Reference After Process Start (Priority: P1)

As an operator, I want to assign a newly learned business reference to an eligible running process so that I can find it without restarting it.

**Why this priority**: Business references may become known only after process creation; direct assignment provides the basic late-assignment workflow.

**Independent Test**: Exercise scalar, repeated-key, and newline-separated stdin assignment against isolated fixtures; verify exact targets, values, previews, and per-instance outcomes.

**Acceptance Scenarios**:

1. **Given** an active root instance without a BID and business-ID uniqueness disabled, **When** I preview and execute `update pi --key 123 --set-bid 'ORDER-123'`, **Then** the preview performs no mutation and execution targets exactly instance 123; once assignment is confirmed, `get pi --bid 'ORDER-123'` can find it.
2. **Given** repeated keys or stdin containing duplicates, **When** I request assignment, **Then** all input is validated before any mutation, distinct process keys are frozen before confirmation, and each receives at most one submission.
3. **Given** a child, inactive instance, existing BID including an identical value, or uniqueness enabled even for an unused value, **When** assignment is submitted, **Then** backend rejection remains a failure; no parent or family receives a substitute write.
4. **Given** several targets with mixed acceptance, **When** execution finishes, **Then** results retain each target's outcome and partial evidence rather than claiming complete success.
5. **Given** assignment combined with `--bid`, `--vars`, or `--vars-file`, or invalid trailing key input, **When** I execute, **Then** validation rejects the whole input before mutation.
6. **Given** valid assignment input, **When** I use normal execution or `--no-wait`, **Then** output distinguishes the requested assignment, accepted submission, and any confirmed observation; a successful preview never guarantees backend acceptance.

---

### User Story 2 - Assign the Reference While Completing Work (Priority: P1)

As a worker or operator, I want completion and first BID assignment to succeed together so that work cannot complete while its required assignment fails.

**Why this priority**: Separate actions would permit inconsistent operational outcomes.

**Independent Test**: Exercise scalar, collection, stdin, and selector completion through isolated command and public-library paths; inspect combined submissions, prepared payload stability, and rejection behavior.

**Acceptance Scenarios**:

1. **Given** a job belonging to an eligible process, **When** I use `update job --key 456 --complete --set-bid 'ORDER-123' --vars '{"approved":true}'`, **Then** one native completion request includes BID and variables together while retaining other supported completion fields.
2. **Given** assignment rejection, including an identical existing BID, **When** combined completion is attempted, **Then** the whole request is rejected and an open job remains open; no separate assignment or fallback completion without BID occurs.
3. **Given** explicit collections, stdin, or tenant-scoped selector discovery, **When** I prepare completion, **Then** complete selection, BID, and variables are frozen before confirmation; every submitted job carries the approved BID through either submission path.
4. **Given** caller changes to the original BID or nested variables after preparation, **When** the prepared operation is submitted, **Then** the approved payload remains unchanged.
5. **Given** duplicate job keys and two distinct jobs in one process, **When** completion executes, **Then** duplicate keys submit once, distinct jobs remain separate targets, and one may succeed while another conflicts and remains open, with separate outcomes.
6. **Given** keyless `--complete --retries 0`, **When** I add `--set-bid`, **Then** zero remains an exact search predicate; keyed completion retains its existing retry-flag restriction.
7. **Given** retry-update, timeout-update, job-failure, or BPMN-error mode, **When** assignment is supplied, **Then** validation rejects the combination before mutation.
8. **Given** scalar input or a collection deduplicated to one job, **When** results return, **Then** scalar input retains its scalar result and collection input retains its bulk result.
9. **Given** a child-process job or a job already completed before submission, **When** combined completion is rejected, **Then** no parent is targeted and the request is not reported as successful completion and assignment.

---

### User Story 3 - Trust Evidence and Automation Results (Priority: P1)

As an operator or automation author, I want requested intent and observed facts kept separate so that I can safely interpret conflicts, partial failures, and uncertain responses.

**Why this priority**: Assignment is non-idempotent; replay or unsupported success claims can mislead subsequent operations.

**Independent Test**: Simulate acceptance, rejection, lost responses, empty and incomplete discovery, and mixed outcomes; capture configured logger destinations and result streams with isolated terminal fixtures.

**Acceptance Scenarios**:

1. **Given** old or absent resource snapshots, **When** assignment is previewed or reported, **Then** requested BID stays separate from previously observed BID; job, task, and listener observations are not backfilled from process data.
2. **Given** a timeout or lost response on direct assignment or scalar/bulk BID-bearing completion, **When** outcomes are assessed, **Then** the mutation is not automatically replayed; a later job `COMPLETED` state alone leaves assignment unconfirmed and its outcome unknown.
3. **Given** authoritative completed empty selection, **When** execution or dry-run runs in supported output and automation modes, **Then** it succeeds without prompting or mutation and claims no accepted or pending work, including with no-wait.
4. **Given** failed or incomplete discovery, **When** earlier pages contain candidates, **Then** nothing is submitted and incomplete discovery is not presented as an empty success.
5. **Given** human, quiet, verbose, JSON, or keys-only output, **When** execution succeeds or fails partially, **Then** existing format and exit contracts remain intact, diagnostics use configured logging, prompts use plain configured stderr, and results use stdout.
6. **Given** partial failure in shared-contract JSON mode, **When** the command exits, **Then** exactly one envelope retains available results and the original failure classification, without a second error rendering.
7. **Given** help and documentation, **When** I review assignment, **Then** I can distinguish `--bid` selection from `--set-bid` assignment and understand root eligibility, uniqueness disabled, atomic completion, and no-wait or uncertain evidence.

---

### User Story 4 - Validate Assignment Consistently (Priority: P1)

As a CLI or public-library user, I want consistent value and version rules so that unsupported assignment cannot silently turn into another operation.

**Why this priority**: Invalid assignment must never be dropped or confused with a variable update.

**Independent Test**: Exercise omitted and explicit invalid inputs, Unicode boundaries, unsupported adapters, and representative non-assignment workflows through command and direct library entry points.

**Acceptance Scenarios**:

1. **Given** valid padded, comma-containing, mixed-case, numeric-looking, or Unicode input, **When** assignment is requested, **Then** the exact opaque value is preserved without trimming, splitting, normalization, or key interpretation.
2. **Given** empty, whitespace-only, invalid UTF-8, or more than 256 UTF-16 code units, **When** either entry point receives assignment, **Then** it fails before mutation; otherwise valid values of exactly 256 units are accepted.
3. **Given** Camunda 8.8 or 8.9, **When** assignment is supplied through CLI or direct public-library calls, **Then** it fails clearly before mutation; 8.10 and later supported adapters carry assignment.
4. **Given** omitted assignment, **When** existing workflows execute, **Then** their behavior remains unchanged and completion omits the `businessId` field entirely.
5. **Given** a process variable named `businessId`, **When** existing variable-update behavior executes, **Then** it remains separate from native assignment.

### Edge Cases

- Identical reassignment is a conflict, not idempotent success. Dry-run cannot reserve eligibility against submission-time changes.
- Uniqueness enabled prevents late assignment even when the requested BID is unused.
- Child instances never redirect to roots; two job keys in one process are not one target.
- Supplementary Unicode characters occupy two UTF-16 units; surrounding spaces on nonblank input count and remain unchanged.
- Invalid trailing stdin input prevents all mutations rather than permitting an already-read prefix to execute.
- Sparse pages with continuation are not completed discovery. Documented limits preserve their existing bounded-selection meaning.
- Lost responses, caller payload changes, already-completed jobs, stale snapshots, user aborts, and missing reports must not manufacture assignment success.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: CLI and public-library operations MUST support first BID assignment on 8.10 and later supported adapters, reject supplied assignment on 8.8/8.9 before mutation, and preserve omitted-assignment behavior.
- **FR-002**: Assignment input MUST distinguish omission from explicit empty, reject empty/whitespace-only values, invalid UTF-8, and values over 256 UTF-16 code units, and preserve accepted input exactly.
- **FR-003**: Submission-time backend eligibility MUST govern acceptance: an active root instance, no existing BID including an identical value, and business-ID uniqueness disabled. Dry-run MUST NOT guarantee acceptance.
- **FR-004**: Direct assignment MUST target exact supplied process keys; combined completion MUST target the job's own instance. Neither may redirect to a parent or traverse a family.
- **FR-005**: `update pi --set-bid <value>` MUST support scalar, repeated-key, and newline-separated stdin explicit input, validating the complete input and deduplicating by process-instance key before mutation.
- **FR-006**: Process assignment MUST reject combination with `--bid`, `--vars`, or `--vars-file`; existing BID selection for variable updates MUST retain its meaning.
- **FR-007**: Direct assignment MUST support dry-run, normal execution, and no-wait; use service-owned planning and bounded concurrency; freeze targets before confirmation; and retain per-instance outcomes on partial failure.
- **FR-008**: `update job --complete --set-bid <value>` and corresponding public-library capabilities MUST support scalar, explicit collection, stdin, and selector-based completion through both existing submission paths.
- **FR-009**: Job assignment MUST reject retry-update, timeout-update, failure, and BPMN-error modes. Existing selectors and exclusivity MUST remain intact, including exact zero-retry selection for keyless completion and existing restrictions for keyed completion.
- **FR-010**: Requested BID MUST survive parsing, validation, public and domain requests, prepared batches, previews, and submission. Prepared completion MUST freeze BID and variables, including nested values, against caller changes.
- **FR-011**: Every job MUST receive BID and completion variables in one native completion request, retaining other supported completion fields. When assignment is omitted, `businessId` MUST be absent from the payload.
- **FR-012**: Assignment rejection MUST reject the whole completion. The client MUST NOT split it into separate mutations, retry without BID, or fall back to plain completion.
- **FR-013**: Bulk completion MUST preserve tenant-scoped discovery, limits, confirmation, and frozen complete selection; deduplicate by job key; retain distinct jobs sharing a process; and report each outcome separately with every submission carrying the requested BID.
- **FR-014**: Scalar input MUST retain its scalar result and collection input its bulk result, including collections deduplicated to one job.
- **FR-015**: Previews and results MUST distinguish requested BID from previously observed BID and distinguish accepted submission, confirmed observation, rejection, and unknown outcome. Existing job, task, and listener snapshots MUST NOT be backfilled from process data.
- **FR-016**: BID-bearing mutations MUST NOT be automatically replayed after timeout or lost response. A later job `COMPLETED` observation alone MUST NOT establish that this request assigned the BID; scalar and bulk completion MUST retain unknown assignment outcome when evidence is insufficient.
- **FR-017**: Dry-run MUST mutate nothing; no-wait MUST report only established evidence. Failed or incomplete discovery MUST submit nothing. Authoritative completed empty selection MUST succeed without prompting, mutation, or claims of accepted/pending work.
- **FR-018**: Existing confirmation, auto-confirm, automation, quiet, verbose, JSON, keys-only, dry-run, no-wait, and abort contracts MUST remain intact for supported combinations. Keys-only MUST retain its documented meaning and emit zero bytes when empty.
- **FR-019**: Human mutation/dry-run summaries, verbose outcomes, warnings, and errors MUST use configured command-context logging; selected-format results MUST use stdout; prompts MUST remain plain configured stderr with existing eligibility and input behavior.
- **FR-020**: Failures MUST preserve original exit classification and available partial evidence. Shared JSON MUST emit exactly one envelope with established empty/null conventions and no duplicate error rendering; quiet MUST retain requested machine results.
- **FR-021**: Normal and verbose human failures MUST use compact aggregate diagnostics with bounded grouped causes, without raw transport details on either stream. Redacted DEBUG diagnostics MUST remain available via debug flag or configured level independently of verbose and keep result stdout clean.
- **FR-022**: Help, examples, README, and generated CLI documentation MUST explain selection versus assignment, root eligibility, uniqueness disabled, identical-ID conflicts, exact-value rules, atomic completion, snapshot limitations, and no-wait/unknown evidence.
- **FR-023**: Native assignment MUST remain separate from a variable named `businessId`. Clearing, replacement, synthetic propagation, and new BID mutation fields on other operations MUST remain excluded.

### Key Entities *(include if feature involves data)*

- **Assignment intent**: Optional exact reference requested by the caller, distinct from selectors and observed values.
- **Eligible instance**: Active root without BID, subject to uniqueness configuration and submission-time eligibility.
- **Frozen assignment plan**: Validated distinct process keys and requested value retained before confirmation.
- **Prepared completion batch**: Frozen job selection, requested BID, variables, and other supported completion data.
- **Per-target outcome**: Evidence of acceptance, observation, rejection, or uncertainty for one assignment or completion.
- **Observed snapshot**: A resource's previously recorded BID or absence, without an assumed guarantee of freshness.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can preview, assign, and find an eligible process by its new reference in the three-command primary workflow without restarting it.
- **SC-002**: Every accepted target receives the exact requested value; duplicate target keys produce zero duplicate submissions, and child targets produce zero writes to parents.
- **SC-003**: Every rejected combined-completion case produces zero fallback completions and leaves previously open jobs open; distinct jobs sharing a process retain individual outcomes.
- **SC-004**: Every invalid-value and unsupported-version case fails before mutation; accepted boundary values remain exact, and omitted-assignment cases retain established behavior.
- **SC-005**: Every lost-response case produces at most one submission per target and never infers confirmed assignment solely from job completion. Every prepared payload remains stable after caller changes.
- **SC-006**: All completed empty selections produce zero prompts and mutations; all incomplete discoveries produce zero mutations. Supported JSON output contains one envelope, and empty keys output contains zero bytes.
- **SC-007**: Every output-mode acceptance case retains original failure classification and partial evidence, uses established destinations, and separates intent from observed facts.
- **SC-008**: Published guidance explains both workflows and all eligibility restrictions, allowing operators to choose an assignment path and interpret uncertainty without reading source code.

## Assumptions

- [Issue #360](../360-business-id-creation-views/spec.md) provides creation-time BID and observed views; [issue #361](../361-business-id-selection/spec.md) provides filtering and selection. Both are merged prerequisites per issue #362.
- The issue's pinned Camunda 8.10.3 eligibility behavior is the baseline. Later supported adapters does not mean adding a new supported version or changing the default.
- Existing authorization, tenant scope, limits, wait behavior, confirmation, and result conventions remain authoritative unless extended above. A preview validates intent rather than reserving eligibility.
- No new BID mutation field is added to user-task completion, variable updates, cancellation, deletion, incident resolution, suspension, or resumption.
- Acceptance validation uses generated-client mocks, test-owned HTTP servers, isolated command/library tests, and terminal/subprocess fixtures. It covers uniqueness enabled, identical reassignment, child targeting, already-completed jobs, payload capture/omission, lost responses, sparse discovery, partial failures, and Unicode boundaries. No live environment is an implementation completion gate.
- Output coverage captures stdout/stderr separately, checks configured and inherited logger destinations in plain and structured formats, quiet and WARN-level suppression, repeated backend failures with distinctive HTTP details, and redacted DEBUG through both configuration paths. Real terminal stdin verifies prompts and empty-scope completion; JSON decoding requires EOF after one envelope. Nonempty, explicit-key, abort, and error regressions remain covered.
