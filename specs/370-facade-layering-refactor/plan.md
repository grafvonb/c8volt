# Facade layering refactor for v5.0.0

## Baseline and scan

2026-11-03: clean `develop`, no existing diff or dirty changes; created
`370-facade-layering-refactor` using the next local feature number (not an issue association).
No commits, pushes, staging, resets, reverts or discards are authorized.

Read the maintenance prompt, prompt-local AGENTS, repository rules, Ralph implementation
rules and active 369 feature context. This is a manual maintenance run, not a Ralph
iteration; complete every slice. Preserve flag-intent behavior and historical artifacts.

Full-tree inventory: 14 public packages / 127 Go files and 76 internal directories /
547 Go files, including generated clients, test-only directories, adapters, waiter,
walker, paging, HTTP, authentication, common and operational workflow packages.
Scanned imports, execution loops, concurrency, worker selection, activities, conversion,
public models and workflow delegation; traced the remaining candidates to their owners.
No production internal-to-facade/command imports. Existing internal services already own
pools, polling, waits, traversal, paging, filtering during discovery, impact planning,
enrichment, repair and cleanup. Generated code needs no edits.

## Ordered implementation checklist

- [x] S1: Resource bulk result identity. `c8volt/resource/bulk.go:DeleteProcessDefinitions`
  repeats service key deduplication, and `convert.go:fromResourceDeleteResponses` reconstructs
  identity by input position. Internal `ResourceDeleteResponse` already has Key. Add
  `internal/services/processdefinition/delete_reports.go:DeleteProcessDefinitionReports`
  to return keyed slots even for failures/unattempted pool slots, then map these
  mechanically in the facade. Preserve the lower-level response APIs, whose empty
  keys distinguish unattempted work for existing internal consumers. No new public type or interface.
  Risk: medium (duplicate keys, fail-fast slots, truncated forced cleanup results).
  Add service tests and facade mapping coverage. Run targeted processdefinition delete
  and resource facade tests, including race checks for worker paths.
- [x] S2: Resource plan tenant evidence. `c8volt/resource/model.go:TenantEvidence` and
  its accumulator duplicate internal `processdefinition/delete.go` aggregation. Export
  that narrowly scoped internal operation and delegate using mechanical plan-item and
  tenant-evidence conversions. Keep model methods/public fields/JSON unchanged, including
  manually constructed plans; do not cache stale evidence or add reads. Reuse the existing
  common accumulator; retain legacy aggregate-only handling at the public boundary.
  No new public type/interface. Risk: medium (first observation, kind-qualified identity,
  aggregate-only fallback, nil/empty collections). Add isolated service and facade tests;
  run tenant-evidence and resource tests after migration.
- [x] S3: Job scalar approval. `c8volt/job/client.go:toDomainWorkerOutcomeRequest`,
  `UpdateJob`, and `completion_plan.go:FreezeCompletionPayload` own reusable assignment
  eligibility and frozen-approval validation. Move policy and payload resolution into
  `internal/services/job/approval.go`, retaining private approval state and snapshot/mapping
  at the public boundary. Add an internal approval input type and preparation/validation
  functions; no versioned API or public interface changes. Preserve rejection before
  dispatch, frozen key/mode/value checks, exact values, copied payloads, ordinary outcome
  behavior, partial evidence and existing option forwarding. Risk: medium (approval
  tampering, request mutation after approval, ownership, invalid JSON, direct assignment).
  Add direct service policy tests, retain/extend facade tests, run targeted job/service
  tests and isolated command completion/assignment tests.
- [x] Completion: repeat full-tree scan, review changed output/error boundaries, run
  broader `go test ./c8volt/... ./internal/... -race -count=1` and affected command tests,
  formatting and `git diff --check`; record actual results and any blockers.

## Intentional retained constructs

Public-model collection operations (state predicates, filter convenience methods, report
counts/totals), JSON wire shaping, pointer/map/slice copying and page visitor conversion
remain public API/model mapping; none fetch pages or schedule backend work. Constructor
service wiring and missing-dependency guards remain composition concerns. Public options
map controls and callbacks without executing them. Scalar wrappers select one result from
an internal collection operation. Existing exported conversion APIs with internal input
types are retained for source compatibility; changing/removing signatures is outside this
behavior-preserving refactor. Legacy error/exit helpers in ferrors are separate reporting
APIs, not backend workflow orchestration; this run does not change their output contract.

## Validation boundary

Only isolated Go unit, mock, fake HTTP, subprocess and terminal tests. Integration-tagged
auth tests are excluded; no cluster/profile discovery, live service, smoke or integration
scripts. Check inherited Go flags and environment before tests. CLI syntax, metadata and
output do not change, so generated CLI docs do not require regeneration.

## Results

All three slices completed. No blocked slices. Initial checkout was clean; no pre-existing changes needed preservation.

S1: targeted processdefinition/resource worker and mapping tests passed with `-race`.
New canceled-scope regression failed before implementation and passes afterward.

S2: targeted resource and processdefinition tenant-evidence checks passed. Regression
coverage includes duplicate cross-kind keys, aggregate-only plans, edited plan observations,
copy isolation and nil/empty representation. Initial nil/empty regression was corrected.

S3: targeted `internal/services/job`, `c8volt/job`, and affected command tests passed.
The public plan now privately holds a domain payload; the facade only maps/snapshots
attachments and invokes internal approval policy. Local error classification, exact
messages and empty rejected-result payloads remain unchanged. No output emitter changed.

## Final audit and validation evidence

Final inventory: 14 public package directories / 129 Go files (73 production),
76 internal package directories / 551 Go files (342 production). Re-read/scanned
all production files for imports, loops, concurrency, traversal and delegation.
Facade counts: zero pool calls, poller/waiter usage, backend activity orchestration,
worker selection, custom goroutine/channel scheduling, unbounded backend traversal
loops, key deduplication, generated-client imports or versioned-service imports.
Internal production code has zero imports of facade/command packages. Manual review
of public loops confirms conversion, model convenience operations and option mapping
only. No remaining backend-workflow violation was identified.

Compatibility exceptions retained intentionally:

- Existing facade `New*` constructors accept internal service interfaces as dependency
  injection seams; public operational API methods remain facade/domain-independent.
- `c8volt/process/client.go:MapStateResponseToReport` and
  `MapStateResponsesToReports` retain their existing internal-domain input signatures.
  They only convert models; removing/changing them would change public source compatibility.
- `c8volt/foptions/options.go:MapFacadeOptionsToCallOptions` retains its established
  option bridge signature for service wiring. It executes no backend workflow.
- Public collection filters/state predicates/report totals and JSON/copy adapters are
  local model conveniences, not backend query strategy or execution. Missing-dependency
  checks remain constructor/wiring preconditions. Legacy ferrors reporting/exit helpers
  were reviewed as separate APIs and were not changed by this refactor.

Validation commands executed (all final checks PASS):

```sh
GOFLAGS= go test ./internal/services/processdefinition ./c8volt/resource -run 'TestDeleteProcessDefinitionsOwnsResultIdentity|TestDeleteReportsMapServiceIdentity|TestClient_DeleteProcessDefinitions|TestDeleteProcessDefinitionResources' -race -count=1
GOFLAGS= go test ./c8volt/resource ./internal/services/processdefinition -run 'TestPlanTenantEvidence|TestDeleteProcessDefinitionPlan_TenantEvidence|TestPreviewDeleteProcessDefinitionsAggregatesTenantEvidence|TestProcessDefinitionPlanTenantEvidence' -count=1
GOCACHE=/tmp/c8volt-gocache GOFLAGS= go test ./c8volt/job -run 'TestCompletionAssignment|TestClient' -count=1
GOCACHE=/tmp/c8volt-gocache GOFLAGS= go test ./internal/services/job ./c8volt/job -run 'TestScalarCompletionApproval|TestAdministrativeJobAssignmentGuard|TestCompletionAssignment|TestCompletionApprovalLocalErrorContract|TestClient' -count=1
GOCACHE=/tmp/c8volt-gocache GOFLAGS= go test ./cmd -run 'TestCompletionAssignment|TestScalarJob|TestUpdateJob|TestBulkCompletionDiagnosticLogging|TestBulkCompletionSummaryLogging' -count=1
GOCACHE=/tmp/c8volt-gocache GOFLAGS= go test ./c8volt/... ./internal/... -race -count=1
GOCACHE=/tmp/c8volt-gocache GOFLAGS= go test ./cmd -race -count=1
git diff --check
```

All touched Go files were formatted with targeted `gofmt -w`. An early build-cache
sandbox denial and the first broad suite's loopback-bind denials were resolved through
normal permission reruns; the final broad run exited 0. These were test-owned HTTP
servers, never live Camunda targets. New regression checks exposed missing result
identity and nil/empty conversion differences during development; both were resolved.
A test assertion using pointer-only context comparison was corrected before the final pass.

Error/output review: resource reports preserve response count/order, Ok/status and
original errors; job preparation still rejects before API dispatch and normalizes through
ferrors. Approval errors keep exact prior text/class and empty rejected result payloads.
Returned partial backend evidence follows the unchanged view/logger/exit route. No new
print, logger or error emitter was introduced; command regression suites cover the existing
human, quiet/verbose, machine envelope, prompt and diagnostic behavior. No generated docs
refresh is needed because CLI metadata, flags, help and output contracts are unchanged.
Real-engine behavior was not tested; it is outside the implementation validation boundary.

All changes are unstaged on `370-facade-layering-refactor`. No commit, push, reset,
revert, discard or staging was performed.
