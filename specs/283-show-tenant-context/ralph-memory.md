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

## Decisions
- Phase 1 setup was treated as the first work unit because T001 was the first incomplete task and Phase 2 depends on it.
- Iteration 2 paired T002 with T005 so the new TDD tests were validated into a passing package state before commit.
- Iteration 3 paired T003 with T006 because the accumulator tests can only be completed when the service helper passes them.
- Iteration 4 completed the remaining Phase 2 foundation by pairing T004 renderer/envelope tests with T007/T008 CLI implementation and T009 focused validation.
- Iteration 5 paired T010 with T013 so cancel selector tests and implementation were validated together without committing failing tests.

## Gotchas
- Follow `specs/ralph-implementation-rules.md` in addition to this feature's artifacts; it is binding for Ralph iterations.
- Context warnings are derived in stable order by the domain constructor and left nil when absent so `omitempty` omits them from structured output.
- `TenantEvidenceSnapshot` carries unexported per-target state for merge dedupe; merge snapshots produced by `TenantEvidenceAccumulator.Snapshot()` instead of constructing snapshots manually.
- The existing generic `renderHumanWarningLine` strips a leading `WARNING:` for older messages; tenant warning rendering uses a tenant-specific wrapper around `renderHumanLogLine` so the feature's exact human contract remains visible.
- Destructive cancel search progress tests expect stdout to stay empty; route pre-confirmation tenant context to `cmd.ErrOrStderr()` on non-dry-run selector cancellation.

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
- `go test ./cmd -run 'TestCancelProcessInstance' -count=1`
- `go test ./cmd -count=1`
- `go test ./internal/domain ./c8volt/tenant ./internal/services/common ./cmd -count=1`
- `git diff --check`

## Do Not Repeat
- Do not add accumulator or renderer behavior to the domain/public model; T003/T006 own service evidence aggregation and T004/T007/T008 own command rendering/envelope plumbing.
- Do not render destructive cancel selector tenant context to stdout; the progress contract reserves stdout for command results and keeps compact progress on stderr.

## Current Handoff
- Next iteration should continue Phase 3 / US1 at T011, adding named/empty tenant selector and frozen-scope confirmation tests for delete workflows in `cmd/delete_processinstance_selector_test.go` and `cmd/delete_processinstance_test.go`; T012 and T014-T016 remain open in the same story.
