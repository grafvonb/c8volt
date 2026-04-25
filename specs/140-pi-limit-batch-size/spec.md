# Feature Specification: Process-Instance Limit and Batch Size Flags

**Feature Branch**: `140-pi-limit-batch-size`  
**Created**: 2026-04-25  
**Status**: Draft  
**Input**: User description: "GitHub issue #140: feat(cli): add --limit and replace --count with --batch-size for process-instance commands"

## GitHub Issue Traceability

- **Issue Number**: 140
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/140
- **Issue Title**: feat(cli): add --limit and replace --count with --batch-size for process-instance commands

## Clarifications

### Session 2026-04-25

- No critical ambiguities detected worth formal clarification. The GitHub issue defines the affected commands, flag names, short options, validation rules, output expectations, documentation updates, and required test coverage.
- Q: How should `--total` behave when combined with `--limit`? -> A: Reject as mutually exclusive.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Limit Search Results Across Pages (Priority: P1)

As a CLI user searching process instances, I want `--limit` to cap the total number of matched instances returned or processed across all pages so that a command can intentionally stop after the first matching `N` process instances.

**Why this priority**: Total-match limiting is the main behavior requested and directly reduces operational risk for read and destructive search-mode commands.

**Independent Test**: Run `get process-instance`, search-based `cancel process-instance`, and search-based `delete process-instance` against fixtures spanning multiple pages and verify that `--limit` stops at the configured total without fetching or processing later pages.

**Acceptance Scenarios**:

1. **Given** more than 25 process instances match a search, **When** the user runs `c8volt get pi --state active --limit 25`, **Then** only the first 25 matched process instances are returned overall.
2. **Given** more than 10 process instances match a cancellation search, **When** the user runs `c8volt cancel pi --state active --limit 10 --auto-confirm`, **Then** only the first 10 matched process instances are selected for cancellation.
3. **Given** more than 25 completed process instances match a deletion search, **When** the user runs `c8volt delete pi --state completed --limit 25 --auto-confirm`, **Then** only the first 25 matched process instances are selected for deletion.
4. **Given** a configured limit is reached before the backend has no more matches, **When** the command finishes the limited set, **Then** it stops cleanly without prompting for continuation or fetching another page.

---

### User Story 2 - Distinguish Batch Size From Total Limit (Priority: P2)

As a CLI user working with paged process-instance searches, I want `--batch-size` to control per-page fetching while `--limit` controls the total number of matched instances so that the two concepts are clear and can be combined safely.

**Why this priority**: Replacing the ambiguous `--count` flag with `--batch-size` removes the current source of confusion while preserving existing page-size behavior through clearer naming.

**Independent Test**: Run affected commands with `--batch-size`, `-n`, `--limit`, and `-l` in separate and combined forms, then verify page-size selection and total-limit behavior are independent.

**Acceptance Scenarios**:

1. **Given** a user runs `c8volt get pi --state active --batch-size 250`, **When** matching process instances are fetched, **Then** each page request uses a batch size of 250.
2. **Given** a user runs `c8volt get pi --state active -n 250`, **When** matching process instances are fetched, **Then** `-n` behaves as the short option for `--batch-size`.
3. **Given** a user runs `c8volt delete pi --state completed --batch-size 250 --limit 25 --auto-confirm`, **When** the command searches process instances, **Then** it fetches in batches of up to 250 but stops after 25 total matched instances.
4. **Given** existing page-size configuration is present and the user does not specify `--batch-size`, **When** an affected command runs in search mode, **Then** the configured default page size continues to apply as it did for the previous page-size flag.

---

### User Story 3 - Reject Ambiguous or Invalid Flag Combinations (Priority: P3)

As a CLI user or script author, I want invalid or removed flag combinations to fail clearly so that scripts do not accidentally rely on ambiguous process-instance command behavior.

**Why this priority**: The flag rename intentionally removes `--count` and constrains `--limit` to search mode, so clear failures are required for safe migration.

**Independent Test**: Run affected commands with removed `--count`, non-positive `--limit`, invalid `--batch-size`, and `--limit` combined with `--key`, then verify the repository's standard invalid-arguments behavior.

**Acceptance Scenarios**:

1. **Given** a user runs any affected process-instance command with `--count`, **When** argument parsing and validation run, **Then** the command fails clearly because `--count` is no longer accepted.
2. **Given** a user runs a direct key-based workflow with `--key` and `--limit`, **When** validation runs, **Then** the command rejects the combination because `--limit` applies only to search/list mode.
3. **Given** a user supplies `--limit 0`, `--limit -1`, or another non-positive value, **When** validation runs, **Then** the command reports that `--limit` must be a positive integer.
4. **Given** a user supplies an invalid `--batch-size`, **When** validation runs, **Then** the command applies the same numeric validation rules previously used for the paging-size flag.

---

### User Story 4 - Discover the New Semantics in Help and Docs (Priority: P4)

As a CLI user reading examples, help, or documentation, I want process-instance command docs to distinguish total limits from per-batch sizing so that I choose the right flag for scripting and operations.

**Why this priority**: This change removes a user-facing flag and adds new semantics, so documentation must move in lockstep with command behavior.

**Independent Test**: Inspect command help, README process-instance paging examples, and generated CLI docs to verify they use only `--batch-size`/`-n` and `--limit`/`-l` for affected command paths.

**Acceptance Scenarios**:

1. **Given** a user views help for `get process-instance`, `cancel process-instance`, or `delete process-instance`, **When** flags are listed, **Then** help text clearly distinguishes batch size from total limit.
2. **Given** a user reads README examples for process-instance paging, **When** examples mention affected command paths, **Then** they use `--batch-size` and `--limit` rather than `--count`.
3. **Given** generated CLI docs are refreshed, **When** the affected pages are inspected, **Then** removed `--count` references are absent for the affected command paths.

### Edge Cases

- A limit smaller than the batch size must return or process only the limited number of matches and must not continue to a later page.
- A limit equal to the number of matches returned on a page must stop without a continuation prompt, even if the backend indicates more matches remain.
- A limit larger than the first batch must continue to later pages only until the limit is reached or no more matches remain.
- A final page containing more matches than the remaining limit must be truncated to the remaining limit before output or destructive processing.
- `--limit` must not alter `--total`; count-only output remains a separate mode from limited detail output, and combining `--total` with `--limit` must be rejected as mutually exclusive.
- Interactive one-line mode, `--auto-confirm`, `--json`, and `--automation` must all stop once the configured limit is satisfied.
- Direct key-based workflows must remain non-paged and must reject `--limit`.
- `--count` must not remain as an alias on the affected command paths.
- Worker help text and any related output must stop referring to `--count` for the affected command paths.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add `--limit` and short flag `-l` to search-mode `get process-instance`, `cancel process-instance`, and `delete process-instance`.
- **FR-002**: `--limit` MUST cap the total number of matched process instances returned or processed across all pages for the current command execution.
- **FR-003**: The command MUST stop fetching later pages once the configured limit is reached.
- **FR-004**: The command MUST not prompt for continuation after the configured limit has been reached.
- **FR-005**: The command MUST apply `--limit` consistently in interactive one-line mode, `--auto-confirm`, `--json`, and `--automation`.
- **FR-006**: The system MUST reject `--limit` when direct `--key` workflows are used.
- **FR-007**: The system MUST validate `--limit` as a positive integer.
- **FR-008**: The system MUST add `--batch-size` as the per-batch search size flag for the affected process-instance command paths.
- **FR-009**: The system MUST keep `-n` as the short flag for `--batch-size`.
- **FR-010**: `--batch-size` MUST preserve the current paging behavior previously exposed through `--count`.
- **FR-011**: `--batch-size` MUST be combinable with `--limit`.
- **FR-012**: The system MUST apply the same numeric validation rules to `--batch-size` that it currently applies to paging size.
- **FR-013**: The system MUST remove `--count` from the affected `get process-instance`, `cancel process-instance`, and `delete process-instance` command interfaces without keeping it as an alias.
- **FR-014**: Use of removed `--count` on affected command paths MUST fail clearly with the repository's standard invalid-arguments behavior.
- **FR-015**: `--total` MUST remain separate from `--limit`, and commands MUST reject `--total` combined with `--limit` as mutually exclusive.
- **FR-016**: Progress output MUST truthfully indicate when execution stopped because no more matches remained, the user aborted continuation, or the configured limit was reached.
- **FR-017**: Command examples and help text MUST clearly distinguish per-batch size from total match limit.
- **FR-018**: README examples, explanations of process-instance paging behavior, and generated CLI docs MUST be updated to use only `--batch-size`/`-n` and `--limit`/`-l` for affected command paths.
- **FR-019**: Automated test coverage MUST cover `get pi`, `cancel pi`, and `delete pi` with `--limit` across multiple pages.
- **FR-020**: Automated test coverage MUST cover stop-without-prompt behavior once a limit is reached.
- **FR-021**: Automated test coverage MUST cover `--auto-confirm`, `--json`, and `--automation` combined with `--limit`.
- **FR-022**: Automated test coverage MUST cover combined `--batch-size` and `--limit`.
- **FR-023**: Automated test coverage MUST cover rejection of `--limit` with `--key`.
- **FR-024**: Automated test coverage MUST cover rejection of removed `--count` on affected command paths.
- **FR-025**: Automated test coverage MUST cover rejection of `--total` combined with `--limit`.

### Key Entities *(include if feature involves data)*

- **Process-Instance Match Limit**: The user-provided positive integer that caps how many matched process instances may be returned or processed across the entire command execution.
- **Process-Instance Batch Size**: The user-provided or configured page size used for each paged search request.
- **Remaining Limit Window**: The number of additional matched process instances that may still be returned or processed before the limit is satisfied.
- **Limited Page Result**: A page of matching process instances truncated to the remaining limit before rendering, cancellation, or deletion.
- **Limit Stop Reason**: The operator-visible completion state indicating that execution stopped because the configured total match limit was reached.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated tests show `get pi --limit N` returns exactly the first `N` matched process instances across multiple pages.
- **SC-002**: Automated tests show search-based `cancel pi --limit N --auto-confirm` processes exactly the first `N` matched process instances across multiple pages.
- **SC-003**: Automated tests show search-based `delete pi --limit N --auto-confirm` processes exactly the first `N` matched process instances across multiple pages.
- **SC-004**: Automated tests show no continuation prompt occurs after the configured limit is reached.
- **SC-005**: Automated tests show `--batch-size` and `-n` control per-batch page size independently from `--limit` and `-l`.
- **SC-006**: Automated tests show removed `--count` fails on the affected command paths.
- **SC-007**: Automated tests show `--limit` with `--key` fails on direct key-based workflows.
- **SC-008**: Automated tests show `--total` with `--limit` fails as a mutually exclusive flag combination.
- **SC-009**: Help output, README examples, and generated CLI docs distinguish batch size from total limit and no longer document `--count` for affected command paths.

## Assumptions

- The affected command paths are `get process-instance`, `cancel process-instance`, and `delete process-instance`, including their `pi` aliases.
- Existing paged search orchestration from the process-instance paging feature remains the preferred implementation seam.
- Existing direct key-based process-instance workflows are outside the limit scope and must remain non-paged.
- The previous shared page-size configuration remains valid and should map to the new `--batch-size` terminology in user-facing documentation.
- `--count` remains outside this change for unrelated command families, such as `run process-instance`, where it has a different meaning.
- Repository-standard invalid argument handling is sufficient for removed and invalid flags as long as the error is clear.
- Documentation generated from Cobra command metadata should be regenerated rather than edited by hand when command help changes.
