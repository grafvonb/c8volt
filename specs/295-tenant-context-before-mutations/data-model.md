# Data Model: Tenant Reporting Stages

## Existing Selection Context

Owner: `cmd/cmd_tenant_context.go`, using public `tenant.Context` from `c8volt/tenant/context.go`.

Fields remain unchanged: `Mode`, `Filter`, `ConfiguredTenantID`, `TargetTenantID`, `ResolvedTenantIDs`, `UnknownTargetCount`, `CrossTenant`, and `Warnings` (code/message). Discovery uses a named filter or no filter; explicit keys use the established not-applied filter. This feature does not use creation mode, but shared renderer changes must preserve it for other commands.

Command-local override provenance retains configured value, explicit flag presence/value, and all-tenants choice. Its existing representation is not added to public schemas. Meaningful overrides render only where discovery-filter semantics apply.

## Existing Tenant Evidence

Owner: internal domain `TenantEvidence`, mapped to public `process.TenantEvidence`.

Fields: `ResolvedTenantIDs`, `UnknownTargetCount`, and `Targets` containing target `Key` and `TenantID`. Use service-owned plan evidence as-is. Existing `opsTenantEvidenceSummary` deduplicates observations by nonempty target key when targets are present, otherwise uses aggregate evidence. Context normalization produces distinct sorted tenant IDs and existing warning codes.

Invariants:

- Missing metadata stays unknown; configured tenant never fills it in.
- Empty validated evidence means zero known affected tenants and zero unknown targets, not discovery failure.
- Negative unknown counts remain normalized by existing rules.
- Actual default-tenant evidence uses the established representation; an empty selection filter does not imply the default tenant.
- More than one distinct known tenant sets cross-tenant semantics; unknown metadata may coexist with that condition.

## Proposed Tenant Scope Progress Payload

Owner: `internal/domain/ops_progress.go`, mirrored in `c8volt/ops/progress_model.go`.

Add a `tenant_scope` event kind and optional `TenantScope` payload, with an evidence field typed to the corresponding existing internal/public `TenantEvidence`. The event is an additive member of the existing progress union; other payloads are absent for this kind. A nonnil payload with empty evidence explicitly indicates a successfully established empty scope.

Relationship: one service invocation publishes at most one validated scope notification, derived from its delete plan or repair frozen set. Interactive command execution can invoke the service twice; both events describe existing service work and the CLI prevents duplicate human output. This notification does not authorize mutation or replace existing plan validation.

Delivery invariants:

- Synchronous callback completion precedes the first mutation, including repair variable updates.
- Existing plan/impact validation must succeed before emission; partial failure evidence is not published as validated.
- Payload slices are copied at the facade boundary; callback consumers cannot modify service-owned plan evidence.
- The callback is optional and excluded from serialized request data as today.
- This event is not appended to final CLI results or audit report schemas.

## Proposed Human Emission State

Owner: command execution context only; never serialized.

Fields: `selectionRendered` and `affectedRendered` booleans, initialized false per execution. The existence of the staged ops state distinguishes the new path from unrelated legacy full-context rendering. The full attached tenant context remains independent and immutable to renderers.

| Transition | Condition | Output and state |
| --- | --- | --- |
| Initialized → selection rendered | Local validation complete; permitted durable channel | Emit override lines and selection label; set selection flag |
| Selection rendered → affected rendered | Validated tenant-scope event; permitted channel | Attach full context; emit known summary and unknown warning; set affected flag |
| Selection rendered → empty scope complete | Successful event with no targets | Attach full context; set affected flag without tenant/unknown lines |
| Any stage → protected-mode execution | Channel prohibits human tenant output | Attach available data, emit nothing, do not claim suppressed output was rendered |
| Affected rendered → confirmation/final result | Same command execution | Preserve full context, omit already emitted human stages |
| Selection rendered → discovery/validation failure | Scope unavailable | Preserve existing failure behavior; no validated affected claim |
| Completed → next command invocation | Execution initializes again | Fresh state; no cross-execution suppression |

No database, persistent state, migration, or public tenant-context schema changes are required.
