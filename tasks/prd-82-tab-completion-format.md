# PRD: Fix Terminal Command Completion Suggestion Formatting

## Overview

Fix the malformed `c8volt` terminal Tab-completion experience so interactive shell completion returns readable user-facing suggestions instead of noisy help-like output or internal completion entries. The work must keep Cobra itself unchanged, stay inside `c8volt` command/completion metadata and integration points, preserve concise descriptions where available, and prove the shipped behavior with focused completion regression coverage plus repository-required validation.

## Goals

- Restore readable interactive completion suggestions for `c8volt`.
- Prevent full help or usage output from replacing the normal completion suggestion list.
- Prevent internal completion helper entries from appearing as normal user-facing suggestions.
- Preserve concise candidate descriptions where `c8volt` already provides them.
- Keep the fix bounded to repository-native command/completion metadata and integration points.
- Prove the behavior with focused regression coverage for one top-level path, one nested path, and one flag-completion path.
- Finish with targeted validation and `make test`.

## User Stories

### US-001: Lock the completion fix to `c8volt` metadata and integration points
**Description:** Establish the shared completion seam in the existing root command and command checks so the feature is implemented entirely inside `c8volt` without changing Cobra behavior.

**Acceptance Criteria:**
- The implementation identifies `cmd/root.go` and `cmd/cmd_checks.go` as the primary shared completion boundary for this feature.
- The design keeps Cobra framework behavior unchanged and does not introduce a parallel completion subsystem.
- The resulting approach remains compatible with the existing `c8volt` command tree and utility-command handling.

### US-002: Stop usage-style output from replacing normal completion suggestions
**Description:** Correct the root-command completion behavior so representative top-level completion flows render readable suggestions instead of malformed help-like output.

**Acceptance Criteria:**
- A representative top-level completion path no longer emits usage-style output in place of normal suggestions.
- The resulting suggestion output remains readable and does not break prompt layout.
- Focused command-level regression tests prove the top-level completion rendering behavior.

### US-003: Filter internal completion helpers from normal interactive suggestions
**Description:** Ensure normal user-facing completion output excludes internal completion plumbing while keeping relevant visible candidates intact.

**Acceptance Criteria:**
- Internal helper entries such as `__complete` do not appear as normal user-facing suggestions in representative interactive completion flows.
- Representative top-level and nested completion paths still surface relevant user-facing commands or subcommands.
- The implementation keeps the filtering behavior inside `c8volt` command/completion metadata or integration points.

### US-004: Preserve concise descriptions without falling back to full help text
**Description:** Keep useful concise candidate descriptions where `c8volt` already provides them, while preventing completion output from expanding into full help or usage text.

**Acceptance Criteria:**
- Representative completion candidates show concise descriptions when `c8volt` provides them.
- Suggestions without descriptions still render cleanly.
- No representative completion path falls back to full help or usage text to populate descriptions.

### US-005: Prove the completion contract across representative command paths
**Description:** Add focused regression coverage for one top-level path, one nested path, and one flag-completion path so the fixed completion behavior remains durable.

**Acceptance Criteria:**
- Automated regression coverage exists for one top-level completion path, one nested subcommand path, and one flag-completion path.
- The covered scenarios assert both expected visible candidates and forbidden output such as internal helper names or usage dumps.
- The regression suite uses the repository’s existing root-command or helper-process testing patterns in `cmd/`.

### US-006: Keep help and generated CLI docs aligned with shipped completion behavior
**Description:** Update any affected help text and generated CLI docs so published CLI guidance matches the final completion contract.

**Acceptance Criteria:**
- Any user-visible help text affected by the final metadata change is updated in `cmd/root.go`.
- Generated CLI reference output is refreshed with `make docs` when public help text changes.
- `README.md` is updated only if current completion guidance would otherwise become inaccurate or incomplete.

### US-007: Prove completion with targeted validation and repository checks
**Description:** Finish the feature with explicit proof that the final completion behavior is correct, documented, and repository-compliant.

**Acceptance Criteria:**
- The quickstart validation steps in `specs/82-tab-completion-format/quickstart.md` are runnable against the final implementation.
- Focused completion regression tests in `./cmd` pass.
- `make test` passes.

## Functional Requirements

- FR-001: Interactive completion for `c8volt` MUST present shell suggestions in a readable terminal-friendly format.
- FR-002: The implementation MUST avoid rendering full help or usage output as the normal completion suggestion list.
- FR-003: Completion suggestions MUST remain focused on valid user-facing commands, flags, or values for the current cursor position.
- FR-004: Internal completion helper commands MUST not appear as normal user-facing suggestions in standard interactive completion flows.
- FR-005: Prompt readability MUST be preserved by avoiding malformed spacing, line breaks, or suggestion output that disrupts terminal layout.
- FR-006: The corrected behavior MUST be proven for at least one top-level path, one nested subcommand path, and one flag-completion path.
- FR-007: The implementation MUST keep Cobra framework behavior unchanged and MUST deliver the fix through `c8volt` command/completion metadata or integration points.
- FR-008: When `c8volt` provides completion description text, the implementation MUST preserve concise descriptions without falling back to full help or usage output.
- FR-009: The feature MUST reuse the existing command structure and repository-native `cmd/` patterns instead of adding a new completion subsystem.
- FR-010: Automated regression coverage MUST live at the command surface where the completion behavior is observable to users.
- FR-011: User-facing help or generated CLI docs MUST be updated when the final metadata change alters public CLI guidance.
- FR-012: Repository validation MUST include focused completion regression tests and `make test`.

## Non-Goals

- Modifying Cobra framework behavior or introducing a Cobra patch.
- Adding a new top-level command family or separate completion subsystem.
- Redesigning unrelated command help output outside the completion bug scope.
- Changing unrelated command semantics, API behavior, or non-completion workflows.
- Removing concise descriptions from completion output when `c8volt` already provides them.
- Broad documentation rewrites unrelated to the final shipped completion behavior.

## Implementation Notes

- Treat `cmd/root.go` and `cmd/cmd_checks.go` as the primary shared boundary for completion visibility and integration behavior.
- Use representative command paths already present in the repository, including top-level command flows and flag completion such as `walk process-instance --mode`, to prove the contract.
- Keep candidate filtering and description behavior repository-native; prefer existing Cobra metadata and command registration patterns over new abstractions.
- Use `cmd/get_test.go` and `cmd/mutation_test.go` as the primary completion regression surface because the user-visible behavior is at the command boundary.
- Regenerate `docs/cli/` with `make docs` only when help text changes are part of the final implementation.
- Preserve repository guidance from `AGENTS.md`, including keeping Cobra unchanged, favoring incremental changes, and running `make test` before completion.

## Validation

- Add or update focused completion regression tests in `cmd/get_test.go` and `cmd/mutation_test.go`.
- Cover one top-level completion path, one nested command path, and one flag-completion path.
- Assert both expected visible candidates and forbidden output such as internal helper names or usage dumps.
- Run the validation flow documented in `specs/82-tab-completion-format/quickstart.md`.
- Run `go test ./cmd -run 'Test.*Completion' -count=1`.
- Run `make test`.
- If help text changes, run `make docs` and verify generated CLI docs match the shipped behavior.

## Traceability

- GitHub Issue: #82
- GitHub URL: https://github.com/grafvonb/c8volt/issues/82
- GitHub Title: bug(completion): bad suggestion formatting when pressing Tab for command completion in the terminal
- Feature Name: 82-tab-completion-format
- Feature Directory: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format
- Spec Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/spec.md
- Plan Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/plan.md
- Tasks Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/82-tab-completion-format/tasks.md
- PRD Path: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/tasks/prd-82-tab-completion-format.md
- Source Status: derived from Speckit artifacts

## Assumptions / Open Questions

- The exact root-command or metadata change that removes the noisy completion output is assumed to be discoverable within the existing `cmd/` surface without requiring a Cobra change.
- Generated CLI docs are assumed to need regeneration only if the final implementation changes public help text rather than completion behavior alone.
