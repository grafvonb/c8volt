# Feature Specification: Process-Instance Dry Run Scope Preview

**Feature Branch**: `138-pi-dry-run-scope`
**Created**: 2026-04-25
**Status**: Draft
**Input**: User description: "GitHub issue #138: feat(cmd): add --dry-run to cancel/delete process-instance with family-aware scope and orphan-parent handling"

## GitHub Issue Traceability

- **Issue Number**: 138
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/138
- **Issue Title**: feat(cmd): add --dry-run to cancel/delete process-instance with family-aware scope and orphan-parent handling

## Clarifications

### Session 2026-04-25

- No critical ambiguities detected worth formal clarification. The GitHub issue defines the affected commands, dry-run mutation boundary, scope-resolution behavior, orphan-parent handling, output requirements, documentation updates, and required test coverage.
- Q: How should structured search-mode dry-run output represent previews across pages? -> A: Aggregate summary plus nested per-page previews.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Preview Keyed Destructive Scope (Priority: P1)

As a CLI operator, I want `cancel process-instance --dry-run` and `delete process-instance --dry-run` to preview the same resolved scope that real execution would use for a direct `--key` request so that I can inspect parent escalation and affected family members before any destructive action is submitted.

**Why this priority**: Direct key workflows are the highest-risk destructive path because a child key can expand to a parent/root and then to a wider process-instance family.

**Independent Test**: Run both commands with `--key` and `--dry-run` against fixtures that include child-to-root escalation and full-family expansion, then verify the previewed roots and affected keys match the real command preflight scope while no mutation or wait occurs.

**Acceptance Scenarios**:

1. **Given** a requested child process instance whose real cancellation would escalate to a root, **When** the user runs `cancel process-instance --key <child> --dry-run`, **Then** the output reports selected process instances, process-instance trees to cancel, process instances in scope, and that no cancellation was submitted.
2. **Given** a requested child process instance whose real deletion would select a root for family deletion, **When** the user runs `delete process-instance --key <child> --dry-run`, **Then** the output reports the same root and family members that real deletion would affect.
3. **Given** a requested root process instance with multiple descendants, **When** either command runs with `--dry-run`, **Then** the output reports the root and all affected family members without prompting for confirmation or polling for completion.

---

### User Story 2 - Preview Search-Based and Paged Scope (Priority: P2)

As a CLI operator or automation author, I want search-based dry runs to use the same per-page scope calculation as destructive preflight so that previewed counts and keys stay accurate for paged selections.

**Why this priority**: Search-mode commands can select many process instances, and operators need a safe way to preview the expanded destructive scope before allowing action.

**Independent Test**: Run search-based `cancel process-instance --dry-run` and `delete process-instance --dry-run` across multiple pages and compare requested counts, resolved roots, and affected family keys with the existing preflight scope calculation.

**Acceptance Scenarios**:

1. **Given** a search matches process instances across more than one page, **When** `cancel process-instance --dry-run` runs, **Then** each page uses the same dependency expansion as real cancellation preflight and the final preview reports the total requested, root, and affected counts.
2. **Given** a search matches roots and children in the same result set, **When** `delete process-instance --dry-run` runs, **Then** the previewed deletion roots and affected family members match real deletion scope calculation.
3. **Given** structured output is requested for a search-mode dry run spanning multiple pages, **When** the command completes, **Then** it returns an aggregate summary with nested per-page previews.
4. **Given** a search-mode dry run completes, **When** the command exits, **Then** it has not submitted cancel requests, delete requests, confirmation prompts, or wait polling.

---

### User Story 3 - Preserve Orphan-Parent Warning Behavior (Priority: P3)

As a CLI operator investigating process-instance dependencies, I want dry runs to preserve current orphan-parent handling so that partial but actionable scopes can still be inspected safely.

**Why this priority**: Missing parent instances are already a known operational condition; dry run must not become stricter or hide partial traversal warnings.

**Independent Test**: Run keyed and search-based dry runs against fixtures with missing ancestors and verify partial scopes, warning text, missing ancestor keys, and failure behavior match current dependency-expansion/preflight semantics.

**Acceptance Scenarios**:

1. **Given** one or more parent process instances are missing but actionable roots or family members are resolved, **When** a dry run executes, **Then** it returns the resolved partial scope and marks the scope as partial.
2. **Given** missing ancestor keys were discovered during traversal, **When** human-readable output is rendered, **Then** the warning and missing ancestor keys are shown clearly.
3. **Given** dependency expansion resolves no actionable roots or family members, **When** a dry run executes, **Then** it fails consistently with the current unresolved/orphan handling.

---

### User Story 4 - Consume Dry-Run Results in Human and Structured Output (Priority: P4)

As a CLI user or automation author, I want dry-run output to expose requested keys, resolved roots, affected family members, traversal status, and warnings so that humans and scripts can decide whether to run the destructive command.

**Why this priority**: The feature is only safe and useful if its preview communicates the planned scope clearly in both output modes.

**Independent Test**: Run dry runs in human-readable and structured output modes, then verify required counts, keys, traversal outcome, warning text, and missing ancestor keys are present and consistent.

**Acceptance Scenarios**:

1. **Given** a complete resolved scope, **When** human-readable dry-run output is rendered, **Then** it shows selected process instance count, process-instance tree count for the operation, total process instances in scope, and that the scope is complete.
2. **Given** a partial resolved scope, **When** structured output is rendered, **Then** it includes requested keys, resolved roots, collected family keys, traversal outcome, warning text, and missing ancestor keys.
3. **Given** a user reads command help or docs, **When** they inspect cancel/delete process-instance usage, **Then** `--dry-run` is documented with examples that make clear no destructive action is submitted.

### Edge Cases

- A requested child process instance must report the same parent/root escalation that real execution would apply.
- A requested root with descendants must report the full affected family scope.
- Search-mode dry run must aggregate scope across all selected pages using the same preflight expansion behavior as real execution.
- Structured search-mode dry run must expose an aggregate summary plus nested per-page previews so automation can inspect totals and page-level traversal outcomes.
- Missing ancestors with a partial actionable scope must produce a partial preview rather than a hard failure.
- Missing ancestors with no actionable scope must fail consistently with existing unresolved/orphan behavior.
- Dry run must not ask for confirmation, submit cancel/delete mutations, or wait for completion status.
- Dry run must work with direct `--key` usage and search-based/paged selection.
- Dry run must preserve existing output mode behavior, including structured output for automation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose a documented `--dry-run` flag on `cancel process-instance`.
- **FR-002**: The system MUST expose a documented `--dry-run` flag on `delete process-instance`.
- **FR-003**: Dry run MUST use the same ancestry, root, family, and dependency-expansion logic as the matching real command path.
- **FR-004**: Dry run MUST report the same parent/root escalation that real execution would apply for requested child process instances.
- **FR-005**: Dry run MUST report the same affected process-instance family scope that real execution would affect.
- **FR-006**: `delete process-instance --dry-run` MUST use the same root-selection logic as real delete execution.
- **FR-007**: Search-mode dry run MUST use the same per-page scope calculation already used by destructive command preflight.
- **FR-008**: Dry run MUST support direct `--key` usage.
- **FR-009**: Dry run MUST support search-based and paged selection.
- **FR-010**: Dry run MUST NOT submit cancel requests.
- **FR-011**: Dry run MUST NOT submit delete requests.
- **FR-012**: Dry run MUST NOT perform confirmation polling or wait for completion.
- **FR-013**: Dry run MUST preserve current orphan-parent behavior for partial traversal results.
- **FR-014**: If one or more parent process instances are missing but actionable roots or family members are resolved, dry run MUST return the resolved partial scope instead of failing.
- **FR-015**: If dependency expansion resolves no actionable roots or family members, dry run MUST fail consistently with current unresolved/orphan handling.
- **FR-016**: Human-readable output MUST clearly show selected process instance count, process-instance tree count for the operation, total process instances in scope, count of selected process instances already in final state when applicable, delete-only count of in-scope process instances not in final state, and whether the resolved scope is complete or partial.
- **FR-017**: Human-readable output MUST clearly show warnings and missing ancestor keys when applicable.
- **FR-018**: Structured output MUST expose requested keys, resolved roots, collected family keys in scope, keys and states for selected instances already in final state, keys and states for delete instances requiring cancellation before delete, traversal outcome, warning text, and missing ancestor keys.
- **FR-019**: Structured output for search-mode dry run spanning multiple pages MUST expose an aggregate summary plus nested per-page previews.
- **FR-020**: Tests MUST cover keyed flows for both cancel and delete dry run.
- **FR-021**: Tests MUST cover search-based and paged flows for both cancel and delete dry run.
- **FR-022**: Tests MUST cover child-to-root escalation cases.
- **FR-023**: Tests MUST cover full-family scope cases.
- **FR-024**: Tests MUST cover partial orphan-parent resolution cases.
- **FR-025**: CLI docs and README examples MUST be updated to document dry-run usage and non-mutating behavior.

### Key Entities *(include if feature involves data)*

- **Dry-Run Request**: The user command invocation that asks for a destructive process-instance scope preview without submitting mutations.
- **Requested Process Instance Key**: A key selected directly or through search before dependency expansion.
- **Resolved Root**: A root or parent process instance that real execution would use as the action target after escalation.
- **Affected Family Member**: A process instance included in the resolved family scope that real execution would cancel or delete.
- **Traversal Outcome**: The scope completeness status, including complete, partial, or unresolved failure.
- **Missing Ancestor Key**: A parent key referenced during traversal that cannot be loaded but must be surfaced in warnings and structured output.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `cancel process-instance --dry-run --key <child>` reports the same resolved root and affected family as real cancellation preflight while submitting zero cancel requests.
- **SC-002**: Automated tests show `delete process-instance --dry-run --key <child>` reports the same resolved root and affected family as real deletion preflight while submitting zero delete requests.
- **SC-003**: Automated tests show search-based dry runs across multiple pages report the same requested counts, resolved roots, and affected family keys as the existing destructive preflight scope calculation.
- **SC-004**: Automated tests show dry run performs no confirmation polling or wait behavior.
- **SC-005**: Automated tests show partial orphan-parent traversal returns a partial scope, warning text, and missing ancestor keys when actionable roots or family members exist.
- **SC-006**: Automated tests show unresolved orphan traversal fails when no actionable roots or family members can be resolved.
- **SC-007**: Structured output tests verify requested keys, resolved roots, family keys, traversal outcome, warning text, and missing ancestor keys are present.
- **SC-008**: Structured search-mode dry-run tests verify aggregate totals and nested per-page previews are present when multiple pages are processed.
- **SC-009**: Help output, generated CLI docs, and README examples document `--dry-run` for both cancel and delete process-instance commands.

## Assumptions

- Existing dependency-expansion and preflight logic remains the source of truth for destructive process-instance scope calculation.
- Existing orphan-parent warning semantics from current traversal/preflight flows must be reused rather than redefined.
- The affected command paths are `cancel process-instance` and `delete process-instance`, including their aliases where the repository documents aliases.
- Dry run is a preview mode only; it does not change the destructive command confirmation model outside bypassing mutation and wait behavior for the preview.
- Documentation generated from command metadata should be regenerated through the repository's existing docs path after command metadata changes.
