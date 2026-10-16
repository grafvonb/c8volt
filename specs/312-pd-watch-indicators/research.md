# Research: Process Definition Watch Change Indicators

## 1. Ownership and existing refresh boundary

**Decision**: Keep comparison state local to one invocation in `cmd/get_processdefinition_watch.go`; keep display projection and final rendering in `cmd/cmd_views_processdefinition.go`.

**Evidence**: `executeGetProcessDefinitionWatch` already calls `CollectProcessDefinitionWatchSnapshot` serially through `toolx/watch`, returns immediately on collection errors, then repaints and renders. `internal/services/processdefinition/search.go` owns complete collection and canonical ordering. `c8volt/process` exposes the snapshot through the existing facade.

**Rationale**: This is presentation history over completed results, not backend traversal, polling, or retry policy. Existing lifecycle, services, and public contracts need no changes.

**Alternatives considered**: Service/domain change metadata would couple backend facts to CLI display choices. A generic diff framework or changes to the watch runner would broaden the scope without a second consumer.

## 2. Comparing displayed values

**Decision**: Project each definition to owned, unstyled display values indexed by key. Compare semantic display fields before alignment and ANSI decoration. Preserve the incoming ordered slice for rendering and use the map only for lookup/counting.

**Evidence**: `flatRowPD` displays key, tenant, BPMN process ID, combined version/tag, and optional active/completed/canceled/incident statistics. `zeroAsMinus` determines count display. `formatFlatRows` computes widths from string lengths. The statistics tail currently bundles several values into one column.

**Rationale**: Comparing full models would flag undisplayed changes; comparing rendered lines would flag padding and ordering changes. Preserve field identity for individual statistics so a changed active count does not bold unchanged completed/canceled counts. Share a small PD-specific projection between ordinary row composition and watch comparison if needed, preserving ordinary bytes.

**Alternatives considered**: Comparing whole statistics-tail strings is adequate for row markers but too coarse for changed-value highlighting. Parsing final aligned strings is fragile and loses field identity.

## 3. Baseline lifecycle

**Decision**: Use an explicit initialized flag and a key-to-display-values map. Prepare a candidate comparison after complete collection, render it, and replace the baseline only after repaint and watch rendering succeed. Failed collection or output must not advance it. Copy projected values rather than retaining mutable model/statistics pointers.

**Rationale**: An empty successful snapshot is a baseline, while no successful snapshot is not. The first result has zero changes; later empty results remove all previous keys. Committing after successful output avoids acknowledging an unseen refresh.

**Alternatives considered**: Treating an empty map as uninitialized would incorrectly classify definitions after an empty success. Updating incrementally during collection or before output can consume failed/partial results.

## 4. Styling and alignment

**Decision**: Use a small watch-specific bold capability decision based on `cmd.OutOrStdout()` exposing a terminal file descriptor, using the already available `golang.org/x/term`. Disable bold when `TERM` is empty or `dumb`, or `NO_COLOR` is nonempty. Test capability detection separately from deterministic rendering with an explicit styling parameter. Do not add a dependency, flag, global style destination, or change watch eligibility/repaint controls.

**Evidence**: No general bold/color policy exists in `cmd` or `toolx`. `toolx/logging/activity.go` detects terminal capability on its actual writer. Watch repaint in `cmd/cmd_views_rendermode.go` currently emits ANSI unconditionally; it is separate from this feature's bold text.

**Rationale**: Actual output-writer capability avoids assuming terminal stdin means terminal stdout. Conservative fallback preserves meaningful plain markers. Calculate widths and padding from plain cells, then decorate only changed visible tokens; keep padding and adjacent fields outside bold spans. Prefix each formatted row with a one-character marker plus one space, including two leading spaces for unchanged rows.

**Alternatives considered**: Unconditional bold contaminates captured output. Injecting escape sequences before `formatFlatRows` corrupts width calculations. Changing shared formatting to be ANSI-aware expands the regression surface unnecessarily.

## 5. Output compatibility and documentation

**Decision**: Replace only watch's delegation to ordinary list output with a focused view. Print the summary, current rows, and existing `found: N` footer. Empty successful refreshes print summary plus `found: 0`. Preserve normal human/JSON/keys-only/XML behavior and current watch flag exclusions. Update README, command Long help and output-contract notes, then regenerate docs with `make docs-content` during implementation.

**Evidence**: `processDefinitionWatchView` currently delegates to `listProcessDefinitionsView`; `listOrJSONFlat` supplies `found: N`. Existing watch/list parity tests and command metadata explicitly describe identical output, so those assertions must be replaced by explicit watch contracts while preserving ordinary tests.

**Alternatives considered**: Adding marker fields to facade models or shared JSON envelopes would break unrelated consumers. Removing the footer is unnecessary output churn.

## 6. Validation scope

**Decision**: Reuse command watch harnesses with controlled refresh/sleep functions for successful, failed, empty, reordered, and recovered sequences; add exact view assertions for markers, counts, selective bold, and alignment. Capture stdout and stderr separately. Add writer-error coverage for baseline commitment. Retain existing selector, timing, retry, termination, and incompatible-mode tests.

**Rationale**: These tests prove the user-visible behavior at its owning layer without live Camunda mutations or sleeps. Targeted command tests plus generated documentation review cover the planned blast radius. Broader tests are warranted only if implementation changes shared runtime behavior or exposes unresolved failures.

**Alternatives considered**: Live deployment/deletion tests are slow and unnecessary for deterministic presentation comparisons. Renderer-only tests cannot prove that failed refreshes retain the baseline.

All planning questions are resolved; no external library or backend API changes are required.
