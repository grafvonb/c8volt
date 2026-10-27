# Feature Specification: Resume Process Instances

**Feature Branch**: `347-resume-process-instances`

**Created**: 2026-10-27

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/347 — feat(process): add process-instance resumption commands"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Resume explicitly selected instances (Priority: P1)

As an operator, I want to resume suspended instances by key so that I can restart their execution without manually sending low-level requests or changing unselected relatives.

**Why this priority**: Explicit resumption is the core capability and the smallest useful operator workflow.

**Independent Test**: Supply single, repeated, comma-separated, and stdin keys against controlled outcomes. Verify exact targeting, version checks, preview, approval, and submission results.

**Acceptance Scenarios**:

1. **Given** a suspended instance on configured Camunda 8.10, **When** `c8volt resume process-instance --key <key>` runs with approval, **Then** that instance is submitted for resumption and its outcome is reported; `resume pi` provides equivalent behavior.
2. **Given** repeated/comma-separated `--key`/`-k` values and newline-separated stdin through `-`, **When** selection contains duplicates and distinct parent/child keys, **Then** unique keys retain first-input reporting order, both parent and child remain targets, and no relatives are discovered or automatically included.
3. **Given** an already-active, terminal, terminating, missing, or unauthorized exact-key target, **When** submission is rejected, **Then** that rejection remains a failure even if a later observation shows a desired state.
4. **Given** valid exact keys, **When** `--dry-run` is used, including with `--no-wait`, **Then** only those keys are previewed without discovery, mutation, approval prompts, acceptance claims, or claimed eligibility verification.
5. **Given** missing selection, malformed keys, or empty/whitespace-only stdin, **When** either mode is requested, **Then** an established input error occurs without discovery, mutation, approval, or successful-no-op claims.
6. **Given** configured Camunda 8.8 or 8.9, **When** resumption or its dry-run is requested through the CLI or public library, **Then** the established unsupported-capability error occurs before tree discovery or unsupported requests.
7. **Given** eligible interactive approval, **When** the operator declines or reaches the existing abort condition, **Then** no mutation occurs and established default-answer, EOF, and abort behavior is preserved; auto-confirm and automation retain existing unattended behavior.

---

### User Story 2 - Preview and resume a selected subtree (Priority: P1)

As an operator, I want an explicit tree option that discovers suspended descendants and executes the scope I approve so that I do not have to collect descendant keys manually.

**Why this priority**: Resumption does not cascade, and expanded mutation scope must be understandable and stable before approval.

**Independent Test**: Use nested, overlapping, mixed-state hierarchies, including an active parent with suspended descendants. Verify classifications and frozen eligible keys, then exercise failed discovery and empty eligible scopes.

**Acceptance Scenarios**:

1. **Given** selected instances with children and grandchildren, **When** `resume pi --key <key> --tree` resolves scope, **Then** each selected instance and all descendants returned within existing traversal limits are included. Selecting a child includes its subtree without ancestors or siblings unless separately selected.
2. **Given** overlapping subtrees and duplicate seeds, **When** planning completes, **Then** each discovered key is accounted for once and each eligible key is one logical mutation target.
3. **Given** active or terminal parents with suspended descendants, **When** planning runs, **Then** discovery traverses the reachable subtree before filtering states; only observed `SUSPENDED` members are eligible, while active, terminal, unknown, and other ineligible members are accounted for as excluded.
4. **Given** supported tree input, **When** `--tree --dry-run` runs, **Then** the plan distinguishes supplied keys, discovered scope, eligible targets, and excluded instances without mutation or confirmation prompts. Discovery reads still occur, including with no-wait.
5. **Given** a successfully resolved nonempty eligible scope, **When** confirmation is eligible, **Then** one prompt names the expanded unique eligible count, and execution uses those frozen keys without rediscovery after approval.
6. **Given** new descendants or state changes after planning, **When** execution runs, **Then** new descendants are not silently added and rejected submissions remain failures; an earlier suspended observation cannot establish acceptance.
7. **Given** any discovery failure, denied discovery permission, planning interruption, or explicitly partial/unresolved traversal, **When** planning ends, **Then** no mutation or mutation-confirmation prompt occurs, even if another selected subtree resolved successfully.
8. **Given** successful discovery with zero suspended targets, **When** normal execution, dry-run, or no-wait runs, **Then** it completes as a successful no-op without confirmation, mutation, or accepted/pending-work claims.
9. **Given** descendants suspended independently before this operation, **When** they are discovered under selected seeds, **Then** they remain eligible regardless of which prior operation suspended them; no previous suspension scope is inferred.

---

### User Story 3 - Distinguish acceptance from observed resumption (Priority: P1)

As an operator or automation author, I want submission and resulting-state evidence so that I can distinguish resumed execution from accepted but unconfirmed work.

**Why this priority**: Acceptance alone does not prove resumption, and fast completion must not cause a false failure.

**Independent Test**: Exercise controlled submission/observation sequences in exact-key and tree modes, including immediate completion, concurrent cancellation, persistent suspension, ambiguous transport failures, and interrupted waits.

**Acceptance Scenarios**:

1. **Given** accepted submission, **When** subsequent bounded observation returns `ACTIVE` or immediate `COMPLETED`, **Then** resumption is confirmed without requiring an intermediate `ACTIVE` observation, and completion is identified when observed.
2. **Given** acceptance followed by `TERMINATED` or `CANCELED`, **When** observed, **Then** waiting stops and cancellation is reported distinctly without claiming normal completion or confirmed resumption.
3. **Given** acceptance followed by persistent `SUSPENDED`, `ABSENT`, `UNKNOWN`, missing suspension date alone, or unsuccessful reads, **When** waiting ends through its bound, interruption, or read failure, **Then** acceptance and the last observed state survive without false confirmation; no observation remains distinguishable from an observed unknown state.
4. **Given** rejection or transport failure without a successful submission response, **When** results are reported, **Then** rejection or uncertain acceptance is retained; a later desired state cannot turn it into confirmed acceptance or resumption.
5. **Given** `--no-wait`, **When** submission succeeds, **Then** results report acceptance without post-submission confirmation reads or observed-resumption claims. Earlier tree discovery remains permitted.
6. **Given** delayed confirmation or retryable submission failure, **When** waiting or retries occur, **Then** observation never resubmits the mutation, and established submission retry policy preserves uncertainty about ambiguous outcomes.
7. **Given** a fresh read after resumption, **When** the backend omits `suspendedDate`, **Then** existing inspection output omits it instead of retaining or manufacturing a timestamp.

---

### User Story 4 - Use reliable collection controls and output (Priority: P2)

As an operator or script author, I want familiar collection controls and trustworthy output so that partial failures remain actionable and machine results remain parseable.

**Why this priority**: Multiple-key and tree operations can partially succeed; reporting must preserve evidence without overwhelming operators or breaking scripts.

**Independent Test**: Execute mixed-outcome collections with worker limits, fail-fast, and interruption. Capture stdout, stderr, and configured logger destinations separately across supported modes and real-terminal prompt scenarios.

**Acceptance Scenarios**:

1. **Given** mixed accepted, confirmed, rejected, uncertain, and unattempted targets, **When** execution partially fails, **Then** ordered per-key outcomes and partial results survive with established failure class and exit behavior.
2. **Given** worker controls or fail-fast, **When** a collection executes, **Then** established concurrency and stop-scheduling behavior apply equally to exact-key and tree inputs; unattempted targets remain distinguishable from accepted work.
3. **Given** JSON output, **When** success, no-op, or partial failure renders, **Then** stdout contains exactly one shared envelope retaining appropriate payload and error evidence, without a second envelope or diagnostic contamination.
4. **Given** keys-only output, **When** preview renders, **Then** stdout lists exact planned keys or eligible tree keys; execution lists accepted keys only, one per line. No qualifying keys produces zero bytes, and accepted keys alone do not assert confirmation.
5. **Given** human, dry-run, or verbose output, **When** summaries render, **Then** they use configured INFO logging and its level, format, destination, and activity behavior, with empty human stdout. WARN level and quiet suppress informational records; quiet retains required errors and explicit machine results.
6. **Given** repeated backend failures carrying raw request details, **When** normal, verbose-only, or quiet output renders, **Then** one compact aggregate error retains counts and bounded grouped causes without raw HTTP details on either stream. Both debug-flag and configured-debug modes expose only established redacted diagnostics independently of verbose.
7. **Given** eligible interactive approval with redirected stdout, **When** a prompt is shown, **Then** it remains plain text on configured or inherited stderr, independently of logger records and selected-format results.
8. **Given** eligible activity display, **When** approved execution and waiting begin, **Then** the shared indicator starts after approval, covers execution/waiting, and stops before final rendering on success, failure, and interruption; existing mode restrictions remain effective.
9. **Given** a resolved empty eligible tree scope, **When** human, JSON, keys-only, quiet, quiet-plus-machine, or supported automation/auto-confirm/no-wait combinations run, **Then** output remains mode-correct without confirmation or mutation, including when terminal stdin would otherwise permit approval.

---

### User Story 5 - Understand permissions and operational limits (Priority: P2)

As an operator, I want documentation to explain permissions and limits so that I can choose a mode and interpret its outcome correctly.

**Why this priority**: Resumption can release buffered work, requires different discovery and confirmation permissions, and does not promise successful process completion.

**Independent Test**: Review help, README, examples, generated documentation, and relevant fixture changes against the stated behavior without executing live fixtures.

**Acceptance Scenarios**:

1. **Given** an operator following documentation, **When** choosing a mode, **Then** they can identify Camunda 8.10 support, suspended-only eligibility, permissions, input forms, preview, approval, bounded wait, and no-wait semantics.
2. **Given** a changing or large tree, **When** limitations are reviewed, **Then** guidance states up to 1,000 direct children per parent lookup, no coverage guarantee beyond that limit, a frozen non-atomic scope, possible omission of later descendants, and no automatic rollback after partial failure.
3. **Given** observed resumption, **When** its meaning is reviewed, **Then** guidance explains that it does not guarantee incident-free execution, process completion, or immediate subscription readiness; buffered work may drain and failures can leave an instance suspended.
4. **Given** integration scenarios and fixtures intended to cover CLI resumption, **When** updated, **Then** they invoke the command for that coverage without making live execution an implementation completion gate.

### Edge Cases

- Selected ancestors and descendants remain distinct exact-key targets; overlapping tree scopes deduplicate identical discovered keys.
- Active or terminal ancestors must not hide suspended descendants. Unknown state is a tree exclusion, never a resumption success.
- Missing roots, partial discovery, read/search authorization failure, and interrupted planning prevent all mutation. Failed planning and invalid input are not successful empty scopes.
- Discovery may become stale before submission; exact-key rejections and tree state-race failures remain authoritative.
- An instance can complete before the first post-acceptance observation. Concurrent cancellation must remain distinct from normal completion.
- Acceptance survives later read failure, timeout, interruption, or disappearance. Ambiguous submission does not become known acceptance by inference.
- No-wait skips confirmation reads but not discovery. Dry-run always remains a preview, including with no-wait.
- Fail-fast may leave queued targets unattempted while in-flight work finishes. Known outcomes cannot be lost or described as rolled back.
- Rendering must not initiate discovery or mutations. Empty keys output contains no newline; quiet retains explicit machine results.
- Missing `suspendedDate` is not sufficient confirmation evidence, and no replacement timestamp is manufactured.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provide `c8volt resume process-instance` and alias `resume pi`, accepting repeated/comma-separated `--key`/`-k` and newline-separated stdin through `-`, deduplicating identical keys in first-input reporting order.
- **FR-002**: Missing selection, malformed keys, and empty/whitespace-only stdin MUST retain existing input errors without discovery, approval, or mutation. Arbitrary search/filter selection is excluded.
- **FR-003**: CLI and public-library resumption entry points MUST support configured Camunda 8.10 and reject configured 8.8/8.9 through the established capability error before tree discovery or unsupported requests, including dry-run.
- **FR-004**: Without `--tree`, target exactly the supplied unique keys, preserving separately selected parents and children without traversal, root escalation, or ancestor collapsing. Preserve backend rejection outcomes for active, terminal, terminating, missing, and unauthorized targets.
- **FR-005**: With `--tree`, include selected seeds and descendants returned by existing traversal, merge overlaps, and traverse reachable members before state filtering. Only observed `SUSPENDED` members become eligible; all others MUST be accounted for as excluded without mutation requests.
- **FR-006**: Resolve the entire tree scope before approval or mutation. Discovery failures, denied permissions, interruption, and explicitly partial/unresolved traversal MUST abort all mutations, even when other seeds resolved successfully.
- **FR-007**: Tree planning MUST distinguish supplied keys, discovered members, eligible targets, and excluded members. Freeze eligible keys and counts before one approval for the expanded unique count, then execute the same scope without rediscovery. Independently suspended descendants remain eligible.
- **FR-008**: Treat discovery as advisory. Subsequent state changes and rejection responses MUST retain their failure outcomes; neither preflight nor later desired states can turn rejected or ambiguous submission into acceptance.
- **FR-009**: Dry-run MUST preview exact keys without mandatory reads, or discover and preview expanded tree scope, without mutation or approval prompts. It MUST NOT claim acceptance or verified exact-key eligibility, including with no-wait.
- **FR-010**: A successfully resolved tree with zero eligible targets MUST complete as a no-op without confirmation, mutation, or accepted/pending claims. Preserve command-appropriate empty machine payload conventions, zero-byte keys output, and quiet suppression. Failed discovery, abort, or absent reports MUST NOT establish a successful no-op.
- **FR-011**: Preserve established confirmation eligibility, default answers, EOF/abort behavior, auto-confirm, and automation. Declined approval MUST submit no mutation. Prompts remain plain text on configured stderr even when stdout is redirected.
- **FR-012**: Only a successful submission response establishes acceptance. Preserve rejection and uncertainty when no such response exists. Default bounded post-submission observation MUST confirm resumption only upon `ACTIVE` or `COMPLETED`; immediate completion requires no intermediate active observation.
- **FR-013**: Observed `TERMINATED` or `CANCELED` after acceptance MUST stop waiting and report cancellation distinctly without claiming normal completion or confirmed resumption. Persistent suspension, absent/unknown states, missing suspension dates, and failed reads MUST NOT establish confirmation.
- **FR-014**: Timeout, interruption, and read-authorization failure MUST preserve acceptance and last-observed-state evidence, distinguishing no observation from an observed state. Waiting MUST remain bounded by established configuration.
- **FR-015**: `--no-wait` MUST report acceptance without confirmation reads or observed-resumption claims. Observation MUST never resubmit mutations; submission retries follow established policy and retain uncertainty about ambiguous outcomes.
- **FR-016**: Preserve ordered collection outcomes, partial results, worker-count policy and controls, fail-fast, and interruption behavior for single-key, multiple-key, and tree execution. Distinguish rejected, accepted, confirmed, uncertain, and unattempted work as applicable; partial failure retains established failure class and exit behavior without rollback.
- **FR-017**: JSON MUST render exactly one shared envelope, retaining partial results and established error evidence, without second error rendering. Preserve omission/null/empty-payload conventions and supported output-mode precedence.
- **FR-018**: Keys-only preview MUST list planned exact keys or eligible tree keys; execution MUST list accepted keys only, including accepted-but-unconfirmed targets. Emit one key per line and zero bytes when none qualify. Document that accepted keys alone do not prove confirmation.
- **FR-019**: Human mutation summaries, previews, and verbose outcomes MUST use configured command-context INFO logging, honoring inherited/configured level, plain or structured format, destination, and activity handling. Human stdout remains empty; machine results remain stdout payloads and prompts remain plain stderr text. Quiet suppresses information while retaining required errors and explicit machine results.
- **FR-020**: Emit one compact aggregate failure diagnostic through the configured logger, grouping repeated causes with counts and a bounded useful reason. Normal, verbose-only, and quiet output MUST exclude raw URLs, methods, bodies, cursors, and transport chains on both streams. Established redacted DEBUG diagnostics remain independently accessible through the debug flag or configured debug level.
- **FR-021**: Shared activity MUST cover approved execution and waiting, start after approval, stop before final output on every exit path, and preserve existing mode eligibility and cleanup. Do not add default per-request or per-key lifecycle chatter.
- **FR-022**: Fresh reads MUST preserve existing `suspendedDate` mapping by omitting the field when the backend no longer supplies it. Do not manufacture timestamps or treat their absence as confirmation. Existing inspection and expectation commands remain usable to observe resulting states.
- **FR-023**: Help, README, examples, command metadata, and generated documentation MUST describe supported versions, eligibility, exact-key/tree scope, input forms, preview, worker/fail-fast controls, output, bounded wait/no-wait, and accepted-versus-confirmed evidence.
- **FR-024**: Document `SUSPEND_PROCESS_INSTANCE` permission on `PROCESS_DEFINITION` when authorization is enabled, additional instance-read permission for default confirmation, and read/search discovery permissions for tree execution and tree dry-run.
- **FR-025**: Document the limit of up to 1,000 direct children per parent lookup, frozen non-atomic scope, possible omission of descendants created after discovery, independently suspended descendants, races, and no automatic rollback. Do not promise incident-free execution, completion, immediate subscription readiness, or coverage beyond the limit; explain buffered work and failures that leave suspension in place.
- **FR-026**: Relevant integration resumption scenarios and fixtures MUST use the command where CLI coverage is intended. Updating them does not authorize execution or make live validation an implementation gate.
- **FR-027**: Preserve suspension, cancellation, forced deletion, traversal, and inspection behavior. Exclude native batch operations, suspension command changes, definition lifecycle output, automatic incident repair, automatic relative resumption without `--tree`, root escalation, repeated discovery, previous-suspension tracking, rollback, and pagination changes.

### Key Entities

- **Selected key / tree seed**: An explicit instance identifier; an exact-key target or a boundary for subtree discovery.
- **Discovered scope**: Deduplicated selected instances and descendants, observed states, and discovery completeness within existing limits.
- **Resumption plan**: Supplied keys, discovered scope, exclusions, and frozen ordered eligible targets used for preview, approval, and accounting.
- **Submission outcome**: Per-target evidence of acceptance, rejection, uncertainty, or work not attempted.
- **Resumption observation**: A post-acceptance state and available suspension-date evidence; it can confirm active execution/completion, identify cancellation, or leave resumption unconfirmed.
- **Result collection**: Ordered per-key outcomes retaining scope distinctions, acceptance, last observation, confirmation, and failure evidence after partial success or interruption.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every confirmed resumption in acceptance coverage has accepted submission and a later `ACTIVE` or `COMPLETED` observation for that key. Zero false acceptance or confirmation claims occur across rejection, ambiguity, cancellation, persistent suspension, absence/unknown, date-only, failed-read, timeout, and interruption scenarios.
- **SC-002**: Exact-key scenarios affect exactly the supplied unique keys. Tree scenarios account for 100% of discovered members, with every eligible target represented once and zero mutations outside the frozen eligible scope or to excluded members.
- **SC-003**: Every preview, invalid-input, unsupported-version, declined-approval, failed/incomplete-discovery, and resolved-empty-scope case produces zero mutations. Preview and resolved-empty-scope cases produce zero approval prompts even with interactive input eligible.
- **SC-004**: Every eligible interactive tree execution asks once for the frozen eligible count and uses that scope without additional discovery. Zero later descendants are silently added to approved work.
- **SC-005**: All default waits terminate within their configured bound or earlier on decisive observation, interruption, or failure. No-wait scenarios perform zero confirmation reads; observation causes zero mutation resubmissions.
- **SC-006**: Every execution target retains its final known disposition after partial failure, fail-fast, or interruption; 100% of known acceptance evidence survives unsuccessful confirmation.
- **SC-007**: Every supported output scenario emits uncontaminated machine results, exactly one document when JSON is effective, and one qualifying key per line when keys-only is effective. Human information and errors use the configured logger, with zero duplicate failures or raw HTTP-detail leakage outside DEBUG.
- **SC-008**: An operator following the examples can preview and resume a subtree without manually collecting descendants and identify required permissions, exclusions, wait/no-wait meaning, and discovery limits from documentation.
- **SC-009**: Existing suspension, cancellation, deletion, traversal, and inspection acceptance scenarios retain their behavior. Relevant fixture scenarios intended to test CLI resumption use the command, while live execution remains a separate workflow.

## Assumptions

- [Issue #347](https://github.com/grafvonb/c8volt/issues/347), retrieved on 2026-10-27 with no comments, is authoritative and provides sufficient detail without unresolved clarification markers. Its implementation constraints remain binding planning inputs; component and interface design belong in the plan.
- Existing suspension conventions supply key validation, deduplication, confirmation, automation, worker limits, retry/wait configuration, output precedence, and errors. Exact-key preview needs no preflight reads. The current wait default is two minutes through `--backoff-timeout`, with per-request `--timeout`; reuse established configuration semantics.
- Preserve existing supported combinations: JSON takes precedence over keys-only, requires dry-run or unattended approval, and is incompatible with verbose. Quiet preserves explicit machine output. Do not invent combinations to resolve this feature.
- Existing descendant discovery returns up to 1,000 direct children per parent lookup, rather than a whole-tree cap. Explicitly partial/unresolved results still fail planning; this feature does not add pagination or certify discovery beyond that limit.
- Validation uses isolated unit tests, generated-client mocks, test-owned HTTP fake servers, and isolated CLI/subprocess/terminal tests. No live target discovery, provisioning, integration/smoke execution, or real-service mutation is authorized. Updating fixtures is source work only; actual engine behavior remains unverified absent a separate target-specific request.
- Isolated coverage must include request mapping, CLI/library version guards, nested/overlapping mixed-state trees, failed discovery, frozen-scope races, worker/fail-fast behavior, partial outcomes, immediate completion, cancellation, stale-date omission, invalid/empty input, preview, no-wait, and supported mode combinations. These behavioral checks do not require runtime tests for this specification-only change.
- Prompt checks use real terminal stdin, separately captured stdout/stderr, redirected stdout, and configured/inherited destinations. Empty scopes remain prompt-free even when interactive approval is eligible. Machine checks decode exactly one JSON envelope with EOF, retaining partial results and original failure exits.
- Logging checks exercise command execution with configured/inherited destinations, plain and structured formats, INFO suppression at WARN/quiet, and repeated errors carrying distinctive URL/method/body/key markers. Check both streams under normal and verbose-only output, both DEBUG entry points and redaction, and activity cleanup. Assertions must detect direct-print bypasses, raw aggregate leaks, and duplicate diagnostics.
