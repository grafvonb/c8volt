# Research: Tenant Context Before Ops Mutations

## Scope and resolved questions

Research used the current repository and feature specification. No new technology or external dependency is needed. Questions resolved: where validated tenant evidence becomes available, how to surface it during auto-confirm without rediscovery, how to deduplicate two reporting stages, and how to protect output and audit contracts.

## Decision 1: Publish validated tenant evidence through existing progress callbacks

**Decision**: Add a typed `tenant_scope` event to the existing internal and public ops progress envelopes. Its payload contains an owned snapshot of existing `TenantEvidence`. Deliver it synchronously once each service invocation has established its complete applicable plan, before its first mutation. Map it mechanically through the facade.

**Rationale**: Auto-confirm invokes discovery and mutation inside a single service call. The CLI receives the final result too late. All affected requests already carry `Progress`; `c8volt/ops/convert.go` already maps these callbacks and event envelopes. A dedicated event distinguishes validated evidence from discovery preflight, page counters, and execution progress. It introduces no requests or workflow decisions.

**Alternatives considered**: Always performing a separate dry-run adds discovery and can change execution behavior. Interpreting discovery page events exposes incomplete scope. Reusing the repair frozen-scope event is too late because variable updates precede it. A separate callback on every request duplicates the existing notification channel. Adding tenant wording to services violates layer ownership.

### Verified integration points

| Workflow | Evidence | Service file and boundary | CLI selection semantics |
| --- | --- | --- | --- |
| All process definitions purge | `DeletePlan.TenantEvidence` | `internal/services/ops/all_process_definitions_purge.go`: after delete-plan construction and applicable validation, before dry-run return or mutation | Explicit keys when `Selection.Key` is present; discovery otherwise |
| Orphan process instances purge | `DeletionPlan.TenantEvidence` | `internal/services/ops/orphan_purge.go`: after dependency expansion validation; before deletion, retaining dry-run and force gates | Discovery |
| Incident process instances purge | `DeletePlan.TenantEvidence` | `internal/services/ops/incident_purge.go`: after delete-plan validation and applicable blockers; before dry-run/no-work return or mutation | Explicit keys when `Selection.Keys` is nonempty; discovery otherwise |
| Retention policy | `DeletePlan.TenantEvidence` | `internal/services/ops/retention_policy.go`: after retention delete-plan validation; before deletion, retaining dry-run and force gates | Discovery |
| Incident repair | `FrozenSet.TenantEvidence` | `internal/services/ops/repair.go`: `repairExplicitIncidents` and `repairFilteredIncidents`, after frozen-set construction, before dry-run return or `executeRepairVariableUpdates` | Existing keyed/search discovery mode |
| Process-instance repair | `FrozenSet.TenantEvidence` | `internal/services/ops/repair.go`: `finishProcessInstanceIncidentRepair`, after complete frozen-set discovery, before dry-run return or `executeRepairVariableUpdates` | Existing keyed/search discovery mode |

Keep existing validation order intact. Dry-run plans may legitimately describe work requiring force; they can publish successfully constructed preview evidence without authorizing execution. On actual execution, preserve blockers and publish only at a successful boundary before mutations. An empty successful plan may publish empty evidence; it must not generate an unknown warning. A discovery or plan error must never publish partial evidence as validated. Do not move a force gate or introduce a new one merely to centralize emission.

## Decision 2: Split human reporting into selection and affected-scope stages

**Decision**: Track command-local selection and affected-stage emission separately. Reuse existing context construction, override provenance, line wording, and warning severity. Partition line construction by semantic responsibility, rather than matching rendered text. Preserve the existing full-render path for other commands.

**Rationale**: `cmd/ops_tenant_context.go:printOpsTenantContext` and `cmd/cmd_views_tenant_context.go:renderTenantContext` currently share a single `tenantContextHumanRendered` boolean. Setting it after selection would suppress the later scope. `cmd/cmd_tenant_context.go:tenantContextHumanLines` already builds override lines, primary scope, distinct tenant summaries, and unknown warnings.

**Alternatives considered**: Clearing the boolean before each print repeats selection and overrides. Stripping evidence from attached context damages final structured output and audit reports. Global rendering state leaks across command executions. A generic output deduplication framework is unnecessary for two fixed stages.

## Decision 3: Keep reporting policy in the CLI

**Decision**: Emit selection after local validation and request-mode resolution, before activity/progress wrappers and any service invocation. Route both stages through the mode-derived durable stderr channel. Compose the tenant-event handler with existing progress callbacks without replacing their behavior.

**Rationale**: `opsProgressChannelForMode` already protects JSON, automation, keys-only, and quiet execution. Current interactive code passes a hardcoded human channel, which should be replaced for the affected flows. Repair progress ownership is already in `cmd/ops_repair_progress.go`; purge/retention progress is in `cmd/ops_processinstance_purge_progress.go` and APD progress in its dedicated progress file.

**Alternatives considered**: Printing on stdout contaminates machine consumption. Writing selection from a service event can occur after CLI discovery activity starts. Printing only at confirmation retains the auto-confirm bug.

## Decision 4: Preserve reports and existing frozen-scope reuse

**Decision**: Keep `attachOps*ResultTenantContext`, cloned report contexts, legacy tenant-ID rules, and audit writers authoritative for full final evidence. Human emission state never changes serialized fields. Interactive planning and execution retain their current request/frozen-key reuse, and repeated events from those invocations do not repeat human summaries for the same command.

**Rationale**: Existing report writers already retain separate tenant evidence, and `writeMarkdownTenantContext` uses common wording. APD and other interactive flows reuse discovered candidates but may repeat impact validation; the requirement is zero additional passes compared with that baseline, not a redesign of existing validation.

**Alternatives considered**: Removing tenant fields from final results to suppress output would lose audit data. Rebuilding mutation plans in commands violates layering. Changing frozen-scope mechanics belongs to another feature.

## Decision 5: Prove event ordering at mutation boundaries

**Decision**: Add command tests for all six workflows, service tests for event-before-mutation, and facade tests for mapping and snapshot isolation. Use existing subprocess, fake server, activity sink, safe collection, and counter helpers. Assert output while the first mutation is intercepted, not only the final concatenated text.

**Rationale**: Final text alone cannot prove when it appeared. Variable updates are the earliest repair mutation when enabled. Existing tests already cover tenant semantics, quiet failures, automation, JSON, and audit serialization, providing regression baselines.

**Alternatives considered**: Renderer-only tests miss command wiring. Live destructive cluster tests are unnecessary for this timing correction; deterministic fake backends exercise actual command entry points without requiring a disposable cluster.
