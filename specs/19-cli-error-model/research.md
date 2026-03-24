# Research: Review and Refactor CLI Error Code Usage

## Decision 1: Keep `c8volt/ferrors` as the single CLI classification and exit-mapping layer

- **Decision**: Consolidate the shared CLI failure model around `c8volt/ferrors`, using it as the only package responsible for mapping normalized error classes to exit codes and operator-facing failure output.
- **Rationale**: The repository already routes most command failures through `ferrors.HandleAndExit`, and the package already owns the current exit-code mapping. Extending that package is lower risk than introducing a parallel classifier because all command families can keep their existing `HandleAndExit` integration pattern while the model becomes more intentional and complete.
- **Alternatives considered**:
  - Introduce a new CLI error package beside `ferrors`: rejected because it would split ownership and force broad call-site churn without adding clear value.
  - Keep per-command ad hoc error mapping: rejected because it preserves the inconsistency this issue is meant to fix.
  - Push CLI classification into `internal/exitcode`: rejected because exit codes are only one part of the contract; message and class normalization also belong in the same boundary.

## Decision 2: Use a small bounded set of CLI error classes that map to existing exit codes

- **Decision**: Define a compact shared set of CLI failure classes such as invalid input, local precondition or configuration failure, unsupported capability, not found, conflict, timeout, unavailable or retryable remote failure, and internal error, then map those classes onto the existing `internal/exitcode` constants where possible.
- **Rationale**: The current repository already exposes stable exit-code constants and documents script-friendly failure handling. Reusing those exit codes preserves compatibility while giving maintainers and machine callers a clearer semantic layer than raw wrapped error strings alone.
- **Alternatives considered**:
  - Expand the numeric exit-code set substantially: rejected because it would increase compatibility risk and documentation churn.
  - Keep only textual classes without stable exit semantics: rejected because automation depends on predictable exit behavior.
  - Collapse everything into a generic failure and rely on message text: rejected because it does not satisfy the scripting and AI-agent goals in the spec.

## Decision 3: Normalize upstream errors into CLI classes through repository-native sentinels

- **Decision**: Continue translating domain and service sentinel errors into CLI-facing classes through repository-native sentinels, extending the current `ferrors.FromDomain` pattern to cover unsupported-version, validation, malformed-response, local configuration, and representative command-level flag validation paths.
- **Rationale**: The repository already defines meaningful sentinels in `internal/domain/errors.go`, `internal/services/errors.go`, and `cmd/cmd_errors.go`. A shared normalization layer can map those to stable CLI classes without requiring every command to understand the entire downstream error surface.
- **Alternatives considered**:
  - Match error strings at command call sites: rejected because it is brittle and hard to extend safely.
  - Remove sentinel usage and classify only by HTTP status: rejected because not all important failures come from remote HTTP responses.
  - Require every service package to return CLI-specific errors directly: rejected because it would leak CLI concerns into internal service boundaries.

## Decision 4: Preserve `--no-err-codes` as an exit-code override, not a classification bypass

- **Decision**: Keep `--no-err-codes` as a final exit-code suppression behavior that returns `0` while preserving the shared error classification and consistent failure message rendering underneath.
- **Rationale**: Existing documentation already promises that `--no-err-codes` suppresses error codes. The clarified spec requires compatibility with that behavior, so the model should not treat the flag as permission to skip classification or render unrelated ad hoc output.
- **Alternatives considered**:
  - Make `--no-err-codes` disable both classification and non-zero exit behavior: rejected because it would undermine the shared failure model and create a second parallel behavior path.
  - Remove or weaken `--no-err-codes`: rejected because it would break documented operator workflows.

## Decision 5: Sweep all command families through common failure helpers instead of redesigning command trees

- **Decision**: Apply the shared model across the existing command surface by standardizing how `PersistentPreRunE`, command validation, service creation, remote API calls, and render/write failures are wrapped before `HandleAndExit`, without changing Cobra structure, command names, or flag layouts.
- **Rationale**: The constitution and repo guidance both favor compatible, repository-native changes. The issue is about failure behavior, not command-tree redesign, so the rollout should happen by improving existing command handlers and shared helpers rather than by restructuring commands.
- **Alternatives considered**:
  - Rewrite commands into a new command execution framework: rejected because it is far broader than the issue and increases regression risk.
  - Limit the rollout to a subset of command families: rejected because the clarified spec now requires all existing CLI commands.

## Decision 6: Validate the error model primarily through subprocess CLI tests

- **Decision**: Add focused subprocess-based CLI tests for representative root, get, run, deploy, cancel, delete, expect, walk, embed, and config failure paths, while keeping unit coverage around shared error classification helpers where helpful.
- **Rationale**: `ferrors.HandleAndExit` terminates with `os.Exit`, and repository guidance already calls out subprocess helpers for these paths. Subprocess execution is the most reliable way to prove exit codes, stderr output, and `--no-err-codes` compatibility for real command wiring.
- **Alternatives considered**:
  - Test only helper functions and mock exit behavior: rejected because it would miss Cobra wiring and root pre-run failures.
  - Depend only on broad `make test`: rejected because the feature needs targeted regression proof for failure semantics.

## Decision 7: Update user-facing scripting documentation where failure semantics change

- **Decision**: Plan updates to `README.md` and `docs/index.md` for the scripting and error-code sections, and regenerate `docs/cli/` only if command help text or flag descriptions are intentionally changed during implementation.
- **Rationale**: The feature changes user-visible failure semantics and the repository already documents error-code behavior. The generated CLI docs are sourced from Cobra metadata, so they only need regeneration when help text changes rather than when runtime failure mapping alone changes.
- **Alternatives considered**:
  - Skip documentation because commands themselves stay the same: rejected because the failure contract is user-visible.
  - Hand-edit generated CLI docs: rejected because the repository already requires regeneration from Cobra metadata.
