# Feature Specification: Version-Aware Process-Instance Paging and Overflow Handling

**Feature Branch**: `101-processinstance-paging`  
**Created**: 2026-04-12  
**Status**: Draft  
**Input**: User description: "GitHub issue #101: feat(cmd/process-instance): add version-aware paging + overflow handling for get/cancel/delete"

## GitHub Issue Traceability

- **Issue Number**: 101
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/101
- **Issue Title**: feat(cmd/process-instance): add version-aware paging + overflow handling for get/cancel/delete

## Clarifications

### Session 2026-04-12

- Q: How should search-based cancel/delete behave when the user declines the next-page prompt after one or more pages were already processed? → A: Stop normally and report that only the processed pages were completed.
- Q: Should the configurable default process-instance page size use one shared key or separate keys per affected command? → A: One shared config key for get, cancel, and delete process-instance paging.
- Q: Should operator-facing paging output show only the current page count or both current-page and cumulative processed counts? → A: Show both the current page count and the cumulative processed count.
- Q: Should `get process-instance` use the same continuation prompt behavior as search-based `cancel` and `delete`? → A: `get process-instance` also prompts between pages unless `--auto-confirm` is set.
- Q: What should the command do if overflow still cannot be determined after the version-specific fallback is attempted? → A: Stop and warn that more matching items may remain.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Page Through Matching Process Instances (Priority: P1)

As a Camunda operator, I want `c8volt get process-instance` to fetch and present process instances page by page when more matches exist than fit in one page so I can inspect the full result set without silent truncation.

**Why this priority**: Safe, visible paging for read-only search is the core behavior requested in the issue and provides immediate value without changing write actions first.

**Independent Test**: Can be fully tested by running `c8volt get process-instance` in search mode against matching data sets larger than one page on supported versions and verifying that the command reports the current page size, returns the current page of results, and communicates whether more matches remain.

**Acceptance Scenarios**:

1. **Given** a supported environment where more than one page of process instances matches the search, **When** the user runs `c8volt get process-instance` without `--count`, **Then** the command uses the configured default page size of `1000`, returns the first page of matching instances, and clearly reports that more matching instances exist.
2. **Given** a supported environment where more than one page of process instances matches the search, **When** the user runs `c8volt get process-instance --count 250`, **Then** the command uses `250` as the current page size instead of the configured default and clearly reports that page size in its output.
3. **Given** a supported environment where the total matching process instances fit within one page, **When** the user runs `c8volt get process-instance`, **Then** the command returns the matching instances without prompting for another page and clearly reports that no additional matching items remain.
4. **Given** a supported environment where `get process-instance` has more matching instances after the current page and `--auto-confirm` is not set, **When** the current page has been returned, **Then** the command prompts before fetching the next page using the same continuation behavior as the search-based write commands.

---

### User Story 2 - Continue Search-Based Cancel and Delete Safely Across Pages (Priority: P2)

As a Camunda operator, I want search-based `cancel process-instance` and `delete process-instance` to stop after each page and require explicit continuation when more matches remain so I can manage large changes without silent bulk actions.

**Why this priority**: Search-based write actions carry higher operational risk, so safe continuation behavior is essential once paging expands beyond the first page.

**Independent Test**: Can be fully tested by running `cancel process-instance` and `delete process-instance` in search mode against data sets that overflow a single page, then verifying that the command reports the current page progress, prompts before continuing when `--auto-confirm` is not set, and continues automatically when `--auto-confirm` is set.

**Acceptance Scenarios**:

1. **Given** a supported environment where search-based cancellation matches more instances than fit in one page and `--auto-confirm` is not set, **When** the current page of matching instances has been processed, **Then** the command prompts the user before fetching and processing the next page.
2. **Given** a supported environment where search-based deletion matches more instances than fit in one page and `--auto-confirm` is set, **When** the current page of matching instances has been processed, **Then** the command continues automatically to the next page without an interactive prompt and clearly reports that it is auto-continuing.
3. **Given** a supported environment where a search-based cancel or delete command reaches the final page of matches, **When** the current page has been processed, **Then** the command stops without an extra continuation prompt and clearly reports that no more matching instances remain.
4. **Given** a supported environment where search-based cancel or delete has already processed one or more pages and additional matches remain, **When** the user declines the continuation prompt, **Then** the command stops normally and clearly reports that only the already processed pages were completed.

---

### User Story 3 - Receive Version-Aware Overflow Handling and Clear Operator Feedback (Priority: P3)

As a Camunda operator, I want overflow detection and paging behavior to follow the actual capabilities of Camunda 8.7 and 8.8 so I can trust the command behavior for my configured version and understand what the command is doing at each step.

**Why this priority**: Version-aware behavior and clear operator messaging are required to avoid misleading results and to keep the user experience consistent across supported versions.

**Independent Test**: Can be fully tested by exercising the affected commands on both supported versions with overflowing and non-overflowing searches, then verifying that overflow detection follows the version-specific behavior contract and that the command output consistently shows the page size used, processed item count, overflow status, and whether the command is prompting or auto-continuing.

**Acceptance Scenarios**:

1. **Given** the configured Camunda version is `8.7` and a search matches more instances than fit in one page, **When** the user runs one of the affected commands, **Then** the command applies the version-specific overflow detection behavior defined for `8.7` and clearly communicates whether more matching instances exist.
2. **Given** the configured Camunda version is `8.8` and a search matches more instances than fit in one page, **When** the user runs one of the affected commands, **Then** the command applies the version-specific overflow detection behavior defined for `8.8` and clearly communicates whether more matching instances exist.
3. **Given** the configured Camunda version is outside the supported scope for this feature, **When** the user runs one of the affected search-based commands expecting paging support, **Then** the command preserves existing behavior for unsupported versions and does not claim support beyond `8.7` and `8.8`.

### Edge Cases

- A result set whose total match count is exactly equal to the current page size must not be reported as overflow.
- A result set that exceeds the current page size by only one item must still be reported as overflow and must offer continuation.
- A user who declines the continuation prompt after one or more pages have been processed must leave the remaining matching items untouched and must receive a clear partial-completion summary.
- If the configured default page size is present but the command also supplies `--count`, the command-line value must take precedence for that execution only.
- Commands that target process instances directly instead of using search mode must keep their current non-paged behavior.
- If overflow cannot be determined using the preferred method for a supported version, the command must use the version-appropriate fallback instead of silently assuming that no more matches exist.
- If overflow still cannot be determined after the version-appropriate fallback is attempted, the command must stop and clearly warn that more matching items may remain.
- Operator-facing status output must stay aligned across `get process-instance`, `cancel process-instance`, and `delete process-instance` so users do not have to learn different paging cues for each command.
- `8.9` behavior remains out of scope and must not be described as supported by this feature.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add paging-aware search behavior to `c8volt get process-instance`.
- **FR-002**: The system MUST add paging-aware search behavior to `c8volt cancel process-instance` when the command is operating in search mode.
- **FR-003**: The system MUST add paging-aware search behavior to `c8volt delete process-instance` when the command is operating in search mode.
- **FR-004**: The system MUST default the process-instance page size to `1000` when neither a command override nor a configured default changes it.
- **FR-005**: The system MUST allow a configurable default page size for affected process-instance commands through one shared config key used by `get process-instance`, search-based `cancel process-instance`, and search-based `delete process-instance`.
- **FR-006**: The system MUST let `--count` override the configured default page size for the current command execution.
- **FR-007**: The system MUST detect when additional matching process instances exist beyond the current page instead of silently truncating processing to the first page.
- **FR-008**: The system MUST use the preferred overflow indicator available for the configured Camunda version and fall back to a version-appropriate alternative when the preferred indicator is unavailable.
- **FR-008a**: If overflow still cannot be determined after the version-appropriate fallback is attempted, the system MUST stop and clearly warn that more matching process instances may remain.
- **FR-009**: After completing the current page for any affected command, the system MUST prompt the user before fetching or processing the next page whenever overflow exists and `--auto-confirm` is not set.
- **FR-010**: After completing the current page for any affected command, the system MUST continue automatically without prompting whenever overflow exists and `--auto-confirm` is set.
- **FR-010b**: `get process-instance` MUST use the same continuation prompt and `--auto-confirm` behavior as search-based `cancel process-instance` and `delete process-instance`.
- **FR-010a**: When a user declines a continuation prompt after one or more pages have already been processed, the system MUST stop without error and report that only the processed pages were completed.
- **FR-011**: The system MUST stop paging once no further matching process instances remain.
- **FR-012**: The system MUST clearly report the current page size used for each affected command execution.
- **FR-013**: The system MUST clearly report how many items were returned or processed in the current page.
- **FR-013a**: The system MUST clearly report the cumulative number of items returned or processed across all completed pages in the current command execution.
- **FR-014**: The system MUST clearly report whether additional matching process instances remain after the current page.
- **FR-015**: The system MUST clearly report whether execution is prompting for continuation or auto-continuing.
- **FR-016**: The system MUST implement overflow detection and paging behavior explicitly for Camunda `8.7`.
- **FR-017**: The system MUST implement overflow detection and paging behavior explicitly for Camunda `8.8`.
- **FR-018**: The system MUST preserve current behavior for versions outside the supported scope of this feature rather than implying support for them.
- **FR-019**: The system MUST preserve current behavior for direct, non-search process-instance workflows that are outside the paging scope of this issue.
- **FR-020**: The system MUST keep operator-facing paging behavior consistent across `get process-instance`, search-based `cancel process-instance`, and search-based `delete process-instance`.
- **FR-021**: The system MUST include automated coverage for default paging, configured default paging, `--count` override behavior, overflow detection, prompt-driven continuation, auto-confirm continuation, exact-boundary non-overflow, cross-page processing, and version-specific behavior for `8.7` and `8.8`.

### Key Entities *(include if feature involves data)*

- **Process-Instance Page Request**: The operator-selected or default page size applied to one execution of a search-based process-instance command.
- **Paged Result Window**: The current set of matching process instances returned or acted on before a continuation decision is required.
- **Overflow State**: The determination of whether additional matching process instances remain beyond the current page for the configured version.
- **Continuation Decision**: The operator-visible decision point that either prompts for the next page or continues automatically when `--auto-confirm` is active.
- **Version Paging Rule**: The version-specific rule set that defines how `8.7` and `8.8` determine overflow and continue paging.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can run `get process-instance` in search mode with more than `1000` matches and receive all matching results through explicit paging instead of silent first-page truncation.
- **SC-002**: Users can configure a default page size for affected process-instance commands, and the commands apply that value whenever `--count` is not provided.
- **SC-002a**: The configured default page size is applied consistently across `get process-instance`, search-based `cancel process-instance`, and search-based `delete process-instance` when `--count` is not provided.
- **SC-003**: Users who supply `--count` receive paging behavior based on that value rather than the configured default for that execution.
- **SC-004**: Users of search-based `cancel process-instance` and `delete process-instance` are prompted between pages whenever additional matches remain and `--auto-confirm` is not set.
- **SC-005**: Users of search-based `cancel process-instance` and `delete process-instance` with `--auto-confirm` enabled continue across overflowing result sets without interactive prompts.
- **SC-006**: Operator-facing output for all affected commands always identifies the page size used, the number of items handled in the current page, the cumulative number of items handled so far, whether more matches remain, and whether the command is prompting or auto-continuing.
- **SC-006a**: Users who decline a continuation prompt after one or more processed pages receive a non-error partial-completion summary that makes it clear the remaining matching items were not processed.
- **SC-007**: Supported Camunda `8.7` and `8.8` environments both demonstrate explicit, version-aware overflow handling in automated coverage.
- **SC-007a**: When overflow cannot be determined even after the version-specific fallback path, the command stops without silent continuation and warns operators that more matching items may remain.
- **SC-008**: No user-facing documentation or specification text for this feature claims support for Camunda `8.9`.

## Assumptions

- The existing `--count` flag already represents the per-command page-size override and remains the preferred command-line override for this feature.
- Search-based `cancel process-instance` and `delete process-instance` already have a notion of interactive confirmation and `--auto-confirm`, and this feature extends that pattern to multi-page continuation rather than introducing a new confirmation model.
- When an operator declines a continuation prompt, that choice is treated as an intentional stop rather than a command failure.
- Operators need both per-page visibility and running-total visibility during multi-page command execution.
- Operators benefit from one consistent continuation model across read and write process-instance paging flows.
- Each supported Camunda version exposes enough information to determine overflow directly or through a safe version-appropriate fallback.
- If a supported-version overflow signal remains indeterminate after the fallback path, the safest acceptable outcome is to stop and warn rather than silently continue or silently finish.
- The configured default page size applies only to the affected process-instance commands in scope for this issue.
- The affected commands share one paging-default configuration value rather than defining separate defaults per command.
- Direct, non-search process-instance commands remain out of scope for paging and should not change as part of this feature.
- Camunda `8.9` support remains explicitly out of scope even if adjacent code paths already mention later versions elsewhere in the repository.
