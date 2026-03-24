# Data Model: Fix Terminal Command Completion Suggestion Formatting

## Overview

This feature corrects interactive CLI completion behavior. The model is transient and request-oriented rather than persistent.

## Entities

### Completion Context

- **Purpose**: Represents the command-line state Cobra receives when `c8volt` completion is triggered.
- **Fields**:
  - `commandPath`: the current top-level or nested command path being completed
  - `args`: already-parsed arguments that determine the current completion position
  - `toComplete`: the partial token currently being completed
  - `completionKind`: whether the request is for commands, flags, or flag values
- **Validation rules**:
  - The context must resolve to one user-facing completion mode at a time.
  - Representative regression coverage must include one top-level path, one nested path, and one flag-completion path.

### Completion Candidate

- **Purpose**: Represents one user-facing suggestion shown in the terminal.
- **Fields**:
  - `name`: the command, flag, or value shown to the user
  - `description`: optional concise explanation shown alongside the candidate
  - `isInternalHelper`: whether the candidate is an internal completion-only helper that must not appear as a normal suggestion
  - `renderable`: whether the candidate is valid for user-facing interactive display
- **Validation rules**:
  - `isInternalHelper=true` candidates must not appear in normal interactive suggestion output.
  - If `description` is present, it must stay concise and must not expand into full usage or help output.

### Completion Output View

- **Purpose**: Captures the rendered suggestion output that the terminal receives from `c8volt` completion.
- **Fields**:
  - `candidates`: ordered list of user-facing completion candidates
  - `hasUsageDump`: whether full help or usage text leaked into the suggestion stream
  - `hasPromptBreakage`: whether spacing or line formatting breaks normal prompt readability
  - `sourceContext`: the completion context that produced the output
- **Validation rules**:
  - `hasUsageDump` must be false for the representative interactive flows in scope.
  - `hasPromptBreakage` must be false for the representative interactive flows in scope.
  - Output must remain readable whether candidates have descriptions or not.

### Completion Regression Scenario

- **Purpose**: Defines one automated verification slice for the feature.
- **Fields**:
  - `scenarioType`: top-level command, nested subcommand, or flag completion
  - `inputShape`: representative partial command line used to trigger completion
  - `expectedCandidates`: user-facing suggestions that should remain visible
  - `forbiddenOutput`: internal helper names, usage text, or malformed formatting that must not appear
- **Validation rules**:
  - At least one scenario of each required type must be covered.
  - Forbidden output must be asserted explicitly so the original regression shape is guarded.

## Relationships

- A `Completion Context` produces one `Completion Output View`.
- A `Completion Output View` contains zero or more `Completion Candidate` entries.
- A `Completion Regression Scenario` validates the relationship between a `Completion Context` and its `Completion Output View`.

## Notes

- No persistent storage, API schema, or long-lived domain entity is added by this feature.
- The feature should remain inside current Cobra and `cmd/` patterns unless implementation reveals a small reusable helper is clearly justified.
