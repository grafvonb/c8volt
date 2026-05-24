# Feature Specification: Ops Paged Discovery Scope

**Feature Branch**: `228-ops-paged-discovery`

**Created**: 2026-05-24

**Status**: Draft

**Input**: GitHub issue [#228](https://github.com/grafvonb/c8volt/issues/228), "fix(ops): page discovery so ops workflows process the full matching scope by default"

## Issue Traceability

- **GitHub Issue**: #228
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/228
- **Issue Title**: fix(ops): page discovery so ops workflows process the full matching scope by default

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Purge Incident-Bearing Process Instances Uses Full Scope (Priority: P1)

As a Camunda operator, I want `ops purge process-instances-with-incidents` and its aliases to discover all matching incident-bearing process instances by default so that dry-run previews and confirmed deletion cover the same full operational population that inspection commands report.

**Why this priority**: This is the reported defect and the highest-risk destructive workflow because silently limiting scope can leave thousands of matching runtime records untouched.

**Independent Test**: Create a test population that spans multiple discovery pages, run `ops purge piwi --dry-run` without `--limit`, and verify the frozen preview includes every matching incident and process-instance target.

**Acceptance Scenarios**:

1. **Given** more matching incident-bearing process instances exist than fit in one discovery page, **When** an operator runs `ops purge piwi --dry-run` without `--limit`, **Then** the preview is based on the complete matching population rather than only the first page.
2. **Given** the same filters are used for `ops purge piwi --dry-run` and a matching `get pi ... --total` inspection path, **When** both commands complete, **Then** the frozen process-instance candidate scope is consistent with the inspection total.
3. **Given** an operator confirms a non-dry-run purge after seeing the frozen preview, **When** the command mutates runtime state, **Then** it uses the previously frozen target set and does not perform a second discovery that changes scope.

---

### User Story 2 - Repair Workflows Discover All Matching Candidates (Priority: P2)

As a Camunda operator, I want `ops repair incident` and `ops repair process-instance` search modes to discover all matching candidates by default so repair actions do not silently ignore records beyond the first page.

**Why this priority**: Repair commands are operational remediation workflows; partial discovery can make a remediation appear complete while matching incidents or process instances remain unresolved.

**Independent Test**: Use multi-page incident and incident-bearing process-instance fixtures, run each repair workflow in preview or dry-run mode, and verify the frozen candidate set includes every match unless an explicit `--limit` is supplied.

**Acceptance Scenarios**:

1. **Given** more matching incidents exist than fit in one discovery page, **When** an operator runs `ops repair incident` search mode without `--limit`, **Then** all matching incidents are discovered before preview or mutation.
2. **Given** more matching incident-bearing process instances exist than fit in one discovery page, **When** an operator runs `ops repair pi` search mode without `--limit`, **Then** all matching process instances are discovered before preview or mutation.
3. **Given** repair confirmation is required, **When** the operator approves the displayed scope, **Then** the command reuses the frozen repair candidates and does not expand or shrink the scope after confirmation.

---

### User Story 3 - Process Definition Purge Discovers All Matching Definitions (Priority: P3)

As a Camunda operator, I want `ops purge all-process-definitions` and its aliases to page through all matching process definitions by default so cleanup decisions apply to the full matching definition population.

**Why this priority**: Process-definition cleanup is destructive and should share the same complete-by-default discovery semantics as other broad ops workflows.

**Independent Test**: Provide a multi-page process-definition population, run `ops purge apd --dry-run` without `--limit`, and verify the frozen preview covers every matching process definition.

**Acceptance Scenarios**:

1. **Given** more matching process definitions exist than fit in one discovery page, **When** an operator runs `ops purge apd --dry-run` without `--limit`, **Then** the preview includes every matching process definition.
2. **Given** an explicit `--limit N`, **When** any affected purge or repair workflow discovers candidates, **Then** the frozen set contains at most N matching candidates and the output identifies the scope as user-limited.

---

### User Story 4 - Operators Can Audit Discovery Completeness (Priority: P4)

As an operator or automation author, I want dry-run, confirmation, JSON, Markdown report, and help output to describe whether discovery completed fully or was explicitly limited so that I can understand and audit the exact operational scope.

**Why this priority**: Full discovery is only safe if every human and automation path can see the same frozen scope and its completeness status.

**Independent Test**: Run affected workflows across human, JSON, report-file, automation, dry-run, and confirmed execution paths and verify each output describes the same frozen candidate set and whether it was complete or user-limited.

**Acceptance Scenarios**:

1. **Given** discovery completes without `--limit`, **When** an affected command renders human, JSON, or Markdown report output, **Then** the output exposes that discovery completed fully for the frozen scope.
2. **Given** discovery stops because `--limit` was supplied, **When** an affected command renders human, JSON, or Markdown report output, **Then** the output exposes that the scope was intentionally user-limited.
3. **Given** an operator reads help for an affected ops workflow, **When** the help text describes discovery flags, **Then** it explains that discovery pages by default, `--batch-size` tunes page size, and `--limit` is the way to intentionally cap scope.

### Edge Cases

- Matching populations can span many pages; discovery must continue until completion unless an explicit limit is reached.
- A final page can contain fewer records than the configured page size; this must be treated as completion, not as an error.
- Duplicate candidates encountered across discovery relationships must not cause duplicate mutation attempts or inflated preview counts.
- Empty result sets must still report a frozen complete scope of zero candidates.
- `--limit 0` and invalid limit values must follow existing command validation behavior.
- Automation mode must remain non-interactive and structured while still using the same frozen scope semantics.
- Report-file generation must describe the same candidate set shown by dry-run or confirmation output.
- Smoke-test process-definition cleanup eligibility is related safety work and should be reviewed for a follow-up unless it is explicitly pulled into this feature during planning.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Affected ops search-mode discovery MUST page through all matching candidates by default before preview, confirmation, mutation, automation output, or report rendering.
- **FR-002**: `--batch-size` MUST control discovery page size only and MUST NOT cap the total candidate scope.
- **FR-003**: `--limit N` MUST cap the frozen candidate set at no more than N matching candidates and MUST identify the scope as user-limited.
- **FR-004**: Dry-run output MUST preview the exact frozen candidate set that would be used for mutation.
- **FR-005**: Interactive confirmation MUST describe the exact frozen candidate set that mutation will use.
- **FR-006**: Confirmed mutation MUST reuse the frozen candidate set and MUST NOT perform another discovery pass that can expand, shrink, or reorder the approved scope.
- **FR-007**: Automation mode MUST remain non-interactive and MUST expose structured discovery completeness and explicit-limit status.
- **FR-008**: JSON output MUST expose whether discovery completed fully or stopped because of an explicit user limit.
- **FR-009**: Markdown report output MUST expose whether discovery completed fully or stopped because of an explicit user limit.
- **FR-010**: `ops purge process-instances-with-incidents`, `ops purge pi-with-incidents`, and `ops purge piwi` MUST use complete-by-default paged discovery.
- **FR-011**: `ops repair incident` and `ops repair inc` search modes MUST use complete-by-default paged incident discovery.
- **FR-012**: `ops repair process-instance` and `ops repair pi` search modes MUST use complete-by-default paged incident-bearing process-instance discovery.
- **FR-013**: `ops purge all-process-definitions`, `ops purge all-pds`, and `ops purge apd` MUST use complete-by-default paged process-definition discovery.
- **FR-014**: Equivalent filters between affected ops workflows and matching inspection totals MUST identify consistent candidate populations when no explicit limit is supplied.
- **FR-015**: Help text for affected workflows MUST document complete-by-default discovery, `--batch-size` page-size semantics, and `--limit` scope-cap semantics.
- **FR-016**: Tests MUST cover multi-page discovery for affected process-instance, incident, and process-definition paths.
- **FR-017**: Tests MUST cover frozen-scope reuse after confirmation.

### Key Entities *(include if feature involves data)*

- **Frozen Candidate Set**: The complete or explicitly limited set of candidates discovered before preview, confirmation, mutation, automation output, and report generation.
- **Discovery Completeness**: The status that records whether candidate discovery reached the end of the matching population or stopped because the operator supplied a limit.
- **Explicit Scope Limit**: The operator-provided cap from `--limit`, separate from page-size tuning.
- **Discovery Page Size**: The per-request discovery size controlled by `--batch-size`.
- **Ops Report**: Human or machine-readable output that records the frozen scope and discovery completeness for auditing.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In multi-page fixtures without `--limit`, each affected workflow discovers and freezes 100% of matching candidates before preview or mutation.
- **SC-002**: In equivalent-filter fixtures, affected process-instance ops workflow counts match the corresponding inspection total when no explicit limit is supplied.
- **SC-003**: With `--limit N`, each affected workflow freezes no more than N matching candidates and marks the result as user-limited.
- **SC-004**: In confirmation tests, candidate discovery occurs before confirmation and is not repeated after approval.
- **SC-005**: Human, JSON, and Markdown report outputs all expose discovery completeness or explicit-limit status for affected workflows.
- **SC-006**: Help output for affected workflows accurately describes complete-by-default discovery and the distinct meanings of `--batch-size` and `--limit`.
- **SC-007**: The closest relevant automated tests for affected ops workflows pass, including multi-page process-instance, incident, and process-definition coverage.

## Assumptions

- Operators expect broad ops workflows to process the complete matching population unless they intentionally provide `--limit`.
- Existing `ops execute retention-policy` and `ops purge orphan-process-instances` behavior is a useful reference because those workflows already page process-instance discovery until completion unless limited.
- The service layer owns full paged discovery and frozen-scope behavior; command renderers display returned scope and completeness rather than inferring it from counts.
- Existing mutation safety semantics remain: preview first, confirm the exact frozen scope where interactive confirmation is required, then mutate only that frozen scope.
- The related smoke-test process-definition cleanup eligibility issue is review-worthy but not required for this feature unless planning identifies it as inseparable from shared discovery behavior.

## Implementation Governance

- Planning, task generation, and every Ralph implementation iteration MUST read and apply `specs/ralph-implementation-rules.md`.
- Ralph MUST NOT be launched unless `--implementation-context specs/ralph-implementation-rules.md` is included in the implementation instructions.
- Commit subjects for this issue-backed work MUST use Conventional Commits format and append `#228` as the final token.
