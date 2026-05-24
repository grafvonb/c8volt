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
- Camunda 8.7 process-instance direct lookup intentionally returns an unsupported tenant-safe boundary; this feature should preserve that behavior.
- Camunda 8.8/8.9 process-instance state lookups delegate to direct keyed lookup, so any explicit-input tenant semantics should be changed at keyed lookup behavior rather than adding state-specific checks.
- Process-definition and resource direct get paths map returned tenant metadata from keyed/id backend endpoints without a local selected-tenant equality check.
- Search-derived process-instance dry-run command tests can isolate c8volt-produced candidates by capturing top-level search requests separately from parent/child dependency-expansion searches.
- Direct process-instance key command modes use `foptions.WithIgnoreTenant()` for traversal, dry-run planning, and enrichment so operator-supplied keys are not narrowed by selected-tenant search filters.
- v88/v89 process-instance search services honor `services.WithIgnoreTenant()` by omitting the tenant filter, while normal search-derived command paths continue to pass tenant-scoped options.

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
---
## Iteration 2 - 2026-05-24 21:56 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**: 
- [x] T005: Identify and record current local tenant mismatch checks or tenant-equality assumptions in `specs/156-tenant-admin-keys/progress.md`
- [x] T006: Add or update shared test fixtures for tenant-a selected context with tenant-b returned metadata in `cmd/get_processinstance_test.go`
- [x] T007: Add or update facade/service stubs for explicit tenant mismatch behavior in `c8volt/process/client_test.go` and `c8volt/resource/client_test.go`
- [x] T008: Verify no new c8volt-side authorization layer is needed and record the chosen repository-native path in `specs/156-tenant-admin-keys/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- cmd/get_processinstance_test.go
- c8volt/process/client_test.go
- c8volt/resource/client_test.go
- specs/156-tenant-admin-keys/tasks.md
- specs/156-tenant-admin-keys/progress.md
**Learnings**:
- Current local tenant assumptions are search-side filters, v8.7 unsupported direct lookup safeguards, and backend not-found handling in wrong-tenant v8.8/v8.9 fixtures; no new c8volt-side authorization layer is needed.
- The repository-native path is to keep commands classifying input mode, let facades translate public options, and let versioned services own search tenant filters versus keyed/id backend calls.
- Validation passed with `go test ./cmd ./c8volt/process ./c8volt/resource -run 'Test(GetProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest|Client_LookupProcessInstance_UsesSearchBackedLookup|Client_LookupProcessInstanceStateByKey_MapsSearchBackedState|Client_GetResource)$' -count=1`.
---
---
## Iteration 3 - 2026-05-24 22:03 CEST
**User Story**: User Story 1 - Preserve Tenant-Scoped Discovery Boundaries
**Tasks Completed**: 
- [x] T009: Add tenant-scoped process-instance search/list test in `cmd/get_processinstance_test.go`
- [x] T010: Add search-derived `cancel pi --dry-run` tenant-scoped candidate test in `cmd/cancel_test.go`
- [x] T011: Add search-derived `delete pi --dry-run` tenant-scoped candidate and dependency-scope test in `cmd/delete_test.go`
- [x] T012: Ensure `get pi` search/list mode continues passing selected tenant through existing filters/options in `cmd/get_processinstance_search.go` and affected process-instance services
- [x] T013: Ensure search-derived `cancel pi` preserves the tenant-scoped discovered candidate set in `cmd/cancel_processinstance.go` and `c8volt/process/client.go`
- [x] T014: Ensure search-derived `delete pi` preserves the tenant-scoped discovered candidate set and intended dependency scope in `cmd/delete_processinstance.go` and `c8volt/process/client.go`
- [x] T015: Run `go test ./cmd -run 'Test(GetProcessInstance|CancelProcessInstance|DeleteProcessInstance).*Tenant' -count=1`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- cmd/get_processinstance_test.go
- cmd/cancel_test.go
- cmd/delete_test.go
- specs/156-tenant-admin-keys/tasks.md
- specs/156-tenant-admin-keys/progress.md
**Learnings**:
- `get pi` search already sends the effective tenant through versioned process-instance search requests; US1 only needed regression coverage.
- Search-derived cancel/delete dry-run flows freeze keys from the tenant-scoped search page before dependency expansion, and child-scope searches keep the same tenant filter.
- Validation passed with `go test ./cmd -run 'Test(GetProcessInstance|CancelProcessInstance|DeleteProcessInstance).*Tenant' -count=1`.
---
---
## Iteration 4 - 2026-05-24 22:12 CEST
**User Story**: User Story 2 - Treat Explicit Process-Instance Keys As Admin Input
**Tasks Completed**: 
- [x] T016: Add `get pi --key` selected-tenant mismatch test in `cmd/get_processinstance_test.go`
- [x] T017: Add `walk pi --key` selected-tenant mismatch test in `cmd/walk_test.go`
- [x] T018: Add `expect pi --key` selected-tenant mismatch test in `cmd/expect_test.go`
- [x] T019: Add `cancel pi --key --dry-run` selected-tenant mismatch test in `cmd/cancel_test.go`
- [x] T020: Add `delete pi --key --dry-run` selected-tenant mismatch test in `cmd/delete_test.go`
- [x] T021: Update direct process-instance lookup and enrichment paths to avoid c8volt-side tenant mismatch rejection in `c8volt/process/client.go` and affected `internal/services/processinstance/` packages
- [x] T022: Preserve existing direct-key cancellation preflight, dry-run, confirmation, force, and verification behavior in `cmd/cancel_processinstance.go` and `c8volt/process/client.go`
- [x] T023: Preserve existing direct-key deletion preflight, dry-run, dependency expansion, force, and verification behavior in `cmd/delete_processinstance.go` and `c8volt/process/client.go`
- [x] T024: Run `go test ./cmd ./c8volt/process ./internal/services/processinstance -run 'Test.*(Key|Direct).*Tenant|Test.*Tenant.*Key' -count=1`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- c8volt/process/client.go
- c8volt/process/client_test.go
- cmd/cancel_processinstance.go
- cmd/cancel_test.go
- cmd/cmd_options.go
- cmd/cmd_views_processinstance_dryrun.go
- cmd/delete_processinstance.go
- cmd/delete_test.go
- cmd/expect_processinstance.go
- cmd/expect_test.go
- cmd/get_processinstance.go
- cmd/get_processinstance_enrichment.go
- cmd/get_processinstance_test.go
- cmd/walk_processinstance.go
- cmd/walk_test.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v89/service.go
- specs/156-tenant-admin-keys/tasks.md
- specs/156-tenant-admin-keys/progress.md
**Learnings**:
- Facade direct process-instance lookup had been using the shared search-backed helper; explicit admin input now delegates to versioned service direct lookup so v88/v89 backend authorization owns tenant mismatch behavior.
- Direct-key traversal and dry-run planning need `IgnoreTenant` only for operator-supplied keys; search-derived cancel/delete paths keep ordinary tenant-scoped options.
- Validation passed with `GOCACHE=/private/tmp/c8volt-go-build-cache go test ./cmd ./c8volt/process ./internal/services/processinstance -run 'Test.*(Key|Direct).*Tenant|Test.*Tenant.*Key' -count=1`.
---
