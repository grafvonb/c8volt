# Data Model: Suspend Process Instances

**Feature**: [spec.md](spec.md) | **Design decisions**: [research.md](research.md)

These are additive domain/public models. Existing `ProcessInstance`, `Reporter`, cancellation/deletion reports, and shared JSON envelope schemas remain unchanged. Go declarations and converters will be implemented with their required comments in the corresponding work unit.

## Selected scope

An ordered collection of explicit instance keys. CLI input uses repeated/comma-separated `--key`/`-k`, followed by newline-separated stdin keys when `-` is supplied. Trim and validate using the established 16-digit key convention, then deduplicate by first occurrence. Distinct parent and child keys never collapse. The internal service also validates public-library input, independently of CLI validation.

The preview returns one result per unique key and performs no remote reads. The CLI copies these keys before confirmation; execution uses that exact scope. There is no hierarchy, selection cursor, root list, or eligibility assertion in a suspension preview.

## Per-key result

Internal type: `domain.ProcessInstanceSuspensionResult`. Public type: `process.SuspensionResult`. Public string-enum types and constants mirror the domain meaning without exposing internal types.

| Go field / JSON field | Type | Meaning and invariant |
| --- | --- | --- |
| `Key` / `key` | string | Requested instance key, populated even for skipped or failed work. |
| `Status` / `status` | string enum | `preview`, `accepted`, `confirmed`, `failed`, `timed_out`, `interrupted`, or `not_attempted`; workflow disposition, not engine state. |
| `Submission` / `submission` | string enum | `not_submitted`, `accepted`, `rejected`, or `unknown`; retained independently of confirmation. |
| `Confirmation` / `confirmation` | string enum | `not_requested`, `unconfirmed`, or `confirmed`. Preview, rejected/unknown submission, skipped work, and no-wait use `not_requested`; accepted default waiting starts as `unconfirmed`. |
| `StatusCode` / `statusCode,omitempty` | int | Last valid submission HTTP status, including 204 after acceptance; never overwritten with a confirmation-read status. Zero means no valid response status. |
| `HasObservation` / `hasObservation` | bool | True only when a matching instance was read or a typed not-found observation established ABSENT. False differs from an actual UNKNOWN observation. |
| `LastObservedState` / `lastObservedState,omitempty` | state/string | Last observed state, empty when `hasObservation=false`. SUSPENDED proves confirmation only after accepted submission. |
| `SuspendedDate` / `suspendedDate,omitempty` | string | Existing mapped date from the latest successful matching-instance observation; never synthesized. Omit after an ABSENT observation; preserve along with the prior state on a later read failure. |
| `Error` / `error,omitempty` | string | Per-key machine error evidence. Never use as a human logger message or verbose detail. |

`Submission` and `Confirmation` are authoritative fields; do not infer them from HTTP status, timestamps, error text, or `Status` alone. Do not add a redundant `ok` field. A method may derive success for internal counting: preview is a valid preview; confirmed is successful waiting; accepted with confirmation not requested is successful no-wait. Accepted-but-unconfirmed is a failure of the requested default workflow.

## Collection result

Internal `ProcessInstanceSuspensionResults` and public `SuspensionResults` contain `Items` with JSON field `items`. Initialize the collection as an empty array, not null. Return one item per normalized selected key in first-input order. No asynchronous batch identifier is created.

No persisted report store or database is added. Results exist for the call duration and are returned even when an aggregate error is nonnil. Use existing joined error causes and facade classification; never replace them with only rendered summary text.

A direct v810 library call with an empty collection returns `items: []`, no error, and no I/O after capability validation. CLI missing/empty stdin remains an input error. Unsupported v88/v89 empty and preview calls still return unsupported capability. No-wait by itself does not make an empty result an accepted submission.

## Submission evidence transitions

| Event | Submission evidence | Result behavior |
| --- | --- | --- |
| Preview | not_submitted | preview / not_requested; no I/O. |
| Work not scheduled or context canceled before sending | not_submitted | not_attempted or interrupted; retain the key and cause. |
| Valid 204 without call error | accepted | accepted / not_requested for no-wait, otherwise proceed to observation with unconfirmed confirmation. |
| Explicit rejection, with no earlier ambiguous attempt | rejected | failed (or timed_out/interrupted if the retry budget/context ended); no observation reconciliation. |
| Network failure, malformed response, unexpected success code, or call error with a partial response | unknown | failed/timed_out/interrupted; never claim accepted. |
| Unknown attempt followed by rejection or an unsent local pacing failure | unknown | Preserve uncertainty and latest response/error evidence. |
| Unknown attempt followed by a valid 204 | accepted | Acceptance is now proven; confirmation still requires observation unless no-wait. |
| Locally unsent retry attempt | unchanged from earlier attempts | Retain not_submitted, rejected, or unknown evidence as appropriate; do not manufacture uncertainty. |

Recognize explicit HTTP rejection using the established mutation evidence classification: relevant 4xx responses and 503 RESOURCE_EXHAUSTED are rejection evidence; generic server/proxy failure is not proof that nothing happened. Retry policy is unchanged. The accumulator lives outside the retry callback so canceled backoff and request pacing cannot discard earlier facts.

## Confirmation state machine

Observation begins only after a valid accepted submission, unless no-wait ends the call immediately. The waiter receives no mutation callback.

| Observation/event | Result |
| --- | --- |
| Matching instance with SUSPENDED | status confirmed, submission accepted, confirmation confirmed. Preserve existing date if supplied; a date is not required to confirm state. |
| Matching instance with COMPLETED, CANCELED, or TERMINATED | Stop: status failed, submission accepted, confirmation unconfirmed, last state retained. Return an existing conflict-class cause for the incompatible observed terminal outcome. |
| ACTIVE, UNKNOWN, an unrecognized nonterminal state, or timestamp without SUSPENDED | Continue within the bound; do not confirm. Normalize unrecognized state evidence to UNKNOWN for this result, preserving existing shared mapping behavior. |
| Typed not-found | Record ABSENT, omit date, continue within the bound; never call `State.IsTerminal()` for this decision. |
| Permission denial or other non-not-found read failure | Stop failed/unconfirmed with accepted submission and previous observation intact. Do not classify absence from error substrings. |
| Timeout / polling attempt limit | Stop timed_out/unconfirmed with accepted submission and last observation intact; preserve timeout classification. |
| Caller cancellation | Stop interrupted/unconfirmed with accepted submission and last observation intact. |
| Mismatched or malformed instance identity | Stop failed/unconfirmed with malformed-response evidence; do not store it as a valid observation for the selected key. |

Apply the positive configured backoff timeout and the earlier caller deadline to the entire confirmation phase, including in-flight reads and transport read retries. A positive max-retries setting may end it sooner. This is a per-target confirmation bound; a queued collection is also subject to caller cancellation/deadline and is not promised to finish within one target's bound.

## Reporting invariants

- Accepted count includes confirmed and accepted-but-unconfirmed targets. Failed count includes failed/timed-out/interrupted started workflows. Not-attempted items count as skipped; an interrupted unsent item is failed, not accepted. The categories are explained rather than incorrectly summed as disjoint acceptance buckets.
- Keys-only execution includes exactly items with submission accepted; preview includes all selected preview keys. This intentionally reports acceptance, not confirmation.
- Human reasons come from typed workflow/submission/state facts and existing error classes. Raw `Error`, joined causes, and backend strings remain outside normal/verbose human output.
- Fail-fast stops scheduling but cannot undo accepted work or erase in-flight outcomes. Fill unscheduled slots explicitly and retain a nonnil aggregate context/error result.
- Slice conversion copies the collection; no generated-client types or internal enum types appear in public signatures.
