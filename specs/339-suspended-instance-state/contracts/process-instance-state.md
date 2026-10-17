# Contract: Existing Process-Instance Interfaces

## Public library and internal service boundary

Public signatures remain unchanged. Extend the public state vocabulary and ProcessInstance metadata additively. Existing lookup, search, state wait and combined state/incident expectation methods preserve suspension in their results. Both single and collection entry points obey configured-version capability checks.

The internal service contract gains `ValidateProcessInstanceStates(states domain.States) error`; it performs no I/O. All three adapters and required test doubles implement it. It rejects only the new unsupported suspension capability on 8.8/8.9, preserving existing handling of other states. It is not a new public facade method.

| Operation | Configured 8.10 | Configured 8.8/8.9 |
| --- | --- | --- |
| Explicit SUSPENDED filter | Send suspended equality filter | Unsupported before backend discovery/search |
| Explicit SUSPENDED expectation | Succeed when observed (and other requested conditions match) | Unsupported before lookup/workers |
| Mixed desired states containing SUSPENDED | Existing OR state semantics | Unsupported, no silent removal |
| ACTIVE filter | ACTIVE only | Existing behavior |
| Existing ALL/unfiltered selection | Includes suspended matches | Existing behavior, no added guard failure |
| Creation visibility | Suspension proves observation | Existing defaults remain valid |
| Unrecognized observed state | Existing fallback | Existing fallback |

Validation must cover page search, shared page traversal/total, direct-incident search, single/bulk state waits and single/bulk combined expectations. The internal creation waiter is not an explicitly requested suspended expectation and must not inherit that guard.

## CLI

Existing `get process-instance --state suspended` and `expect process-instance --key <key> --state suspended` use the new value. Preserve aliases, normal flag validation precedence and supported output modes. Public state syntax is case-insensitive. Help and validation list the value with the Camunda 8.10 restriction; value completion offers command-appropriate states without configuration/bootstrap/network requirements. Do not invent new flags or enable unsupported command-specific combinations.

No suspend/resume commands are added. The feature does not authorize newly canceling/deleting suspended instances or expanding purge scopes.

## Output

A structured instance with a timestamp includes the additive field, for example:

```json
{"state":"SUSPENDED","suspendedDate":"2026-10-17T08:30:00Z"}
```

This is a payload excerpt, not a replacement command envelope. A subsequent response lacking the timestamp omits `suspendedDate`; a null or zero source date does not become a zero-date string. Existing result fields, null/array conventions and error envelopes remain unchanged.

Default human rows gain no timestamp column/detail. Keys-only remains one key per line, or zero bytes for no results. Empty successful JSON emits exactly one existing successful envelope with the established empty payload and EOF; quiet combined with JSON still emits the explicit JSON result. Preserve stderr/stdout separation and prompt eligibility. Read-only empty searches must not introduce prompts or additional discovery requests.

## Acceptance boundaries

Suspension never satisfies a terminal-only expectation. Creation visibility is not execution completion. A simultaneous incident expectation must still match. No accumulated suspension duration or historical intervals are derived. Repeated reads verify timestamp clearing; existing definition watch has no instance metadata field to extend.
