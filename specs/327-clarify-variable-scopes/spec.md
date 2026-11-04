# Feature Specification: Clarify Variable Destination Scopes

**Feature Branch**: `codex/327-clarify-variable-scopes`

**Created**: 2026-10-11

**Status**: Draft

**Input**: [GitHub issue #327](https://github.com/grafvonb/c8volt/issues/327) — Clarify variable destination scopes in `update pi` and `update ut` previews.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Identify Each User-Task Write Destination (Priority: P1)

As an operator reviewing a user-task variable update, I want to see where every addition and change will be written so I can understand effects on inherited data before confirming.

**Why this priority**: Changing an inherited value can affect work outside the selected task; a bare scope key does not distinguish process-root inheritance from intermediate-element inheritance.

**Independent Test**: Preview one task with process-root, intermediate-element, and task-local variables, including a missing name, and identify every write destination without performing the update.

**Acceptance Scenarios**:

1. **Given** a task inheriting `customerTier` from process instance 123, **When** a change is previewed, **Then** the entry identifies `inherited, pi:123` and preserves the old-to-new value display.
2. **Given** a task inheriting `approvalLimit` from intermediate element 456, **When** a change is previewed, **Then** the entry identifies `inherited, element:456` without guessing the element's business-process type.
3. **Given** task-local scope 789, **When** an existing local variable changes or a missing name is added, **Then** each entry identifies `local, element:789`; a new name is not presented as a process-root write.
4. **Given** the same planned changes, **When** an operator compares the normal confirmation preview with `--dry-run`, **Then** scope notation and local/inherited distinctions agree, dry-run applies no changes, and existing summary wording is preserved.

Illustrative entries, retaining established change and addition tokens:

```text
~ customerTier (inherited, pi:123): "silver" -> "gold"
~ approvalLimit (inherited, element:456): 100 -> 200
+ incident (local, element:789): "buba"
```

---

### User Story 2 - Identify Process-Instance Root Writes (Priority: P1)

As an operator updating process-instance variables, I want every detailed addition and change to identify the selected process-instance root so the destination is explicit in either update workflow.

**Why this priority**: Both commands need a consistent vocabulary for scopes without suggesting that their write-selection behavior has changed.

**Independent Test**: Preview a process-instance update containing one changed variable and one addition, through normal confirmation and dry-run.

**Acceptance Scenarios**:

1. **Given** selected process instance 123, **When** additions and changes are shown in a single-resource preview, **Then** every planned write is labeled `pi:123` and retains the existing value and change notation.
2. **Given** multiple selected process instances, **When** verbose previews are requested, **Then** every detailed addition and change identifies its own selected process-instance root in both normal and dry-run previews.

---

### User Story 3 - Review Bulk Updates Without Losing Compatibility (Priority: P2)

As an operator or automation author, I want bulk previews to remain compact while exposing inherited destinations, and existing scripts to keep receiving the same results.

**Why this priority**: Scope clarity must not make bulk review noisy or alter machine-readable contracts and execution controls.

**Independent Test**: Preview multiple tasks sharing inherited destinations, compare ordinary and verbose human output, and exercise existing machine-output and confirmation combinations.

**Acceptance Scenarios**:

1. **Given** several tasks with inherited planned changes, **When** an ordinary bulk preview is shown, **Then** existing summaries remain compact, inherited destinations use `pi:<key>` or `element:<key>`, repeated destinations retain existing deduplication and ordering, and no per-variable details are added by default.
2. **Given** the same bulk selection with `--verbose`, **When** the plan is shown, **Then** every addition and change includes its destination and task-local or inherited relationship as applicable.
3. **Given** any supported JSON, keys-only, quiet, or combined output mode, **When** preview or execution completes, **Then** existing output precedence and contracts are preserved, including no added scope fields or human text in machine results and zero-byte keys output for no work.
4. **Given** a confirmed, declined, unattended, no-wait, failed, or no-op update, **When** the command completes, **Then** confirmation behavior, prompt streams, mutation destinations, result statuses, and errors match existing behavior.
5. **Given** equivalent selections and variable data before and after the change, **When** previews are produced, **Then** destination labeling adds zero backend requests.
6. **Given** command help and user documentation, **When** operators look up preview behavior, **Then** affected guidance consistently explains destination notation and inherited versus local writes.

### Edge Cases

- Tasks may inherit the same variable name from different scopes; each detailed entry must identify its actual destination rather than classify it by name alone.
- Multiple tasks can share an inherited destination; compact output must retain existing stable unique destination reporting without implying additional mutations.
- An element scope is not necessarily the task's own scope. Intermediate inherited scopes use `element:<key>` with an inherited distinction; task-local writes use the same scope prefix with a local distinction.
- A plan containing unchanged or untouched values must not present them as planned writes. Existing no-op counts, wording, and prompt-free behavior remain intact.
- Empty payloads, lookup failures, declined confirmation, and dry-run remain distinguishable from accepted mutations; adding scope labels must not change their outcomes.
- Scope labels reflect the collected plan, not a new lookup or a guarantee against concurrent changes. Existing scope-resolution failures must not be hidden by fabricated labels.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Single-resource and verbose human previews for both update commands MUST identify the destination of every planned variable addition and change, in normal confirmation previews and dry-run previews.
- **FR-002**: Process-instance root destinations MUST use `pi:<key>`. Task-local and intermediate-element destinations MUST use `element:<key>`.
- **FR-003**: User-task detailed entries MUST preserve the distinction between local and inherited variables and distinguish process-root inheritance, intermediate-element inheritance, and task-local writes.
- **FR-004**: Process-instance previews MUST identify the selected process-instance root for every detailed write. Scope labels MUST NOT imply a change to variable-selection or write semantics.
- **FR-005**: Human previews MUST preserve existing summary wording, counts, change/addition tokens, value formatting, and compact bulk behavior. Bulk task previews MUST expose inherited destinations using the same scope notation; per-variable bulk detail remains available through `--verbose`.
- **FR-006**: Labels MUST use information already collected for planning, including known process-instance identity. Labeling MUST add zero backend requests and MUST NOT infer an intermediate element's business-process type.
- **FR-007**: Process-instance updates MUST continue updating or creating variables at the selected root. User-task updates MUST continue changing existing effective variables at their returned owning scopes and creating missing names at the task's element-instance scope.
- **FR-008**: Command grammar, flags, aliases, confirmation behavior and streams, output-mode precedence, result statuses, JSON schemas and contents, and keys-only contracts MUST remain unchanged. Scope clarification MUST NOT introduce new machine-output fields.
- **FR-009**: Dry-run, no-op, abort, error, and partial-result behavior MUST remain unchanged; a scope annotation MUST NOT claim that a previewed write was submitted or confirmed.
- **FR-010**: Affected command help, examples, README guidance, and generated CLI documentation MUST describe the delivered preview notation consistently.

### Key Entities

- **Variable update plan**: The selected resources and requested values classified as additions, changes, unchanged, or untouched, together with their known destinations.
- **Destination scope**: The identity of the owning process-instance root or element where a planned value will be written.
- **Variable relationship**: Whether a user-task variable belongs to the task's local element scope or is inherited from another scope.
- **Human preview**: A compact summary and, when appropriate, variable details that allow an operator to understand planned effects before mutation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In every acceptance case, operators can identify the destination of 100% of detailed additions and changes directly from the preview, including all three user-task scope relationships and process-instance root writes.
- **SC-002**: Normal and dry-run previews agree on 100% of destination labels for equivalent plans; bulk previews expose every distinct inherited write destination without enabling verbose detail.
- **SC-003**: Across compatibility cases, there are zero changes to mutation destinations, confirmation decisions, result statuses, or machine-readable results attributable to scope labeling.
- **SC-004**: Producing scope annotations requires zero additional backend requests for single-resource, bulk, verbose, and dry-run previews.
- **SC-005**: All affected help and documentation examples use the same scope notation as the previews, enabling operators to distinguish local and inherited effects without a separate scope-inspection step.

## Assumptions

- The existing update commands and their variable plans provide the compatibility baseline. Known owning process-instance identity can accompany the plan without exposing new machine-readable fields.
- Scope is limited to preview clarity. New commands, scope-selection flags, element-variable inspection, and changes to mutation or confirmation workflows are excluded.
- Exact placement of annotations follows established compact formatting; the scope prefixes and local/inherited distinction are required, while the issue's sample layout is illustrative.
- Implementation planning must preserve repository architecture boundaries: planning metadata stays with existing service and domain/facade ownership, and human formatting stays in command views. Package design belongs in the implementation plan.
- Targeted validation must cover human confirmation and dry-run previews, single and bulk selections, verbose output, request counts, and machine-output compatibility. Documentation is regenerated from command metadata where affected.
