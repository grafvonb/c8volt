# Facade and Adapter Contract: User-Task Assignment

**Feature**: [spec.md](../spec.md) | **Models**: [data-model.md](../data-model.md)

These are planned additions. Existing completion and variable interfaces retain their behavior.

## Public facade

Add to `c8volt/task.API` with normal repository naming/import aliases:

```go
AssignUserTasks(ctx context.Context, keys typex.Keys, assignee string, wantedWorkers int, opts ...foptions.FacadeOption) (UserTaskAssignmentResults, error)
UnassignUserTasks(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...foptions.FacadeOption) (UserTaskAssignmentResults, error)
```

Both methods are collection-first; a single key uses the same execution. Facades map inputs/options, delegate, map all returned items and convert errors through `ferrors.FromDomain`, including when partial results accompany an error. No scheduling, retries, state reads or confirmation in the facade. No new prepare/execute pair or generated/internal types in public signatures.

Add `foptions.WithUserTaskAssignmentOverride()` with default false and the corresponding field/mapping to internal call options. It is the only new assignment-override control; generic force and automation do not set it. In assignment, the adapter always emits explicit false or true. Unassignment rejects enabled override rather than silently ignoring it.

Reuse existing dry-run, fail-fast, worker selection, explicit-key tenant, detail-suppression and context options. Both operations reject no-wait. Zero wantedWorkers chooses the existing automatic bound; negative fails. The CLI rejects explicitly supplied zero separately.

All keys must be valid before any mutation; domain/facade key values are exact 16-digit strings rather than silently normalized identifiers. Empty selection and blank assignee fail. Snapshot and deduplicate keys in first-input order. Treat nonblank assignee as opaque. Validate the supplied service dependency as in existing coordinators. On validation failure return an initialized empty Items collection and the original classified error; do not invent per-key mutation evidence.

Facade dry-run returns preview/not_submitted for all unique valid keys without backend requests. Normal execution returns ordered per-key results for the whole validated selection, including not-attempted work. Caller cancellation, failure exits and submission uncertainty follow [data-model.md](../data-model.md).

## Internal service contract

Extend `internal/services/usertask.API` and each versioned API with:

```go
AssignUserTask(ctx context.Context, key string, assignee string, opts ...services.CallOption) (domain.UserTaskAssignmentResult, error)
UnassignUserTask(ctx context.Context, key string, opts ...services.CallOption) (domain.UserTaskAssignmentResult, error)
```

Add matching collection wrappers/coordinator in the version-neutral usertask package. The shared coordinator owns complete validation, snapshots, dry-run, bounds, fail-fast and ordered result aggregation. It calls the appropriate concrete per-key method without task reads. Options are evaluated before scheduling; the effective assignment override must not change between keys or attempts.

A request body and per-call option snapshot must not share mutable caller state across workers. Stop scheduling separately from parent-context cancellation; the parent context controls already submitted requests. Preserve started markers, final evidence and caller errors rather than treating zero pool slots as completed work.

## Versioned generated-client interfaces

For each of v88/v89/v810, extend `GenUserTaskClientCamunda` with the existing generated `AssignUserTaskWithResponse` and `UnassignUserTaskWithResponse` methods using that version's types. Add concrete adapter methods and update every necessary mock/test double in the same compiling unit. No new method is required on Tasklist clients.

| Action | Native operation | Request | Success |
| --- | --- | --- | --- |
| assign | POST `/user-tasks/{key}/assignment` | Nonnull `assignee`; explicit boolean `allowOverride`; optional custom action unset | 204 with valid metadata, no overriding call error |
| unassign | DELETE `/user-tasks/{key}/assignee` | No body | Same |

A generated response's existence is not proof of success. Preserve status/body/call error through common evidence handling. Auth, missing resource, conflict, malformed response, transport, timeout and cancellation errors retain normal classification; backend rejection for same-assignee/already-unassigned is not suppressed.

Use backend-authorized explicit keys and do not add tenant-filter queries or Tasklist resolution. Supported-version factories and generic unsupported-version handling remain unchanged.

## Common submission helper

Reuse `RetryCamundaMutation` and `httpc.HttpStatusErr` through narrow private native user-task evidence handling in `internal/services/common`. Keep completion's public/internal result types and externally observable behavior behind its existing wrapper. Assignment wrappers produce only assignment-family results and correct operation labels.

Preserve the existing retry policy: temporary/timeout transport errors, 429 and resource-exhausted 503 may retry; generic 504 and 409 do not gain retries. Original transport error goes to the retry helper. Keep last response and ambiguity outside the callback return value to survive backoff cancellation. Later rejection cannot resolve earlier ambiguity, while later documented success may establish accepted submission. No state reconciliation or mutation compensation.

## Compatibility and isolated checks

Extending `task.API` requires command stubs such as `cmd/process_api_stub_test.go` to implement the new methods. Internal contract extensions affect facade and service read/search/variable/completion doubles; generated-client extensions affect each adapter's mocks. Keep unrelated fixtures fail-closed on unexpected new calls.

Test default false and explicit true option mapping, opaque identity fidelity, no-wait/override rejection, ordered partial conversion and original class preservation at the facade. Test all generated request mappings with test-owned HTTP servers, including zero extra reads/mutations. Keep completion submission/response regressions after helper extraction. No runtime claims are made by this contract document.
