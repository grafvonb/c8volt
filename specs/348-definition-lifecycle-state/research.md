# Research: Process-Definition Lifecycle State

**Date**: 2026-10-27
**Feature**: [spec.md](spec.md)

## 1. Preserve the supplied value with an optional string

**Decision**: Add `State string` with `json:"state,omitempty"` to the existing internal and public process-definition models. Do not reuse the process-instance `State` type or add a definition state machine.

**Rationale**: `internal/clients/camunda/v810/camunda/client.gen.go` already provides `ProcessDefinitionResult.State` as a string-backed generated type. Empty, omitted, and null source values become the empty string and must all be omitted publicly. A nonempty unfamiliar value must survive without validation against a closed enumeration. The current v88/v89 definition result types have no lifecycle state, so unchanged converters naturally leave the new field empty.

**Alternatives considered**: A closed enum loses future values; an instance-state type conflates runtime execution with definition lifecycle; a pointer or availability wrapper adds no distinction required by this specification. No generated-client changes or new dependencies are needed.

## 2. Map once through the existing definition read path

**Decision**: Map `string(r.State)` in `internal/services/processdefinition/v810/convert.go:fromProcessDefinitionResult` and copy `x.State` in `c8volt/process/convert.go:fromDomainProcessDefinition`.

**Rationale**: Lookup and search both use the versioned converter. Collection, page, traversal-result, latest-selection, and watch-snapshot conversions all reuse the facade converter. `internal/services/processdefinition/search.go` retains whole domain objects while sorting, paging, selecting latest versions, and assembling snapshots. None of these operations needs state-specific logic. Statistics enrichment assigns `pd.Statistics` without replacing the definition.

**Testing detail**: v810 search uses `decodeSearchProcessDefinitionsResponse` to decode `resp.Body` again. Fixtures must serialize state into the actual response body; setting only `JSON200.Items` does not prove search propagation. Compare the existing mocked request set for each selected mode rather than hard-coding one universal statistics request count: statistics include separate observations and can paginate.

**Alternatives considered**: Extra lifecycle lookups violate the zero-additional-request requirement. New service/facade methods, version gates that reject older reads, or state-based sorting/filtering would unnecessarily change the current read contract.

## 3. Use one stable optional human column

**Decision**: Extend `processDefinitionDisplay` in `cmd/cmd_views_processdefinition.go` with state. Place lifecycle after version and before the statistics tail. Always reserve the state cell internally, even when empty; do not print an unavailable-state placeholder.

**Rationale**: `cmd/cmd_views_flat.go` already omits all-empty columns across a listing and keeps remaining columns aligned. `compactFlatRow` omits empty cells for a single row. A fixed lifecycle cell prevents statistics from shifting into the state column when some rows omit state. All-state-unavailable output retains the existing human layout. Watch must move statistics formatting from column index 4 to 5 and format lifecycle at index 4.

**Alternatives considered**: Conditionally appending state before statistics breaks mixed-row alignment. Adding lifecycle inside the statistics brackets conflates separate observations. Changing the shared flat renderer or adding a second definition renderer is unnecessary.

## 4. Extend existing watch comparison rather than its lifecycle

**Decision**: Add state to the existing display identity, `processDefinitionChangedFields`, `any()`, `changedFields()`, and value highlighting. Preserve the current candidate/commit model and key-based row membership.

**Rationale**: `cmd/get_processdefinition_watch.go` already builds fresh candidates from each successful snapshot, counts each changed row once, and commits the baseline only after the refresh is rendered successfully. A string comparison covers state-only changes, supplied-to-omitted transitions, unfamiliar states, and skipped transitions. A returned `DELETED` row stays present because row removal is determined by absent keys, not state. No new polling, timers, retry rules, or state ordering is required.

**Alternatives considered**: Keeping a separate lifecycle baseline risks stale values and double-counted updates. Synthesizing `DELETING` or requiring `ACTIVE → DRAINING → DELETED` would invent observations not guaranteed by eventually consistent reads.

## 5. Correct the touched watch logger bypasses narrowly

**Decision**: Route slow-refresh warnings through `renderHumanWarningLine` and verbose timing through `renderHumanLine`. Render the per-refresh summary through a small error-returning helper in the existing definition view file, using the command-context logger's configured handler and level check.

**Rationale**: Current `processDefinitionWatchView` constructs its own INFO plain-time handler. Current slow/verbose refresh diagnostics print directly to stderr. These are pre-existing bypasses of configured destination, level, and format and conflict with FR-015 and repository logging requirements. Retry/stop diagnostics already use the established helpers.

The summary helper must retrieve the context logger with `logging.FromContext`, handle a missing logger as an error rather than use the returned default logger, check `logger.Enabled(ctx, slog.LevelInfo)`, and send the INFO record through `logger.Handler().Handle`. Preserve message text, activity-aware configured writer, inherited destination, and the existing repaint → summary → result-rows ordering. If INFO is disabled, suppression is successful and must not prevent rows or baseline commit. If the enabled summary write fails, propagate the error so the baseline is not committed. Do not add a second logger, an unconditional timestamp, or severity text to the message.

**Alternatives considered**: A direct `renderHumanLine` replacement loses summary write-error propagation because that helper returns no error. Leaving the new plain handler or direct stderr writes in place would preserve a known configuration violation. A general logger redesign is out of scope; only these existing watch paths need correction.

**Compatibility note**: The default configured plain-time logger keeps its normal timestamped form. Explicit structured/plain formats, WARN thresholds, and configured destinations now take effect for these messages. Tests that previously assumed an unconditional plain-time summary must install that format explicitly.

## 6. Reuse isolated tests and the existing draining scenario

**Decision**: Extend the existing definition adapter/facade tests, fake-backed command fixtures, watch harness, and real-PTY tests. Change the existing draining integration observation to bounded `get pd --key ... --json` execution with one-envelope decoding. Keep independent disappearance/cleanup evidence.

**Rationale**: `cmd/get_processdefinition_suspended_test.go` already provides test-owned lookup/search/count responses and request counters. Existing watch tests cover timing, failed reads, repaint failures, optional columns, ANSI highlighting, and separate output streams. `integration/cli/real_state_destructive_test.go:TestRealStateProcessDefinitionDraining` already has a held definition; only its CLI-acceptance observation currently uses a direct generated read.

The required zero-active/nonzero-suspended case is deterministic isolated coverage, not a new live acceptance gate. Keep the live fixture's existing state preparation and cleanup semantics for this issue. In particular, its cleanup currently selects ACTIVE instances; do not introduce suspended fixture work without an owned-state recovery change. No new explicit `--no-wait` scenario or mode matrix is needed. Existing submission flags in the draining fixture are setup behavior, not new coverage.

**Alternatives considered**: A new integration framework, a second draining suite, an engine-backed transition test, or observing every intermediate state adds complexity or violates the implementation testing boundary.

## 7. Validation and delivery size

**Decision**: Planning changes receive document/link/whitespace checks only. Implementation starts with targeted tests in affected packages, adds configured-logger regression assertions for the touched watch paths, regenerates CLI docs after command metadata changes, and compiles edited integration tests without executing them.

**Rationale**: Constitution v2.0.0 requires proportional validation. Existing package-local fixtures cover the changed paths. A full race suite is warranted only if implementation broadens shared helpers or mechanics, or focused failures leave wider behavior unresolved. No live target, cluster inspection, deployment, or cleanup run belongs to implementation validation.

**Alternatives considered**: Running broad/live suites as a planning gate wastes effort and conflicts with repository policy. Treating compilation alone as runtime verification would overstate coverage.

## Sources and resolved questions

- [Feature specification](spec.md) and [issue #348](https://github.com/grafvonb/c8volt/issues/348), including the backend-deletion clarification.
- [Camunda resource deletion](https://docs.camunda.io/docs/components/concepts/resource-deletion/): runtime lifecycle, internal transition, draining, and separate historical-data removal.
- [Definition lookup](https://docs.camunda.io/docs/apis-tools/orchestration-cluster-api-rest/specifications/get-process-definition/) and [definition search](https://docs.camunda.io/docs/apis-tools/orchestration-cluster-api-rest/specifications/search-process-definitions/): existing read operations.
- Repository paths and symbols cited above, inspected without executing tests or accessing a live environment.

All design questions are resolved. No user clarification, new dependency, new endpoint, or constitution exception is required.
