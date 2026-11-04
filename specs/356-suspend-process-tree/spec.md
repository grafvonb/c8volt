# Feature Specification: Suspend Process-Instance Trees

**Feature Branch**: `356-suspend-process-tree`

**Created**: 2026-10-26

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/356 — feat(process): support process-instance tree suspension with --tree"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Suspend a selected subtree (Priority: P1)

As an operator, I want to suspend a selected process instance and its descendants in one operation so that I do not have to discover and supply every child key manually.

**Why this priority**: The main value is explicit subtree targeting without changing relatives outside the selected scope.

**Independent Test**: Use controlled nested parent/child relationships with active targets and verify that tree execution targets exactly the selected instances and reachable descendants, deduplicating overlaps and leaving ancestors and siblings untouched.

**Acceptance Scenarios**:

1. **Given** an active instance with active children and grandchildren, **When** the operator runs `c8volt suspend process-instance --key <key> --tree` and approves execution, **Then** every unique active instance discovered within that subtree is a suspension target, including the selected instance.
2. **Given** a selected child with a parent, siblings, and its own descendants, **When** `suspend pi --tree` executes, **Then** the selected child and its descendants are considered without escalating to the parent or including siblings.
3. **Given** repeated/comma-separated key selections and newline-separated stdin through `-`, including duplicates and overlapping selected subtrees, **When** tree scope is resolved, **Then** every unique discovered instance is accounted for once and every unique eligible target has one logical execution target.
4. **Given** an already-suspended or terminal parent with active reachable descendants, **When** tree planning runs, **Then** traversal continues through that parent, the active descendants remain targets, and the ineligible parent receives no suspension request.
5. **Given** existing exact-key input without `--tree`, **When** suspension runs, **Then** only supplied keys are targeted and existing rejection behavior for ineligible supplied keys remains unchanged.

---

### User Story 2 - Review and approve the expanded scope (Priority: P1)

As an operator, I want to preview which instances will be affected and approve one frozen target set so that execution cannot silently expand after I confirm it.

**Why this priority**: Hierarchy expansion increases mutation scope and must be visible before any suspension request.

**Independent Test**: Resolve mixed-state overlapping subtrees using controlled observations, review the preview, change the available descendants before approval, and verify that execution uses only the originally frozen active keys. Exercise discovery failure and empty eligible scope independently.

**Acceptance Scenarios**:

1. **Given** supported tree input, **When** `--tree --dry-run` is requested, including with `--no-wait`, **Then** the expanded discovered scope, active targets, already-suspended instances, and excluded instances are reported without mutation or mutation-confirmation prompts; discovery reads are permitted.
2. **Given** a fully resolved scope containing active targets, **When** interactive confirmation is eligible, **Then** one plain confirmation names the expanded mutation count, and no suspension request occurs before approval.
3. **Given** an approved frozen target set, **When** descendants appear or observations change after planning, **Then** no rediscovery expands the run; execution uses the approved keys and reports submission or confirmation failures caused by subsequent state changes.
4. **Given** multiple selected subtrees and a discovery failure in any one of them, **When** planning ends, **Then** execution fails before any suspension request or mutation-confirmation prompt, even if earlier subtrees were resolved.
5. **Given** an explicitly partial or unresolved discovery result, **When** tree execution is requested, **Then** it is not treated as an approved complete plan and no mutation is submitted.
6. **Given** successful planning with no active targets, **When** normal execution or no-wait is requested, **Then** the command reports a successful no-op with the discovered classifications and no confirmation, suspension request, or accepted/pending-mutation claim.
7. **Given** declined approval, **When** the operator aborts, **Then** no suspension requests are submitted and existing abort behavior is preserved.
8. **Given** missing selection, malformed keys, or empty/whitespace-only stdin, **When** tree suspension is requested, **Then** the existing input error occurs before discovery, confirmation, or mutation; this is not a successful empty-scope result.

---

### User Story 3 - Retain truthful collection outcomes (Priority: P2)

As an operator or automation author, I want tree suspension to retain the existing collection controls and evidence so that a partially successful tree does not appear fully suspended.

**Why this priority**: Independent mutations can fail or remain unconfirmed; the larger scope must preserve per-target accountability and existing script behavior.

**Independent Test**: Use controlled accepted, rejected, delayed, timed-out, interrupted, and unattempted targets in one frozen scope. Verify the resulting evidence, output modes, worker controls, and absence of mutations outside the approved scope.

**Acceptance Scenarios**:

1. **Given** discovered ACTIVE, SUSPENDED, and terminal instances, **When** tree execution completes, **Then** only planned active targets receive mutation requests; already-suspended instances are reported separately, terminal instances are excluded, and neither category is counted as a newly accepted or newly confirmed suspension.
2. **Given** an active observation followed by mutation rejection, **When** execution reports results, **Then** the rejection remains a failure even if a later state is SUSPENDED; discovery alone proves neither acceptance nor confirmation.
3. **Given** accepted submissions followed by mixed confirmation outcomes, **When** the run partially fails, **Then** per-target acceptance, last observation, confirmation, and failure evidence survive with the established failure exit behavior.
4. **Given** no-wait tree execution, **When** requests finish, **Then** results describe acceptance without claimed observed suspension or post-submission confirmation reads; the earlier tree discovery reads remain part of planning.
5. **Given** worker limits, fail-fast, or interruption, **When** execution stops or limits concurrency, **Then** existing collection behavior applies and targets not attempted remain distinguishable from accepted work.
6. **Given** a delayed confirmed run, **When** activity display is eligible, **Then** the existing indicator covers execution and waiting after approval and stops before final output; protected modes retain existing suppression.
7. **Given** human, verbose, JSON, keys-only, or quiet output, **When** the run renders results or failures, **Then** existing logger, prompt, output-mode precedence, machine-result, and single-error-envelope rules remain effective.

---

### User Story 4 - Understand permissions and practical limits (Priority: P2)

As an operator, I want help and documentation to explain the added discovery permissions and scope limits so that I can choose between exact-key and tree suspension deliberately.

**Why this priority**: A tree consists of independent instances, and a discovered scope cannot promise an atomic pause or coverage beyond the documented traversal limit.

**Independent Test**: Review command help, README guidance, and generated command documentation for the tree examples, permission distinction, frozen-scope behavior, supported versions, and explicit limits.

**Acceptance Scenarios**:

1. **Given** an operator comparing exact-key and tree modes, **When** they review documentation, **Then** they can identify selected-plus-descendant scope, no ancestor/root escalation, mixed-state handling, overlap deduplication, and unchanged default behavior.
2. **Given** an operator using tree preview or execution, **When** they review permissions, **Then** they can distinguish discovery/read/search permissions from suspension permission and the additional reads required for default confirmation.
3. **Given** a changing hierarchy or one beyond the existing traversal limit, **When** the operator reviews limitations, **Then** documentation states that the run acts on a discovered, frozen scope, is not atomic, may omit descendants created after discovery, and retains the existing limit of up to 1,000 direct children per parent lookup without promising coverage beyond it.
4. **Given** configured Camunda 8.8 or 8.9, **When** tree suspension or tree dry-run is requested, **Then** the established unsupported-capability error occurs before descendant discovery or any suspension request.

### Edge Cases

- Duplicate seeds, repeated stdin keys, and a selected ancestor plus selected descendants must not duplicate execution targets.
- A single-node tree is valid; selecting a child never implies root escalation.
- Already-suspended or terminal ancestors must not hide reachable active descendants.
- Unknown or other non-ACTIVE observed states are excluded and identified without being described as already suspended or successfully mutated.
- A missing selected instance, failed child discovery, denied discovery permission, interruption during planning, or explicitly partial/unresolved traversal prevents all mutations.
- No active targets after successful planning is a valid no-op; missing selection, malformed keys, and empty stdin remain input errors.
- An active target can become suspended, terminal, missing, or unauthorized after approval; existing submission and confirmation failures remain authoritative.
- New descendants discovered only after approval are outside the frozen scope; no automatic second discovery or rollback occurs.
- An accepted mutation followed by timeout, cancellation, terminal observation, or failed read retains acceptance without false confirmation.
- Dry-run may read the hierarchy but never mutates; no-wait avoids confirmation reads but does not avoid discovery reads.
- Existing traversal limits bound the discoverable scope; this feature does not add a large-tree pagination requirement.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Support opt-in `--tree` on `c8volt suspend process-instance` and its `pi` alias with existing repeated/comma-separated key and stdin conventions.
- **FR-002**: Tree scope MUST include each selected instance and its discovered descendants, including nested children. Selecting a child MUST NOT include its ancestors or siblings unless independently selected through another seed.
- **FR-003**: Merge overlapping selected subtrees and deduplicate keys so every discovered instance is classified once and every eligible instance is one logical target. Existing submission retry policy remains applicable.
- **FR-004**: Resolve all selected subtree scopes before confirmation or mutation. Discovery errors, denied discovery, planning interruption, and explicitly partial/unresolved traversal MUST prevent mutation across the whole invocation.
- **FR-005**: Traverse through all discovered states before determining eligibility. Plan mutation only for instances observed ACTIVE; report SUSPENDED instances separately and exclude terminal and other non-ACTIVE states without requests or new-mutation success claims.
- **FR-006**: Discovered state is advisory. Preserve authoritative rejection, uncertainty, acceptance, and confirmation behavior if a target changes state after planning; do not rewrite rejected or unaccepted work as successful suspension.
- **FR-007**: Tree dry-run MUST resolve and report the expanded scope and state classifications using discovery reads, with no suspension request or mutation-confirmation prompt, including with no-wait.
- **FR-008**: Freeze the ordered eligible keys and the scope counts before approval. Confirmation MUST occur once for the expanded unique mutation count using existing prompt eligibility, default-answer, EOF, auto-confirm, automation, and abort behavior. Decline MUST submit no mutations.
- **FR-009**: Execute only the frozen eligible keys without rediscovery or automatic scope growth. Preserve planned counts and classifications in final output so the reported scope agrees with the preview and confirmation.
- **FR-010**: Successful planning with zero eligible targets MUST complete as a truthful no-op without confirmation, mutation, or accepted/pending-work claims. Preserve the command-appropriate successful empty machine payload, exactly one JSON envelope, zero-byte keys output, and quiet suppression of informational summaries.
- **FR-011**: Missing selection, malformed keys, and empty/whitespace-only stdin MUST retain existing input-error behavior without discovery, confirmation, or mutation. No successful no-op may be inferred from invalid input or failed planning.
- **FR-012**: Preserve existing worker controls, fail-fast, and interruption behavior for the frozen collection. Retain per-target evidence and distinguish unattempted targets from accepted work.
- **FR-013**: Default execution MUST retain bounded confirmation of SUSPENDED after accepted submission. No-wait MUST retain acceptance-only results without confirmation reads. Planning observations MUST NOT be substituted for post-submission confirmation.
- **FR-014**: Preserve per-key partial results, original failure classes/exits, and accepted-but-unconfirmed evidence. A failed target MUST NOT erase successful targets or turn a partially failed run into unconditional tree success.
- **FR-015**: Human mutation summaries, previews, and verbose details MUST use established configured INFO logger helpers, honoring level, format, destination, and activity handling. Failures MUST use compact grouped configured diagnostics; raw HTTP details remain in the established redacted DEBUG path independently of verbose.
- **FR-016**: Prompts MUST remain plain text on configured stderr; JSON and keys-only results remain stdout results. Preserve JSON-over-keys mode precedence and quiet behavior. JSON MUST contain exactly one shared envelope with meaningful partial evidence and no second error rendering.
- **FR-017**: Keys-only tree preview MUST list frozen eligible targets; execution MUST list accepted targets under the existing acceptance contract. Do not emit already-suspended, excluded, rejected, uncertain, or unsubmitted targets as accepted keys.
- **FR-018**: Reuse existing indicator behavior for tree execution and waiting, starting only after approval and stopping before final rendering. Preserve suppression for protected output/automation modes and existing indicator controls; do not add per-request or per-key default chatter.
- **FR-019**: Preserve Camunda 8.10 suspension capability requirements and reject configured 8.8/8.9 before tree discovery, including dry-run. Existing public-library suspension guards remain effective.
- **FR-020**: Without `--tree`, preserve exact-key targeting, input handling, rejection of ineligible supplied keys, preview semantics, and all existing output/wait controls. Do not introduce discovery reads into the exact-key preview path.
- **FR-021**: Help, README, examples, and generated command documentation MUST explain selected-plus-descendant scope, independent child selection, mixed-state classification, expanded-count confirmation, and additional discovery/read/search permissions. Existing suspension permission and confirmation-read requirements remain documented.
- **FR-022**: Document the current traversal limit of up to 1,000 direct children per parent lookup, non-atomic execution, state races, and omission of descendants created after discovery. Preserve existing traversal behavior; pagination changes and guarantees beyond the documented limit are outside scope.
- **FR-023**: Preserve existing exact-key suspension, cancellation, deletion, and standalone walk behavior. Do not add automatic ancestor escalation, repeated discovery, resumption, rollback, native batch suspension, arbitrary search-based selection, or unrelated lifecycle changes.

### Key Entities

- **Selected seed**: An explicit process-instance key from flags or stdin whose own instance and descendants define a subtree boundary.
- **Discovered scope**: The deduplicated union of selected seeds and descendants returned within existing traversal limits, with observed states and scope-resolution outcome.
- **Tree suspension plan**: The frozen eligible target keys, already-suspended instances, excluded instances, and decisive counts used for preview, approval, and final accounting.
- **Eligible target**: A unique discovered instance observed ACTIVE during planning; that observation does not guarantee later acceptance or confirmation.
- **Execution outcome**: Per-target submission evidence, post-submission observation and confirmation, or failure/unattempted evidence, separate from planning classifications.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In all supported hierarchy acceptance scenarios, 100% of discovered active targets inside the selected subtree union are represented once, with zero mutations to unselected ancestors, siblings, or other instances outside the frozen target set.
- **SC-002**: Every preview, failed/incomplete discovery, declined approval, unsupported-version, invalid-input, and successfully resolved empty-eligible-scope scenario produces zero suspension requests. All preview and empty-eligible-scope cases produce zero mutation-confirmation prompts.
- **SC-003**: In every approved run, the executed target set and unique mutation count match the frozen plan and confirmation; zero targets are added by rediscovery after approval.
- **SC-004**: All discovered instances are accounted for as eligible, already suspended, or excluded, without counting an existing state as newly accepted or confirmed suspension. Every frozen eligible target retains its final known disposition after partial failure or interruption.
- **SC-005**: Every confirmed target has both accepted submission and a later SUSPENDED observation; no-wait results make zero confirmation claims or confirmation reads. Existing bounded-wait and failure behavior remains effective in all tree outcome scenarios.
- **SC-006**: All supported output-mode acceptance scenarios retain one machine-result document where requested, exact accepted-key output where requested, configured human logger routing, and plain prompt routing, with zero diagnostic or activity contamination of machine results.
- **SC-007**: Existing exact-key suspension, cancellation, deletion, and walk acceptance scenarios retain their behavior, and no new relative discovery is introduced without explicit tree selection.
- **SC-008**: An operator following the documentation can preview and suspend a subtree without manually collecting descendant keys, while identifying its permissions, exclusions, frozen scope, and traversal/non-atomic limitations.

## Assumptions

- Issue [#356](https://github.com/grafvonb/c8volt/issues/356), retrieved on 2026-10-26, is the authoritative scope. It extends #346 only through explicit tree selection; the earlier exact-key-only default remains valid.
- Existing parent/child relationships used by the walk feature define descendants. A selected instance belongs to its own subtree, and traversal must continue through ineligible ancestors to reach eligible children.
- Existing traversal currently requests up to 1,000 direct children per parent lookup. The expected workload fits that constraint; no pagination redesign, beyond-limit completeness guarantee, or large-tree implementation gate is introduced.
- Within the documented traversal behavior, failures and explicit partial/unresolved outcomes are errors rather than authorization to mutate a partially discovered tree. No additional completeness claim beyond the retained traversal limit is made.
- Non-ACTIVE states other than SUSPENDED are excluded with their observed classification. Missing selected data or failed discovery is a planning failure rather than an excluded target that permits partial execution.
- Already-suspended and excluded members do not themselves make a resolved tree run fail. Mutation or confirmation failures for frozen active targets retain the existing failure semantics.
- Ordering and deduplication follow existing input/traversal conventions; no atomicity or parent-before-child execution guarantee is added. Submission retries follow the existing policy for each unique logical target.
- This extends the existing suspension and walk capabilities. Planning should retain their established collection execution, confirmation, worker, logging, and indicator behavior without introducing a separate execution model.
- Validation uses controlled, isolated fixtures and command/terminal tests only. Live services, clusters, and live mutation or cleanup are not implementation completion gates.
