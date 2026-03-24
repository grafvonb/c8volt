# CLI Command Contract: Terminal Completion Suggestion Formatting

## Scope

This contract defines the expected public CLI behavior for interactive `c8volt` shell completion after the formatting fix.

## Preferred Behavior

- **Entry point**: Existing `c8volt` shell completion output generated from the Cobra-based command tree
- **Purpose**: Provide readable interactive command, subcommand, flag, and flag-value suggestions without exposing internal completion plumbing or verbose help output
- **Expected Behavior**:
  - Uses the existing `c8volt` command tree and completion integration
  - Keeps Cobra framework behavior unchanged
  - Surfaces normal user-facing candidates for the active completion context
  - Shows concise candidate descriptions where `c8volt` already provides them
  - Avoids full usage/help dumps in place of the suggestion list

## Output Rules

- Interactive completion output must show only user-facing candidates relevant to the current cursor position.
- Internal helper entries such as completion-only plumbing commands must not appear as normal suggestions.
- If descriptions are present, they must remain concise and must not expand into full help or usage text.
- Candidate lists must remain readable whether descriptions are present or absent.
- Prompt layout must not be broken by malformed spacing or line formatting in the suggestion output.

## Scope Guard

- The fix must be implemented through `c8volt` command/completion metadata or integration points.
- Cobra framework behavior itself is out of scope for this feature.
- The feature must not introduce a parallel completion subsystem or a new top-level command family.

## Representative Regression Signals

- One top-level command completion path returns readable user-facing suggestions.
- One nested subcommand completion path returns readable user-facing suggestions.
- One flag-completion path returns readable user-facing suggestions.
- None of the covered paths emit internal helper names as normal suggestions.
- None of the covered paths replace the suggestion list with usage/help output.

## Help and Documentation Rules

- If the implementation changes user-visible command descriptions or completion-command help text, Cobra help text must be updated first.
- Generated CLI docs must be refreshed with `make docs` when public help changes.
- `README.md` updates are required only if existing completion guidance would otherwise become inaccurate or incomplete.
