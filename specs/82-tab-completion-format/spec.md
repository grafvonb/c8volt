# Feature Specification: Fix Terminal Command Completion Suggestion Formatting

**Feature Branch**: `82-tab-completion-format`  
**Created**: 2026-03-24  
**Status**: Draft  
**Input**: User description: "https://github.com/grafvonb/c8volt/issues/82"

## GitHub Issue Traceability

- **Issue Number**: `82`
- **Issue URL**: `https://github.com/grafvonb/c8volt/issues/82`
- **Issue Title**: `bug(completion): bad suggestion formatting when pressing Tab for command completion in the terminal`

## Clarifications

### Session 2026-03-24

- Q: Should this feature change Cobra completion behavior itself or stay within `c8volt` completion metadata and integration points? → A: Keep Cobra unchanged; limit the fix to `c8volt` command/completion metadata and integration points.
- Q: How should `c8volt` render completion candidates when descriptions are available? → A: Show candidate names plus concise descriptions when available, but never dump full usage/help text.
- Q: What representative completion flows must regression coverage include? → A: One top-level command path, one nested subcommand path, and one flag-completion path.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Get clean completion suggestions in the terminal (Priority: P1)

As a terminal user, I want pressing Tab to show clean command completion suggestions so that I can continue typing without reading malformed or noisy output.

**Why this priority**: The issue is a direct usability bug in the primary completion flow, so restoring clean suggestions is the most immediate user value.

**Independent Test**: A reviewer can install the generated shell completion, trigger completion for a representative command in the terminal, and confirm the suggestions appear as a readable candidate list rather than a dumped usage block.

**Acceptance Scenarios**:

1. **Given** shell completion is installed for a supported terminal workflow, **When** the user presses Tab for a partially typed `c8volt` command, **Then** the terminal shows clean completion suggestions instead of malformed help-like output.
2. **Given** multiple matching completions exist, **When** the suggestion list is rendered, **Then** each suggestion is presented in a readable format that does not break the terminal prompt layout.

---

### User Story 2 - See only relevant completion candidates (Priority: P2)

As a CLI user, I want completion output to show relevant commands, flags, and values so that I can discover valid inputs without seeing internal or confusing entries.

**Why this priority**: Correct formatting is not enough if the completion list still exposes confusing or unintended suggestions that reduce trust in the feature.

**Independent Test**: A reviewer can trigger completion at representative command and flag positions and confirm the suggestions contain relevant user-facing candidates without internal completion plumbing or unrelated help text.

**Acceptance Scenarios**:

1. **Given** the user requests command completion in a normal interactive flow, **When** suggestions are shown, **Then** internal completion helper commands are not presented as normal user choices.
2. **Given** the user completes flags or subcommands, **When** the candidate list is rendered, **Then** the output stays focused on valid user-facing options for that position.

---

### User Story 3 - Keep completion behavior dependable across common command paths (Priority: P3)

As a maintainer, I want the completion experience to stay dependable across representative command paths so that future changes do not reintroduce broken suggestion rendering.

**Why this priority**: Completion bugs are easy to miss without explicit regression expectations, so the fix needs durable validation around representative workflows.

**Independent Test**: Reviewers can exercise representative top-level commands, subcommands, and flag completions and confirm that completion stays readable and context-appropriate across those flows.

**Acceptance Scenarios**:

1. **Given** a representative top-level or nested command path, **When** completion is triggered, **Then** the rendered suggestions remain readable and context-appropriate.
2. **Given** the completion behavior is updated, **When** automated or scripted verification is run for the representative flows in scope, **Then** regressions in suggestion rendering or candidate selection are detected.

### Edge Cases

- Completion may be triggered when the user has typed trailing spaces, partial flags, or partially completed subcommands, and suggestion rendering must remain readable in each case.
- Commands with many global flags must not cause the suggestion view to collapse into raw usage output or duplicated noise.
- Internal completion plumbing may exist for shell integration but must not leak into normal user-facing candidate lists.
- The fix must preserve useful descriptive context where supported without overwhelming the user with full help text in the suggestion area.
- Candidate descriptions may be present for some suggestions and absent for others, and both cases must render cleanly without falling back to verbose help output.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST present shell completion suggestions in a readable terminal-friendly format when users trigger completion for `c8volt` commands.
- **FR-002**: The system MUST avoid rendering full help or usage output as the normal completion suggestion list during interactive completion flows.
- **FR-003**: The system MUST keep completion suggestions focused on valid user-facing commands, flags, or values for the current cursor position.
- **FR-004**: The system MUST prevent internal completion helper commands from appearing as normal user-facing suggestions in standard completion flows.
- **FR-005**: The system MUST preserve prompt readability by avoiding malformed spacing, line breaks, or suggestion text that disrupts the terminal layout.
- **FR-006**: The system MUST behave consistently across representative top-level commands, nested command paths, and flag-completion flows in scope.
- **FR-007**: The system MUST define how the corrected completion behavior is verified for representative interactive completion scenarios.
- **FR-008**: The system MUST keep the change bounded to completion rendering and candidate-selection behavior rather than altering unrelated command semantics.
- **FR-009**: The system MUST deliver the fix through `c8volt` command/completion metadata and integration points without modifying Cobra framework completion behavior itself.
- **FR-010**: When `c8volt` provides completion description text, the system MUST render concise candidate descriptions without falling back to full usage or help output.
- **FR-011**: Regression verification for this feature MUST cover at least one representative top-level command path, one nested subcommand path, and one flag-completion path.

### Key Entities *(include if feature involves data)*

- **Completion Suggestion**: A user-facing candidate presented when the shell requests command, flag, or value completion.
- **Completion Context**: The current command-line position that determines which suggestions are valid and how they should be displayed.
- **Terminal Completion View**: The rendered suggestion output shown to the user after pressing Tab in an interactive terminal session.
- **Internal Completion Helper**: A shell-integration-only completion path or helper entry that should not surface as a normal user-facing suggestion.

## Assumptions

- The issue scope is limited to command completion behavior in terminal usage and does not require redesigning unrelated help output.
- The primary user-visible bug is malformed or noisy completion suggestion rendering rather than missing business functionality.
- Supported completion workflows should continue to provide discoverability, but only through user-facing candidates that match the current completion context.
- Cobra's standard completion framework behavior is treated as fixed for this feature, so any correction must come from `c8volt` configuration, metadata, or integration choices.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In the representative completion flows covered by this feature, pressing Tab no longer produces usage-style output in place of normal completion suggestions.
- **SC-002**: Reviewers can confirm that representative completion lists contain only relevant user-facing candidates for the current cursor position.
- **SC-003**: Internal completion helper commands no longer appear as normal suggestions in the representative interactive flows in scope.
- **SC-004**: Verification for the representative completion flows detects regressions in suggestion readability or candidate relevance before release.
- **SC-005**: The agreed regression coverage set includes at least one top-level command path, one nested subcommand path, and one flag-completion path.
