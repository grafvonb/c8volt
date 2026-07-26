# Feature Specification: Slow Process Instance Analysis

**Feature Branch**: `244-slow-process-analysis`

**Created**: 2026-07-18

**Status**: Draft

**GitHub Issue**: [#244](https://github.com/grafvonb/c8volt/issues/244) - feat(ops): add `ops analyse slow-process-instances` to localize slow runs for one process definition

**Input**: GitHub issue #244 requests a read-only ops command that identifies slow process instances and shows where their elapsed time was spent, while presenting factual timings without selecting a bottleneck or claiming causality.

## Clarifications

### Session 2026-07-18

- Q: How should process instances with unavailable whole duration sort? -> A: After all available durations
- Q: How should process-definition search behave when no process instances match? -> A: Successful empty analysis

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Analyze Known Process Instances By Key (Priority: P1)

As a c8volt operator, I want to analyze one or more known process instance keys so I can inspect elapsed time for exact runs without first searching by process definition.

**Why this priority**: Direct keyed analysis is the smallest useful workflow and supports pipelines from existing process-instance discovery commands.

**Independent Test**: Can be fully tested by analyzing one known process instance key, repeated keys, stdin keys, and a mixture of flag and stdin keys, then verifying deduplication, validation, sorting, and read-only output.

**Acceptance Scenarios**:

1. **Given** a supported Camunda 8.8 or 8.9 environment with a known process instance, **When** the operator runs `c8volt ops analyse slow-process-instances --key <process-instance-key>`, **Then** the command renders that process instance as the root of an analysis section with its available identifiers, state, timestamps, incident marker, and `dur:`.
2. **Given** several process instance keys supplied through repeated `--key` values and newline-separated stdin using `-`, **When** the operator runs the analysis command, **Then** duplicate keys are analyzed once and results are sorted by whole duration from longest to shortest.
3. **Given** `c8volt get pi --state active --keys-only` emits process instance keys, **When** the output is piped into `c8volt ops analyse slow-process-instances -`, **Then** the analysis accepts the pipeline and emits a read-only timing analysis for the selected keys.

---

### User Story 2 - Discover Slow Runs For One Process Definition (Priority: P2)

As a c8volt operator, I want to select process instances for one process definition so I can localize slow runs within a meaningful operational cohort.

**Why this priority**: Operators usually compare executions within the same process definition before deciding which run needs deeper investigation.

**Independent Test**: Can be fully tested by selecting a process definition with exactly one supported selector, applying process-instance search filters, and verifying that selected instances are frozen before detail analysis.

**Acceptance Scenarios**:

1. **Given** a process definition with active and terminal process instances, **When** the operator runs the analysis command with exactly one of `--bpmn-process-id` or `--pd-key`, **Then** the command analyzes the selected process instances for that process definition.
2. **Given** date, state, incident, batch-size, and limit filters, **When** the operator runs process-definition search mode, **Then** those filters affect only process-instance discovery and do not truncate explicit-key analysis or timeline details.
3. **Given** the selected process instances have been discovered, **When** element inspection begins, **Then** the command analyzes the frozen selection with one captured analysis time for all active durations.

---

### User Story 3 - Inspect Timelines, Transitions, And Slow Details (Priority: P3)

As a c8volt operator, I want each analyzed process instance to include chronological element rows and adjacent transition timings so I can see where elapsed time appears in the run without inferring causality.

**Why this priority**: Root process durations identify slow runs, but operators need element and transition timing to localize where time was spent.

**Independent Test**: Can be fully tested by analyzing a process instance with completed, active, terminated, overlapping, missing-timestamp, and incident-bearing elements, then applying detail filters and verifying the visible timeline.

**Acceptance Scenarios**:

1. **Given** a process instance with runtime elements, **When** the command renders the analysis, **Then** element rows appear in the same chronological order as `c8volt get pi --with-elements` and include duration values where timestamps support them.
2. **Given** adjacent chronological elements with valid timestamps where the next start is at or after the previous end, **When** the command renders transition timing, **Then** it emits a compact `A -> B: duration` line without claiming BPMN causality.
3. **Given** detail filters such as `--element-id`, `--type`, `--element-state`, or `--dur-element-longer`, **When** the command renders details, **Then** calculations still use the complete unfiltered timeline and filtering never creates synthetic transitions across hidden elements.

---

### User Story 4 - Consume Stable Human, JSON, And Keys-Only Results (Priority: P4)

As a c8volt operator or automation author, I want slow-run analysis in established output modes so I can inspect results interactively or compose them in scripts.

**Why this priority**: The command is most useful when it works both for human investigation and for downstream automation.

**Independent Test**: Can be fully tested by running equivalent selections in human, JSON, and keys-only modes and verifying stable ordering, stable fields, relative-duration indicators, and final counts.

**Acceptance Scenarios**:

1. **Given** multiple selected process instances, **When** default human output is requested, **Then** process instances are sorted longest to shortest, details are grouped under each root, relative-duration indicators appear when enough comparable measurements exist, and a final process-instance count is shown.
2. **Given** JSON output is requested, **When** analysis completes, **Then** each process instance contains stable process, element, transition, duration, comparison, and analysis-time fields suitable for machine consumption.
3. **Given** keys-only output is requested, **When** analysis completes, **Then** only unique process instance keys are emitted one per line in longest-to-shortest order and detail filters do not change which root keys are emitted.

### Edge Cases

- Camunda 8.7 must return the established unsupported-version error for this command.
- The British and American spellings, `ops analyse slow-process-instances` and `ops analyze slow-process-instances`, must behave the same.
- Exactly one target selection mode must be used: explicit keys or process-definition search.
- Explicit-key mode must accept repeated `--key` or `-k` values, one stdin marker `-`, or both.
- Explicit-key mode must reject empty stdin and invalid, missing, or unauthorized keys rather than silently discarding them.
- Explicit-key mode must not require a process-definition selector and may include keys from different process definitions.
- Explicit-key mode must not be combined with process-definition selectors or process-instance search filters.
- Process-definition search mode must require exactly one of `--bpmn-process-id` or `--pd-key`.
- Process-definition search mode with no matching process instances must succeed with an empty result instead of failing.
- Date filters must accept precise timestamps and `YYYY-MM-DD`; relative-day filters are out of scope.
- `--no-incidents-only` is supported; `--incidents-only` is out of scope for this command.
- Process instances with valid end dates use elapsed time from start to end; active instances without end dates use captured analysis time; terminal instances without usable end dates have unavailable duration and sort after all process instances with available durations.
- Completed, canceled, and terminated process instances are included when selected.
- Elements with valid end dates use elapsed time from start to end; active elements without end dates use captured analysis time; otherwise element duration is unavailable.
- Terminated elements with valid timestamps must be measured.
- Overlapping elements must not produce negative transition durations.
- Missing, invalid, or filtered-out elements must not be bridged by synthetic transition timings.
- Detail filters must keep the process-instance root even when no detail rows match.
- `--dur-longer` applies only to process-instance root visibility and excludes roots without an available measured duration.
- `--dur-element-longer` applies only to element and transition durations, never to process-instance root visibility.
- `--duration-after` remains a backward-compatible alias for `--dur-element-longer` and cannot be combined with it.
- Human duration bars must be omitted for zero-duration rows and for single-root root comparisons.
- JSON relative indicators must be omitted when fewer than three comparable measurements exist.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a read-only slow process instance analysis command under both `c8volt ops analyse slow-process-instances` and `c8volt ops analyze slow-process-instances`.
- **FR-002**: The command MUST support Camunda 8.8 and Camunda 8.9.
- **FR-003**: The command MUST return the established unsupported-version error for Camunda 8.7.
- **FR-004**: The command MUST present factual timings only and MUST NOT select a bottleneck, name a cause, or imply that a timing proves BPMN or system causality.
- **FR-005**: The command MUST require exactly one selection mode: explicit process-instance keys or process-definition search.
- **FR-006**: Explicit-key mode MUST accept repeated `--key` and `-k` values.
- **FR-007**: Explicit-key mode MUST accept newline-separated process-instance keys from stdin using one positional `-` argument.
- **FR-008**: Explicit-key mode MUST allow flag keys and stdin keys in the same request.
- **FR-009**: Explicit-key mode MUST deduplicate process-instance keys before lookup.
- **FR-010**: Explicit-key mode MUST validate every supplied key and reject empty stdin.
- **FR-011**: Explicit-key mode MUST accept no positional argument other than one `-`.
- **FR-012**: Explicit-key mode MUST NOT silently discard invalid, missing, or unauthorized keys.
- **FR-013**: Explicit-key mode MUST NOT require a process-definition selector and MUST allow keys from different process definitions.
- **FR-014**: Explicit-key mode MUST preserve the established tenant contract for explicit process-instance keys.
- **FR-015**: Process-definition search mode MUST require exactly one selector: `--bpmn-process-id` or `--pd-key`.
- **FR-016**: Process-definition search mode MUST support process-instance filters for state, start date, end date, no-incidents-only, batch size, and limit.
- **FR-017**: The process-instance state filter MUST accept `active`, `completed`, `canceled`, `terminated`, and `all`.
- **FR-018**: Date filters MUST accept precise timestamps and `YYYY-MM-DD`.
- **FR-019**: The command MUST NOT add relative-day filters or `--incidents-only`.
- **FR-020**: Explicit-key mode MUST reject process-definition selectors and process-instance search filters.
- **FR-021**: `--batch-size` and `--limit` MUST apply only to search discovery and MUST NOT truncate explicit keys, element rows, or transition timings.
- **FR-022**: Process-definition search mode with no matching process instances MUST complete successfully with an empty analysis result.
- **FR-023**: Empty human output MUST show the final process-instance count, empty JSON output MUST return an empty process-instance list, and empty keys-only output MUST print nothing.
- **FR-024**: The command MUST freeze the selected process instances before element inspection.
- **FR-025**: The command MUST capture one analysis time for the full analysis and use it consistently for active process and element durations.
- **FR-026**: Each process instance MUST be the root of its output section.
- **FR-027**: Process-instance whole duration MUST be calculated from start to valid end date, or from start to captured analysis time when active without an end date, or marked unavailable when terminal without a usable end date.
- **FR-028**: Process instances MUST be sorted by available duration longest to shortest, followed by process instances with unavailable duration, with deterministic tie-breaking within each group.
- **FR-029**: Process-instance root rows in human output MUST follow the established `c8volt get pi` style and include available key, tenant, BPMN process ID, version, state, start/end dates, parent information, incident marker, and `dur:`.
- **FR-030**: Runtime element rows MUST appear under each process instance in chronological order by start date, with element-instance key as the deterministic tie-breaker.
- **FR-031**: Each element row MUST include element-instance key, element ID, element type, state, start date, end date when available, `dur:`, and incident marker when present.
- **FR-032**: Element duration MUST be calculated from start to valid end date, or from start to captured analysis time when active without an end date, or marked unavailable when timestamps do not support a duration.
- **FR-033**: Adjacent transition timing MUST be calculated only from adjacent elements in the complete chronological timeline.
- **FR-034**: Transition timing lines MUST be emitted only when both adjacent timestamps are valid and the next start is at or after the previous end.
- **FR-035**: Human transition timing MUST use the compact form `FromElement -> ToElement: duration` without `between:` or `transition:` prefixes.
- **FR-036**: The command MUST NOT emit negative transition durations and MUST NOT bridge over missing, invalid, or filtered-out elements.
- **FR-037**: The command MUST support detail filters `--element-id`, `--type`, `--element-state`, and `--dur-element-longer`.
- **FR-038**: `--state` MUST remain the process-instance state filter; element state filtering MUST use `--element-state`.
- **FR-039**: Duration filter values MUST use Go duration syntax such as `500ms`, `30s`, `5m`, `1h`, `1h30m`, or `24h`; calendar units such as `1d` MUST NOT be accepted.
- **FR-040**: `--dur-element-longer` MUST apply only to element and transition durations and MUST never remove the process-instance root.
- **FR-041**: Detail filters MUST be applied after constructing the complete timeline and calculating durations, transition timings, and relative indicators.
- **FR-042**: Element rows MUST be shown only when they match all element predicates.
- **FR-043**: Original transition timing lines MUST be shown when at least one endpoint matches the active detail filters.
- **FR-044**: Filtering MUST preserve original endpoint identifiers and MUST never create synthetic transitions across hidden elements.
- **FR-045**: Process-instance roots MUST remain visible even when no detail rows match.
- **FR-045a**: The command MUST support a root duration filter `--dur-longer` that includes only process-instance roots with an available whole-process duration greater than the supplied threshold.
- **FR-045b**: `--duration-after` MUST remain a backward-compatible alias for `--dur-element-longer` and MUST be rejected when combined with `--dur-element-longer`.
- **FR-046**: Human output MUST place visual duration-share indicators directly after durations without labels such as `cohort`, `peer`, `similar`, `compared`, `rank`, or `share`.
- **FR-047**: JSON process-instance relative indicators MUST compare selected process instances with the same process-definition key.
- **FR-048**: JSON element relative indicators MUST compare executions with the same process-definition key, element ID, and type.
- **FR-049**: JSON transition relative indicators MUST compare measurements with the same process-definition key and the same from/to element IDs and types.
- **FR-050**: Process-duration share indicators MUST show each element or transition timing divided by the process instance's whole duration when a valid share can be calculated.
- **FR-051**: JSON relative percentile MUST be based on shorter values plus half of equal values divided by all comparable values, rounded to the nearest whole percent.
- **FR-052**: Human duration bars MUST render ten cells based on the duration's rounded percentage of its comparison duration.
- **FR-053**: JSON relative comparison indicators MUST be omitted when fewer than three comparable measurements exist.
- **FR-054**: Human output MUST include process instances sorted longest to shortest, root rows, chronological element rows, compact arrow timing lines, duration-share indicators, and a final process-instance count.
- **FR-054a**: Human output MUST render each process instance as an unindented root row with detail rows nested under a tree-shaped `└─ elements:` section using `├─` and `└─` child connectors.
- **FR-054b**: Human root rows MUST render a ten-cell duration bar only when multiple visible roots have available durations; the root bar percentage MUST be the root duration divided by the longest visible root duration.
- **FR-054c**: Human detail rows MUST render a ten-cell duration bar only when the detail duration is positive and the root duration is available; the detail bar percentage MUST be the detail duration divided by that root process-instance duration.
- **FR-054d**: Human output MUST omit bars for zero-duration rows and unavailable durations, while JSON output MUST keep explicit relative percentile and comparison fields for machine consumers.
- **FR-055**: JSON output MUST provide stable process-instance data and an ordered timeline containing element and transition timing entries.
- **FR-056**: JSON output MUST include process-instance duration and milliseconds, captured analysis time, element fields and timestamps, element duration and milliseconds, transition endpoints and timestamps, transition duration and milliseconds, relative percentile, comparison sample count, and process-duration share.
- **FR-057**: Keys-only output MUST emit only unique process-instance keys, one per line, longest to shortest.
- **FR-058**: Detail filters MUST NOT change which process-instance root keys are emitted in keys-only output.
- **FR-059**: The feature MUST NOT include listener or job execution analysis, BPMN path reconstruction, mutation or repair workflows, report-file generation, or unrelated process-instance and element filters.
- **FR-060**: User-facing command help, examples, generated CLI documentation, and automated tests MUST be updated because this introduces a new command and output contract.

### Key Entities

- **Process Instance**: The root execution selected by explicit key or process-definition search. Key attributes include process instance key, tenant, BPMN process ID, process definition key and version, state, start and end dates, parent information, incident marker, and whole duration.
- **Runtime Element Instance**: A chronological execution occurrence under a process instance. Key attributes include element-instance key, element ID, type, state, start and end dates, duration, and incident marker.
- **Transition Timing**: The elapsed time between adjacent chronological element executions when the previous end and next start are valid and non-overlapping.
- **Selection Cohort**: The frozen set of process instances used for sorting, timeline inspection, and comparison calculations.
- **Duration-Share Indicator**: A compact human measure showing a root duration relative to the longest visible root, or a detail duration relative to its root process-instance duration.
- **Relative-Duration Indicator**: A structured machine-readable measure showing where a process, element, or transition duration sits among comparable measurements.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can analyze a single known process instance by key and see whole-duration, element-duration, and transition-timing information in one command.
- **SC-002**: Repeated `--key`, stdin `-`, and mixed key inputs analyze each unique process instance exactly once.
- **SC-003**: 100% of invalid selection combinations fail before producing misleading analysis output.
- **SC-004**: Process instances, human output, JSON output, and keys-only output preserve the same root ordering for equivalent selections: available durations longest to shortest, then unavailable durations with deterministic tie-breaking.
- **SC-005**: Empty process-definition searches complete successfully with a zero process-instance count in human output, an empty process-instance list in JSON output, and no keys-only output lines.
- **SC-006**: 100% of active process instances and active elements with valid start dates use the same captured analysis time for duration calculation within one run.
- **SC-007**: 100% of transition timing lines use the `A -> B: duration` shape and omit negative, bridged, or causality-claiming timings.
- **SC-008**: Detail filtering never changes relative-duration calculations, never creates synthetic transitions, and keeps the process-instance root visible.
- **SC-009**: Human bars always describe duration share, while JSON relative indicators appear only when at least three comparable measurements exist and tied durations receive the same percentile.
- **SC-010**: JSON consumers can read captured analysis time, process durations, ordered timeline entries, comparison sample counts, relative percentiles, and process-duration shares using explicit field names.
- **SC-011**: Camunda 8.7 attempts return the established unsupported-version result without mutating operational state.
- **SC-012**: The command remains read-only across all successful and failed scenarios.

## Assumptions

- Operators already have configured c8volt access and permissions sufficient to inspect process instances and runtime element instances.
- Existing `get pi` and `get pi --with-elements` presentation rules remain the source of truth for root rows, element ordering, tenant handling, incident markers, and compact timestamps unless this specification explicitly changes them.
- Process-definition search mode is intended to compare one process-definition cohort, while explicit-key mode remains an ad hoc investigation mode that can include keys from different process definitions.
- Human duration-share indicators are informational and not additive because element executions and transition scopes can overlap.
- Documentation and generated command references will be updated in the same feature work because the command, flags, examples, and output contracts are user-facing.

## Out of Scope

- Selecting or naming one bottleneck.
- Claiming a timing caused a slowdown.
- Listener or job execution analysis.
- BPMN path reconstruction.
- Mutation or repair workflows.
- Report-file generation.
- Reproducing every `get pi` or `get element` filter.
