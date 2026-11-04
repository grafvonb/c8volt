# Feature Specification: Element Scope Variables

**Feature Branch**: `329-element-scope-variables`

**Created**: 2026-10-25

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/329 — inspect and update variables at explicit runtime element scopes"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect variables owned by an element (Priority: P1)

As an operator investigating a running process, I want to see variables owned directly by a selected element, including an embedded subprocess or task, so I can distinguish its local data from values held elsewhere.

**Why this priority**: Existing process-root and effective task views do not provide an arbitrary element-local view. Reliable inspection prevents repairs based on the wrong scope.

**Independent Test**: Read a known element and a filtered element collection with local variables spread across multiple pages, same-name variables in surrounding scopes, and elements with no local variables. Verify only local data is shown and every selected element remains visible.

**Acceptance Scenarios**:

1. **Given** local, ancestor, sibling, descendant, and called-process variables, **When** `get element --with-vars` or `get ei --with-vars` selects an element, **Then** only its directly owned variables are returned, with consistent owning-process identity.
2. **Given** local values span several pages, including sparse or empty intermediate pages with continuation, **When** inspection completes, **Then** the entire local collection has been read; malformed progress or later-page failure is reported rather than presenting an empty or successfully complete scope.
3. **Given** a selected element has no local variables, **When** enriched output is rendered, **Then** the element remains visible and enriched JSON contains an empty variable array. Without enrichment the variable field is omitted.
4. **Given** a search has ordering, a selection limit, and paging controls, **When** variables are requested, **Then** the same elements are selected in the same order, and each returned element's variables are complete regardless of the element limit.
5. **Given** keys-only or numeric total output is effective, **When** `--with-vars` is also supplied, **Then** that output remains unchanged and no variable retrieval occurs. Existing conflicts involving total, JSON, keys-only, and listeners still apply.
6. **Given** human or JSON output requests variables and listeners together, **When** the selection is read, **Then** both enrichments describe the same elements without repeating base element discovery.

---

### User Story 2 - Repair precisely the selected local scopes (Priority: P1)

As an operator, I want to preview and apply the same variable values to explicitly selected element scopes, so I can repair subprocess data or create a task-local override without changing a parent variable.

**Why this priority**: Process-instance updates target roots, and user-task updates preserve an existing variable's owning scope. Neither provides this explicit local-override workflow.

**Independent Test**: Select several element keys, including duplicates, and request a mix of additions, changes, and already-matching values. Preview, approve, and inspect the submitted destinations and values. Repeat with a planning failure on the last element.

**Acceptance Scenarios**:

1. **Given** explicit repeated, comma-separated, or stdin keys and one variable object, **When** `update element` or `update ei` prepares a plan, **Then** keys are deduplicated in first-input order and the same object is considered for every selected element.
2. **Given** a name exists locally with a different complete value, **When** the update is approved, **Then** it changes only at that element. Unmentioned names and other scopes remain unchanged.
3. **Given** a name is absent locally but exists in a parent, even with the requested value already, **When** an update is planned and applied, **Then** it is a local addition and the parent remains unchanged.
4. **Given** an element has both changed and already-matching requested values, **When** its prepared update executes, **Then** only planned additions and changes are submitted. Elements with no changes receive no write.
5. **Given** a requested element is missing, unauthorized, inconsistently identified, or has incomplete variable discovery, **When** any selected element fails planning, **Then** no selected element receives a write or mutation-approval prompt.
6. **Given** a complete plan, **When** dry-run is requested or approval is declined, **Then** no writes occur. Dry-run identifies exact destinations and does not prompt; decline preserves abort behavior rather than becoming a successful no-op.
7. **Given** the configured discovery tenant differs from an explicitly selected element's tenant, **When** authorized lookup, planning, and confirmation occur, **Then** the configured discovery filter does not hide its variables or produce false additions. Search-based reads still honor their discovery tenant selection.

---

### User Story 3 - Know what was accepted and confirmed (Priority: P2)

As an operator, I want outcomes for every selected element to distinguish submission from verified values and failures, so I can respond to partial completion without blindly repeating accepted writes.

**Why this priority**: A successful write and successful read-back are separate facts. Partial failures, cancellation, and concurrent work must not obscure those facts.

**Independent Test**: Use a multi-element selection with successful writes, a rejected write, and a confirmation failure. Compare results with observed submissions, then repeat with no-wait, sequential workers, bounded concurrency, and fail-fast.

**Acceptance Scenarios**:

1. **Given** accepted writes, **When** ordinary execution finishes successfully, **Then** complete local reads confirm the requested values at the selected scopes. Matching parent values, truncated values, or mismatched scope observations cannot establish success.
2. **Given** `--no-wait`, **When** the update executes, **Then** planning and approval remain intact, no post-submission confirmation reads occur, and accepted writes are reported as submitted rather than confirmed.
3. **Given** some writes are accepted and another submission or confirmation fails, **When** results are rendered, **Then** accepted evidence and failed/skipped outcomes remain in selection order with the original failure exit classification.
4. **Given** fail-fast stops scheduling or cancellation/timeout interrupts execution, **When** outcomes are reported, **Then** in-flight or accepted work is not described as rolled back, unscheduled work is identifiable, and unknown acceptance is not represented as definite rejection.
5. **Given** worker and invocation-wide request-rate controls, **When** all phases run, **Then** the existing bounds, deadlines, and retry conventions remain effective; pacing waits do not erase earlier accepted outcomes.
6. **Given** repeated backend failures containing recognizable HTTP details, **When** human normal, verbose, or quiet output is used, **Then** one compact grouped ERROR remains visible without raw HTTP details. Structured output retains one complete failure envelope with all relevant outcome evidence.

---

### User Story 4 - Use previews, no-ops, and automation safely (Priority: P2)

As a script author or interactive operator, I want each supported output mode and no-work path to be predictable, so I can inspect or repair elements without contaminated results or unnecessary prompts.

**Why this priority**: Empty scopes and partial failures are common operational cases, and automation depends on precise streams and exit behavior.

**Independent Test**: Exercise empty searches, empty local collections, all-unchanged plans, and empty payload objects in normal execution and dry-run across output modes, including real terminal stdin with redirected stdout.

**Acceptance Scenarios**:

1. **Given** an empty local collection and a nonempty requested object, **When** planning completes, **Then** requested names are additions rather than a no-op. Conversely, a valid all-unchanged or empty-object plan prompts and writes zero times.
2. **Given** dry-run or a successful no-op, including `--no-wait`, **When** results are rendered, **Then** no submission, accepted mutation, or pending mutation is claimed and no post-submission confirmation reads occur.
3. **Given** JSON output, including quiet plus JSON, **When** success, empty results, no-op, or partial failure is returned, **Then** stdout has exactly one shared envelope followed by end-of-output, preserving payload shape and applicable failure exit status.
4. **Given** keys-only output, **When** a read, dry-run update, or executed update finishes, **Then** it lists respectively selected element keys, changed destination keys, or mutation-accepted keys once each. Accepted writes remain listed after confirmation failure; no qualifying keys means zero bytes.
5. **Given** real terminal stdin and redirected stdout, **When** confirmation or paging is eligible, **Then** plain prompts appear only on configured or inherited stderr and retain existing defaults, decline/EOF, auto-confirm, and automation behavior.
6. **Given** quiet output or diagnostic logging configuration, **When** an operation runs, **Then** informational output follows quiet settings, explicitly requested machine results remain, and required errors retain their configured severity and destination without duplicate emission.

### Edge Cases

- Runtime element keys, BPMN element IDs, and user-task keys are distinct identities; an invalid or missing update target must never trigger discovery fallback.
- A selected element can own an empty collection while inheriting many values. An empty local collection is not an empty element selection.
- A call-activity element and its called process root are distinct scopes; inspection and mutation never cross that boundary.
- Sparse pages, changing or malformed continuation metadata, and canceled/later failed reads cannot establish complete emptiness.
- Backend truncation and display shortening are different. Neither a matching prefix nor an inherited match proves local equality.
- `{}` may produce no work after valid planning; `{"name":null}` sets a value and does not delete it. Missing, conflicting, malformed, non-object, or unreadable payload sources fail.
- A variable or element may change, disappear, or lose authorization after planning. Preserve the approved destinations and report the failure; do not redirect or silently replan.
- A cross-tenant explicitly selected element with no local variables must remain distinguishable from an unauthorized or incorrectly filtered read.
- Partial submission, confirmation failure, fail-fast, and request-pacing timeout can coexist; retain known outcomes and uncertainty separately.
- No-op and dry-run output must not accidentally become an accepted envelope because no-wait is enabled.
- Parent/listener output restrictions still apply when variables are added. Enrichment suppression under keys-only or total must not bypass unrelated flag validation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `get element` and alias `get ei` MUST support `--with-vars` for existing exact-key and list/search selection without changing selection grammar, ordering, limits, or paging policy.
- **FR-002**: Enrichment MUST include only variables directly owned by each selected runtime element and consistent with its owning process. It MUST exclude inherited, sibling, descendant, and called-process scopes.
- **FR-003**: Every enriched scope MUST be read completely before it is presented as complete, including continuation through sparse pages. Invalid/non-progressing pagination, cancellation, or later read failure MUST remain failures rather than empty or successful partial enrichment. Element limits MUST NOT truncate variable collections.
- **FR-004**: Reads MUST request complete values and preserve backend incompleteness evidence. Shared value-formatting/display-shortening conventions MUST remain consistent; JSON MUST retain received values and backend truncation metadata. Incomplete values MUST NOT establish unchanged or confirmed outcomes.
- **FR-005**: Selected elements with no local variables MUST remain visible. Enriched JSON MUST use an empty variable array after a successful empty read and omit enrichment when not requested/performed, preserving existing item/collection envelopes and other fields.
- **FR-006**: Effective keys-only and total output with `--with-vars` MUST skip variable retrieval and preserve existing output and validation. Human/JSON `--with-vars --with-listeners` MUST support both enrichments on the same selection without duplicate base discovery. Existing listener conflicts with total/keys-only remain.
- **FR-007**: `update element` and alias `update ei` MUST accept repeated/comma-separated `--key`/`-k` and newline-separated stdin keys through positional `-`, merging and deduplicating in first-input order. Empty/missing/invalid selection MUST fail without discovery fallback.
- **FR-008**: Update MUST require exactly one of `--vars` or `--vars-file` containing a non-null JSON object, preserve established variable-update value semantics, and apply the same object to every unique selected element. Locally determinable input errors MUST fail before remote work. Empty objects are valid; null properties set values without deletion.
- **FR-009**: Planning MUST successfully establish every selected element's identity, authorization, and complete local variables before any mutation or mutation-approval prompt. A missing target, denied read, inconsistent identity/tenant evidence, or incomplete discovery anywhere MUST prevent all writes.
- **FR-010**: Plans MUST distinguish additions, changes, unchanged requested values, and untouched values. A locally absent name MUST be an addition even when a parent holds the same name/value. Only complete local equality establishes unchanged status.
- **FR-011**: Preview MUST identify exact destinations using `element:<key>` and established compact summary/detail conventions. Approved destinations and mutation values MUST remain frozen; execution MUST submit only additions/changes, grouped as one logical write per changed unique element subject to existing retry policy. Unchanged elements and unrelated names/scopes MUST receive no writes.
- **FR-012**: Execution MUST NOT replan after approval, redirect rejected writes, or imply transactional rollback or protection against concurrent external changes.
- **FR-013**: Ordinary successful updates MUST confirm requested values through complete reads matching the selected local scope and value, without resubmitting accepted writes. Confirmation failure MUST retain submission evidence and a failing outcome.
- **FR-014**: `--no-wait` MUST skip only post-submission confirmation reads. Planning, approval, mutation requests, and mutation-error handling remain required; accepted results MUST be distinguishable from confirmed ones.
- **FR-015**: Updates MUST preserve `--workers`/`-w`, `--no-worker-limit`, fail-fast, cancellation, timeout, backoff, retry, and invocation-wide request-rate conventions across all phases. One worker MUST execute sequentially; a configured bound MUST limit concurrent work. Fail-fast MUST stop further scheduling after observed failure while retaining in-flight outcomes.
- **FR-016**: Ordered outcomes MUST account for unchanged, accepted/submitted, confirmed, mutation-failed, confirmation-failed, and unattempted/skipped work. Partial failure MUST preserve accepted writes, original error classes/exits, and uncertainty about acceptance after interruption.
- **FR-017**: Explicit-key reads, planning, and confirmation MUST preserve backend authorization without applying the configured discovery-tenant filter. List/search discovery MUST preserve its tenant filter; enrichment MUST remain bound to selected element identity. Report known tenant evidence without extra discovery solely for presentation.
- **FR-018**: Successful no-op MUST require completed authoritative planning. It MUST produce no mutation prompt, write, or post-submission confirmation read. An empty local collection with requested names MUST produce additions. Missing targets, errors, user abort, or absent reports MUST NOT be reclassified as successful no-op.
- **FR-019**: Dry-run MUST preserve planning reads, preview exact intended changes, and perform zero mutations, mutation prompts, or post-submission confirmation reads. Neither preview nor no-op MUST claim submitted/accepted/pending mutations, including with no-wait.
- **FR-020**: Human, JSON, keys-only, quiet, verbose, and automation output MUST follow established precedence and use mode-aware rendering for early returns as well as ordinary results. Empty selections preserve existing payload shape and field omission rules; no fictitious reports or shared envelope schema changes represent no work. Quiet suppresses information without suppressing explicit machine results or required errors.
- **FR-021**: JSON MUST contain exactly one shared envelope followed by EOF for successful, empty, no-op, and failed results, retaining partial outcomes and original failure classification. No second error output may follow that envelope. Preserve variable-update JSON confirmation guardrails and rejection of JSON combined with verbose.
- **FR-022**: Keys-only reads MUST emit selected element keys even for empty local collections; update previews MUST emit changed destination keys; executed updates MUST emit mutation-accepted keys even if confirmation subsequently fails. Emit each qualifying key once, one per line, with zero bytes for no keys and no update keys for unchanged/unsubmitted work.
- **FR-023**: Prompts MUST be plain text on configured/inherited stderr with no logger prefix. Preserve terminal-input eligibility when stdout is redirected, default answers, EOF/decline behavior, and auto-confirm/automation. Result stdout MUST remain free of prompts and diagnostics.
- **FR-024**: Human errors, warnings, and diagnostics MUST use the configured command-context logger with correct severity/destination, including partial-result and exit-after-render paths. Failed batches MUST emit one compact aggregate ERROR with nonzero counts and bounded grouped reasons, retained under quiet. Existing bootstrap fallback behavior remains.
- **FR-025**: Normal and verbose-only human output MUST expose no raw joined/transport errors, URLs, HTTP methods, bodies, cursors, or wrapped transport chains on either stream. Functional per-resource detail belongs behind verbose. Existing machine-readable error evidence MUST remain available without schema changes to address human formatting.
- **FR-026**: Existing DEBUG diagnostics/redaction MUST remain available through both `--debug` and configured DEBUG level independently of verbose, honoring flag precedence and configured destination. Diagnostics MUST stay off stdout and introduce no additional remote requests.
- **FR-027**: Camunda 8.8, 8.9, and 8.10 MUST be supported. Preserve generic unsupported-version/bootstrap behavior, including configured 8.7 failing before remote operations with the existing local-precondition error and exit 1; introduce no operation-specific 8.7 behavior.
- **FR-028**: Existing PI root-scope updates, UT effective-scope updates, task completion, and element/listener selection MUST retain their behavior. New element operations MUST NOT traverse or mutate called-process roots implicitly.
- **FR-029**: Command discovery metadata, parent help, examples, README guidance, and generated CLI documentation MUST describe supported local-scope behavior, input/output combinations, no-op and partial outcomes, tenant handling, and no-wait accurately.

### Key Entities *(include if feature involves data)*

- **Selected runtime element**: A concrete execution instance identified by an element key, owning process, known tenant, and selection position.
- **Local variable observation**: A name, value, owning scope, owning process, and completeness indication directly associated with a selected element.
- **Requested variable object**: The shared set of names and intended values applied independently to every selected element.
- **Prepared element update plan**: Frozen destinations and classifications identifying additions, changes, unchanged requested values, and untouched observations across the complete selection.
- **Element update outcome**: Ordered evidence distinguishing no change, submission acceptance, confirmation, failure, and skipped work while retaining partial acceptance or uncertainty.
- **Tenant context**: Known authorization/discovery context and observed target-tenant evidence, without treating configured discovery filters as explicit-key authorization limits.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In all inspection acceptance cases, operators see every selected element and all of its readable local variables, with zero variables incorrectly attributed from other scopes and identical results regardless of page partitioning.
- **SC-002**: Every applied variable/destination matches an approved addition or change. There are zero unintended parent, sibling, descendant, or called-process changes, including when a new local override equals an existing parent value.
- **SC-003**: Every invalid-input, planning-failure, declined-approval, dry-run, and successful-no-op case produces zero writes; every dry-run/no-op produces zero mutation prompts and zero post-submission verification reads.
- **SC-004**: All confirmed outcomes have complete evidence at the selected scope and value. No-wait cases perform zero post-submission verification reads and never claim confirmation.
- **SC-005**: Every unique selected update key is accounted for in execution outcomes, in input order, and 100% of known accepted writes remain identifiable after another failure or failed confirmation.
- **SC-006**: Every supported output/interaction combination yields uncontaminated results: one structured result where requested, one qualifying key per line, and zero bytes when no keys qualify. Quiet does not hide required failures or explicit machine results.
- **SC-007**: Repeated batch failures produce exactly one compact human error at normal/verbose/quiet levels with zero raw HTTP-detail leakage. Diagnostic routing, severity, and redaction remain correct for every supported logger configuration in acceptance coverage.
- **SC-008**: Authorized explicit-key operations across tenants produce the same scope decisions regardless of configured discovery tenant; denied operations never appear as successful empty reads or additions. Existing read/update/completion regressions remain compatible across supported versions.

## Assumptions

- The authoritative source is [issue #329](https://github.com/grafvonb/c8volt/issues/329), retrieved on 2026-10-25 after its requirements update. Its review against `aaa18a7c` establishes a still-unimplemented capability; passing baseline tests is not acceptance of this feature.
- Existing root-variable operations, effective task-variable operations, and exact-local mutation capability are dependencies, but their user-visible scope semantics differ. Architecture, helper reuse, version-adapter boundaries, and generated-documentation obligations in the issue and AGENTS.md remain binding planning constraints; this specification does not select an implementation design.
- Current AGENTS.md and constitution apply. Existing PI/UT failure-output violations are not precedent for the new command. Correct behavior here does not authorize unrelated PI/UT refactoring or global error-class changes.
- Scope excludes new scope-selection flags, recursive discovery, variable deletion, implicit task assignment/completion, a general workflow framework, atomic multi-element transactions, and concurrency locking. No new display-control flag is required by this specification; existing value-rendering semantics are reused.
- Validation must use isolated fixtures, mocks, and test-owned HTTP/CLI/subprocess/terminal exercises only. Live/local Camunda, clusters, integration/smoke targets, environment-backed validation, and live cleanup are excluded from implementation completion gates. Inherited test flags/profiles/environment must not enable them indirectly; real-engine behavior left unverified must be reported accurately.
- Acceptance checks inspect exact scope filters, submitted values, identity evidence, pagination, request counts, outcomes, and exits. Cover normal and dry-run; human/JSON/keys-only/quiet; quiet plus machine output; and supported auto-confirm/automation/no-wait combinations, including nonempty, sparse, explicit-key, abort, and error regressions.
- Interactive validation uses real terminal stdin with stdout/stderr captured separately, testing configured and inherited stderr. Pipe-only input or mocked terminal eligibility is insufficient. Diagnostic validation uses repeated distinctive URL/method/body/key markers, both DEBUG entry points, plain/structured formats, and assertions that detect direct-print routing, duplicate diagnostics, raw-error leakage, or silent quiet failures.
- The existing unsupported-version bootstrap may run before output configuration; preserving that behavior takes precedence over introducing a new JSON bootstrap contract here.
