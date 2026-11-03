# Public and Native Contracts

## Process facade

Add the following proposed methods to `c8volt/process.API`, inherited by the aggregate facade:

```text
PlanProcessInstanceBusinessIDAssignments(ctx, keys typex.Keys, businessID *string, opts ...foptions.FacadeOption)
    -> (BusinessIDAssignmentPlan, error)
ExecuteProcessInstanceBusinessIDAssignments(ctx, plan BusinessIDAssignmentPlan, wantedWorkers int, opts ...foptions.FacadeOption)
    -> (BusinessIDAssignmentResults, error)
```

Both are thin mechanical delegation/conversion with ferrors conversion. New domain/service contracts own input normalization, copied planning, worker pools and observation. The required pointer distinguishes absent assignment from explicit empty and rejects both for direct assignment. Keep all interface implementations, constructors and mocks aligned within a compiling unit.

Plans retain exact ordered distinct keys, copied requested value and independently observed facts. A complete empty plan is valid. Invalid, incomplete, inconsistent or unsupported prepared inputs cannot submit. Invocation-local plans confer no additional authorization and do not guarantee eligibility. Execution returns partial ordered results and original error together.

Version-specific process ports validate capability and submit exact-key assignment once. Native v810 request:

```text
POST /process-instances/{exactKey}/business-id-assignment
Content-Type: application/json
{"businessId":"<exact accepted value>"}
```

Use existing generated instruction and response parser, HTTP evidence and problem mapping. Do not require a JSON success body or consult a parent's key. Accepted normal execution may confirm using exact-key reads; no-wait skips confirmation. Ambiguous submission stays unknown, with no automatic replay or invented attribution.

## Existing job facade

Extend public/domain `WorkerOutcomeRequest`, `CompletionRequest`, `PreparedCompletionBatch`, scalar plan and mappings with optional `SetBusinessID *string`. Completion modes accept it; all other modes reject it. Preparation and execution on v88/v89 reject it before backend I/O, including caller-constructed batches. Preserve error conversion and return evidence alongside errors.

`PrepareJobCompletionBatch` and `ExecuteJobCompletionBatch` keep existing signatures and selection semantics; optional fields are additive. The private submission port changes to accept a copied typed payload containing variables and assignment. All adapters and test doubles update together.

BID-bearing scalar execution uses an approved copied payload. Plan attachment cannot authorize a different mode, key, or value from the approved operation. Nested variables and pointers must not alias caller inputs. Plain completion retry/reconciliation behavior remains unchanged.

## Native completion

```text
POST /jobs/{exactJobKey}/completion
{"variables":{"approved":true},"businessId":"ORDER-123"}
```

Use generated v810 body and existing optional-field serialization helper. Preserve other supported fields, including any already-present job lease token. Without assignment, omit businessId entirely, rather than null or empty. One request per job combines completion and assignment; rejection cannot trigger a plain completion, separate mutation or replay.

## Evidence and compatibility

Add optional `requestedBusinessId` and `assignment` output fields only when assignment is requested. Proposed assignment evidence contains `status` and optional known reason/HTTP evidence, with statuses:

| Status | Evidence / required behavior |
| --- | --- |
| not_submitted | No mutation sent, including preview, precondition or not-started outcome |
| accepted | Native assignment or combined completion acknowledged |
| confirmed | Accepted direct assignment subsequently observed on the exact process key |
| rejected | Authoritative native rejection; retain failure even if later state changed |
| unknown | Missing/ambiguous acknowledgement; retain original error and never replay |

For accepted but unconfirmed direct assignment, status remains accepted and separate confirmation evidence records failure/skipped. Existing completion submission/observation fields remain independent. A job COMPLETED after lost response may be recorded as an observation while assignment stays unknown and unsatisfied. Preexisting completed jobs cannot satisfy assignment. Machine diagnostic evidence retains existing schemas and original error class; human formatters consume bounded known facts rather than raw errors.

Optional assignment output must not change absent-field/null conventions for existing callers. New process API methods are a source compatibility obligation for external custom interface implementations; document it. Added struct fields can affect unkeyed external literals. No generated types or internal sentinels leak into public signatures.

## Contract validation

Use exact request capture and generated-client doubles to prove wire preservation, native omission, no redirect, one-shot submission, all adapter guards, copy stability, complete discovery and response classification. Assert partial result plus original error at facade and command boundaries. See [data model](../data-model.md) and [quickstart](../quickstart.md).
