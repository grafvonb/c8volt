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
- `git diff --check`

## Do Not Repeat
- Do not add accumulator or renderer behavior to the domain/public model; T003/T006 own service evidence aggregation and T004/T007/T008 own command rendering/envelope plumbing.
- Do not render destructive cancel selector tenant context to stdout; the progress contract reserves stdout for command results and keeps compact progress on stderr.

## Current Handoff
- Continue Phase 5 / US3 at T024, adding facade conversion tests for PI plan tenant evidence and slice-copy isolation in `c8volt/process/client_test.go` and `c8volt/process/model_test.go`; then implement T028 public model/conversion wiring before moving to later US3 command surfaces.
