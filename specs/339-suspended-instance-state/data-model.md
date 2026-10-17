# Data Model: Suspended Process-Instance State

## Process-instance state

Add domain and public `StateSuspended = "SUSPENDED"`. Public parsing accepts case variations and surrounding whitespace via the established parser. Include it in valid values while preserving the existing canceled/cancelled/terminated behavior and UNKNOWN fallback.

Suspension is nonterminal. Keep domain/public terminal sets unchanged, including their existing different treatment of ABSENT. State selection and expectation comparison must not equate suspension with ACTIVE, COMPLETED, CANCELED, TERMINATED or ABSENT.

No new transition engine is introduced. Reads reflect observed backend transitions (for example ACTIVE → SUSPENDED → ACTIVE); c8volt does not infer or submit those transitions in this feature. Creation observation accepts SUSPENDED as presence, without implying completion.

## Process instance

Existing identity, tenant, parent relationships and date fields remain unchanged.

| Field | Domain/public representation | Structured output | Rule |
| --- | --- | --- | --- |
| State | Existing State plus SUSPENDED | Existing state field | Preserve backend state through conversion |
| SuspendedDate | New string alongside existing date strings | `suspendedDate,omitempty` | Present timestamp formatted UTC RFC3339Nano; absent/null/zero maps to empty and is omitted |

The generated v810 source is `ProcessInstanceResult.SuspendedDate *time.Time`. Older adapters leave the added field empty. Map `fromDomainProcessInstance` and `toDomainProcessInstance`; nested/enriched payloads using the public ProcessInstance inherit the field without separate envelopes.

Every read/search result is a fresh snapshot. Replacing a result whose timestamp was present with one whose timestamp is absent must clear the value. Do not infer history, accumulated duration, or state from timestamp presence alone. No persistent migration is required.

## Selection and expectation

Keep existing filter/request structures and OR semantics for desired state sets. A suspended member is explicit regardless of case or other members. On configured 8.8/8.9 it causes unsupported-capability failure before discovery/lookup, even in an empty result or empty-key path. Unset filters and existing ALL selection keep their existing representation and behavior; do not expand an implicit ALL into a list triggering the older-version guard.

On v810 use the existing state equality-filter representation with SUSPENDED. Preserve CANCELED-to-TERMINATED request compatibility. ACTIVE remains exact.

## Reports and errors

Keep existing report schemas, statuses and envelope shapes. A successful suspended expectation preserves SUSPENDED. A terminal wait observing suspension remains unsatisfied; retain existing failure/timeout report semantics rather than rewriting unsuccessful reports as successful suspended reports.

Capability failures wrap the established domain unsupported sentinel and pass through existing public error conversion. Do not redefine the sentinel or invent a feature-specific exit code.
