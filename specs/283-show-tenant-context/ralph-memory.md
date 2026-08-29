# Ralph Memory

Feature: 283-show-tenant-context
Started: 2026-08-29T12:20:03Z

## Codebase Patterns
- Progress tracking for this feature lives in `specs/283-show-tenant-context/progress.md` and now contains artifact links, work-unit status, validation results, and codebase-pattern sections before iteration entries.
- Tenant context model ownership is split as planned: canonical validation and warning derivation live in `internal/domain/tenant_context.go`; the public mirror lives in `c8volt/tenant/context.go`; mechanical conversion lives in `c8volt/tenant/convert.go`.
- Conversion tests in `c8volt/tenant` use package-internal unexported converter access, matching the existing tenant facade test style.
- Service-owned tenant evidence aggregation lives in `internal/services/common/tenant_context.go`; callers add one observation per affected target key, duplicate keys keep the first observation, empty tenant IDs count as unknown, snapshots sort known tenants, and snapshot merges preserve per-key dedupe.

## Decisions
- Phase 1 setup was treated as the first work unit because T001 was the first incomplete task and Phase 2 depends on it.
- Iteration 2 paired T002 with T005 so the new TDD tests were validated into a passing package state before commit.
- Iteration 3 paired T003 with T006 because the accumulator tests can only be completed when the service helper passes them.

## Gotchas
- Follow `specs/ralph-implementation-rules.md` in addition to this feature's artifacts; it is binding for Ralph iterations.
- Context warnings are derived in stable order by the domain constructor and left nil when absent so `omitempty` omits them from structured output.
- `TenantEvidenceSnapshot` carries unexported per-target state for merge dedupe; merge snapshots produced by `TenantEvidenceAccumulator.Snapshot()` instead of constructing snapshots manually.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `git diff --check -- specs/283-show-tenant-context/progress.md`
- `go test ./internal/domain ./c8volt/tenant -run 'Test.*TenantContext|Test.*Context' -count=1`
- `go test ./internal/domain ./c8volt/tenant -count=1`
- `go test ./internal/services/common -run 'TestTenantEvidenceAccumulator' -count=1`
- `go test ./internal/domain ./c8volt/tenant ./internal/services/common -count=1`

## Do Not Repeat
- Do not add accumulator or renderer behavior to the domain/public model; T003/T006 own service evidence aggregation and T004/T007/T008 own command rendering/envelope plumbing.

## Current Handoff
- Next iteration should continue Phase 2 at foundational renderer/envelope test task T004 in `cmd/cmd_views_tenant_context_test.go` and `cmd/command_contract_test.go`; T007/T008 remain the matching implementation tasks.
