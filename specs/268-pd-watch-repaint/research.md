# Research: Process Definition Watch Repaint

## Decision: Reuse Normal Human Result Rendering For Each Refresh

Each watch refresh should render the same process-definition result body as the equivalent non-watch human command.

**Rationale**: The issue explicitly requires watch result rows to match normal human output and removes watch-only labels such as `snapshot 1:`. The existing `listProcessDefinitionsView` path already renders process-definition rows plus `found: N`, so reusing that shape keeps watch and non-watch output aligned.

**Alternatives considered**:

- Keep `processDefinitionWatchSnapshotView` as a separate labeled renderer. Rejected because it is the current defect and introduces watch-only result rows.
- Add timestamps or tick counters to each row. Rejected because the result body would no longer match non-watch human output.
- Remove `found: N` from watch output. Rejected because normal human list output already includes it.

## Decision: Repaint The Terminal Before Rendering Each Successful Refresh

Watch mode should clear/reposition the terminal for each successful refresh before writing the current result body.

**Rationale**: This matches Linux `watch` expectations and prevents the terminal from accumulating stale snapshots. Keeping repaint behavior in the command rendering layer preserves existing service/facade boundaries.

**Alternatives considered**:

- Append a blank line between refreshes. Rejected because it remains a growing log stream.
- Repaint only after the first refresh. Rejected because a stale first view can remain when watch starts in a non-empty terminal.
- Move repaint behavior into `toolx/watch`. Rejected because terminal rendering is command UX, while `toolx/watch` should stay output-agnostic.

## Decision: Keep Refresh Cycles Serial

The implementation should rely on the existing watch runner's sequential tick execution and add tests only if new timing/status hooks are introduced.

**Rationale**: `toolx/watch.Run` calls the tick function synchronously and sleeps only after the tick completes, so refresh cycles do not overlap. The feature needs to preserve and validate that property while adding duration measurement.

**Alternatives considered**:

- Start refreshes on a timer in separate workers. Rejected because it can overlap broad lookups and violates the spec.
- Skip a refresh when the previous one is still running. Rejected as unnecessary with the existing serial loop and would create more complex operator semantics.

## Decision: Measure Refresh Duration Around Collection And Rendering

Refresh duration should cover the operator-visible refresh work: collecting the current snapshot and rendering the repainted result view.

**Rationale**: The operator experiences a refresh as complete only when the new view is visible. Measuring both collection and rendering catches broad lookups and statistics-heavy views without exposing backend-specific timing concepts.

**Alternatives considered**:

- Measure only backend collection. Rejected because the user-visible refresh is not complete until rendering finishes.
- Measure the sleep interval too. Rejected because slow-refresh detection should compare work duration against the configured interval, not include the deliberate wait.

## Decision: Default Slow-Refresh Warnings Are Streak-Based

Default human mode should warn once when a continuous slow-refresh streak starts, suppress repeated warnings while refreshes continue exceeding the interval, and allow a new warning after one refresh completes within the interval.

**Rationale**: This clarification gives operators a clear signal without flooding stderr during long-running broad lookups. It also gives tests a precise reset rule.

**Alternatives considered**:

- Warn on every slow refresh. Rejected because default human mode would become noisy.
- Warn only once for the entire watch run. Rejected because it hides a later slow condition after recovery.
- Show only verbose timing. Rejected because default users still need to know when the configured interval cannot be met.

## Decision: Verbose Mode May Report Per-Refresh Timing

Verbose watch mode should be allowed to emit more detailed refresh timing/status outside the result body.

**Rationale**: The spec calls for more detailed per-refresh timing/status in verbose mode. Routing it away from result stdout preserves the result body contract while helping operators diagnose broad or statistics-heavy watch runs.

**Alternatives considered**:

- Put timing in the repainted result body. Rejected because the result body must match non-watch output.
- Suppress timing entirely. Rejected because slow refresh handling is an explicit acceptance criterion.

## Decision: Preserve Machine-Oriented Watch Rejections

`--watch` must continue rejecting `--json`, `--keys-only`, `--xml`, `--quiet`, and `--automation` before lookup work starts.

**Rationale**: Watch is an interactive human view. Preserving the existing rejection behavior protects deterministic script output and avoids inventing a streaming machine contract.

**Alternatives considered**:

- Emit newline-delimited JSON for watch. Rejected because the command's JSON mode is a finite document contract.
- Stream keys-only rows. Rejected because keys-only output is for finite pipelines.
- Allow quiet watch. Rejected because watch requires visible human output.
