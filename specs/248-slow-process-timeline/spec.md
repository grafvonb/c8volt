# Feature Specification: Slow Process Timeline Readability

**Feature Branch**: `248-slow-process-timeline`

**Created**: 2026-07-22

**Status**: Draft

**GitHub Issue**: [#248](https://github.com/grafvonb/c8volt/issues/248) - feat(ops): improve slow-process analysis timeline readability

**Input**: GitHub issue #248 requests a more compact default human view for `c8volt ops analyse slow-process-instances`, making slow contributors obvious first while preserving access to the complete chronological timeline.

## Clarifications

### Session 2026-07-22

- Q: Which completed contributors should default human output show? → A: Contributors at or above 1%

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read A Hotspot-Oriented Human Summary (Priority: P1)

As a c8volt operator investigating a slow process instance, I want the default human output to emphasize the process-instance root and slowest element contributors so I can identify likely areas for follow-up without visually parsing every instant or near-instant timeline row.

**Why this priority**: The command is an ops diagnostic, and its default view must make the most important timing information obvious during live investigation.

**Independent Test**: Can be fully tested by analyzing a process instance with many element and transition rows, then verifying that the default human output shows the root row, a nested slowest-elements section, slow contributors, active or incident rows, and a hidden-row summary instead of the full chronological timeline.

**Acceptance Scenarios**:

1. **Given** a process instance with many completed element rows and zero-duration gateway or transition rows, **When** the operator runs `c8volt ops analyse slow-process-instances -k <process-instance-key>` with default human output, **Then** the output shows the process-instance root followed by a `slowest elements:` section that places the largest duration contributors before low-duration details.
2. **Given** the summarized timeline omits rows from the complete analysis, **When** the operator reads the default human output, **Then** the output includes a concise hidden-row count and names `--with-full-timeline` as the way to inspect the complete chronological detail.
3. **Given** an element row is active or has an incident, **When** default human output is rendered, **Then** that row remains visible in the summary even when it is not one of the longest completed contributors.

---

### User Story 2 - Inspect The Complete Timeline On Demand (Priority: P2)

As a c8volt operator performing audit or deep debugging work, I want an explicit full-timeline option so I can still inspect the complete chronological element and transition detail when the compact summary is not enough.

**Why this priority**: The readability improvement must not remove the current diagnostic depth needed for audit, debugging, or edge-case investigation.

**Independent Test**: Can be fully tested by running the same analysis with and without `--with-full-timeline`, then verifying that the flag restores the complete chronological detail while the default mode remains compact.

**Acceptance Scenarios**:

1. **Given** a process instance with zero-duration gateways, transitions, and fast elements, **When** the operator runs `c8volt ops analyse slow-process-instances -k <process-instance-key> --with-full-timeline`, **Then** the human output includes the complete chronological timeline detail comparable to the current command behavior.
2. **Given** the operator runs the command with existing element filters and `--with-full-timeline`, **When** the output is rendered, **Then** the full timeline honors the existing filter meanings and does not invent summary-only filtering behavior.
3. **Given** a process instance has no hidden rows after filtering, **When** the full timeline is requested, **Then** the output does not show a hidden-row summary.

---

### User Story 3 - Preserve Script And Machine Output Contracts (Priority: P3)

As an automation author, I want JSON and keys-only output to remain unchanged so existing scripts continue to consume slow-process analysis without migration.

**Why this priority**: The feature improves human readability only; machine-readable and pipeline-safe contracts must stay stable.

**Independent Test**: Can be fully tested by comparing JSON and keys-only output before and after enabling this feature for equivalent selections and filters.

**Acceptance Scenarios**:

1. **Given** JSON output is requested for slow-process analysis, **When** the command runs with or without `--with-full-timeline`, **Then** the JSON payload shape, ordering, fields, and values remain unchanged.
2. **Given** keys-only output is requested, **When** the command runs with or without `--with-full-timeline`, **Then** only unique process-instance keys are emitted one per line in the established order and no summary text appears.
3. **Given** existing process-instance and element duration filters are used, **When** JSON or keys-only output is requested, **Then** those filters keep their current meaning.

### Edge Cases

- Process instances with no element rows must still show the process-instance root and must not show a misleading hidden-row count.
- Process instances where every detail row is instant or below 1% process-duration share must show the root, any active or incident rows, and a hidden-row summary when completed rows are omitted.
- Active rows and incident rows that are also among the slowest contributors must appear only once in the default summary.
- Default human output must not show element instance keys unless the row needs the key to identify an incident or the operator explicitly requests the full timeline.
- Hidden-row counts must include omitted element and transition timeline rows from the analyzed detail set and must not count the process-instance root.
- Detail filters must not create synthetic transitions across hidden rows.
- The `--with-full-timeline` flag must not change process-instance selection, sorting, duration calculation, filtering, JSON output, or keys-only output.
- The British and American command spellings must expose equivalent behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Default human output for slow-process analysis MUST render each analyzed process instance as the root row before any detail summary.
- **FR-002**: Default human output MUST replace the full chronological detail section with a nested `slowest elements:` section.
- **FR-003**: The `slowest elements:` section MUST show completed element contributors whose process-duration share is at least 1%, ordered from largest to smallest contribution.
- **FR-004**: The default summary MUST include active element rows even when their current process-duration share is below 1%.
- **FR-005**: The default summary MUST include incident-bearing element rows even when their process-duration share is below 1%.
- **FR-006**: The default summary MUST omit zero-duration transition rows and completed rows below 1% process-duration share unless a row is visible because it is active or incident-bearing.
- **FR-007**: The default summary MUST show enough context for each visible row to identify the BPMN element, including element type, element ID, state, duration when available, and compact start or end timing when useful.
- **FR-008**: Default human output MUST avoid showing element instance keys unless needed to identify an incident-bearing row.
- **FR-009**: Default human output MUST include a concise hidden-row summary whenever analyzed timeline rows are omitted from the human view.
- **FR-010**: The hidden-row summary MUST report the count and kind of omitted rows in terms operators can understand, such as instant, fast, or timeline rows.
- **FR-011**: The hidden-row summary MUST tell operators to use `--with-full-timeline` when they need the complete chronological detail.
- **FR-012**: The system MUST provide a `--with-full-timeline` option for human output on `ops analyse slow-process-instances` and the equivalent American spelling.
- **FR-013**: When `--with-full-timeline` is set, human output MUST show the complete chronological timeline detail that remains available for audit and debugging.
- **FR-014**: Full-timeline human output MUST include zero-duration gateways, transitions, and other rows that default human output may summarize or hide.
- **FR-015**: Existing process-instance selection filters, including duration filters for root visibility, MUST keep their current meaning in both default and full-timeline human output.
- **FR-016**: Existing element and detail filters, including element-duration filters, MUST keep their current meaning before the default summary decides which eligible human rows to show.
- **FR-017**: Detail filtering MUST still be based on the complete analyzed timeline and MUST NOT create synthetic transitions across hidden or filtered rows.
- **FR-018**: JSON output MUST remain unchanged by this feature.
- **FR-019**: Keys-only output MUST remain unchanged by this feature.
- **FR-020**: The `--with-full-timeline` option MUST NOT add any text, fields, or behavioral differences to JSON or keys-only output.
- **FR-021**: Command help, examples, generated CLI documentation, and user-facing documentation MUST describe the compact default view and the `--with-full-timeline` option.
- **FR-022**: Automated validation MUST cover default human summaries, full-timeline human output, hidden-row summaries, active rows, incident rows, JSON stability, keys-only stability, and documentation metadata for the new flag.

### Key Entities

- **Slow Process Analysis Result**: The complete read-only analysis for one or more process instances, including root process-instance rows and analyzed detail rows.
- **Process Instance Root Row**: The top-level human row for an analyzed process instance, containing the identifiers, state, timestamps, parent context, incident marker, and whole duration operators use as the investigation anchor.
- **Timeline Row**: An analyzed element or transition detail row that may appear in full chronological output and may be summarized or hidden in default human output.
- **Hotspot Summary Row**: A visible default human row selected because it is a large duration contributor, active, or incident-bearing.
- **Hidden-Row Summary**: A compact message that tells operators how many analyzed detail rows are omitted from default human output and how to request the full timeline.
- **Full Timeline Mode**: The explicit human-output mode that shows complete chronological detail for audit and debugging.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For process instances with at least 10 analyzed detail rows, default human output allows operators to identify every visible completed contributor at or above 1% process-duration share without reading the complete chronological timeline.
- **SC-002**: Default human output reduces visible detail rows by at least 50% for timelines where at least half of analyzed detail rows are zero-duration or very fast, while preserving active and incident-bearing rows.
- **SC-003**: 100% of default human outputs that omit analyzed detail rows include a hidden-row summary with the omitted-row count and `--with-full-timeline` guidance.
- **SC-004**: 100% of full-timeline human runs retain access to complete chronological detail, including rows hidden by the default summary.
- **SC-005**: JSON output for equivalent selections remains byte-for-byte compatible except for values that already vary by analysis time or live process state.
- **SC-006**: Keys-only output for equivalent selections remains exactly one process-instance key per line with no additional text.
- **SC-007**: Command help and generated command documentation both include `--with-full-timeline` and an example or description of when to use it.

## Assumptions

- This feature changes only the human presentation of the existing slow-process analysis command unless explicitly stated otherwise.
- The existing slow-process analysis behavior from `specs/244-slow-process-analysis` remains the baseline for selection, duration calculation, ordering, filtering, JSON output, and keys-only output.
- Operators value a compact default view during interactive diagnosis, but still need full chronological detail for audit and deep debugging.
- Very-fast rows are rows that provide little default diagnostic value compared with the process-instance duration, such as zero-duration gateway or transition rows and completed sub-1% contributors, unless they are active or incident-bearing.
- Documentation updates are required because this changes default human output and adds a user-facing flag.

## Out of Scope

- Changing JSON output shape, field names, ordering, or semantics.
- Changing keys-only output behavior.
- Changing process-instance selection, root sorting, duration calculations, or existing filter meanings.
- Removing access to the complete chronological timeline.
- Adding new analysis domains such as job execution, listener execution, BPMN path reconstruction, mutation workflows, or report-file generation.
