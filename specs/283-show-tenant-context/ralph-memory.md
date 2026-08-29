# Ralph Memory

Feature: 283-show-tenant-context
Started: 2026-08-29T12:20:03Z

## Codebase Patterns
- Progress tracking for this feature lives in `specs/283-show-tenant-context/progress.md` and now contains artifact links, work-unit status, validation results, and codebase-pattern sections before iteration entries.
- Tenant context model ownership is split as planned: canonical validation and warning derivation live in `internal/domain/tenant_context.go`; the public mirror lives in `c8volt/tenant/context.go`; mechanical conversion lives in `c8volt/tenant/convert.go`.
- Conversion tests in `c8volt/tenant` use package-internal unexported converter access, matching the existing tenant facade test style.
- Service-owned tenant evidence aggregation lives in `internal/services/common/tenant_context.go`; callers add one observation per affected target key, duplicate keys keep the first observation, empty tenant IDs count as unknown, snapshots sort known tenants, and snapshot merges preserve per-key dedupe.
- CLI tenant context plumbing lives in `cmd/cmd_tenant_context.go`; it uses the public `c8volt/tenant.Context` shape, stores a cloned value on Cobra context, and falls back to `context.Background()` for direct test commands with nil context.
- Human tenant-context rendering lives in `cmd/cmd_views_tenant_context.go`; it suppresses JSON, keys-only, and quiet modes, preserves the exact `WARNING:` prefix for tenant warnings, and skips the duplicate unfiltered warning because the semantic line already carries that wording.
- Shared full-contract JSON envelopes now have optional root `tenantContext` between `command` and `payload`; `renderResultEnvelope` automatically fills it from the attached command context while leaving unattached commands on the previous shape.
- Selector-based cancel planning attaches discovery tenant context in `cmd/cancel_processinstance_selector.go`; dry-run previews use normal human rendering, while destructive pre-confirmation context writes to stderr to preserve existing stdout-clean progress contracts.
- `tenantContextPrimaryHumanLine` centralizes the first human tenant-context line so workflows can choose stdout/stderr routing without changing the exact contract wording.
- Shared process-instance dry-run summaries now render an attached discovery tenant context before human plan output, include it through the shared JSON envelope, suppress it in quiet mode, and keep keys-only summaries to affected keys only.
- Process-instance mutation progress can render an attached discovery context before verbose preflight scope on stderr; selector-specific renderers mark the context as already emitted to avoid duplicate human lines.
- Deploy and embedded deploy attach creation-mode context after local file/fixture validation and before `DeployProcessDefinition`; human mode renders `Create in tenant: ...` before the POST, while JSON envelopes receive the attached context through `renderCommandResult`.
- Deployment result evidence now reuses `processDefinitionDeploymentTenantIDs` in `cmd/cmd_views_deploy.go` to populate resolved tenant IDs from returned deployment resources without changing deployment payloads or keys-only streams.
- PI dry-run domain plans now carry `domain.TenantEvidence`; `internal/services/processinstance.DryRunCancelOrDeletePlan` fills it from already-loaded traversal chains, preferring any known tenant metadata for each affected key and counting legacy key-only targets as unknown.
- The public process facade mirrors dry-run plan evidence with `process.TenantEvidence` on `DryRunPIKeyExpansion`; `fromDomainTenantEvidence` copies the resolved tenant ID slice and keeps command-owned operation semantics separate from service evidence.
- Direct PI cancel/delete/resolve plans attach explicit-key tenant context only when the planning options carry `IgnoreTenant`, then merge resolved plan evidence with command-configured tenant semantics before dry-run, confirmation, or JSON result rendering.
- PI variable update previews derive actual tenant evidence from already-loaded process-scope variables, count one unknown target per explicit key when no usable tenant metadata is visible, and keep the evidence in unexported preview fields so JSON payload shape stays stable.
- Process-definition delete plans expose `TenantEvidence()` from public resource plans by aggregating known process-definition plan-item tenant IDs plus nested PI cancellation evidence; PD-only missing tenant metadata is left for the later cross/unknown warning story rather than emitting a new no-state-check warning now.
- Resource facade conversion for nested PD cancellation plans must copy `TenantEvidence`; otherwise command-level PD context loses PI tenant evidence during domain-to-public mapping.
- Job update planning attaches explicit-key tenant context from the already-loaded current job before any dry-run, interactive plan, auto-confirmed mutation, or JSON result rendering; no job search or mutation request receives a tenant filter for this reporting.
- `renderAttachedTenantContext` now lives in `cmd/cmd_views_tenant_context.go` because it is shared by PI, PD, and job views; command tests that reuse global Cobra commands should reset command contexts to avoid stale tenant rendered markers.
- PI tenant evidence now carries per-target observations (`TenantEvidenceTarget`) so `PlanProcessInstanceMutationPages` can merge tenant IDs and unknown counts by unique affected key across selected pages; public process facade conversion copies the target slice.
- Search-derived PI dry-run commands defer first human tenant rendering until aggregate page evidence is attached, allowing summary rendering to show sorted resource tenants plus coexisting cross-tenant and unknown warnings.
- Process-definition preview plans now aggregate tenant evidence from process-definition items and nested PI cancellation plans in `internal/services/processdefinition/delete.go`; public `resource.DeleteProcessDefinitionPlan.TenantEvidence()` mirrors the same unknown-PD-item behavior for command rendering.
- PD delete command tenant context is attached after the frozen impact plan for both explicit keys and selector/search discovery; `renderDeleteProcessDefinitionImpact` emits the attached sorted tenant summary and coexisting cross/unknown warnings before compact impact/force lines.
- Destructive PI search confirmations now attach resolved page or aggregate tenant evidence before rendering the tenant block to stderr, so compact confirmations show sorted resource tenants plus cross/unknown warnings without contaminating stdout.
- Delete process-instance search carries service aggregate tenant evidence in the command result and preserves it when merging page previews into the final frozen delete plan.
- Resource facade conversion for PD delete plans copies nested PI `TenantEvidence.Targets`; public aggregation can now dedupe nested known and unknown targets exactly instead of falling back to summary counts.
- Configuration diagnostics now attach configuration-mode tenant context: `config show` embeds it in sanitized YAML via `config.ToSanitizedYAMLWithTenantContext`, `config validate` renders the semantic line before the validation outcome, and `config test-connection` renders it in human mode while including the same object at the raw JSON root.
- Keep `config` package YAML tests free of imports from `c8volt/tenant`; use plain serialized maps there to avoid a test-only import cycle through the public tenant facade.

## Decisions
- Phase 1 setup was treated as the first work unit because T001 was the first incomplete task and Phase 2 depends on it.
- Iteration 2 paired T002 with T005 so the new TDD tests were validated into a passing package state before commit.
- Iteration 3 paired T003 with T006 because the accumulator tests can only be completed when the service helper passes them.
- Iteration 4 completed the remaining Phase 2 foundation by pairing T004 renderer/envelope tests with T007/T008 CLI implementation and T009 focused validation.
- Iteration 5 paired T010 with T013 so cancel selector tests and implementation were validated together without committing failing tests.
- Iteration 6 paired T011 with T014 so delete selector tests and implementation were validated together without committing failing tests.
- Iteration 7 completed the remaining US1 work by pairing T012 with T015 and T016, validating shared dry-run/progress output modes plus cancel/delete selector targets.
- Iteration 8 completed the deploy half of US2 by pairing T017 with T019; T021 remains open because run structured results are not implemented yet.
- Iteration 9 completed the run half of US2 by pairing T018 with T020-T022; deploy and run creation contexts are now both validated before US3 starts.
- Iteration 10 paired T023 with T027 so service tests for PI dry-run tenant evidence were committed only after the domain/service implementation passed.
- Iteration 11 paired T024 with T028 so PI facade tests for plan evidence and slice isolation were committed only after public model/conversion wiring passed.
- Iteration 12 paired T025 with T029 and T030 so PI command explicit-key tests, tenant-context attachment, and variable-update evidence rendering were validated together.
- Iteration 13 completed the remaining US3 work by pairing T026 with T031-T033; process-definition delete and job update now report explicit-key tenant context using already-loaded plan/current-job data.
- Iteration 14 completed T034 only; implementation support was added for PI page-result evidence aggregation and dry-run summary attachment, but T036/T037 remain open for full confirmation-boundary behavior.
- Iteration 15 completed T035 only; supporting PD preview aggregation and command attachment were added so the new PD tests pass, but T036-T039 remain open for the rest of US4 validation and implementation.
- Iteration 16 completed the remaining US4 tasks T036-T039; PI destructive confirmations and PD facade conversion now preserve and render cross-tenant plus unknown metadata warnings from frozen evidence.
- Iteration 17 paired T040 with T044 so configuration diagnostic tests were committed only after sanitized YAML, validate, and test-connection configuration context passed.

## Gotchas
- Follow `specs/ralph-implementation-rules.md` in addition to this feature's artifacts; it is binding for Ralph iterations.
- Context warnings are derived in stable order by the domain constructor and left nil when absent so `omitempty` omits them from structured output.
- `TenantEvidenceSnapshot` carries unexported per-target state for merge dedupe; merge snapshots produced by `TenantEvidenceAccumulator.Snapshot()` instead of constructing snapshots manually.
- The existing generic `renderHumanWarningLine` strips a leading `WARNING:` for older messages; tenant warning rendering uses a tenant-specific wrapper around `renderHumanLogLine` so the feature's exact human contract remains visible.
- Destructive cancel search progress tests expect stdout to stay empty; route pre-confirmation tenant context to `cmd.ErrOrStderr()` on non-dry-run selector cancellation.
- Destructive delete search also freezes all page-level plans before one aggregate confirmation; render the discovery tenant context after freezing the aggregate scope and before that confirmation, routed to stderr for non-dry-run output.
- `resetProcessInstanceCommandGlobals()` does not reset `flagQuiet`; tests that set quiet must restore it explicitly or use a helper that does.
- Root command instances retain Cobra context values across in-process tests, including the tenant-context rendered marker; deploy/embed tests clear command contexts before asserting pre-call rendering order.
- Run command tests that assert per-execution tenant-context rendering must clear retained Cobra contexts before each case; otherwise the human-rendered marker can suppress later subtests.
- In command unit tests that install `config.Config` on context, use `ToContextWithLogWriter(..., buf)` when asserting human output; otherwise `renderHumanLine` routes through the logger and the Cobra output buffer remains empty.
- Process-definition direct-key no-state-check plans may contain only a key and no tenant metadata; do not count that as an unknown-target warning until US4 expands the PD warning contract, because existing stdin/no-state-check output asserts no warning chatter.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `git diff --check -- specs/283-show-tenant-context/progress.md`
- `go test ./internal/domain ./c8volt/tenant -run 'Test.*TenantContext|Test.*Context' -count=1`
- `go test ./internal/domain ./c8volt/tenant -count=1`
- `go test ./internal/services/common -run 'TestTenantEvidenceAccumulator' -count=1`
- `go test ./internal/domain ./c8volt/tenant ./internal/services/common -count=1`
- `go test ./cmd -run 'Test(NewTenantContexts|WithTenantContextEvidence|RenderTenantContext|RenderSucceededResult_.*TenantContext)' -count=1`
- `go test ./internal/domain ./c8volt/tenant ./internal/services/common ./cmd -run 'Test.*TenantContext|Test.*Context|TestTenantEvidenceAccumulator|Test.*CommandContract|TestRenderSucceededResult_.*TenantContext' -count=1`
- `go test ./cmd -run 'TestCancelProcessInstance(DryRun_SearchTenantContextPrecedesPreview|Search_TenantContextPrecedesConfirmation)' -count=1`
- `go test ./cmd -run 'TestDeleteProcessInstance(DryRun_SearchTenantContextPrecedesPreview|Search_TenantContextPrecedesConfirmation)' -count=1`
- `go test ./cmd -run 'TestDeleteProcessInstance' -count=1`
- `go test ./cmd -run 'Test(Cancel|Delete)ProcessInstance' -count=1`
- `go test ./cmd -run 'Test.*(DryRun.*Tenant|MutationProgress|ProcessInstanceDryRunSummary)' -count=1`
- `go test ./cmd -run 'TestDeployProcessDefinitionCommand_(CreationContextPrecedesDeploymentRequest|JSONEnvelopeIncludesCreationContext|QuietSuppressesCreationContext)|TestListProcessDefinitionDeploymentsView_(JSONEnvelopeIncludesAttachedTenantContext|KeysOnlySuppressesTenantContext)|TestEmbedDeployCommand_CreationContextPrecedesDeploymentRequest' -count=1`
- `go test ./cmd -run 'Test.*(Deploy|Embed)' -count=1`
- `go test ./cmd -run 'TestRunProcessInstanceCommand_(CreationContextPrecedesCreateRequest|JSONEnvelopeIncludesCreationContext|ProtectedModesSuppressCreationContext)|TestProcessInstanceTenantIDs_CollectsCreatedInstanceTenantEvidence' -count=1`
- `go test ./cmd -run 'Test.*(Deploy|Embed|Run.*ProcessInstance)' -count=1`
- `go test ./cmd -run 'TestCancelProcessInstance' -count=1`
- `go test ./cmd -count=1`
- `go test ./internal/services/processinstance -run 'TestDryRunCancelOrDeletePlan|TestPlanProcessInstanceMutationPages' -count=1`
- `go test ./internal/domain ./internal/services/common ./internal/services/processinstance ./c8volt/process -count=1`
- `go test ./internal/services/processinstance/... -count=1`
- `go test ./internal/domain ./c8volt/tenant ./internal/services/common ./cmd -count=1`
- `go test ./c8volt/process -run 'TestClient_DryRunCancelOrDeletePlan_(ReturnsStructuredExpansion|MapsTenantEvidenceCopy)|TestDryRunPIKeyExpansionConversionCopiesTenantEvidence|TestClient_PlanProcessInstanceMutationPages_DelegatesSearchAndExpansion' -count=1`
- `go test ./c8volt/process -count=1`
- `go test ./internal/domain ./internal/services/common ./internal/services/processinstance ./c8volt/process -count=1`
- `go test ./cmd -run 'Test(CancelProcessInstanceDryRun_ExplicitKeyRendersActualTenantMismatch|DeleteProcessInstanceDryRun_ExplicitKeyRendersUnknownTenantEvidence|ResolveProcessInstancesWithPlan_ExplicitKeyRendersActualTenant|UpdateProcessInstanceVariableDryRun_ExplicitKeyRendersVariableTenant)' -count=1`
- `go test ./cmd -run 'Test.*(Cancel|Delete|Resolve|Update).*ProcessInstance|TestUpdatePICommand|TestUpdateProcessInstanceVariable' -count=1`
- `go test ./cmd -run 'Test(DeleteProcessDefinitionCommand_KeyTenantMismatchUsesAdminScope|UpdateJobCommand_(RetriesDryRunRendersExplicitKeyTenantContext|JSONDryRunIncludesExplicitKeyTenantContext|RetriesConfirmedRendersExplicitKeyTenantContext|TechnicalFailureDryRunRendersExplicitKeyTenantContext))' -count=1`
- `go test ./c8volt/resource -run 'TestDeleteProcessDefinitionPlan_TenantEvidenceAggregatesPlanItemsAndCancellation|TestClient_PreviewDeleteProcessDefinitionsMapsParentElementInstanceKey' -count=1`
- `go test ./cmd -run 'Test.*(Cancel|Delete|Resolve|Update).*ProcessInstance|TestUpdatePICommand|TestUpdateProcessInstanceVariable|Test.*Delete.*ProcessDefinition|Test.*UpdateJob' -count=1`
- `go test ./internal/services/processinstance/... ./internal/services/processdefinition/... -run 'Test.*(Tenant|DryRun|Plan|DeleteProcessDefinition)' -count=1`
- `go test ./c8volt/process ./c8volt/resource ./c8volt/job -run 'Test.*(Tenant|DryRun|Plan|Preview|UpdateJob|DeleteProcessDefinition)' -count=1`
- `go test ./internal/services/processinstance ./cmd -run 'TestPlanProcessInstanceMutationPages_MergesTenantEvidenceAcrossPages|TestCancelProcessInstanceSearchDryRun_RendersMergedTenantWarnings' -count=1`
- `go test ./internal/services/processinstance/... -run 'Test.*(Tenant|DryRun|Plan|MutationPages)' -count=1`
- `go test ./c8volt/process -run 'Test.*(Tenant|DryRun|Plan|MutationPages)' -count=1`
- `go test ./cmd -run 'Test.*(Cancel|Delete).*ProcessInstance|Test.*DryRun.*Tenant|Test.*MutationProgress|TestProcessInstanceDryRunSummary' -count=1`
- `go test ./internal/domain ./c8volt/tenant ./internal/services/common ./internal/services/processinstance ./c8volt/process ./cmd -count=1`
- `git diff --check`
- `go test ./cmd ./config -run 'TestConfig.*(TenantContext|ToSanitizedYAML|Validate|TestConnection)|TestResolveEffectiveConfig_.*Tenant' -count=1`
- `go test ./cmd ./config -count=1`

## Do Not Repeat
- Do not add accumulator or renderer behavior to the domain/public model; T003/T006 own service evidence aggregation and T004/T007/T008 own command rendering/envelope plumbing.
- Do not render destructive cancel selector tenant context to stdout; the progress contract reserves stdout for command results and keeps compact progress on stderr.

## Current Handoff
- Continue Phase 7 / US5 at T041 by adding failing internal/public ops model and conversion tests for the common nested tenant-context object before implementing T045.
