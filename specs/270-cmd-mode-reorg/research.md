# Phase 0 Research: Command Mode And Concern Reorganization

## Decision: Treat Issue #254 Assessment As The Ownership Baseline

**Rationale**: Issue #270 builds directly on the command-layer ownership work from issue #254. The checked-in #254 assessment already classifies command nodes by output contract, automation support, paging behavior, mutation behavior, progress behavior, ownership, execution style, and high-volume risk. Reusing it prevents this feature from re-litigating architecture and keeps the work focused on file cohesion.

**Alternatives considered**:

- Create a new full command assessment before moving files. Rejected because #254 already produced the baseline needed for this feature.
- Start from file size alone. Rejected because the issue explicitly says large files are not automatically wrong when they are cohesive.

## Decision: Keep `cmd` As One Package And Split By File Ownership

**Rationale**: The requested improvement is file cohesion, navigation, test ownership, and maintainability inside the existing command package. A package split would create new import boundaries, increase review scope, and risk behavior changes without a proven ownership boundary.

**Alternatives considered**:

- Move command families into separate packages. Rejected because the issue lists this as a non-goal and no separate package boundary has been proven.
- Introduce a generic command framework. Rejected because it would expand scope and fight the repository-native command patterns.

## Decision: Move Distinct Modes Before Changing Ownership Mechanics

**Rationale**: Mode files are safest when the first step is mechanical: move watch, polling, selector/direct-key, dry-run, progress, or report lifecycles into focused files while preserving behavior. Ownership corrections can then be evaluated with cleaner source boundaries.

**Alternatives considered**:

- Combine moves with behavior cleanups. Rejected because it makes regressions harder to isolate.
- Delay all moves until after backend ownership corrections. Rejected because the immediate problem is command-file cohesion.

## Decision: Keep Render Files Purely Presentational

**Rationale**: Renderer files are part of the CLI output contract. They should own view-model construction, layout, and formatting, but not facade calls, backend orchestration, traversal, polling, mutation planning, or workflow execution.

**Alternatives considered**:

- Leave facade calls in view files when convenient. Rejected because it hides behavioral work inside presentation files and makes output-only changes risky.
- Move all output logic into services. Rejected because services should not own stdout/stderr or render-mode policy.

## Decision: Align Tests With Production Ownership Incrementally

**Rationale**: Large test moves can create noisy diffs that hide behavioral regressions. Tests should move alongside the production concern they verify, one slice at a time, preserving package-level isolation and flag reset behavior.

**Alternatives considered**:

- Perform one large test-only reorganization commit. Rejected because the issue explicitly asks to avoid a single large test-only reorganization commit.
- Leave tests in old files after production moves. Rejected because navigation and ownership remain unclear.

## Decision: Remove Helpers Only After Caller Checks

**Rationale**: This feature may expose dead helpers, but behavior preservation requires checking production, test, examples, and subprocess helper use before deletion. Helper removal should be small and auditable.

**Alternatives considered**:

- Delete helpers based on visual inspection. Rejected because test-only and subprocess-only callers are easy to miss.
- Keep all helpers forever. Rejected because confirmed dead code weakens maintainability and is in scope for the issue.

## Decision: Document Non-Mechanical Ownership Work As Follow-Up

**Rationale**: The issue includes ownership follow-ups such as job update planning, backend-state lookup, mutation-plan construction, orphan filtering, and limit ownership. Some may require facade/service changes and new behavioral validation. Those should not be hidden in file moves.

**Alternatives considered**:

- Fix every ownership concern during reorganization. Rejected because it would mix mechanical moves with architectural changes and enlarge review scope.
- Ignore ownership findings discovered during moves. Rejected because the issue asks for review and follow-up when substantial corrections are found.

## Decision: Validate Behavior With Targeted Output And Documentation Checks

**Rationale**: A behavior-preserving reorganization can still accidentally change output order, prompts, help, generated docs, command metadata, or machine-output cleanliness. Each slice needs targeted tests plus final docs and broad validation.

**Alternatives considered**:

- Rely only on `go test ./cmd`. Rejected because docs diffs and representative output contracts may not be fully covered.
- Run only `make test` at the end. Rejected because slice-level regressions become harder to locate.
