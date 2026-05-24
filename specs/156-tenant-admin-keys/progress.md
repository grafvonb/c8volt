# Progress: Tenant Scope For Discovery And Explicit Admin Keys

## Implementation Context

- Every Ralph iteration MUST read and apply `specs/ralph-implementation-rules.md` before selecting or implementing a work unit.
- Ralph launch instructions MUST include `--implementation-context specs/ralph-implementation-rules.md`.
- Commit subjects for this feature MUST use Conventional Commits format and append `#156` as the final token.

## Codebase Patterns

- Commands own Cobra flags, local validation, explicit search-vs-keyed mode decisions, and output dispatch.
- Direct keys and stdin keys are parsed through existing command helpers such as `readKeysIfDash`, `mergeAndValidateKeys`, and `validateKeys`.
- Public facade options flow through `c8volt/foptions` into `internal/services.CallOption`; do not pass internal options directly from commands.
- Version-specific Camunda behavior belongs in internal service packages under v87, v88, and v89, with unsupported versions returning explicit errors.
- Generated CLI docs must be refreshed with `make docs-content` after command help or metadata changes.
- Process-instance command paths classify explicit operator input before search: `readKeysIfDash` plus `mergeAndValidateKeys(...).Unique()` select keyed/stdin mode, and search mode is only used when no keys are present and sufficient filters exist.
- Search-derived `cancel pi` and `delete pi` freeze keys from `SearchProcessInstancesPage` results before calling the shared plan/dry-run helpers, so later mutation planning starts from the discovered page keys.
- v88/v89 process-instance search services inject the configured tenant into search filters, while direct `GetProcessInstance` calls use keyed backend lookup endpoints without adding a tenant request filter.
- Process-definition search supports tenant bypass through `foptions.WithIgnoreTenant()` mapped to `services.WithIgnoreTenant()`, while direct process-definition XML/key and resource ID lookups call keyed/id backend endpoints.

## Work Log

- 2026-05-24: Created issue-backed Speckit feature for GitHub issue #156.
- 2026-05-24: Clarification gate completed with no critical formal questions.
- 2026-05-24: Architecture grounding reused existing architecture memory; no architecture refresh needed.

---
## Iteration 1 - 2026-05-24 21:52 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**: 
- [x] T001: Read `specs/ralph-implementation-rules.md` and verify no conflict with `specs/156-tenant-admin-keys/spec.md`
- [x] T002: Review explicit key/stdin command mode handling in `cmd/get_processinstance*.go`, `cmd/walk_processinstance.go`, `cmd/expect_processinstance.go`, `cmd/cancel_processinstance.go`, and `cmd/delete_processinstance.go`
- [x] T003: Review process-definition and resource direct-ID paths in `cmd/get_processdefinition.go`, `cmd/delete_processdefinition.go`, `cmd/get_resource.go`, `c8volt/resource/client.go`, and `internal/services/resource/`
- [x] T004: Review selected-tenant option flow in `c8volt/foptions/options.go`, `internal/services/calloption.go`, `internal/services/common/`, and affected v88/v89 service packages
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- specs/156-tenant-admin-keys/tasks.md
- specs/156-tenant-admin-keys/progress.md
**Learnings**:
- No conflict found between `specs/ralph-implementation-rules.md` and the tenant admin keys feature specification.
- Setup validation passed with `go test ./cmd ./c8volt/process ./c8volt/resource ./internal/services/processinstance/... ./internal/services/processdefinition/... ./internal/services/resource/... -run '^$' -count=1`.
---
