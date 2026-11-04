# Data Model: Late Business-ID Assignment

This is invocation-local request/plan/result data. Camunda remains the authority for stored identity and eligibility. See [public contracts](contracts/public-api.md).

## Assignment intent

- Public/domain field: `SetBusinessID *string`; output name: `requestedBusinessId` (optional).
- Nil means absent in existing job operations. Direct assignment requires a value.
- Reject empty, whitespace-only, invalid UTF-8, and more than 256 UTF-16 code units. Preserve accepted bytes, including surrounding whitespace, case and commas.
- Validate assignment with a separate helper; creation/search validation and observed values retain their existing semantics.
- Copy pointers at request conversion and plan preparation boundaries. Do not share mutable caller storage.

## Process assignment plan

Proposed public/domain type: `BusinessIDAssignmentPlan`.

| Field | Meaning / invariant |
| --- | --- |
| RequestedBusinessID | Required exact frozen assignment |
| RequestedKeys | Complete validated keys, normalized and deduplicated in stable order |
| Items | One item per distinct target; never expanded to parent/family |
| Item.Key | Exact process target |
| Item.PreviouslyObservedBusinessID | Optional prior fact; absence is not synthesized |
| Item.State / parent context | Optional preview facts; never a guarantee of eligibility |
| Readiness | Authoritative complete preparation or incomplete diagnostic evidence |

The plan is invocation-local data, not a persisted authorization token. Validate readiness, unique keys, consistent target membership and assignment again on execution; adapter capability guards apply even to caller-constructed plans. Failed preparation cannot authorize mutation. An intentionally empty stdin plan may be complete with zero items; omitted keys without deliberate stdin remain invalid CLI input.

## Process assignment results

Proposed `BusinessIDAssignmentResults` contains ordered items, each retaining target key, requested value, prior observation, submission evidence, optional later observation, HTTP metadata and machine diagnostic evidence. Include not-started slots when fail-fast/interruption prevents scheduling. Preserve partial results alongside original errors.

Submission and observation are separate axes:

| Submission | Observation | Interpretation |
| --- | --- | --- |
| not_submitted | skipped | Preview, empty/no work, local validation/precondition, or not scheduled; distinguish with reason |
| accepted | skipped | Native acknowledgement; no-wait skipped confirmation |
| accepted | confirmed | Exact-key observation matches requested BID after acceptance |
| accepted | unconfirmed | Acceptance retained; observation failed, timed out or did not match |
| rejected | any available fact | Authoritative rejection; identical visible value does not turn it into success |
| unknown | any available fact | Lost/ambiguous response; attribution unproven and original error retained |

A post-acceptance waiter uses exact process reads and configured backoff. It never mutates or redirects. Rejection/unknown never triggers replay. Previous and later observations are not copied into related resource snapshots.

## Job completion payload and prepared batch

Extend existing `WorkerOutcomeRequest`, `CompletionRequest`, `PreparedCompletionBatch`, scalar `UpdatePlan`, and corresponding internal models rather than adding a competing workflow.

- Payload: optional SetBusinessID plus existing variables and supported fields.
- Selection: existing mode, input form, keys/predicates, tenant context, batch size, limit and completion reason remain intact. `Retries *int32` preserves exact zero.
- Candidates: each job's existing `BusinessId` remains its own observed snapshot.
- Freeze BID pointers with `toolx.CopyPtr`, nested variables with `toolx.CopyJSONMap`, and selection slices at conversion, preparation and execution snapshot boundaries. Scalar approved payloads require the same protection.
- Private submit port receives a typed immutable payload rather than variables alone. All adapters/doubles change together.
- Distinct job keys remain distinct even if ProcessInstanceKey is identical. Scalar/collection input provenance persists before deduplication.

## Assignment evidence on job results

Add optional `assignment` evidence to scalar worker outcomes and each bulk outcome when assignment is requested. Reuse the same conceptual statuses as direct assignment: requested, not_submitted, accepted, rejected, unknown; independent confirmed observation must never be fabricated. Concrete Go enum names may follow local naming conventions; serialized meanings below are fixed.

Existing submission and completion observation fields remain unchanged. No assignment field is emitted when intent is absent. A successful atomic acknowledgement establishes assignment accepted and completion accepted. It does not refresh job/task/listener snapshots. Later COMPLETED observation after unknown/rejected submission does not change assignment evidence or erase the original error.

A pre-submit already-completed job has assignment not_submitted/unsatisfied, even where ordinary completion would consider it satisfied. For assignment intent, a `completed` disposition cannot represent inferred assignment success. Preserve original failure classification on rejected or uncertain submissions regardless of reconciliation state.

## Relationships and exclusions

A process plan owns ordered exact process targets. A completion batch owns ordered exact job targets; each job belongs to its existing process. Both own copied assignment intent. Observed process/job/task/listener identities remain independent facts. Native identity is distinct from a process variable named businessId. No clearing, replacement, synthetic propagation, or assignment field on other mutation types is introduced.
