# Feature Specification: Process-Definition Lifecycle State

**Feature Branch**: `348-definition-lifecycle-state`

**Created**: 2026-10-27

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/348 — expose process-definition lifecycle state including DRAINING; lifecycle display is supported on Camunda 8.10, while 8.8/8.9 retain existing output when state is unavailable."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the current definition lifecycle (Priority: P1)

As an operator or script author, I want ordinary process-definition reads to expose the lifecycle state supplied by Camunda so I can distinguish a definition available for new instances from one awaiting removal or already removed from runtime.

**Why this priority**: The backend already provides this information, but users currently need a separate request outside c8volt to see it. Observation through the existing command is the core value of this feature.

**Independent Test**: Supply controlled definition lookup and listing responses with known, unfamiliar, and unavailable lifecycle states. Verify human output and structured results through command execution and public-library reads, including older supported versions, without contacting a real environment.

**Acceptance Scenarios**:

1. **Given** a Camunda 8.10 lookup or listing supplies `ACTIVE`, `DRAINING`, or `DELETED`, **When** the user runs `get pd --key <key>` or `get pd`, **Then** each returned definition shows its supplied lifecycle state in the ordinary compact human row, and `--json` includes that value as `state`.
2. **Given** a definition is returned without lifecycle state, **When** human or JSON output is requested, **Then** state is omitted without a default, null placeholder, or value retained from an earlier response.
3. **Given** a supplied lifecycle value is unfamiliar, **When** the definition is read, **Then** the value is preserved rather than replaced by a known state or rejected merely for being unfamiliar.
4. **Given** Camunda 8.8 or 8.9 does not supply lifecycle state, **When** the same existing reads run, **Then** they remain supported and retain their existing behavior with state omitted.
5. **Given** a returned definition is marked `DELETED`, **When** it is displayed, **Then** its row remains visible; an absent definition instead follows the existing lookup-not-found or listing-empty behavior. A `DELETED` value does not claim that historical data is gone.
6. **Given** a keys-only, quiet, or empty-result invocation, **When** the result is rendered, **Then** existing mode precedence and payload conventions remain intact: keys-only contains only keys and zero bytes for no keys; successful empty JSON contains exactly one successful envelope with its existing payload; quiet preserves explicitly requested JSON.
7. **Given** otherwise identical reads with lifecycle state present or absent, **When** the command finishes, **Then** displaying lifecycle state has added no backend requests, prompts, or mutations, and diagnostics remain separate from selected-format stdout results.

---

### User Story 2 - Understand lifecycle alongside instance statistics (Priority: P1)

As an operator investigating a deletion, I want definition lifecycle and instance counts shown separately so I do not mistake zero active instances for completed draining when suspended work remains.

**Why this priority**: A definition can be waiting on unfinished suspended instances. Combining the two observations without conflating their meanings prevents misleading operational conclusions.

**Independent Test**: Supply definitions with and without lifecycle state and with independently controlled statistics. Exercise lookup and listing output with `--stat` in human and JSON modes, including a `DRAINING` definition with zero active and nonzero suspended instances. Compare request counts with the existing statistics behavior.

**Acceptance Scenarios**:

1. **Given** lifecycle state and statistics are available, **When** lookup or listing uses `--stat`, **Then** the supplied lifecycle state appears alongside separate statistics, with existing count values, labels, availability indicators, and row alignment preserved. Supported `--json --stat` output contains both observations.
2. **Given** the backend supplies `DRAINING`, zero active instances, and two suspended instances, **When** the user reads that definition with `--stat`, **Then** the result retains `DRAINING` and the separate counts without claiming that draining has completed.
3. **Given** counts are zero or statistics are missing or unsupported, **When** a definition is displayed, **Then** those facts neither generate nor overwrite lifecycle state. Existing statistics failures retain their established behavior.
4. **Given** rows differ in lifecycle availability or contain unfamiliar values, **When** a listing is displayed with or without statistics, **Then** the human rows remain compact and aligned, and omission of state does not misidentify a statistic as lifecycle state.
5. **Given** a user consults command help or documentation, **When** they read the lifecycle explanation, **Then** it explains that backend definition deletion blocks new instances while existing unfinished instances drain; it does not automatically cancel or resume those instances. c8volt's existing `delete pd --force` workflow explicitly cancels unfinished instances first and is unchanged.
6. **Given** a definition is draining while an instance is suspended, **When** that instance is resumed, **Then** the display continues to use the backend's supplied definition state; resumption alone is not treated as completion or as a reversal of the definition-deletion request.

---

### User Story 3 - Follow lifecycle changes during watch (Priority: P2)

As an operator watching definitions, I want lifecycle-only changes highlighted on successful refreshes so I can follow draining and runtime removal without confusing unchanged statistics, missing state, failed reads, and disappearance.

**Why this priority**: Watch already reports changing definition observations. Lifecycle must participate in that behavior without changing how watch sessions are controlled.

**Independent Test**: Use deterministic successful and failed refresh sequences through isolated watch execution. Cover state-only and statistics-only changes, omitted state, unchanged refreshes, returned `DELETED` rows, disappearance, and sequences that skip intermediate states, without relying on live timing.

**Acceptance Scenarios**:

1. **Given** the first successful refresh supplies lifecycle state, **When** watch renders it, **Then** the row establishes an unmarked baseline.
2. **Given** a later successful refresh changes only lifecycle state, **When** watch renders it, **Then** the row is counted as updated and uses the existing change marker and supported value highlighting, even if statistics are unchanged.
3. **Given** `--stat --watch`, **When** only statistics, only lifecycle, or both change, **Then** the changed values are identified independently and a changed row contributes once to the update count.
4. **Given** an earlier successful response included state and the next successful response omits it, **When** watch refreshes, **Then** the earlier value is cleared and the omission is recognized as an update. A later supplied value is likewise recognized as a change.
5. **Given** a refresh fails, **When** a later successful refresh is compared, **Then** comparison uses the last successful baseline. An unchanged successful refresh clears prior change markers and highlighting.
6. **Given** a successful response returns a `DELETED` row, **When** it is rendered, **Then** the row remains visible. If the definition subsequently disappears, existing removal counting and stale-row removal apply.
7. **Given** observations move directly from `ACTIVE` to `DELETED`, **When** watch refreshes, **Then** the supplied state is displayed without requiring an observed `DRAINING` or inventing an intermediate transition.
8. **Given** an existing watch session or an incompatible invocation, **When** lifecycle display is enabled by the available data, **Then** timing, retries, stop behavior, and configured diagnostic and watch-summary logging remain unchanged; `--watch` still rejects `--json`, `--keys-only`, `--xml`, `--quiet`, and `--automation`.

### Edge Cases

- One listing contains known states, an unfamiliar supplied value, and a definition with state unavailable.
- A later successful refresh omits state after a failed refresh; the displayed state must clear relative to the last successful baseline.
- Lifecycle changes while counts remain identical, or counts change while lifecycle remains identical.
- A `DRAINING` definition has no active instances but still has suspended instances; zero active count cannot prove completion.
- `DELETED` is returned while historical data remains; returned state and absence must stay distinct.
- A watch observation skips a brief lifecycle transition; no intermediate state may be synthesized.
- Statistics are absent, unsupported, or fail to load; existing availability and failure behavior must not be repurposed as lifecycle evidence.
- Empty listings, explicit-key not-found results, quiet-plus-JSON output, and empty keys-only streams retain their existing contracts.
- Existing non-watch XML output remains unchanged; lifecycle text is not inserted into process XML.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Existing definition lookup and listing operations MUST expose lifecycle state supplied by Camunda 8.10, including through public-library results and ordinary human or supported JSON command output. No new command or opt-in display flag is required.
- **FR-002**: Lifecycle state MUST remain a separate definition attribute, independent of process-instance state and statistics. Supplied `ACTIVE`, `DRAINING`, `DELETED`, and unfamiliar values MUST be preserved without coercion or inferred defaults.
- **FR-003**: When lifecycle state is unavailable, human output MUST omit the state value and JSON MUST omit `state`, without a null placeholder or a stale value. Camunda 8.8/8.9 reads MUST remain compatible with state omitted when unavailable.
- **FR-004**: Human key lookups and listing rows MUST show available lifecycle state compactly and consistently, with stable alignment when state availability varies and when statistics are included.
- **FR-005**: Supported JSON lookup and listing results MUST include available `state` alongside their existing data, including statistics when requested, while preserving existing envelope, empty-result, and omission conventions.
- **FR-006**: Existing keys-only, quiet, XML, error, and output-precedence behavior MUST remain intact. Empty keys-only output MUST be zero bytes; successful empty JSON MUST contain exactly one successful envelope with the established empty payload. Quiet MUST preserve explicitly requested JSON.
- **FR-007**: Lifecycle observation MUST use data already supplied by the selected lookup or listing and MUST add zero requests solely for lifecycle state. It MUST introduce no mutation, confirmation prompt, or additional discovery and MUST preserve existing statistics request behavior.
- **FR-008**: Lifecycle state MUST be visible both with and without `--stat`. Existing active, suspended, completed, canceled, and incident statistics, their labels, and availability indicators MUST remain separate and retain current semantics and formatting.
- **FR-009**: Lifecycle MUST NOT be inferred from a deletion submission, instance counts, unavailable statistics, or resumption. A supplied `DRAINING` state MUST remain visible when active count is zero and suspended count is nonzero.
- **FR-010**: Every successful watch refresh MUST display the latest supplied lifecycle observation. Lifecycle-only changes MUST participate in existing row update counts, change markers, and supported value highlighting.
- **FR-011**: Watch MUST compare lifecycle and statistics independently. A row changing in both dimensions MUST be counted once as an updated row; unchanged values MUST NOT be highlighted as changed merely because the other dimension changed.
- **FR-012**: The first successful watch refresh MUST establish an unmarked baseline. Failed refreshes MUST preserve the prior successful baseline; unchanged successful refreshes MUST clear previous change indicators. A supplied-to-omitted state transition MUST clear the previous value and count as a change.
- **FR-013**: A returned `DELETED` definition MUST remain a returned row. Actual absence MUST retain existing lookup-not-found, listing-empty, and watch-removal behavior. Watch MUST accept skipped intermediate states without synthesizing transitions.
- **FR-014**: Existing watch timing, retries, stopping behavior, and incompatible flag combinations MUST remain unchanged. Watch MUST continue rejecting `--json`, `--keys-only`, `--xml`, `--quiet`, and `--automation`.
- **FR-015**: Selected-format results MUST remain on stdout. Diagnostics and existing watch summaries MUST continue using the configured command logger and its routing, level, format, and destination, without duplicate emission or contamination of results.
- **FR-016**: Command help, source metadata, and generated user documentation MUST explain 8.10 lifecycle support and older-version omission. They MUST define `ACTIVE` as available for new instances, `DRAINING` as blocking new instances while existing unfinished work remains, and `DELETED` as runtime removal that does not prove historical-data removal.
- **FR-017**: Documentation MUST distinguish backend definition deletion, which does not automatically cancel or resume existing instances, from c8volt's existing `delete pd --force` cancellation workflow. This feature MUST preserve suspension, resumption, cancellation, forced deletion, root resolution, and existing deletion-completion checks. It MUST NOT synthesize internal `DELETING` state.
- **FR-018**: Relevant existing integration draining scenarios MUST observe lifecycle through `get pd` wherever CLI coverage is intended. Updating those scenarios MUST NOT become authorization or a requirement to execute live validation during implementation.

### Key Entities *(include if feature involves data)*

- **Process definition**: A deployed process template identified by its definition key, with existing identity/version information and an optional observed lifecycle state.
- **Definition lifecycle observation**: The backend-supplied value describing the definition's runtime availability or removal progress. An unavailable value, a returned `DELETED` value, and an absent definition are different observations.
- **Instance statistics**: Existing counts and availability indicators for instances associated with a definition, including suspended instances. They accompany lifecycle without determining it.
- **Watch baseline**: The last successfully displayed set of definition observations used to identify row additions, updates, unchanged values, and removals on subsequent successful refreshes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In every controlled 8.10 lookup/listing case, operators and scripts receive the supplied known or unfamiliar lifecycle value through their chosen supported output, without issuing a separate lifecycle inspection command.
- **SC-002**: All unavailable-state and 8.8/8.9 compatibility cases omit lifecycle without defaults or stale values; existing key streams and empty-result contracts remain unchanged in every covered case.
- **SC-003**: In every controlled draining case with zero active and nonzero suspended instances, users can see the supplied `DRAINING` state and separate counts together, without a false indication that draining has completed.
- **SC-004**: Every successful state-changing watch observation is reflected on that refresh and each changed row is counted once. Baseline, unchanged, failed-refresh, omission, and disappearance sequences produce the specified markers and retained or cleared values in all covered cases.
- **SC-005**: Adding lifecycle visibility causes zero additional backend requests, prompts, or mutations compared with the same lookup, listing, statistics, or watch operation without that visibility.
- **SC-006**: Help and examples cover all three known lifecycle meanings, 8.10 support, unavailable-state behavior, and the distinction between runtime deletion, historical-data removal, and c8volt's explicit forced-cancellation workflow.

## Assumptions

- Issue [#348](https://github.com/grafvonb/c8volt/issues/348), including its clarified backend-deletion wording, is the source of scope. Related #340, #346, and #347 supply existing statistics and instance-management context; they do not make this an instance-suspension feature.
- Users retain their existing access and configuration. Authorization and server-side visibility rules do not change.
- Existing Camunda 8.10 lookup/listing support already carries the optional lifecycle information needed for this feature. Regeneration of the generated client is not a prerequisite; the implementation plan must reuse existing support and preserve the value through the established application layers.
- Camunda 8.8/8.9 do not provide lifecycle state through their current supported definition reads. Their ordinary reads remain usable without the new field.
- Backend observations can be eventually consistent and may skip intermediate transitions. Lifecycle and statistics are separate observations, not a promise of an atomic combined snapshot.
- Supplied state is authoritative for display only. The feature does not introduce lifecycle filtering or sorting, new commands, native batch operations, new waiting modes, graceful-drain execution, or changes to deletion-completion policy.
- Implementation validation uses isolated unit tests, generated-client mocks, test-owned HTTP fakes, and isolated CLI/terminal execution. Deterministic fixtures cover mapping, omission, unfamiliar values, human/JSON output, statistics, watch sequences, and unchanged request counts without claiming real-engine validation. No real Camunda environment is discovered, provisioned, or contacted; live validation requires a separate explicit request for a specific target.
