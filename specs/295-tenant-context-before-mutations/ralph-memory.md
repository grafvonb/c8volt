# Ralph Memory

Feature: 295-tenant-context-before-mutations
Started: 2026-09-10T15:15:43Z

## Codebase Patterns
- Tenant progress facts originate in `internal/domain` and internal ops services, cross the thin `c8volt/ops` facade by mechanical conversion, and are rendered/policy-gated in `cmd`.
- Human tenant lifecycle state belongs in focused command support/progress files; final formatting remains in `cmd/cmd_views_tenant_context.go`.
- Ops request callbacks use `c8volt/ops.ProgressEvent`; `fromDomainOpsTenantEvidence` and its inverse copy both resolved-ID and target slices at that boundary.
- Staged ops rendering is opt-in per command execution through `initializeTenantContextHumanRenderStages`; commands without that state retain the legacy full-context render-once path.
- Selection and affected lines are partitioned semantically by `tenantContextSelectionHumanLines` and `tenantContextAffectedHumanLines`; permitted empty affected scope marks completion without output.
- `emitOpsTenantScope` owns synchronous snapshot delivery for validated service plans; emit empty evidence on successful empty scopes, emit after execution blockers, and never emit after planning failures.
- Orphan purge and retention policy publish after their existing destructive force blockers but before dry-run/no-work returns or the first root deletion; their expanded plan evidence requires no additional ancestry, descendant, or discovery calls.
- Repair incident paths emit immediately after freezing explicit/search incidents; process-instance paths emit once in `finishProcessInstanceIncidentRepair`. This places the callback before empty/dry-run returns and before variable updates, while discovery failures emit nothing.
- A command opts into early reporting by initializing a fresh staged base context after local/report validation; tenant-scope events then merge evidence into that attached base and emit the affected stage before existing generic progress handling.
- Real-command timing tests use a synchronized output writer plus a reverse-proxy observer to snapshot output at the first backend request and first mutation without racing command output.
- Repair commands derive staged selection directly from the resolved `RepairDiscoveryMode`: search uses discovery semantics, while keyed and stdin modes use explicit-key/not-applied semantics; initialize before configuring progress or starting activity.
- Interactive command tests can snapshot durable stderr at the exact prompt boundary by wrapping the helper subprocess writer with `io.MultiWriter`; accepted planning/execution naturally exercises duplicate tenant-scope callbacks, while combined stdout/stderr occurrence counts also cover final-render suppression.
- Accepted interactive repair executes a dry-run preflight followed by a frozen-key execution: top-level filtered discovery remains single-pass, while explicit target and incident lookups repeat as part of the established two-phase repair workflow; declined runs stop after preflight with no mutation requests.

## Decisions
- Preserve existing discovery, frozen-scope, mutation, output-mode, and audit behavior; the feature adds a synchronous typed progress event and two command-local rendering stages.

## Gotchas
- Protected output must continue through `opsProgressChannelForMode`; tenant context data attachment is independent from whether human lines were emitted.
- Interactive workflows may invoke a service twice, so command-local stage flags must suppress duplicate human output without removing report evidence.

## Reusable Commands
- `go test ./internal/services/ops -run 'Test.*(Purge|Retention|Repair|Tenant|Progress)' -count=1`
- `go test ./c8volt/ops -run 'Test.*(Progress|Tenant|Purge|Retention|Repair)' -count=1`
- `go test ./cmd -run 'Test.*(TenantContext|OpsPurgeAllProcessDefinitions|OpsPurgeOrphanProcessInstances|OpsPurgeProcessInstancesWithIncidents|OpsExecuteRetentionPolicy|OpsRepairIncident|OpsRepairProcessInstance|OpsAuditReport|MarkdownTenantContext)' -count=1`
- `make test`

## Do Not Repeat
- Do not add discovery or metadata retrieval for reporting, infer unknown tenants from configuration, or strip attached evidence to deduplicate human output.

## Current Handoff
- Continue US2 with T024: expand tenant-context renderer and policy coverage across evidence combinations and override transitions; remain within US2 for the next iteration.
