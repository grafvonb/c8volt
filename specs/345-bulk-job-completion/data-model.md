# Data Model: Bulk Job Completion

Public types live in `c8volt/job`; equivalent internal types live in `internal/domain`. Public models never expose generated clients or domain types. Copy maps and slices across the facade boundary. Proposed names define responsibilities; existing repository naming takes precedence during implementation.

## Selection

`CompletionSelection` contains:

| Field | Meaning |
|---|---|
| `mode` | `explicit` or `search` |
| `keys` | Validated explicit string keys; input order retained before deduplication |
| `filter` | Existing normalized job-search predicates; retries remains optional, including explicit zero |
| `batchSize`, `limit` | Search controls; zero limit means uncapped using existing conventions |
| `inputForm` | Single scalar, collection, stdin or filters; preserves rendering compatibility |

Keys and filter selection are mutually exclusive. Tenant settings are resolved by the configured client; they do not qualify as a predicate. Explicit keys use backend authorization and may span tenants.

`CompletionRequest` adds optional variables to the selection. Payload validation follows existing JSON-object parsing. Nonempty variables cannot accompany task-listener batch candidates. Worker count and execution options are separate from selection provenance.

## Prepared batch

`PreparedCompletionBatch` records:

- Selection mode, normalized predicates and effective discovery tenant scope.
- Unique candidate list in frozen order.
- Completion reason: `exhausted`, `limit_reached`, or `explicit_inputs_read`.
- Preparation status: `complete` or `incomplete`; page count and observed count for diagnostics.
- Requested limit, whether it bounded discovery, and payload.
- Actual selected tenant context.

`CompletionCandidate` records key, process-instance key, element-instance key/ID, tenant, type, kind, listener event, observed state and retries. Include only available facts; no fabricated lifecycle dates.

A complete batch may contain zero candidates. An incomplete batch is never executable. Partial observations may aid an error report but cannot be represented as a frozen executable scope. A returned plan is an in-memory value, not a durable authorization token or backend snapshot. The CLI passes it unchanged to the same configured client; execution validates its invariants and still rechecks current scope.

The target limit counts unique matching candidates, regardless of whether the backend will accept completion. Duplicate and out-of-scope rows do not consume it. Execution never replaces removed candidates or appends newly created listeners.

## Submission and observation

`CompletionSubmission` retains `status` (`not_submitted`, `accepted`, `rejected`, `uncertain`), optional HTTP status, and normalized diagnostic evidence. Bulk permits at most one intentional attempt per selected key. An uncertain submission cannot become rejected merely because a later read is missing.

`CompletionObservation` retains `status` (`completed`, `not_completed`, `missing`, `unreadable`, `unknown`, `skipped`), last known state, and phase (`recheck` or `reconciliation`) when applicable.

These axes remain independent:

| Situation | Submission | Observation | Final disposition |
|---|---|---|---|
| Already completed at recheck | not_submitted | completed | completed |
| Accepted, no independent confirmation | accepted | skipped | accepted |
| 404/409 then observed COMPLETED | rejected | completed | completed |
| Transport uncertainty then observed COMPLETED | uncertain | completed | completed |
| Uncertain request, only missing reads | uncertain | missing | unsatisfied |
| Filter/tenant changed | not_submitted | not_completed | unsatisfied |
| Other terminal state | not_submitted | not_completed | unsatisfied |
| Missing/unreadable at recheck | not_submitted | missing/unreadable | unsatisfied |
| Fail-fast prevents dispatch | not_submitted | skipped | not_started |

Observed completion is not attributed to this invocation. Accepted does not mean independently confirmed. Neither implies termination of any process instance.

## Per-key outcome and aggregate

`CompletionOutcome` contains candidate context, `disposition`, submission, observation and a stable `reason` when needed. Reasons distinguish `scope_changed`, `other_terminal`, `unsupported_state`, `missing`, `unreadable`, `payload_incompatible`, `rejected`, `uncertain`, `fail_fast`, `declined`, and `interrupted`. A declined prepared batch retains each key as not started.

`CompletionBatchReport` includes:

- Phase/status: `preview`, `finished`, `aborted`, `preparation_failed`, or `interrupted`.
- Prepared selection metadata and candidate count.
- Counts: `selected`, `accepted`, `completed`, `unsatisfied`, `notStarted`.
- Deterministically ordered per-key outcomes for execution paths.
- Preview candidates for dry-run, without fake mutation outcomes.
- Aggregate error/class through the existing envelope, alongside preserved payload.

For an execution report:

```text
selected = accepted + completed + unsatisfied + notStarted
len(outcomes) = selected
```

Each key contributes to exactly one disposition count. Submission-axis counts, if displayed, must not be added again to that partition. A preview has candidates but no execution outcomes; an incomplete preparation has no executable selection and no invented outcomes.

The new bulk payload uses `candidates` for preview and `outcomes` for execution. The applicable collection is always present and encoded as `[]` when empty; the inapplicable collection is omitted. Counts are numeric and present even when zero. Do not encode an empty collection as `null` or fabricate a per-key row. This is an additive bulk contract; legacy single-key payload and shared envelope omission rules are unchanged.

Overall success means a complete empty selection, or all selected outcomes are accepted/completed. Accepted outcomes explicitly show confirmation skipped. Any unsatisfied/not-started target, abort, preparation failure or caller interruption makes the invocation unsuccessful, subject to the existing explicit exit-code suppression option. An interruption remains an invocation failure even if all already-returned rows are satisfied.

## Lifecycle

```text
parse -> prepare -> incomplete/error: zero writes
                 -> complete empty: successful no-op
                 -> complete nonempty -> dry-run: preview, zero writes
                                      -> declined: aborted, zero writes
                                      -> approved -> recheck -> submit at most once
                                                            -> reconcile if needed
                                                            -> aggregate + error
```

Fail-fast/cancellation stop further scheduling and retain rows for every frozen key. Missing observations never transition directly to confirmed completion.
