# Research: Fix Terminal Command Completion Suggestion Formatting

## Decision 1: Fix the issue in `c8volt` command metadata and integration points, not in Cobra

- **Decision**: Treat Cobra completion behavior as fixed and solve the bug by adjusting the `c8volt` command tree, help text, completion metadata, and adjacent integration points that shape what Cobra emits.
- **Rationale**: The feature clarification explicitly rules out changing Cobra, and the repository already centralizes CLI behavior in `cmd/` through command metadata and root-command wiring.
- **Alternatives considered**:
  - Patch or fork Cobra completion behavior: rejected because the scope guard for this feature explicitly forbids it.
  - Introduce a custom standalone completion subsystem: rejected because it would add repository-incompatible complexity for a narrow CLI defect.

## Decision 2: Keep normal user-facing suggestions, but suppress internal helper leakage and usage-style dumps

- **Decision**: Preserve normal user-facing completion candidates while ensuring internal completion helpers and verbose help or usage text do not surface as ordinary interactive suggestions.
- **Rationale**: The issue report shows the regression as suggestion formatting and internal output leakage, not as missing candidates. The safest behavioral correction is to keep discovery intact while removing noisy or internal-only output.
- **Alternatives considered**:
  - Strip all descriptions and show names only: rejected because the clarification accepted concise descriptions when available.
  - Accept the noisy output as Cobra internals: rejected because the issue scope and clarification assign responsibility to `c8volt` behavior.

## Decision 3: Preserve concise descriptions where `c8volt` already provides them

- **Decision**: Retain concise candidate descriptions when they already exist in `c8volt`, but prevent fallback to full help or usage text in interactive completion output.
- **Rationale**: This balances discoverability with readability and matches the clarified feature requirement for candidate descriptions.
- **Alternatives considered**:
  - Remove all descriptions from completions: rejected because it reduces usability without being required by the feature.
  - Reuse full command help text as the description source: rejected because it recreates the noisy output shape reported in the issue.

## Decision 4: Use focused command-level completion regression tests as the primary proof

- **Decision**: Add command-level regression coverage in `cmd/` for one representative top-level command path, one nested subcommand path, and one flag-completion path, using the repository’s current root-command execution patterns.
- **Rationale**: The user-visible behavior lives at the Cobra command surface, and the repository already uses command-level tests and helper-process tests where exit or rendering behavior matters.
- **Alternatives considered**:
  - Rely only on manual shell testing: rejected because the constitution requires automated validation before completion.
  - Rely only on lower-level unit tests around helper functions: rejected because they would not prove the interactive completion behavior visible to users.

## Decision 5: Treat documentation impact as conditional on public help changes

- **Decision**: Update help text first if user-visible completion behavior or command descriptions change, regenerate `docs/cli/` with `make docs` when that happens, and review `README.md` only if it documents completion usage that would otherwise become stale.
- **Rationale**: The repository treats CLI reference pages as generated from Cobra metadata, and this feature may be resolved entirely through metadata changes that naturally flow into generated docs.
- **Alternatives considered**:
  - Hand-edit generated CLI docs: rejected because the repository already has a generation path.
  - Skip docs review because the fix is small: rejected because the constitution requires documentation parity for user-visible CLI changes.
