# Feature Specification: Omit Empty Initial Search Cursor

**Feature Branch**: `279-omit-empty-cursor`

**Created**: 2026-08-26

**Status**: Draft

**Input**: GitHub issue [#279](https://github.com/grafvonb/c8volt/issues/279): "fix(process): omit empty initial cursor from latest process-definition searches"

## GitHub Issue Traceability

- **Issue Number**: 279
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/279
- **Issue Title**: fix(process): omit empty initial cursor from latest process-definition searches
- **Observed Environment**: Camunda 8 Run 8.9.17 with its default H2/RDBMS secondary storage

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start Processes by BPMN ID Reliably (Priority: P1)

As a c8volt operator, I want process-instance creation by BPMN process ID to validate the latest deployed process definition successfully so that I can start instances without discovering and supplying a process-definition key or exact version as a workaround.

**Why this priority**: The current initial latest-definition lookup can fail with a server error before any process instances are created, blocking an established operator workflow on a supported Camunda environment.

**Independent Test**: Deploy a process definition to Camunda 8 Run 8.9.17 using its default H2/RDBMS storage, then start ten instances by BPMN process ID. The selector is validated and all ten instances are created without a search-server error.

**Acceptance Scenarios**:

1. **Given** a visible latest process definition on Camunda 8.9 with H2/RDBMS storage, **When** an operator starts process instances by BPMN process ID without specifying a version, **Then** c8volt validates the latest definition and starts the requested instances.
2. **Given** ten process instances are requested by BPMN process ID, **When** latest-definition validation succeeds, **Then** c8volt submits creation for all ten instances and preserves its established completion reporting.
3. **Given** latest-definition validation cannot identify a matching definition, **When** an operator starts by BPMN process ID, **Then** c8volt retains its existing not-found behavior and does not submit partial process-instance creation.

---

### User Story 2 - Page Latest Definitions Correctly (Priority: P1)

As an operator searching for latest process definitions, I want c8volt to distinguish the first search page from later pages so that searches work consistently across supported storage backends and continue through all available results.

**Why this priority**: An empty value is not a continuation cursor. Treating it as one can make a valid first-page request fail, while failing to use a real returned cursor can make a multi-page search incomplete.

**Independent Test**: Exercise a latest-definition search with at least two response pages. Verify that the first request contains only the requested page limit, while the next request uses the exact non-empty continuation cursor returned by the first response.

**Acceptance Scenarios**:

1. **Given** a latest-definition search has not received a continuation cursor, **When** c8volt requests the first page, **Then** the search includes its page limit and includes neither an empty forward cursor nor an offset.
2. **Given** Camunda returns a non-empty continuation cursor, **When** c8volt requests the next page, **Then** it sends that cursor unchanged together with the page limit.
3. **Given** Camunda returns no non-empty continuation cursor, **When** c8volt finishes processing the current page, **Then** it does not issue another cursor-based request.
4. **Given** the same latest-definition search is performed against Camunda 8.8, 8.9, or 8.10, **When** the first and continuation pages are requested, **Then** each supported release follows the same first-page and continuation-page contract.

---

### User Story 3 - Preserve Existing Process-Definition Searches (Priority: P2)

As a c8volt operator, I want the correction to be limited to cursor selection for latest-definition paging so that existing filters, sorting, tenant behavior, exact-version selection, key selection, and ordinary searches remain stable.

**Why this priority**: Correcting the initial request must not change other process-definition selection paths that already work across supported Camunda versions and storage backends.

**Independent Test**: Compare latest, ordinary, exact-version, and key-based selections before and after the correction. Only the empty initial latest-search cursor is removed; all other selection and result behavior remains unchanged.

**Acceptance Scenarios**:

1. **Given** an ordinary process-definition search that is not restricted to latest versions, **When** it requests its first page, **Then** its existing offset-based paging behavior remains unchanged.
2. **Given** an exact process-definition version or process-definition key, **When** an operator selects that definition, **Then** selection behavior remains unchanged.
3. **Given** latest-definition filters, tenant scope, and stable sorting, **When** the corrected search runs, **Then** those values and their ordering remain unchanged.
4. **Given** Elasticsearch, OpenSearch, or an RDBMS-backed supported Camunda environment, **When** the corrected initial latest-definition search runs, **Then** it uses the same valid limit-only first-page contract.

### Edge Cases

- Camunda returns an empty continuation cursor rather than omitting it.
- Camunda returns one page of results with no continuation cursor.
- Camunda returns a non-empty continuation cursor with characters that must be preserved exactly.
- The first page is empty but still includes a non-empty continuation cursor.
- A multi-page search returns a final page without a continuation cursor.
- A latest-definition search combines BPMN process ID, tenant scope, and latest-version selection.
- An ordinary search begins at offset zero; it must continue to express offset zero rather than being converted to limit-only paging.
- The configured Camunda release is 8.8, 8.9, or 8.10, including a patch or prerelease within that release line.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The initial page of every latest process-definition search MUST include the requested positive page limit.
- **FR-002**: The initial page of a latest process-definition search MUST NOT include a forward cursor when no non-empty continuation cursor has been received.
- **FR-003**: The initial page of a latest process-definition search MUST NOT include an offset.
- **FR-004**: A subsequent latest process-definition page MUST use cursor-based continuation only when the preceding response provides a non-empty continuation cursor.
- **FR-005**: A continuation request MUST send the preceding response's non-empty continuation cursor unchanged.
- **FR-006**: Latest process-definition paging MUST continue until Camunda returns no non-empty continuation cursor or the established result limit is reached.
- **FR-007**: The corrected first-page and continuation-page behavior MUST apply consistently to Camunda 8.8, 8.9, and 8.10.
- **FR-008**: Existing latest-version selection semantics MUST remain unchanged.
- **FR-009**: Existing latest-definition filters, tenant scoping, result limits, and stable sort fields and directions MUST remain unchanged.
- **FR-010**: Ordinary process-definition searches MUST retain their existing offset-and-limit paging behavior.
- **FR-011**: Exact-version process-definition selection MUST remain unchanged.
- **FR-012**: Process-definition-key selection MUST remain unchanged.
- **FR-013**: Starting process instances by BPMN process ID MUST use the corrected latest-definition search whenever no exact version or key is supplied.
- **FR-014**: A validation failure MUST continue to prevent partial process-instance creation.
- **FR-015**: The correction MUST NOT introduce new flags, prompts, output fields, or operator-visible success wording.
- **FR-016**: Automated verification MUST cover the initial page, a continuation page with a real cursor, the final page, and an ordinary offset-based search for each affected Camunda release.
- **FR-017**: Operational verification MUST reproduce the previously failing workflow against Camunda 8 Run 8.9.17 with default H2/RDBMS storage.

### Key Entities

- **Latest Process-Definition Search**: A search restricted to the latest visible version of matching process definitions, including its filters, tenant scope, stable ordering, result limit, and paging state.
- **Initial Page**: The first search request, which has a page limit but no continuation cursor or offset.
- **Continuation Cursor**: A non-empty opaque value returned by Camunda that identifies the next page and must be passed back unchanged.
- **Ordinary Process-Definition Search**: A search not using latest-version semantics and therefore retaining its established offset-based paging behavior.
- **Process-Definition Selector**: Operator input that identifies a definition by BPMN process ID, exact version, or key before process-instance creation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In all tested Camunda 8.8, 8.9, and 8.10 latest-definition searches, 100% of initial requests contain a page limit and contain neither a forward cursor nor an offset.
- **SC-002**: In all tested multi-page latest-definition searches, 100% of continuation requests use the exact non-empty cursor returned by the preceding response.
- **SC-003**: Starting ten process instances by BPMN process ID against Camunda 8 Run 8.9.17 with default H2/RDBMS storage creates all ten instances without a latest-definition search-server error.
- **SC-004**: All tested ordinary, exact-version, and key-based process-definition selection paths retain their pre-change results and paging behavior.
- **SC-005**: The closest process-definition and process-instance command validation completes with zero regressions across all supported Camunda releases.
- **SC-006**: Operators can use BPMN process ID selection on the affected environment without resorting to a process-definition-key or exact-version workaround.

## Assumptions

- Camunda 8.8, 8.9, and 8.10 accept a positive limit without either an offset or a continuation cursor for an initial latest-definition search.
- A continuation cursor is opaque; c8volt does not decode, normalize, trim, or otherwise alter a non-empty cursor returned by Camunda.
- Existing latest-definition filters and stable ordering are correct and outside the defect being addressed.
- Existing process-instance creation, confirmation, and output behavior are correct once selector validation succeeds.
- The H2/RDBMS failure observed on Camunda 8 Run 8.9.17 is reproducible and provides the primary operational regression check.

## Dependencies

- A Camunda 8 Run 8.9.17 environment using its default H2/RDBMS secondary storage is available for operational verification.
- Representative process definitions can be deployed and selected by BPMN process ID.
- Supported Camunda releases expose a non-empty continuation cursor when another latest-definition page is available.

## Out of Scope

- Changing process-definition filters, tenant selection, sorting, or latest-version semantics.
- Changing ordinary offset-based process-definition searches.
- Changing exact-version or process-definition-key selection.
- Changing process-instance creation, polling, confirmation, or output behavior.
- Adding new search flags or user-visible commands.
- Working around malformed non-empty cursors returned by Camunda.
- Changing Camunda server-side cursor handling or storage-backend behavior.
