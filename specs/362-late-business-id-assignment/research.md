# Research: Late Business-ID Assignment

**Date**: 2026-11-02 | **Source**: [spec.md](spec.md), issue #362, and repository code inspection. No live engine was contacted. Repository research was split between direct assignment and job completion agents as required by the planning workflow.

## 1. Native transport and version support

**Decision**: Use the existing v810 generated `AssignProcessInstanceBusinessIdWithResponse` endpoint for direct assignment and the existing completion body `BusinessId` field for atomic completion. Add explicit unsupported-operation guards to v88/v89 entry points, including execution of caller-supplied prepared batches.

**Rationale**: `internal/clients/camunda/v810/camunda/client.gen.go` already defines the assignment instruction, POST `/process-instances/{processInstanceKey}/business-id-assignment`, response metadata, and completion payload. Assignment returns HTTP evidence without a success object; do not require a success JSON body. Map documented backend problem responses through existing HTTP/domain error helpers. The generated contract establishes transport support, not engine eligibility: active-root/no-BID/uniqueness-disabled rules come from issue #362's pinned 8.10.3 baseline.

**Alternatives considered**: Client regeneration is unnecessary; writing a variable named businessId does not assign native identity; two independent mutations cannot provide atomic completion.

## 2. Assignment-specific validation

**Decision**: Add `domain.ValidateBusinessIDAssignment(*string)` that reuses `ValidateBusinessID` and additionally rejects whitespace-only input without altering accepted bytes. Nil means omission only where assignment is optional; direct assignment requires a nonnil value.

**Rationale**: `internal/domain/business_id.go` already counts supplementary characters as two UTF-16 units and checks UTF-8 and the 256-unit boundary. Creation intentionally accepts whitespace-only values; strengthening its validator would regress existing behavior. Public and service entry points need the assignment rule, not only Cobra validation.

**Alternatives considered**: Trimming values corrupts accepted padded IDs; duplicating Unicode counting invites inconsistent boundaries; globally changing creation validation expands scope.

## 3. Direct assignment ownership

**Decision**: Add a focused process assignment coordinator with read, one-shot assignment, and bounded observation ports. Expose thin prepare/execute facade operations from `c8volt/process`, promoted through the root API. Reuse `toolx/pool`, call options, exact-key authorization, and process read/backoff patterns.

**Rationale**: `internal/services/processinstance/update.go` and `c8volt/process/variable_updates.go` show copied collection plans, ordered partial results, bounded concurrency, and facade error conversion. Assignment has a distinct lifecycle and must not reuse variable planning or root/family resolution. Freeze complete validated input before confirmation. Backend eligibility remains authoritative; instance reads provide preview facts, not a guarantee about uniqueness configuration.

**Alternatives considered**: CLI worker pools or facade polling violate layering. Reusing tree mutation would redirect/expand scope. A scalar-only implementation followed by bulk rebuilding duplicates required work.

## 4. One-shot mutation and honest evidence

**Decision**: Never wrap assignment-bearing submissions in `RetryCamundaMutation`. Classify explicit HTTP acceptance/rejection versus uncertain response using existing completion evidence conventions; preserve the original error. Direct normal execution observes exact-instance BID only after known acceptance, within configured backoff; no-wait returns acceptance with observation skipped. An uncertain submission remains unknown even if later state is consistent with success.

**Rationale**: v810 scalar `submitJobCompletion` currently uses the retry helper, which can replay ambiguous transport failures. Bulk adapters already submit once. Existing suspension/variable mechanics are useful structural examples but their replay policy is unsuitable for one-time assignment. Successful completion acknowledgement establishes atomic acceptance, not a refreshed resource snapshot.

**Alternatives considered**: Repeating an identical BID conflicts; replay cannot make assignment idempotent. A later matching value or completed job does not attribute a change to this request.

## 5. Extend existing completion preparation once

**Decision**: Carry optional `SetBusinessID` separately from observed `BusinessId` through public/domain worker requests, completion requests, prepared batches, previews, results, and conversions. Copy pointers with `toolx.CopyPtr` and nested variables with `toolx.CopyJSONMap`. Extend the private submission port to receive a typed frozen completion payload rather than only variables; change all adapters and doubles in the same compiling unit.

**Rationale**: `internal/services/job/completion/prepare.go` and `execution_plan.go:snapshotExecutionBatch` already copy selection and variables. Scalar mapping presently shares variable maps: snapshot BID-bearing scalar approved payloads before confirmation too. Preserve input form before deduplication, tenant-scoped discovery, exact zero-retry selectors, limits, listener variable restrictions, and job-key deduplication.

**Alternatives considered**: Reusing candidate.BusinessId fabricates an observation. Deduplication by process key drops distinct jobs. A new bulk engine would duplicate existing safe discovery and concurrency.

## 6. Completion satisfaction must depend on assignment intent

**Decision**: Add separate assignment evidence and BID-aware satisfaction rules. Preexisting completed jobs do not satisfy assignment. In `completion/execute.go:executeCandidate` and `reconcile.go:reconcile`, retain completion observations but do not clear uncertain/rejected assignment or its original error merely because a job is now COMPLETED. Plain completion retains its current reconciliation behavior.

**Rationale**: Existing completion-only reconciliation can return satisfied and nil after observing completion. That would falsely report BID assignment. A successful atomic acknowledgement may report accepted assignment; unknown acknowledgement plus completion state must stay unknown and unsatisfied.

**Alternatives considered**: Changing all plain completion semantics violates omitted-input compatibility. Polling stale job BID snapshots or introducing mandatory process lookups would not prove attribution and would add scope.

## 7. Serialization, output, and compatibility

**Decision**: Reuse `job/v810/completion_request.go:completeJobWithResponse`, whose optional-field wrapper already omits absent businessId and jobLeaseToken while retaining generated fields. Add assignment intent/evidence as optional output fields, absent in legacy cases. Preserve scalar versus collection shapes and existing unsupported-mode behavior: bulk completion rejects keys-only.

**Rationale**: Scalar `cmd/update_job_outcome.go:executeUpdateJobWorkerOutcome` currently discards partial result on error; add a BID-aware result/error rendering path. Use existing one-envelope and exit-after-render handling. Bulk `finishUpdateJobBatch` and configured logging helpers already demonstrate aggregate failures. Never interpolate raw outcome errors into human lines.

**Alternatives considered**: Replacing schemas or inventing key output changes unrelated contracts; rendering an envelope and invoking a second error renderer duplicates JSON.

## 8. Validation and documentation

**Decision**: Implement with isolated generated-client mocks, test-owned HTTP servers, command/subprocess and real-terminal fixtures. Update README, command metadata/examples, and regenerate docs in behavior-delivery units. This planning turn uses document checks only.

**Rationale**: Existing completion payload, reconciliation, logging, and terminal suites cover the nearest boundaries. New tests must catch direct-print substitutions, raw HTTP leakage, duplicate envelopes, accidental replay, and mutation after incomplete discovery. No live engine or cluster validation is authorized or required.

**Alternatives considered**: Live smoke tests would violate the implementation testing boundary. Renderer-only tests cannot establish configured logging or prompt routing.

## Resolution status

All design unknowns are resolved from repository facts and the issue's explicit product rules. No clarification is outstanding. Real-engine eligibility remains unverified by this workflow and must not be described as experimentally confirmed.
