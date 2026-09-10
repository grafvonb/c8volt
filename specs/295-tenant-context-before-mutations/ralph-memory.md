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
- Ops tenant evidence prefers keyed target observations when present, ignores empty target keys, and deduplicates by first key observation; aggregate resolved IDs and unknown counts are used only when no targets are supplied, then normalized by the shared tenant-context rules.
- All six interactive ops workflows route pre-prompt fallback reporting through `printOpsTenantContextForCommand`, which derives the same mode-gated durable channel as early selection and progress callbacks; protected modes neither emit nor mark staged output as rendered.
- Repeated preview/execution tenant-scope events may refresh the complete attached evidence, while `selectionRendered` and `affectedRendered` suppress only duplicate human lines and final-view repetitions; explicit-key selection remains independent of tenant override provenance.
- Automation still selects the one-line renderer, so final tenant-context suppression must check `automationModeEnabled(cmd)` in addition to render mode and quiet state; structured JSON must retain serialized warning messages even though stderr has no tenant chatter.
- Declare output modes explicitly for ops workflows when inherited root flags would advertise an unsupported renderer; orphan purge is the only affected workflow with a real keys-only result path.
- Audit JSON and Markdown derive a normalized tenant-context snapshot from frozen report evidence; staged human render flags and later command-context replacement do not prune serialized IDs, unknown counts, cross-tenant state, or warning messages.
- All six real command audit paths preserve applicable tenant mode, filter, resolved IDs, unknown count, cross-tenant state, and warnings across dry-run, empty, blocked, and mutation-failure outcomes; local invalid input preserves an existing report without backend requests.
- README and all six command `Long` descriptions use one consistent operator contract: selection precedes discovery or explicit-key resolution, validated affected tenants precede confirmation and mutation, and auto-confirm skips only the question. `make docs-content` propagates this wording to the six CLI references and `docs/index.md`.
- The Phase 6 ownership audit found no forbidden command/facade dependencies or added retrieval: ordinary command files only initialize or invoke reporting, lifecycle state stays in tenant/progress support files, facade changes only copy/map the typed payload, and internal ops services only emit snapshots around existing validated plans and mutation gates.
- The complete quickstart validation selects nonempty service, facade, and command suites (95, 23, and 230 tests respectively); all targeted suites, the full race-enabled repository suite, and whitespace validation pass together.

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
- Continue Phase 6 with T038: review FR-001–FR-014 and SC-001–SC-007 against the contract coverage matrix, recording concrete test names and outcomes while leaving unresolved work unchecked.
