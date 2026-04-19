# Feature Specification: Push Supported Get Filters Into Search Requests

**Feature Branch**: `116-server-search-filters`  
**Created**: 2026-04-19  
**Status**: Draft  
**Input**: User description: "GitHub issue #116: refactor(cmd/get): push server-capable filters into Camunda search requests"

## GitHub Issue Traceability

- **Issue Number**: 116
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/116
- **Issue Title**: refactor(cmd/get): push server-capable filters into Camunda search requests

## Clarifications

### Session 2026-04-19

- Q: How broad should the implementation scope be for other audited `get` commands that show the same late-filtering pattern? → A: Implement every clearly in-scope audited `get` filter that can be safely pushed down now, and keep tested fallback only where support is missing.
- Q: How should version differences be handled when only some runtime versions expose a reliable request-side representation for a filter? → A: Enable pushdown per version only where the request-side representation is reliable, and keep client-side fallback for each unsupported version.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Return only matching process instances per page (Priority: P1)

As a CLI operator, I want `get process-instance` to send supported filters with the search request so that each fetched page already reflects the command filters I asked for.

**Why this priority**: The issue directly affects command speed, API load, and whether the first page of results is representative of what the user asked to see.

**Independent Test**: Run `get process-instance` with supported filters on versions that can express those filters server-side, then verify the outgoing search criteria include the requested filters and the returned page contains only matching instances without broad overfetching.

**Acceptance Scenarios**:

1. **Given** a supported runtime version and a `get process-instance` request that uses `--roots-only`, `--children-only`, `--incidents-only`, or `--no-incidents-only`, **When** the command builds the search request, **Then** it includes the equivalent request-side filters before fetching the page.
2. **Given** a supported runtime version and a narrow filter that matches few instances, **When** the command fetches a page, **Then** the reported page size and continuation behavior reflect the filtered result set rather than a larger unfiltered page.

---

### User Story 2 - Preserve behavior on unsupported versions (Priority: P2)

As a maintainer, I want each runtime version to use request-side filtering only for filters that version can represent reliably, while unsupported versions keep the existing client-side filtering path so that the refactor improves supported versions without breaking compatibility.

**Why this priority**: The issue explicitly requires version-aware behavior, and preserving current behavior on unsupported versions keeps the change incremental and safe.

**Independent Test**: Exercise the same filtered command paths across supported and unsupported runtime versions and verify each version uses request-side filters only where that version has a reliable representation, while the final results still honor the existing client-side filtering semantics everywhere else.

**Acceptance Scenarios**:

1. **Given** a runtime version that cannot express a requested filter reliably, **When** the command builds the search request, **Then** it leaves that filter out of the request for that version and applies the current client-side filtering behavior after fetching results.
2. **Given** `--orphan-children-only`, **When** the command runs on any version without a reliable request-side equivalent, **Then** the command keeps that filter client-side and preserves current command semantics.

---

### User Story 3 - Apply the same audit rule across get commands (Priority: P3)

As a maintainer, I want other relevant `get` commands and flags reviewed for the same late-filtering pattern so that every clearly in-scope supported filter is moved to request-side filtering in this feature instead of being deferred by default.

**Why this priority**: The issue scope is broader than one example command, and a narrow one-off fix would leave the same inefficiency in similar paths.

**Independent Test**: Review each affected `get` command path and confirm automated coverage proves every clearly in-scope server-capable filter is pushed down in this feature where supported, while unsupported filters remain client-side.

**Acceptance Scenarios**:

1. **Given** another `get` command or filter flag follows the same "fetch broad page, then narrow locally" pattern, **When** that filter has a reliable request-side equivalent, **Then** the command uses the request-side filter instead of keeping the full narrowing step client-side.
2. **Given** an audited `get` command or flag without a reliable request-side equivalent, **When** the audit is completed, **Then** the command remains on the existing client-side filtering path and the retained fallback is covered by tests rather than being silently deferred.

### Edge Cases

- A runtime version may handle some filters server-side but not others, so each filter must be evaluated independently per version rather than moving all filtering to one side unconditionally.
- Combining multiple supported filters in the same command must preserve current result semantics while narrowing the fetched page at request time.
- Unsupported versions must continue to return the same final filtered result set even when the request itself remains broader.
- `--orphan-children-only` must remain client-side unless a reliable request-side equivalent is identified during implementation.
- Paging and continuation prompts must reflect the filtered result set for request-side filters and must not imply extra matching records because of overfetching.
- The audit may find additional `get` commands with mixed support across versions, and each discovered case must preserve existing behavior where request-side support is incomplete.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST push supported `get` command filters into the search request only for runtime versions that expose a reliable request-side equivalent for the specific filter being used.
- **FR-002**: The system MUST preserve current command semantics while moving supported filters from late client-side narrowing to request-side filtering.
- **FR-003**: `get process-instance` MUST use request-side filtering for `--roots-only`, `--children-only`, `--incidents-only`, and `--no-incidents-only` on supported runtime versions.
- **FR-004**: The system MUST keep client-side filtering for any command filter that lacks a reliable request-side equivalent on the active runtime version, even when other versions support request-side pushdown for that same filter.
- **FR-005**: Unsupported versions such as v8.7 MUST preserve the current client-side fallback behavior where request-side filter support is unavailable, incomplete, or not reliable enough for that version.
- **FR-006**: `--orphan-children-only` MUST remain client-side unless downstream work confirms a reliable request-side equivalent.
- **FR-007**: The system MUST improve paging accuracy for request-side filters so fetched-page counts and continuation prompts reflect the matching server-filtered result set.
- **FR-008**: The system MUST audit other relevant `get` commands and filter flags for the same late-filtering pattern rather than limiting the change to the reported example only.
- **FR-009**: The system MUST implement request-side filtering in this feature for every clearly in-scope audited `get` command filter that has reliable support on the active runtime version.
- **FR-009a**: The system MUST retain and test the existing client-side filtering path for audited filters that are not reliably supported request-side on a given runtime version.
- **FR-010**: The system MUST provide automated coverage for both request-side filtering on supported versions and client-side fallback behavior on unsupported versions.

### Key Entities *(include if feature involves data)*

- **Get Command Filter**: A user-facing command flag that narrows results returned by a `get` command.
- **Search Request**: The outbound query criteria sent before a result page is fetched.
- **Client-Side Fallback Filter**: A post-fetch narrowing step retained only when the active version cannot express the requested filter reliably in the search request.
- **Filtered Result Page**: The page of results and continuation state presented to the user after request-side and any required fallback filtering rules are applied.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated coverage confirms that supported `get process-instance` filters are included in the outbound search request only on runtime versions with reliable request-side support for those specific filters.
- **SC-002**: Automated coverage confirms that unsupported runtime versions keep the existing client-side filtering path and still return the same final filtered results for covered cases.
- **SC-003**: Automated coverage confirms `--orphan-children-only` remains client-side unless downstream work establishes a reliable request-side equivalent.
- **SC-004**: Covered `get process-instance` scenarios no longer fetch a broad page only to discard most results locally when a supported request-side filter was provided.
- **SC-005**: Covered paging and continuation scenarios report counts consistent with the server-filtered result set for supported request-side filters.
- **SC-006**: The audit of other relevant `get` commands results in request-side filter adoption during this feature for every clearly in-scope supported case, with an explicit tested fallback for unsupported cases.

## Assumptions

- The feature is a refactor for efficiency and paging accuracy, not a change to the user-facing meaning of existing filter flags.
- Runtime-version support for request-side filtering may differ by filter and command, so the implementation must remain version-aware at the individual filter level.
- Existing client-side filtering logic is considered the correct behavioral fallback whenever request-side support is missing or unreliable.
- The currently known `get process-instance` filters are the minimum required audit scope, and any additional clearly in-scope `get` command filters discovered during implementation should be completed in this feature when reliable request-side support exists.
- Downstream implementation work for this feature must keep Conventional Commit formatting and append `#116` as the final token of every commit subject.
