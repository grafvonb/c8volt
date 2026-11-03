# Data Model: Standardize Flag Intent

This feature adds no persistent entity, backend field, or public-library schema. The following transient command concepts describe parsing and validation; existing facade/domain request types remain authoritative.

## Flag definition and intent

| Field | Meaning / rule |
| --- | --- |
| Command identity | Canonical leaf path; Cobra aliases resolve to the same command scope. |
| Canonical name | Existing or renamed spelling from spec FR-001. |
| Intent | Selection, submitted value, action, enrichment, suppression, scope, execution, or artifact control. |
| Value kind / arity | Existing scalar/object/file/duration/Boolean contract; BPMN action is strictly valueless. |
| Supplied | Presence tracked separately from decoded value. Explicit zero is never omission. |
| Supported modes | Existing restrictions remain unchanged. |
| Removed names | Diagnostic replacement metadata only; not accepted aliases or flag registrations. |

Each supplied input references one flag definition. Removed names reference the canonical replacement for guidance, without creating a valid request. No arbitrary argument values are stored in migration diagnostic messages.

## Normalized mutation intent

| Operation | Normalized fields / unchanged destination |
| --- | --- |
| Variable update/completion | Inline or file source and decoded object map to existing variable/completion requests. Preserve operation-specific numeric decoding, optional payloads, and non-null-object validation. |
| Process initialization | Optional exact BID and variables map to existing creation fields. |
| Task assignment | Assignee value and existing assign/unassign/complete mode state map to current assignment planning inputs. |
| Job retry update/failure | Optional supplied retry count comes only from `set-retries`; existing nonnegative/range validation and pointer semantics remain. |
| Job timeout update | Optional supplied positive duration comes only from `set-timeout`, with existing duration/unit validation. |
| Worker error | Active valueless BPMN action, required code, optional message and supported variables map to unchanged worker outcome fields. Technical failure keeps its existing retry/backoff requirements. |
| Repair | Retry restoration count, optional related-job timeout and optional variables map to existing repair input. Zero retry count means skip restoration. |

The normalized intent is validated before confirmation or submission. BID assignment remains separate from selection and from a process variable named `businessId`; mode exclusions and exact opaque values remain unchanged.

## Selection and execution context

Job selection retains its existing predicate structure, including an optional exact retry count. Only keyless selector completion and job retrieval accept that predicate; a keyed completion does not. Selection retries and submitted retries are different fields and cannot fall back to one another.

HTTP execution timeout is an existing effective configuration value (`http.timeout`), resolved from inherited `--timeout`, environment, profile/file and default under current precedence. It is independent of optional job mutation timeout. Neither supplies the other's omission default. There is no new mutation configuration property.

## Diagnostic context

| Field | Source / invariant |
| --- | --- |
| Selected command | Preserve leaf identity even when parse/Args fails before pre-run. |
| Reporting controls | Known configuration, profile, output and logging controls recovered according to registered token arity into temporary state. |
| Effective logger | Existing command-context logger, configured destination/format/level/activity integration. |
| Output mode | Existing mode precedence and command support; diagnostics never invent a new output schema. |
| Classified error | Invalid input for removed syntax, conflict, missing code, or legacy retry mutation; retain existing no-err-codes override behavior. |
| Rendered status | Existing exit-after-render contract prevents duplicate rendering. |

Diagnostic preparation is local only. It must not build mutation intent from recovered arguments, consume stdin, load variable files, authenticate, or contact backend services. Invalid local configuration uses established bootstrap fallback behavior.

## Transitions

1. Invocation → selected command plus original token sequence.
2. Rejected parsing/argument input → local diagnostic preparation → one classified error result/diagnostic → exit without confirmation or mutation.
3. Parsed canonical input → local configured context → semantic migration/mode validation.
4. Invalid semantics → same single-render error boundary; no remote bootstrap for migration rejection.
5. Valid semantics → existing remote bootstrap and command plan → existing confirmation/dry-run/execution/result lifecycle.

No change is made to service state transitions, outcome evidence, verification/replay behavior, discovery ordering, or partial-failure aggregation.
