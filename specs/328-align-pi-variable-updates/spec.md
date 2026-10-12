# Feature Specification: Align Process Instance Variable Updates

**Feature Branch**: `328-align-pi-variable-updates`

**Created**: 2026-10-11

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/328 — align update pi variable planning and execution with update ut"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Trust the complete preview (Priority: P1)

As an operator updating process instance variables, I want the preview to reflect all existing variables at each selected instance's root scope, so I can distinguish new values, changed values, and values that need no action before approving an update. The root scope is the variable scope belonging to the process instance itself, excluding variables local to nested elements.

**Why this priority**: An incomplete preview can misrepresent the intended update and undermine the operator's approval.

**Independent Test**: Preview a request against multiple selected instances whose existing variables span several result pages and include nested-scope duplicates. Verify classifications against the complete root-scope values and verify that no writes occur.

**Acceptance Scenarios**:

1. **Given** existing root variables across multiple pages, including capped or sparse intermediate pages with continuation available, **When** an operator previews requested values, **Then** all pages are considered before classifying additions, changes, unchanged requested values, and existing variables untouched by the request.
2. **Given** a variable name exists at both root and nested scopes, **When** its requested value is classified, **Then** only the selected instance's root value determines the classification, consistently with the established `update ut` value-comparison behavior.
3. **Given** an existing value is truncated, **When** it appears to match the requested value, **Then** that incomplete value alone cannot establish that the variable is unchanged.
4. **Given** several instances are selected and discovery or planning fails for any one of them, **When** the operation ends, **Then** the failure is reported and no instance receives a write, even if earlier instances were successfully planned.

---

### User Story 2 - Apply exactly the prepared changes (Priority: P1)

As an operator, I want an approved update to apply only the additions and changes shown for each instance, so already-correct values and unrelated variables are left alone.

**Why this priority**: The current behavior can submit unchanged values and write to instances that have no planned changes.

**Independent Test**: Select three instances, two already matching the requested values and one requiring an addition and a change. Verify that only the third instance receives a write and that its submitted values contain only its prepared addition and change.

**Acceptance Scenarios**:

1. **Given** a request includes changed and unchanged instances, **When** the prepared update is executed, **Then** only instances with planned additions or changes receive writes.
2. **Given** an instance's request contains a new variable, a changed variable, and an unchanged variable, **When** it is executed, **Then** the submitted values contain exactly the prepared addition and change, and no unchanged or untouched variable.
3. **Given** a prepared plan identifies destination root scopes and values, **When** the operator approves and the operation proceeds through execution and result verification, **Then** those destinations and values remain the basis of execution and verification; the original full request does not replace the prepared changes.
4. **Given** identically named variables exist in nested scopes, **When** root changes are applied, **Then** writes are local to the recorded process-instance root scopes and do not modify nested variables.
5. **Given** the operator declines the existing approval prompt, **When** execution stops, **Then** no writes occur and the result preserves established abort behavior rather than claiming a successful no-op.

---

### User Story 3 - Understand what actually happened (Priority: P2)

As an operator or automation consumer, I want results to distinguish accepted updates, verified values, unchanged instances, failures, and skipped work, so I can act on partial results without assuming every instance succeeded or failed together.

**Why this priority**: Accurate outcomes are necessary to decide whether follow-up is needed and to avoid mistaking submission for confirmed completion.

**Independent Test**: Exercise successful waiting, `--no-wait`, and a bulk operation with partial failure. Compare reported outcomes with submitted changes and complete root-scope observations.

**Acceptance Scenarios**:

1. **Given** changes are submitted normally, **When** the command waits for their outcome, **Then** it verifies requested values at the intended root scopes using complete reads through one verification workflow, with no redundant verification read after that workflow succeeds.
2. **Given** verification spans several pages or sees matching values only in a nested scope, **When** outcomes are determined, **Then** all root-scope pages are considered and nested matches cannot establish success.
3. **Given** verification returns a truncated value, **When** the command evaluates whether the requested value is present, **Then** truncation alone cannot establish a confirmed outcome.
4. **Given** `--no-wait` is selected, **When** the command executes, **Then** complete planning and selective writes still occur, no result-verification reads occur, and submitted work is reported as accepted rather than confirmed. Existing operator-approval rules still apply.
5. **Given** some submissions succeed and other execution or verification work fails or is skipped, **When** results are rendered, **Then** successful submissions remain represented and failed or skipped work remains identifiable under the existing outcome and exit conventions; the result does not imply an all-or-nothing transaction.

---

### User Story 4 - Safely preview or finish with nothing to change (Priority: P2)

As an operator or script author, I want dry-runs and successful no-ops to preserve the selected output format and finish without misleading status or unnecessary interaction.

**Why this priority**: A harmless update should neither block an unattended workflow nor contaminate machine-readable results.

**Independent Test**: Exercise complete discovery yielding no selected instances and complete planning yielding only unchanged values, in normal execution and dry-run, across supported output and automation options. Observe stdout and stderr separately, including real terminal input.

**Acceptance Scenarios**:

1. **Given** complete discovery or planning establishes there is no work, **When** normal execution or dry-run finishes, **Then** there are no mutation-approval prompts, writes, or mutation-verification reads, including when terminal input would otherwise make confirmation eligible.
2. **Given** a dry-run has planned changes or execution has no changes, **When** results are rendered with or without `--no-wait`, **Then** a preview remains a preview and neither outcome claims submitted, accepted, or pending mutations.
3. **Given** a successful empty result in JSON mode, including JSON combined with quiet mode, **When** output is captured, **Then** stdout contains exactly one successful shared envelope with the established empty payload and no additional text; existing field omission and null-versus-empty-array conventions remain intact.
4. **Given** keys-only output has no keys, **When** the command finishes, **Then** stdout contains zero bytes. Nonempty keys-only results retain exactly one key per line under the existing output contract.
5. **Given** human or quiet output, **When** a successful no-op is rendered, **Then** established human wording is preserved and quiet mode suppresses informational summaries without suppressing explicitly requested machine output.
6. **Given** an update requires operator interaction, **When** its existing prompt eligibility rules allow a prompt, **Then** the prompt uses configured or inherited stderr, and stdout remains exclusively command results. Redirected stdout does not change existing input eligibility, default answers, EOF, or abort behavior.

### Edge Cases

- An empty or short intermediate page with a continuation is not the end of discovery or verification.
- An empty selected scope and a selected instance with no existing root variables differ: requested variables for the latter are additions.
- A requested variable found only in a descendant scope is still absent from the selected root scope.
- Truncated values, including values with a matching visible prefix, cannot prove equality during planning or verification.
- Failure on the final selected instance during planning still prevents all writes.
- An all-unchanged selection must not become an accepted mutation merely because `--no-wait` is present.
- Discovery errors, absent reports, user aborts, and verification failures must not be relabeled as successful no-ops.
- Partial execution or verification failure must preserve evidence of already accepted or confirmed work.
- Supported auto-confirm, automation, quiet, machine-output, dry-run, and no-wait combinations preserve their existing precedence and interaction behavior.
- Explicit-key selection remains supported, including nonempty, unchanged, aborted, and failed operations.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The command MUST complete variable discovery for every selected process instance before classifying requested values, continuing through all available pages, including capped or sparse pages where supported.
- **FR-002**: Discovery and comparison MUST retain process-instance root-scope filtering and classify additions, changes, unchanged requested values, and untouched existing variables consistently with `update ut`.
- **FR-003**: Truncated values MUST NOT, by themselves, establish unchanged or confirmed outcomes.
- **FR-004**: Planning MUST finish successfully for every selected instance before any write is submitted. Any planning failure MUST prevent all writes.
- **FR-005**: Execution MUST submit exactly the prepared additions and changes for each recorded instance root scope, using local scope semantics. It MUST skip unchanged values and instances without planned changes and MUST leave untouched and descendant-scope variables unmodified.
- **FR-006**: Prepared destinations and values MUST remain authoritative through operator approval, execution, and outcome verification. The original input MUST NOT be used to reconstruct a broader mutation payload.
- **FR-007**: Normal execution MUST use one outcome-verification workflow based on complete reads of the intended root scopes and requested values. It MUST NOT perform a redundant confirmation read after successful waiting.
- **FR-008**: `--no-wait` MUST skip only outcome verification: it MUST perform no confirmation reads while preserving complete planning, selective execution, and existing operator-approval behavior.
- **FR-009**: Outcomes MUST preserve the established accepted, confirmed, unchanged, failed, and skipped distinctions, including partial successes and failures, without adding public fields or implying all-or-nothing execution.
- **FR-010**: A successful no-op MUST be based on completed authoritative discovery or planning. It MUST NOT prompt for mutation approval, submit writes, or perform mutation verification. Rendering MUST NOT introduce further discovery.
- **FR-011**: Dry-run MUST remain a preview without writes or claims of mutation submission. Neither dry-run nor successful no-op results may claim accepted or pending mutations, including with `--no-wait`.
- **FR-012**: Human, shared JSON-envelope, keys-only, quiet, and automation contracts MUST remain stable, including mode precedence, existing public fields, empty payload conventions, and established human empty-result wording. Empty JSON results MUST emit exactly one successful envelope; keys-only output with no keys MUST emit zero bytes.
- **FR-013**: Interactive control text MUST use configured or inherited stderr, with stdout reserved for the selected results. Existing prompt eligibility, wording, defaults, input handling, EOF, auto-confirm, automation, and abort behavior MUST remain compatible.
- **FR-014**: Existing command grammar, flags, aliases, payload syntax, explicit-key selection, and root-scope semantics MUST remain compatible. The behavior of `update ut` MUST remain unchanged.
- **FR-015**: Command help, examples, and user documentation MUST describe the corrected preview, selective-write, no-op, and outcome behavior consistently.

### Key Entities *(include if feature involves data)*

- **Selected process instance**: A target instance identified by its key and associated root variable scope.
- **Requested variable value**: A name and desired value supplied by the operator, compared with existing root-scope data.
- **Observed variable**: An existing name, value, scope, and completeness indication; a truncated observation is insufficient proof of equality.
- **Prepared update plan**: The per-instance destinations, intended values, and classifications that identify exactly which additions and changes may be submitted.
- **Update outcome**: The established report of accepted, confirmed, unchanged, failed, or skipped work, retaining partial results.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In all acceptance cases with complete readable values, operators receive the same classifications regardless of how existing variables are split across pages; no truncated observation is treated as proof of equality.
- **SC-002**: Every submitted variable and destination matches a prepared addition or change, with zero writes to unchanged instances or unrelated scopes.
- **SC-003**: Every planning-failure case results in zero writes across the entire selection.
- **SC-004**: Normal successful execution verifies the intended root values without a second post-success verification pass; every no-wait case performs zero outcome-verification reads.
- **SC-005**: Every partial-failure acceptance case allows operators to distinguish successful submissions from failed or skipped work without losing earlier successes.
- **SC-006**: Every successful no-op completes without prompts or writes, and every dry-run reports only a preview. Empty JSON output is exactly one valid envelope, and keys-only output without keys is exactly zero bytes.
- **SC-007**: All supported output and interaction acceptance cases preserve existing command usage and stream separation, so operators and scripts can adopt the corrected behavior without changing invocation syntax or consuming new result fields.

## Assumptions

- [Issue #328](https://github.com/grafvonb/c8volt/issues/328) is the authoritative scope. Its architectural constraints remain binding inputs to implementation planning; this specification describes the operator-visible behavior.
- Existing `update ut` classifications and existing exact-root-scope update capabilities provide the compatibility baseline. The work does not change user-task behavior.
- `update pi` remains explicit-key-only. Missing targets and empty stdin remain validation errors; successful empty-scope behavior does not introduce a new selector mode or turn invalid input into success.
- Existing keys-only dry-run output lists selected keys, including unchanged selections. Zero-byte keys output applies when the selected output contract has no keys to emit, including executed all-unchanged plans.
- No descendant-scope discovery, call-activity traversal, scope-selection flags, element-variable commands, or general workflow framework is introduced.
- Existing timeout, error, and exit conventions continue to apply. This feature does not promise atomic bulk updates, rollback, or protection against concurrent external changes between planning and execution.
- Complete discovery means following the backend's authoritative continuation behavior, not inferring completion from a sparse page or short page size.
- Acceptance validation must inspect submitted values, destinations, request counts, and partial outcomes through command execution paths. Interactive no-op validation uses real terminal stdin with stdout and stderr captured separately; mocked terminal eligibility alone is insufficient.
- The execution-path acceptance matrix includes normal execution and dry-run; human, JSON, keys-only, and quiet modes; quiet combined with machine output; and supported auto-confirm, automation, and no-wait combinations. Existing nonempty, sparse-page, explicit-key, abort, and error coverage remains necessary.
