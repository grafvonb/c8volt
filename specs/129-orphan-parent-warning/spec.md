# Feature Specification: Graceful Orphan Parent Traversal

**Feature Branch**: `129-orphan-parent-warning`  
**Created**: 2026-04-22  
**Status**: Draft  
**Input**: User description: "GitHub issue #129: fix(process-instance): treat orphan-parent tree traversal as warning in walk/delete/cancel flows"

## GitHub Issue Traceability

- **Issue Number**: 129
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/129
- **Issue Title**: fix(process-instance): treat orphan-parent tree traversal as warning in walk/delete/cancel flows

## Clarifications

### Session 2026-04-22

- Q: Should affected traversal and preflight flows expose machine-readable missing ancestor details or only a user-facing warning? → A: Return partial results plus machine-readable missing ancestor keys metadata and a user-facing warning in every affected traversal/preflight flow.
- Q: How should affected traversal and preflight flows behave when ancestors are missing but at least one actionable result was still resolved? → A: Return success for traversal and preflight flows when at least one actionable result is resolved, and warn about missing ancestors.
- Q: How should affected flows behave when no process-instance data can be resolved at all? → A: Return a normal failure instead of warning-based success when nothing is resolved.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect Partial Trees Safely (Priority: P1)

As a CLI operator, I want tree and family traversal to return the process instances that still exist even when an ancestor is missing so that I can inspect incomplete hierarchies without the whole command failing.

**Why this priority**: Read-only inspection is the fastest way to understand orphaned runtime state, and a hard failure prevents operators from seeing the surviving portion of the tree.

**Independent Test**: Run ancestry and family traversal against a process instance whose recorded parent no longer exists, then verify the command returns the reachable instances together with a warning about the missing ancestor boundary.

**Acceptance Scenarios**:

1. **Given** an investigated process instance exists and its recorded parent no longer exists, **When** the operator runs `walk pi --parent`, **Then** the command returns the resolved ancestry it could collect and a warning that one or more parent process instances were not found.
2. **Given** an investigated process instance exists and an ancestor in its family chain is missing, **When** the operator runs `walk pi --family`, **Then** the command returns the reachable family data instead of failing the entire traversal.
3. **Given** a family traversal includes a missing ancestor boundary, **When** the operator requests tree rendering, **Then** the CLI renders the partial tree and surfaces that the tree is incomplete because a parent process instance was missing.

---

### User Story 2 - Keep Orphan Children Actionable (Priority: P2)

As a CLI operator, I want delete and cancel preparation flows to keep orphan children actionable even when an ancestor is missing so that cleanup work can continue without manual recovery steps.

**Why this priority**: Cleanup and destructive preflight flows are a practical operational need, and blocking them on a missing ancestor leaves stranded child instances harder to manage.

**Independent Test**: Run delete and cancel preflight flows on orphan-child scenarios and verify the commands return the resolvable process-instance keys, report the missing ancestor keys, and continue preparing the actionable items.

**Acceptance Scenarios**:

1. **Given** selected process instances include orphan children whose ancestor is missing, **When** the operator runs delete preflight, **Then** the command returns the keys it could resolve and does not fail solely because an ancestor was missing.
2. **Given** selected process instances include orphan children whose ancestor is missing, **When** the operator runs cancel preflight or force root expansion, **Then** the command continues with the resolvable family data and reports which ancestor keys were missing.
3. **Given** process-definition deletion expands active process instances through the same process-instance dependency flow, **When** an expanded family includes a missing ancestor, **Then** the delete preparation continues with partial results instead of treating the missing parent as a hard stop.

---

### User Story 3 - Preserve Strict Single-Resource Semantics (Priority: P3)

As a maintainer, I want direct single-resource lookups and absent-state waiting behavior to remain strict so that this resilience change only affects traversal and dependency-expansion flows.

**Why this priority**: The issue explicitly limits the behavioral change, and preserving strict lookup and waiter semantics prevents unintended regressions in established command contracts.

**Independent Test**: Verify that direct key lookups still fail normally when the target process instance is missing and that absent/deleted wait flows behave exactly as they did before the feature.

**Acceptance Scenarios**:

1. **Given** a process instance key does not exist, **When** the operator runs a direct key-based get command, **Then** the command still returns the normal not-found error.
2. **Given** an operator waits for a process instance to become absent or deleted, **When** the resource disappears as expected, **Then** waiter behavior remains unchanged by the orphan-parent traversal feature.
3. **Given** maintainers review the final behavior, **When** they compare traversal flows with strict single-resource flows, **Then** only traversal and dependency-expansion paths have the new partial-result warning contract.

### Edge Cases

- More than one ancestor in a traversal chain may be missing, and the warning contract must still surface every missing parent key the command can identify.
- The investigated process instance may still exist even when the first missing ancestor is close to the root, so the CLI must preserve reachable descendants and ancestry collected before the missing boundary.
- Tree or list rendering must remain usable when family data is partial rather than presenting an empty result solely because one ancestor lookup failed.
- Delete and cancel preparation must distinguish between process-instance keys that were resolved successfully and ancestor keys that were missing so operators can act on the valid subset knowingly.
- Direct key-based lookup and waiter flows must stay strict even if they share supporting services with traversal-oriented commands.
- Version-specific process-instance service paths for `v87`, `v88`, and `v89` may differ internally, but the user-facing contract for affected traversal and preflight flows must remain consistent where those paths apply.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST treat a missing non-start parent encountered during process-instance traversal as a warning and partial-result condition for traversal and dependency-expansion flows rather than as an automatic command failure.
- **FR-002**: The system MUST return every investigated or related process-instance key that was successfully resolved before traversal stopped at a missing parent boundary.
- **FR-003**: The system MUST return the missing parent key or keys as machine-readable metadata associated with an incomplete traversal so callers can report, reason about, and act on the orphaned boundary explicitly.
- **FR-004**: The system MUST allow `walk pi --parent`, `walk pi --family`, and `walk pi --family --tree` to continue rendering partial ancestry or family output when a non-start parent is missing.
- **FR-005**: The system MUST emit a user-facing warning that the returned tree or family data is incomplete because one or more parent process instances were not found.
- **FR-005a**: The system MUST keep affected traversal and dependency-expansion flows non-fatal when at least one actionable result was resolved, while still emitting the incomplete-tree warning and missing-ancestor metadata.
- **FR-005b**: The system MUST return a normal failure for an affected traversal or preflight flow when no process-instance data could be resolved at all.
- **FR-006**: The system MUST allow delete process-instance preflight flows to continue when selected instances are orphan children, using the resolvable family data without failing solely on a missing ancestor.
- **FR-007**: The system MUST allow cancel process-instance preflight, paged preparation, and force root expansion flows to continue when an ancestor is missing, while preserving visibility into unresolved ancestor keys.
- **FR-008**: The system MUST allow any indirect process-definition deletion flow that depends on the same process-instance dependency expansion to continue with partial results when an ancestor is missing.
- **FR-008a**: The system MUST avoid turning an affected traversal or preflight flow into a failure solely because an ancestor was missing when the command still resolved at least one actionable result.
- **FR-008b**: The system MUST treat a fully unresolved traversal as a failure condition rather than a warning-only partial result.
- **FR-009**: The system MUST preserve a strict not-found error for direct key-based process-instance retrieval when the requested process instance itself is missing.
- **FR-010**: The system MUST preserve the current absent/deleted waiter behavior without loosening or reinterpreting success conditions for wait flows.
- **FR-011**: The system MUST let callers distinguish resolved family keys from missing ancestor keys through one shared structured warning contract so destructive preparation flows can act on the valid subset knowingly.
- **FR-012**: Regression coverage MUST verify the partial-result warning contract across the applicable `v87`, `v88`, and `v89` process-instance service paths.

### Key Entities *(include if feature involves data)*

- **Resolved Process Instance Keys**: The set of investigated, ancestor, descendant, or family process-instance identifiers that were successfully retrieved before traversal encountered a missing parent boundary.
- **Missing Ancestor Keys**: The machine-readable parent process-instance identifiers that could not be resolved during traversal and therefore explain why the returned tree or family data is incomplete.
- **Partial Family Result**: The traversal output that contains valid resolved family data plus one shared structured warning contract, including machine-readable missing ancestor metadata and a user-facing warning about the incomplete boundary.
- **Traversal and Dependency-Expansion Flow**: Any command path that builds ancestry, family, tree, delete-preflight, cancel-preflight, or related expansion data from one or more process-instance keys.
- **Strict Single-Resource Flow**: A direct lookup or waiter path whose not-found or absent-state semantics must remain unchanged by this feature.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated regression coverage shows that traversal commands in scope return partial ancestry or family results with an explicit warning instead of failing hard when a non-start parent is missing.
- **SC-002**: Automated coverage shows delete and cancel preparation flows continue to produce actionable resolved keys while also exposing missing ancestor keys in orphan-child scenarios.
- **SC-002a**: Automated coverage shows every affected traversal and preflight flow exposes the same structured missing-ancestor metadata contract alongside the user-facing warning.
- **SC-003**: Automated coverage shows tree or family rendering remains available for partial traversal data rather than collapsing to a command failure for the orphan-parent condition.
- **SC-003a**: Automated coverage shows affected traversal and preflight flows complete successfully with warnings when at least one actionable result is resolved, and do not fail solely because an ancestor is missing.
- **SC-003b**: Automated coverage shows affected traversal and preflight flows return a normal failure when no process-instance data can be resolved, even if missing-ancestor metadata is available.
- **SC-004**: Automated coverage confirms direct key-based process-instance retrieval still returns the normal not-found error when the requested instance is absent.
- **SC-005**: Automated coverage confirms absent/deleted waiter behavior remains unchanged after the feature is implemented.
- **SC-006**: Regression coverage exercises the applicable `v87`, `v88`, and `v89` process-instance service paths for the orphan-parent warning contract.

## Assumptions

- Orphaned process-instance relationships are a valid operational reality and should be inspectable and cleanable without manual scripting around hard command failures.
- The investigated process instance or child family members may still be valid and actionable even when an ancestor record no longer exists.
- Warning-based partial results are the safest default for traversal and dependency-expansion flows because they preserve useful output while still making the data gap explicit.
- When at least one actionable result is resolved, warning-based partial results should be treated as successful command outcomes for the affected traversal and preflight flows.
- Warning-based success is only appropriate when at least one actionable process-instance result was resolved; fully unresolved traversals should still fail normally.
- Direct single-resource retrieval and waiter flows are intentionally out of scope for this behavior change and should keep their existing strict semantics.
- The same user-facing contract should apply anywhere the shared process-instance family traversal is used for walk, delete, cancel, or indirect process-definition cleanup preparation.
