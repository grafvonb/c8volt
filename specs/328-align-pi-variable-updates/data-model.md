# Data Model: Process Instance Variable Updates

This design extends the in-memory domain/facade models. It adds no database, migration, or persisted execution plan. Exact proposed Go identifiers may follow local naming during implementation; the invariants below are binding.

## Root variable page

| Field | Meaning / validation |
| --- | --- |
| Request | Positive bounded size, nonnegative offset or advancing cursor; checked arithmetic |
| Items | Existing `ProcessInstanceVariable` values, preserving raw value and `APITruncated` |
| RawCount | Backend item count before local filtering; nonnegative |
| ReportedTotal / certainty | Optional exact count or lower bound; absence is not zero |
| Continuation / EndCursor | Backend progress evidence; contradictory or repeated non-progressing metadata is an error |

Version adapters normalize page metadata and retain process-instance, root-scope, and tenant filters. The shared traversal consumes page metadata until authoritative completion. A sparse retained page may contain no root variables while raw progress still continues. Exact totals must be consistent with observed counts; retained lower bounds must not decrease. A zero-item page advertising continuation requires a usable advancing cursor or other supported progress; otherwise fail rather than loop or silently truncate.

## Observed variable

Reuse `ProcessInstanceVariable`: name, raw value, variable key, process-instance key, scope key, tenant ID, and truncation flag. Only variables with both instance key and scope key equal to the selected key participate in root planning and confirmation. Conflicting observations of one root/name must fail planning instead of choosing an arbitrary value. Identical duplicate observations may be normalized deterministically.

## Prepared collection plan

| Field | Meaning / validation |
| --- | --- |
| RequestedKeys | Unique selected keys in stable input order |
| RequestedValues | Deep-copied JSON-compatible name/value input, immutable during use |
| Instances | One per selected key, each with the instance/root destination and classifications |
| Tenant evidence | Facts read during preparation, sufficient for existing approval/envelope behavior |
| Derived counts | Selected/changed instance counts and classification counts, computed from the same plan |

The service returns a plan only after all selected instances are successfully planned. Public facade plan types mirror these facts without exposing internal domain types or generated clients. Copies must isolate nested objects and arrays, not only the outer map.

## Per-instance plan

| Field | Meaning |
| --- | --- |
| ProcessInstanceKey / ScopeKey | Both identify the same selected root; neither may be blank |
| Additions | Requested names absent from that root; desired values |
| Changes | Requested names with different or unproven truncated values; before/after values and truncation indication |
| UnchangedRequested | Requested names with complete, equal observed values |
| Untouched | Existing root names absent from the request |
| MutationVariables | Exactly the union of additions and change after-values; absent/empty for an unchanged instance |
| VerificationValues | All requested values for this instance, including initially unchanged ones |

Names in the four classifications are disjoint. Every requested name appears once in additions, changes, or unchanged. Untouched names never enter a submitted payload. Equality follows UT's JSON-normalization behavior; truncated observations always lack sufficient evidence for unchanged classification. Execution validates correspondence between the classifications, requested values, destination, and payload before any mutation.

## Outcomes and transitions

Retain existing `ProcessInstanceVariableUpdateResult` fields: `key`, `status`, `mutationAccepted`, optional `confirmationStatus`, `statusCode`, `message`, `error`, and `variables`. Retain the enclosing `items,omitempty` field. Do not expose new execution bookkeeping in the command envelope.

| Situation | Status | mutationAccepted | Meaning |
| --- | --- | --- | --- |
| No planned additions/changes | `unchanged` (additive) | false | Successful no-op; no write or wait |
| Writer accepts and no-wait is set | `submitted` | true | Accepted, verification skipped |
| Writer accepts and complete root values match | `confirmed` | true | Default operational success |
| Writer rejects/fails before acceptance is established | `mutation_failed` | false | Mutation failure under existing error semantics |
| Accepted write, later verification fails/times out | `confirmation_failed` | true | Preserve acceptance and confirmation error |
| Fail-fast/cancellation prevents starting this target | `skipped` (additive) | false | Unexecuted work remains identifiable |

`OK()` treats unchanged as successful and retains existing accepted/confirmed success semantics; mutation/confirmation failures and skipped entries are unsuccessful. Mutation summaries and keys-only filtering use `MutationAccepted`, not success totals alone. Result order follows selected-key order regardless of worker completion order. Fail-fast cannot undo concurrent in-flight writes; collect their actual outcomes instead of marking them skipped.

Dry-run never enters execution states. An entirely unchanged CLI run renders the existing no-op preview, with `mutationSubmitted=false`, rather than manufacturing mutation reports. Planning errors and invalid plans do not enter execution. No atomic rollback or concurrency lock against external changes is introduced.
