# Ralph Memory

Feature: 295-tenant-context-before-mutations
Started: 2026-09-10T15:15:43Z

## Codebase Patterns
- Tenant progress facts originate in `internal/domain` and internal ops services, cross the thin `c8volt/ops` facade by mechanical conversion, and are rendered/policy-gated in `cmd`.
- Human tenant lifecycle state belongs in focused command support/progress files; final formatting remains in `cmd/cmd_views_tenant_context.go`.
- Ops request callbacks use `c8volt/ops.ProgressEvent`; `fromDomainOpsTenantEvidence` and its inverse copy both resolved-ID and target slices at that boundary.

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
- Continue with T003 in Phase 2: add staged-renderer regressions before implementing T006; T007 remains the foundational validation checkpoint, so do not start US1 yet.
