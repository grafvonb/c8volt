# Feature Specification: Business-ID Filters and Process Selection

**Feature Branch**: `361-business-id-selection`

**Created**: 2026-11-01

**Status**: Draft

**Input**: [GitHub issue #361](https://github.com/grafvonb/c8volt/issues/361) — Add native business-ID filters and process selection on Camunda 8.10 and later supported versions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find Work by Its Business Reference (Priority: P1)

As an operator or library consumer, I want to find process instances and user tasks by an exact business ID so that I can locate business work without first finding technical instance keys.

**Why this priority**: Accurate filtering is the foundation for both investigation and safe operational selection.

**Independent Test**: Search isolated process-instance and user-task records with matching and nonmatching IDs, combined predicates, multiple tenants, multiple pages, and distinct task-versus-process IDs; verify results, counts, and enriched views.

**Acceptance Scenarios**:

1. **Given** a supported version and records with different IDs, **When** I use `get pi --bid <value>` or `get user-task --bid <value>`, **Then** the search applies native exact matching and returns only records matching that resource's own business ID within the configured tenant scope.
2. **Given** additional supported predicates, **When** I combine them with a business ID, **Then** all predicates apply together using AND semantics, including totals and variable-enriched results.
3. **Given** a business ID containing commas, numeric-looking text, mixed case, whitespace, or Unicode, **When** I search, **Then** the entire supplied string is one opaque value, without trimming, splitting, normalization, key interpretation, or process-variable interpretation.
4. **Given** an omitted business ID, **When** I run an existing search, **Then** existing behavior is preserved; an explicitly empty business ID is rejected instead of becoming an unfiltered search.
5. **Given** a BID-only search or a search combined with direct-incident filtering, **When** any ordinary or optimized search route is used through either the command or public library interface, **Then** the business-ID predicate remains effective and cannot be treated as an absent filter.
6. **Given** duplicate records, sparse intermediate pages, paging, limits, or enrichment, **When** I retrieve matching work, **Then** continuation, ordering, totals, and documented limit semantics remain consistent with the existing command, without silently broadening the search or inventing a complete result.
7. **Given** a user task whose stored business ID differs from its process instance's current ID, **When** I filter user tasks, **Then** matching follows the task's own backend semantics, including with variables shown; process-instance identity is not substituted.
8. **Given** explicit keys or another existing exclusive selection mode, **When** I also supply a BID search selector, **Then** the unsupported combination is rejected before execution rather than choosing one selector silently.

---

### User Story 2 - Walk One Unambiguous Process Family (Priority: P1)

As an operator investigating a business reference, I want to start a process walk from a uniquely matched instance so that I can inspect its family without accidentally inspecting an arbitrary match.

**Why this priority**: A business ID is not a guarantee of uniqueness, and walking the wrong instance can mislead subsequent operational decisions.

**Independent Test**: Resolve zero, one, and multiple distinct matching instances with isolated discovery pages, then inspect walk invocation counts and existing ancestry, descendant, family, flat, tree, and enriched views.

**Acceptance Scenarios**:

1. **Given** completed discovery with exactly one matching instance in tenant scope, **When** I use `walk pi --bid <value>`, **Then** that instance becomes the starting instance and existing walk scope and enrichment options remain available.
2. **Given** two distinct matches, even in the same family, **When** I request a walk, **Then** selection fails as ambiguous before walking; no first-match or common-root shortcut establishes uniqueness.
3. **Given** one distinct instance repeated across pages, **When** discovery completes, **Then** repeated observations do not make the selection ambiguous.
4. **Given** a limit, interruption, timeout, or failure leaves discovery incomplete, **When** only one instance has been observed, **Then** a unique match is not claimed and walking does not start.
5. **Given** authoritative completed discovery with no matches, **When** I request a walk, **Then** the command returns its successful empty-result representation without traversing any family.
6. **Given** an explicit starting key without a business-ID filter, **When** I walk, **Then** existing explicit-key authorization and scope behavior remain unchanged.

---

### User Story 3 - Apply Existing Process Actions to a Safe Selection (Priority: P1)

As an operator, I want to select process instances by business ID for existing actions so that I can act on the complete intended scope and review its impact before submission.

**Why this priority**: Selection errors in mutations can affect unrelated business work or omit intended instances.

**Independent Test**: Exercise each mutation below against isolated matching pages and action responses, including duplicates, family expansion, documented limits, empty results, discovery failure, confirmation, dry-run, and partial action failure; verify selected and submitted keys.

**Acceptance Scenarios**:

1. **Given** several matching instances, **When** I select by BID for any covered collection action, **Then** all selected instances within the documented limit are considered, using the command's existing execution behavior rather than choosing only the first match.
2. **Given** completed discovery, **When** a mutation requires confirmation, **Then** the selected set and expanded impact are frozen before confirmation and submission; new matches appearing later do not silently join that run.
3. **Given** related selected instances or duplicate records across pages, **When** the action expands scope, **Then** selected keys and expanded roots are deduplicated and overlapping scope does not submit duplicate actions.
4. **Given** selected instances whose relatives have different IDs or none, **When** existing family or explicit tree expansion includes those relatives, **Then** the preview distinguishes BID-selected seeds from expanded impact and retains each observed resource's own ID or absence.
5. **Given** a failed, interrupted, or unexpectedly truncated discovery, **When** some or no matches have been observed, **Then** no mutation is submitted, the original failure is retained, and the result is not presented as a complete selection or successful no-op.
6. **Given** completed discovery with zero matches, **When** normal or dry-run execution is requested, including auto-confirm, automation, or no-wait where supported, **Then** no confirmation or mutation occurs and no accepted or pending mutation is reported.
7. **Given** a BID dry-run, **When** discovery is needed to describe scope, **Then** reads may occur but no mutation occurs; existing local exact-key previews keep their current request behavior.
8. **Given** successful submissions followed by failures, **When** execution ends, **Then** partial results and the original error classification remain available, without claiming complete success; no-wait preserves existing acceptance-versus-verification semantics.

#### Required Process Command Coverage

| Command | Selection and execution contract |
| --- | --- |
| `get pi` | Native process-instance search, combined filters, paging, totals, and existing enrichment |
| `walk pi` | Exactly one proven starting instance; existing parent, children, full-family, flat/tree, and enrichment choices |
| `cancel pi` | Complete selected collection within documented limits; existing family expansion and cancellation behavior |
| `delete pi` | Complete selected collection within documented limits; existing expansion, preconditions, and deletion behavior |
| `update pi` | Existing variable-update workflow and variable scope for all selected instances |
| `resolve pi` | Existing resolution workflow and execution scope for the selected instances |
| `suspend pi` | Exact selected instances by default; existing tree expansion only when explicitly requested |
| `resume pi` | Exact selected instances by default; existing tree expansion only when explicitly requested |

---

### User Story 4 - Narrow Operational Work and Preserve Its Evidence (Priority: P2)

As an operator running maintenance or analysis, I want to narrow existing process discovery by business ID and retain that selection in reports so that I can explain which work was considered and why.

**Why this priority**: Operational workflows have separate selection and reporting paths that must remain consistent with interactive searches.

**Independent Test**: Run each listed operation using isolated discovery and outcome fixtures, inspect frozen selections and previews, and compare supported JSON and Markdown reports before and after serialization.

**Acceptance Scenarios**:

1. **Given** process-instance repair, retention-policy execution, orphan-process-instance purge, or slow-process analysis, **When** I supply a business ID through the command or its public library entry point, **Then** the operation applies that predicate to existing process-instance discovery together with its other supported predicates and eligibility rules.
2. **Given** slow-process analysis, **When** I supply only a BID selector or combine BID with one existing definition selector, **Then** discovery is valid and uses the combined scope; existing mutually exclusive definition selectors and explicit-key/search exclusivity remain enforced.
3. **Given** any listed operation, **When** selection is carried into previews, frozen scope, filter descriptions, and supported JSON or saved Markdown reports, **Then** the exact business-ID criterion and selected-versus-expanded provenance survive without being confused with observed resource IDs.
4. **Given** documented limits, **When** operations select work across pages, **Then** limits retain their documented unit: repair and retention cap matching instances frozen for work, orphan purge caps matching child instances inspected, and slow analysis caps matching instances selected for analysis; page size is not a total-scope limit.
5. **Given** sparse or duplicate pages, empty completed discovery, or incomplete discovery, **When** an operational workflow runs, **Then** the same continuation, deduplication, no-op, and pre-mutation safety rules apply as for process commands.

---

### User Story 5 - Keep Version and Automation Contracts Dependable (Priority: P2)

As a script author or operator, I want clear version failures and stable results, prompts, and diagnostics so that adding a BID selector does not make existing automation unreliable.

**Why this priority**: The feature spans queries and mutations whose existing output and failure contracts are relied upon by users.

**Independent Test**: Execute affected command paths with isolated dependencies for versions 8.8, 8.9, and 8.10, capture stdout and stderr independently, and check configured logger destinations, supported output modes, and terminal confirmation behavior.

**Acceptance Scenarios**:

1. **Given** Camunda 8.8 or 8.9, **When** any affected command or public library entry point receives a business-ID filter, **Then** it fails clearly before mutation and never runs an equivalent search with the predicate discarded; existing non-BID workflows and the default version remain unchanged.
2. **Given** a shared-contract JSON command, **When** success, empty completion, validation failure, incomplete discovery, or partial mutation failure occurs, **Then** stdout contains exactly one envelope with established payload and absence conventions, preserved partial evidence and error class, and no trailing diagnostic output. Quiet plus JSON still emits the requested envelope.
3. **Given** keys-only output, **When** results exist, **Then** stdout contains one key per line; with no keys it contains zero bytes. Totals remain counts for the filtered scope, and existing mode precedence is unchanged.
4. **Given** human mutation, dry-run, or verbose outcome output, **When** the command runs, **Then** informational summaries use the configured logger, respect inherited destinations and plain or structured formats, leave human result stdout empty, and are suppressed under quiet or a warning-level threshold.
5. **Given** repeated backend failures containing raw transport details, **When** normal, quiet, or verbose-only human output is selected, **Then** one compact aggregate error retains useful counts and bounded reasons without exposing methods, URLs, bodies, or wrapped transport chains on either stream. Redacted DEBUG details remain available independently through the debug flag or configured debug level.
6. **Given** interactive terminal input with separately captured output, **When** confirmation is eligible for a nonempty selection, **Then** the existing plain prompt uses configured stderr and retains its wording, input handling, and abort behavior; empty completed selections never prompt.
7. **Given** help, examples, README guidance, and generated CLI documentation, **When** I review BID selection, **Then** I can identify exact-match behavior, version support, tenant scope, multiple-match handling, per-command expansion, limits, dry-run discovery, and the explicit exclusions below.

### Edge Cases

- A numeric-looking or comma-containing BID remains one string; explicit empty input cannot accidentally select all work.
- The same business ID may match multiple instances, definitions, or tenants. Configured tenant scope applies during discovery, before explicit-key execution rules can be relevant.
- Repeated observations of one key differ from two distinct keys sharing one BID; two distinct matches in one family still make `walk pi` ambiguous.
- An empty intermediate page with continuation is not end-of-discovery. Errors after earlier matches do not authorize mutation.
- An intentional documented collection limit defines a bounded selection, not a claim that all global matches were found. It cannot establish uniqueness for `walk pi`.
- Selected child instances may expand to the same root; deduplication must not erase selection provenance or duplicate effects.
- Relatives included by expansion may have different business IDs or none. Selection criteria never become fabricated observation data.
- A task's stored business ID may differ from the current process ID; native task filtering and variable enrichment must retain that distinction.
- A changed business ID or newly matching instance after confirmation does not silently replace the frozen selected keys. Existing action-time authorization, state checks, and failure handling still apply.
- A user abort, missing report, or partial failure is not successful empty discovery. Dry-run and no-wait do not relax discovery safety.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Business-ID search and selection MUST be available only on Camunda 8.10 and later versions already supported by c8volt. Unsupported versions MUST reject supplied BID filters through command and public library entry points without silently ignoring them or submitting mutations.
- **FR-002**: Covered commands MUST accept `--bid <value>` and corresponding public library inputs MUST represent the same optional criterion. Omission MUST preserve existing behavior; explicit empty input MUST fail validation rather than become omission.
- **FR-003**: A business ID MUST be matched as one exact opaque string, without trimming, comma splitting, case normalization, numeric-key conversion, or variable-filter interpretation.
- **FR-004**: Process-instance and user-task searches MUST use their native business-ID matching semantics and combine all supported predicates with AND semantics within configured tenant scope.
- **FR-005**: BID predicates MUST remain effective through all supported search representations and execution routes, including BID-only search, filter descriptions and serialization, direct-incident optimization at both command and public library boundaries, paging, totals, limits, and enrichment.
- **FR-006**: User-task search MUST match the task-level business ID and preserve combined predicates, totals, and variable-enriched results without substituting the current process-instance ID.
- **FR-007**: Every command in the required process-command coverage table MUST apply BID selection through its actual execution path. Existing explicit-key/search and other selection-mode exclusivity MUST remain enforced.
- **FR-008**: BID discovery MUST remain tenant-scoped. Existing explicit-key authorization behavior MUST be preserved for explicit-key input and MUST NOT be applied prematurely to bypass tenant restrictions during BID discovery.
- **FR-009**: Collection commands MUST consider the complete selected set within documented limits, not arbitrarily select the first match. Existing ordering and the documented unit of each limit MUST remain intact.
- **FR-010**: Discovery MUST continue through sparse pages with continuation and deduplicate selected keys and expanded roots across pages without discarding meaningful provenance.
- **FR-011**: `walk pi` MUST start only after completed discovery establishes exactly one distinct matching instance. Multiple matches MUST fail as ambiguous even within one family; truncated, failed, or interrupted discovery MUST NOT establish uniqueness.
- **FR-012**: BID-selected execution MUST preserve existing command scope, including applicable family expansion, exact-instance suspend/resume defaults, explicitly requested tree expansion, and existing variable-update scope.
- **FR-013**: Mutation selections and expanded impact MUST be frozen before confirmation and submission. Incomplete discovery MUST NOT submit mutations or be represented as a complete selection. A documented collection limit may define a complete bounded scope but MUST NOT be represented as exhaustive global discovery.
- **FR-014**: Previews and reports MUST distinguish selected seeds from expanded impact, retain the BID criterion used, and preserve each resource's own observed ID or absence without inventing facts from the selector.
- **FR-015**: Completed empty discovery MUST return successful mode-appropriate empty output without confirmation or mutation. It MUST NOT report acceptance or pending work, including under no-wait; dry-run MUST remain a preview.
- **FR-016**: BID dry-run MAY perform necessary discovery but MUST NOT mutate. Existing local exact-key previews MUST retain their current behavior, and rendering MUST NOT add discovery requests.
- **FR-017**: Process-instance repair, retention-policy execution, orphan-process-instance purge, and slow-process analysis MUST support BID filtering through all command and public library request paths while preserving each operation's eligibility rules and documented selection limits.
- **FR-018**: Slow-process analysis MUST accept BID-only discovery and BID combined with one supported definition selector, preserving existing definition-selector and explicit-key/search exclusivity.
- **FR-019**: Operations MUST retain the exact BID selection through frozen scope, previews, filter descriptions, and supported JSON and Markdown reports, including report round trips and alternate public entry paths.
- **FR-020**: Existing dry-run, confirmation, automation, quiet, keys-only, JSON, partial-result, and no-wait contracts MUST remain intact. Discovery and action failures MUST preserve their original classification and available partial evidence without claiming complete success.
- **FR-021**: Shared-contract JSON paths MUST emit exactly one envelope for success, empty results, and failures, preserving established shapes and null-versus-empty conventions. Keys-only MUST emit one key per line or zero bytes when empty; quiet MUST NOT suppress explicitly requested machine results.
- **FR-022**: Human mutation summaries, dry-run summaries, verbose outcomes, warnings, and errors MUST use configured logger routing and levels. Query results remain selected-format stdout results; interactive prompts remain plain configured stderr text with established eligibility and input behavior.
- **FR-023**: Human failures at normal level or with verbose alone MUST use compact aggregate diagnostics without raw transport details. Existing redacted DEBUG diagnostics MUST remain available independently of verbose and MUST NOT contaminate result stdout.
- **FR-024**: Help, examples, README guidance, and generated CLI documentation MUST describe BID selection, version support, tenant scope, limits, ambiguity, expansion, discovery during dry-run, and exclusions.

### Key Entities *(include if feature involves data)*

- **Business-ID criterion**: One optional opaque business reference used for exact search, distinct from explicit keys and from an observed resource's business ID.
- **Search scope**: The BID criterion, configured tenant scope, other supported predicates, ordering, and documented selection limits.
- **Discovery outcome**: Distinct observed matches and whether the intended scope completed, reached a documented collection bound, or remained incomplete due to truncation, interruption, or failure.
- **Frozen selection**: The selected instance keys and selection provenance retained for a particular operation before confirmation and submission.
- **Expanded impact**: Additional instances or roots included by existing family or explicitly requested tree behavior, distinguished from directly matched seeds.
- **Selection report**: Existing preview or result evidence retaining the criteria used, selected-versus-expanded scope, observed resource facts, and partial outcomes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All eight listed process commands, user-task search, and all four listed operations honor the supplied business ID in every applicable acceptance scenario; zero paths silently broaden scope by dropping the predicate.
- **SC-002**: Across exact-value, combined-filter, tenant, paging, and enrichment cases, 100% of directly selected records satisfy the applicable resource-level criteria, and no unrelated tenant's record enters selection.
- **SC-003**: In every incomplete-discovery mutation case, zero mutations are submitted. Every ambiguous or incompletely established walk selection starts zero walks, including two matches within one family.
- **SC-004**: For every completed collection scenario, the selected set equals the distinct intended matches within the documented bound; overlapping expansion produces no duplicate actions and reports distinguish seeds from expanded impact.
- **SC-005**: Every successful empty-scope case produces zero confirmation prompts and zero mutations. Supported machine-output cases produce exactly one result envelope or the documented key/count output, with zero diagnostic records on stdout.
- **SC-006**: All supported report round trips preserve the exact BID criterion and selection provenance; no observed resource ID is fabricated from selection input or a related resource.
- **SC-007**: Every BID attempt on 8.8 or 8.9 fails clearly before mutation, while representative non-BID workflows and explicit-key previews retain existing behavior.
- **SC-008**: Published guidance covers each command and operation in scope, enabling operators to determine support, matching, ambiguity, limits, and expansion without reading source code.

## Assumptions

- This feature depends on [business-ID creation and complete resource views, issue #360](../360-business-id-creation-views/spec.md). The issue's references to “issue 1” mean that prerequisite. Issues #203 and #239 are not dependencies.
- `--bid` follows the existing creation flag spelling. It represents one value, not a list or pattern. Explicit empty input is rejected as a safety default; creation-specific length rules are not automatically added to search by this specification.
- “Later supported versions” does not add a new supported Camunda version or change the configured default version.
- Existing tenant configuration, authorization, ordering, safety rules, wait behavior, and command-specific limits remain authoritative unless explicitly changed above. BID adds a selector, not a new operation or permission model.
- The issue identifies no native BID filter for jobs, incidents, or elements in the pinned backend. Direct BID selectors for those commands and incident-selected operations are excluded; existing process-key selectors remain available.
- Process-definition deletion and global definition purge do not gain BID filtering. Their observed process-instance impact views belong to the prerequisite feature.
- Late assignment of a business ID, assignment during job completion, new generic selector frameworks, and unrelated command refactoring are out of scope.
- Acceptance validation uses isolated service and command tests, test-owned fake pages/responses, and terminal/subprocess fixtures. It covers actual command execution, direct-incident routes, BID-only slow analysis, report round trips, and repeated backend failures. No live environment discovery, provisioning, or validation is an implementation completion gate.
- Output validation captures stdout and stderr separately; checks configured and inherited logger destinations, plain and structured formats, quiet and warning-level suppression, and DEBUG via both supported configuration paths; decodes one JSON envelope followed by EOF; and uses real terminal stdin for prompt and empty-scope checks. Supported mode combinations and existing abort/error regressions remain part of coverage.
