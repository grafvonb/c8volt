# Feature Specification: Process-Definition Lifecycle State

**Branch**: `348-definition-lifecycle-state` · **Issue**: [#348](https://github.com/grafvonb/c8volt/issues/348)
**Created**: 2026-10-27 · **Scope clarified**: 2026-10-28 · **Status**: Implemented

Expose the lifecycle value already returned by Camunda 8.10 in existing human and JSON definition reads. Reuse the shared renderer and watch comparison. This is an observation feature, not a new draining/deletion workflow. c8volt's ordinary delete path blocks unfinished work or cancels it with `--force`; DRAINING is principally useful for deletion initiated through the state-check bypass or another client.

The simplification retains behavior and logger fixes while reducing repeated tests and documentation. Requirements below describe outcomes, not a requirement to repeat every case at every layer.

## User stories and acceptance

| Story | Priority | Acceptance |
| --- | --- | --- |
| US1 — See the current definition lifecycle | P1 / MVP | Lookup/list and public results preserve supplied state, omit unavailable state, and retain existing output modes and request counts. |
| US2 — Read lifecycle alongside statistics | P1 | A DRAINING definition with active=0 and suspended=2 displays state and counts independently, with aligned mixed rows. |
| US3 — Follow changes in existing watch | P2 | State participates in the existing change markers/highlighting; failed refreshes/writes preserve the successful baseline and diagnostics honor logger configuration. |

## Functional requirements

- **FR-001**: Existing lookup/list operations expose Camunda 8.10 state through public results and human/JSON output, without a new command or flag.
- **FR-002**: Preserve ACTIVE, DRAINING, DELETED and unfamiliar values verbatim, independently of instance state/counts.
- **FR-003**: Omit unavailable state in human/JSON output; do not default, emit null, or retain stale values. Existing 8.8/8.9 reads remain compatible.
- **FR-004**: Keep human lookup/list rows compact and aligned, including mixed state availability and statistics.
- **FR-005**: Add optional `state` to existing JSON objects without changing envelopes, empty payloads or statistics serialization.
- **FR-006**: Preserve keys-only, quiet, XML, error and mode precedence. Empty keys output is zero bytes; empty JSON is one envelope; quiet retains requested JSON.
- **FR-007**: Add no backend requests, prompts or mutations; preserve existing statistics/pagination requests.
- **FR-008**: Show state with and without `--stat`, preserving count labels, values and support indicators.
- **FR-009**: Never infer lifecycle from deletion submission, counts, unavailable statistics or resumption. DRAINING can coexist with zero active and nonzero suspended instances.
- **FR-010**: Successful watch refreshes display current state; state-only changes update the row and its existing marker/highlight.
- **FR-011**: Compare state and statistics independently; simultaneous changes count the row once.
- **FR-012**: First watch success is unmarked. Failed fetch/render retains the baseline, unchanged success clears markers, and omission clears the old value and counts as a change.
- **FR-013**: Keep returned DELETED rows. Actual absence retains existing not-found/empty/removal handling. Do not synthesize intermediate transitions.
- **FR-014**: Preserve watch timing, retries, stop behavior and rejection of JSON/keys-only/XML/quiet/automation combinations.
- **FR-015**: Keep selected results on stdout and diagnostics/watch summaries on the configured logger, respecting routing, format and level without duplication. Preserve bounded human errors and machine error evidence.
- **FR-016**: Briefly document 8.10 availability, omission, and meanings: ACTIVE permits starts; DRAINING blocks starts while unfinished work remains; DELETED is runtime removal, not proof of history removal.
- **FR-017**: Distinguish backend deletion from c8volt's existing forced cancellation. Preserve suspension, resumption, cancellation, forced deletion, root resolution and deletion-completion checks. Do not synthesize DELETING.
- **FR-018**: Keep the existing draining integration scenario's CLI observation; it does not authorize live execution during implementation.

## Success criteria and evidence

| Criterion | Evidence |
| --- | --- |
| SC-001 — Supplied values reach supported reads/output | Adapter value table, facade propagation, focused lookup/list command tests. |
| SC-002 — Missing/older state stays omitted; modes remain compatible | Public JSON model test, older adapters, command mode/mixed-row regressions. |
| SC-003 — Draining and suspended counts stay independent | Four lookup/list × human/JSON command cases, mixed rows and existing adapter count cases. |
| SC-004 — Watch updates and baselines stay correct | Existing deterministic transition, write-failure, configured-logger and real-PTY tests with test-owned services. |
| SC-005 — No added requests/prompts/mutations | Existing per-fixture request assertions and unexpected-request rejection. |
| SC-006 — Operator explanation matches behavior | Source help, README and regenerated CLI documentation. |

## Boundaries

No lifecycle filtering/sorting, new execution mode, state machine, generated-client changes or deletion policy. Lifecycle/statistics are separate observations, not an atomic snapshot. Preserve unknown values and distinguish omitted state, returned DELETED and absent definitions.

Validation uses isolated mocks, test-owned HTTP servers and CLI/terminal fixtures only. Do not contact or provision real Camunda services, run live integration tests, or add explicit no-wait scenarios. Real-engine timing remains unverified. Model/interface details are in [data-model.md](data-model.md) and [contracts/](contracts/).
