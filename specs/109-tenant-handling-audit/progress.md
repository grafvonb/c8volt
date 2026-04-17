# Ralph Progress Log

Feature: 109-tenant-handling-audit
Started: 2026-04-17 03:50:08

## Codebase Patterns

- Tenant-safe behavior is centralized below `cmd/`: `cmd/root.go` resolves the effective tenant, `c8volt/process/client.go` forwards without adding tenant logic, and the real enforcement seams live in the versioned process-instance services plus shared walker/waiter helpers.
- The process-instance command surface splits into five audit families with different risk profiles: keyed/search `get`, walker-based `walk`, search-plus-preflight `cancel`, search-plus-preflight `delete`, and create-plus-confirmation `run`.
- Repository support for process-instance services currently stops at `v8.8`: `toolx` normalizes `8.9`, but `internal/services/processinstance/factory.go` and `factory_test.go` only admit `v87` and `v88`.
- Existing regression anchors are already in place for this feature: versioned service tests for request-shape and behavior seams, walker/waiter helper tests for mixed-flow composition, `cmd/*_test.go` for command-family coverage, and `config/config_test.go` for tenant-source precedence.
- Tenant-safe keyed lookup now resolves through the versioned search endpoint rather than unscoped direct-get endpoints, so command and service fixtures that used to stub `GET /v2/process-instances/<key>` need search responses for `processInstanceKey` filters as well.

---

## Iteration 1 - 2026-04-17 03:53:10 CEST
**User Story**: Phase 1 Setup
**Tasks Completed**:
- [x] T001: Inventory tenant-aware command families and shared flow seams in the setup artifacts
- [x] T002: Confirm version support and current process-instance factory boundaries
- [x] T003: Inspect existing tenant-related regression seams
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/109-tenant-handling-audit/research.md
- specs/109-tenant-handling-audit/quickstart.md
- specs/109-tenant-handling-audit/tasks.md
- specs/109-tenant-handling-audit/progress.md
**Learnings**:
- `walk`, `cancel`, and `delete` all depend on shared mixed-flow seams that combine direct lookup, descendant expansion, and waiter/state checks, so later fixes should be concentrated in the service and helper layers.
- `run` already injects `TenantId` on creation payloads; the tenant audit there is mainly about follow-up confirmation behavior rather than Cobra flag wiring.
- The current repo already has enough command/config/service test scaffolding to add tenant-source and supported-vs-unsupported coverage incrementally without introducing new harnesses.
---

## Iteration 2 - 2026-04-17 03:55:51 CEST
**User Story**: Partial progress on Phase 2 Foundational
**Tasks Completed**:
- [x] T004: Define the authoritative tenant-handling contract and unsupported-version boundaries
**Tasks Remaining in Story**: 3
**Commit**: No commit - partial progress
**Files Changed**:
- specs/109-tenant-handling-audit/contracts/tenant-handling.md
- specs/109-tenant-handling-audit/research.md
- specs/109-tenant-handling-audit/plan.md
- specs/109-tenant-handling-audit/tasks.md
- specs/109-tenant-handling-audit/progress.md
**Learnings**:
- The safest stable boundary for the refactor is an operation matrix: search-backed paths remain the tenant-safe baseline, while direct-key, state-check, and composed wait/mutation seams must earn support explicitly per version.
- `v8.9` must stay audit-only in this feature until the process-instance factory and tests grow a real `v89` implementation; version normalization alone is not runtime support.
---

## Iteration 3 - 2026-04-17 04:03:15 CEST
**User Story**: Phase 2 Foundational
**Tasks Completed**:
- [x] T005: Refactor shared process-instance service seams to support tenant-safe direct-get alternatives and explicit unsupported outcomes
- [x] T006: Add or update shared helper support for tenant-safe filters and response normalization
- [x] T007: Add foundational regression coverage for effective tenant source resolution
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/api.go
- c8volt/process/client.go
- c8volt/process/client_test.go
- cmd/config_test.go
- config/app_test.go
- config/config_test.go
- internal/services/common/deps.go
- internal/services/common/filter.go
- internal/services/common/helpers_test.go
- internal/services/common/response.go
- internal/services/processinstance/api.go
- specs/109-tenant-handling-audit/tasks.md
- specs/109-tenant-handling-audit/progress.md
**Learnings**:
- The shared seam can introduce tenant-safe lookup alternatives before the versioned services switch over by exposing search-backed lookup helpers alongside the existing direct-get methods.
- Single-result normalization needs to treat zero matches as `not found` and duplicate search matches as malformed response so later `v88` direct-lookup hardening has one repository-native outcome.
- Tenant source precedence is safest to lock down at three levels: `config.App` normalization for explicit empty values, `config.ResolveEffectiveConfig` for profile/env/base merging, and `cmd` bootstrap tests for root flag inheritance across real command trees.
---

## Iteration 4 - 2026-04-17 04:16:54 CEST
**User Story**: Partial progress on User Story 1 - Keep Tenant-Scoped Lookups Safe
**Tasks Completed**:
- [x] T008: Add `v88` service tests for tenant-safe direct lookup and supported wrong-tenant `not found` behavior
- [x] T009: Add `v87` service tests for explicit unsupported direct/state lookup outcomes
- [x] T010: Add direct process-instance command regression tests for flag/env/profile/base-config tenant sources plus wrong-tenant `not found`
**Tasks Remaining in Story**: 3
**Commit**: No commit - partial progress
**Files Changed**:
- c8volt/process/client.go
- cmd/cancel_test.go
- cmd/get_processinstance_test.go
- cmd/get_test.go
- internal/services/processinstance/v87/contract.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v87/service_test.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- specs/109-tenant-handling-audit/tasks.md
- specs/109-tenant-handling-audit/progress.md
**Learnings**:
- `v88` direct key lookup and keyed state checks can be hardened without new infrastructure by routing them through the existing search path plus `RequireSingleProcessInstance`.
- Narrowing `v87` keyed lookup/state lookup to explicit unsupported outcomes immediately propagates into cancel/delete preflight behavior, so the remaining US1 implementation work needs coordinated fixture updates in `cmd/` before the whole story is clean.
- Reused root command instances in `cmd/get_processinstance_test.go` need targeted persistent-flag resets; full tree resets can corrupt `StringSlice` defaults and create phantom keyed lookups.
---

## Iteration 5 - 2026-04-17 04:24:00 CEST
**User Story**: User Story 1 - Keep Tenant-Scoped Lookups Safe
**Tasks Completed**:
- [x] T011: Implement tenant-safe direct lookup and state lookup behavior in `v88`
- [x] T012: Narrow `v87` direct/state lookup to exact unsupported tenant-unsafe seams
- [x] T013: Normalize facade and command handling for tenant-safe `not found` and unsupported outcomes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/client.go
- cmd/cancel_test.go
- cmd/cmd_processinstance_test.go
- cmd/delete_test.go
- cmd/get_processinstance_test.go
- cmd/get_test.go
- internal/services/processinstance/v87/contract.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v87/service_test.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- specs/109-tenant-handling-audit/progress.md
- specs/109-tenant-handling-audit/tasks.md
**Learnings**:
- Search-backed keyed lookup is now the single tenant-safe authority for `v88`, so direct-key command tests must mock `processInstanceKey` search requests instead of `/v2/process-instances/<key>` GETs.
- `v87` direct get and keyed state checks now fail at the exact unsupported seam, and command regressions that previously depended on those preflight calls need to assert the narrower unsupported outcome rather than broad command-family success.
- Paging assertions must ignore per-key tenant-safe lookups and count only the actual top-level search requests; otherwise prompt/continuation tests overcount the new preflight traffic.
---

## Iteration 6 - 2026-04-17 04:26:49 CEST
**User Story**: User Story 2 - Preserve Tenant Boundaries Through Multi-Step Flows
**Tasks Completed**:
- [x] T014: Add walker and waiter regression tests for tenant-safe ancestry, descendants, family, and state polling behavior
- [x] T015: Add versioned service tests for mixed search plus direct-get flows, cancel preflight, and delete preflight behavior
- [x] T016: Add command-family regression tests for `walk`, `cancel`, `delete`, and `run` tenant propagation across flag/env/profile/base-config sources
- [x] T017: Implement tenant-safe traversal and direct-child lookup behavior
- [x] T018: Implement tenant-aware wait, cancel, and delete preflight handling with narrowly scoped unsupported outcomes
- [x] T019: Align tenant-aware command flows with the service-layer contract
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/109-tenant-handling-audit/tasks.md
- specs/109-tenant-handling-audit/progress.md
**Learnings**:
- The mixed-flow contract is now coherent across layers: walker traversal and waiter polling inherit tenant safety entirely from `GetProcessInstance`, `GetDirectChildrenOfProcessInstance`, and `GetProcessInstanceStateByKey`, so command code can stay thin.
- `v88` remains fully tenant-safe for mixed flows because every follow-up step reduces to tenant-filtered search-backed lookups, while `v87` now fails only when a keyed direct/state seam would otherwise become the authority.
- US2 verification is already anchored in repository-native tests: `walker` and `waiter` unit tests for helper composition, versioned service tests for preflight behavior, and command-family subprocess tests for tenant propagation and unsupported-surface handling.
---
