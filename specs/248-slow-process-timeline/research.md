# Research: Slow Process Timeline Readability

## Decision: Keep the complete analysis payload unchanged and summarize only during human rendering

**Rationale**: The existing slow-process analysis service already builds a complete, filtered, ordered timeline and JSON exposes that payload. The feature changes default human readability only, so selecting hotspot rows in the command renderer preserves JSON, keys-only, selection, duration, and comparison semantics.

**Alternatives considered**:

- Filter the timeline in `internal/services/ops`: rejected because it would risk changing JSON output and downstream machine consumers.
- Add a second service method for summary mode: rejected because the same analysis result already contains all information needed for presentation.

## Decision: Add `--with-full-timeline` as a command-local human rendering flag

**Rationale**: The flag controls only whether human output renders the compact summary or the complete chronological timeline. Keeping it command-local avoids leaking a human-only display preference into service or public facade contracts unless implementation discovers the current render dispatch cannot access it cleanly.

**Alternatives considered**:

- Add the flag to `SlowProcessAnalysisRequest`: rejected unless implementation needs it, because request fields are part of the structured payload and would make a human-only option visible in JSON.
- Use an environment variable or config option: rejected because operators need an explicit, discoverable CLI switch per invocation.

## Decision: Default summary rows are completed contributors at or above 1%, plus active and incident-bearing rows

**Rationale**: Clarification selected the 1% threshold. This keeps meaningful slow contributors visible, hides noisy sub-1% completed rows, and ensures operationally important active and incident-bearing rows are not lost.

**Alternatives considered**:

- Always show exactly the top three completed contributors: rejected because several meaningful contributors above 1% could be hidden in longer timelines.
- Show up to five completed contributors: rejected because a fixed count is less tied to the process-duration share users see in output.

## Decision: Preserve full chronological output as the current timeline style

**Rationale**: Issue #248 explicitly requires audit/debug access to the current detail style. Full-timeline mode should therefore reuse existing element and transition row formatting, including element instance keys and zero-duration rows, while still honoring existing detail filters.

**Alternatives considered**:

- Create a new verbose style for full timeline: rejected because the feature asks to preserve the current detail behavior and changing it would expand test/documentation scope.
- Add separate flags for gateways, transitions, or zero-duration rows: rejected because one explicit full-timeline switch is simpler and matches the issue.

## Decision: Hidden-row summaries are renderer-calculated from omitted visible-detail rows

**Rationale**: Hidden counts are a human readability cue, not analysis data. Calculating them while choosing summary rows avoids service churn and lets tests directly verify default human output.

**Alternatives considered**:

- Add hidden-row counts to JSON: rejected because JSON output must remain unchanged.
- Count all original unfiltered service rows before detail filters: rejected because hidden-row messaging should describe what the default human view omitted from the analyzed detail set visible under the current filters.

## Decision: Documentation and command metadata must describe default compact output and the full-timeline escape hatch

**Rationale**: The default output changes, and the project constitution requires user-visible command behavior to be reflected in help, README, examples, and generated CLI docs.

**Alternatives considered**:

- Update tests only: rejected because operators rely on CLI help and docs for discoverability.
- Hand-edit generated docs: rejected because repository guidance requires regenerating CLI docs with `make docs-content`.
