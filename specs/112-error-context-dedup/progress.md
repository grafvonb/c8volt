# Ralph Progress Log

Feature: 112-error-context-dedup
Started: 2026-04-17 15:38:09

## Codebase Patterns

- Wrapper ownership should stay at the CLI-facing seam: `ferrors` owns shared class prefixes and exit behavior, while command/service wrappers own breadcrumb context and must not restate the same root detail.
- Process-instance duplication clusters around walker breadcrumbs (`get %s`, `list children of %s`, `ancestry fetch`) plus versioned service wrappers such as `fetching process instance with key %s` and `waiting for ... failed`.
- Existing regression anchors already map cleanly by pattern family: `walk` for traversal, `cancel` and `delete` for mutation/wait flows, `get` for single-resource fetch wrappers, and `c8volt/ferrors` plus bootstrap tests for unchanged class and exit behavior.
- The lowest-risk dedup change is to shorten wrappers to stage-only breadcrumbs such as `get process instance`, `cancel wait`, or `get cluster topology`; this preserves ordering while leaving the deepest normalized failure detail intact.
- Helper-level regressions should assert raw wrapper prefixes and stage order directly, because those seams run before `ferrors` normalization and therefore do not carry the shared class prefix yet.
- Equivalent shortening is safest when wrapper labels collapse to the stage noun or a short noun pair, for example `ancestry`, `family`, `process instance state`, `cancel validation`, or `delete wait absent`.

---

## Iteration 1 - 2026-04-17 16:00 CEST
**User Story**: Setup phase audit boundary and regression anchors
**Tasks Completed**:
- [x] T001 Refresh the implementation boundary and affected pattern-family inventory in `specs/112-error-context-dedup/plan.md`, `specs/112-error-context-dedup/research.md`, and `specs/112-error-context-dedup/contracts/cli-error-rendering.md`
- [x] T002 Inventory current duplicated wrapper seams and target owner layers across walker, versioned process-instance services, and CLI wrappers
- [x] T003 Confirm representative regression anchors for each duplication-pattern family in command, helper, and shared error tests
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/112-error-context-dedup/plan.md
- specs/112-error-context-dedup/research.md
- specs/112-error-context-dedup/contracts/cli-error-rendering.md
- specs/112-error-context-dedup/tasks.md
- specs/112-error-context-dedup/progress.md
**Learnings**:
- The setup work is fully documentational, but it established the concrete owner layers that later code cleanup must stay within.
- `cmd/get_test.go` and the versioned service tests already assert today’s wrapper text, so later behavior changes will need deliberate test rewrites rather than additive coverage only.
---

## Iteration 2 - 2026-04-17 15:46 CEST
**User Story**: Foundational rendering contract and shared helper boundary
**Tasks Completed**:
- [x] T004 Define the authoritative prefix-preserving dedup contract and breadcrumb-shortening rules in `contracts/cli-error-rendering.md`, `data-model.md`, and `plan.md`
- [x] T005 Keep shared classification and exit behavior fixed while tightening helper expectations in `c8volt/ferrors/errors.go`, `cmd/cmd_cli.go`, and `cmd/cmd_errors.go`
- [x] T006 Add foundational regression coverage for unchanged shared classification and exit behavior in `c8volt/ferrors/errors_test.go` and `cmd/bootstrap_errors_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/112-error-context-dedup/contracts/cli-error-rendering.md
- specs/112-error-context-dedup/data-model.md
- specs/112-error-context-dedup/plan.md
- specs/112-error-context-dedup/tasks.md
- specs/112-error-context-dedup/progress.md
- c8volt/ferrors/errors.go
- c8volt/ferrors/errors_test.go
- cmd/cmd_cli.go
- cmd/cmd_errors.go
- cmd/bootstrap_errors_test.go
**Learnings**:
- The foundational contract is now explicit that `ferrors` and the command/bootstrap helpers own class selection, prefix rendering, and exit behavior only; later dedup work must stay in upstream wrappers.
- Locking normalized-error stability in focused tests gives later story work room to shorten breadcrumbs and remove duplicate detail text without accidentally moving the shared error boundary.
---

## Iteration 3 - 2026-04-17 15:55 CEST
**User Story**: User Story 1 - Keep CLI failures readable
**Tasks Completed**:
- [x] T007 Add traversal-family regression tests in `cmd/walk_test.go` and `internal/services/processinstance/walker/walker_test.go`
- [x] T008 Add mutation-family regression tests in `cmd/cancel_test.go`, `cmd/delete_test.go`, `internal/services/processinstance/v87/service_test.go`, and `internal/services/processinstance/v88/service_test.go`
- [x] T009 Add single-resource fetch regression tests in `cmd/get_test.go` and related `get_*` command tests
- [x] T010 Refactor lookup and traversal wrappers in `internal/services/processinstance/walker/walker.go`, `internal/services/processinstance/v87/service.go`, `internal/services/processinstance/v88/service.go`, and `internal/services/processinstance/v89/service.go`
- [x] T011 Refactor mutation and wait follow-up wrappers in the versioned process-instance services and aligned wait-facing seams
- [x] T012 Sweep representative CLI fetch and orchestration wrappers in `cmd/get_*` commands and `c8volt/resource/client.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/resource/client.go
- cmd/bootstrap_errors_test.go
- cmd/cancel_test.go
- cmd/delete_test.go
- cmd/get_cluster_license.go
- cmd/get_cluster_topology.go
- cmd/get_processdefinition.go
- cmd/get_resource.go
- cmd/get_test.go
- cmd/walk_test.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v87/service_test.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/walker/walker.go
- internal/services/processinstance/walker/walker_test.go
- specs/112-error-context-dedup/tasks.md
- specs/112-error-context-dedup/progress.md
**Learnings**:
- Command-level regressions are important here because some process-instance flows fail during key-validation orchestration before reaching the obvious cancel/delete service entrypoints.
- Short stage-only breadcrumbs are enough to keep failures diagnosable once the deepest layer already owns the resource-specific detail.
---

## Iteration 4 - 2026-04-17 16:01 CEST
**User Story**: User Story 2 - Preserve where the failure happened
**Tasks Completed**:
- [x] T013 Add helper-level tests for ordered and equivalent breadcrumb preservation in `internal/services/processinstance/walker/walker_test.go` and `internal/services/processinstance/v87/service_test.go`
- [x] T014 Add command-level regression tests for recognizable breadcrumb stages after shortening in `cmd/walk_test.go`, `cmd/cancel_test.go`, and `cmd/delete_test.go`
- [x] T015 Adjust breadcrumb wording in shared traversal and process-instance wrappers to preserve equivalent stage meaning with less noise in `internal/services/processinstance/walker/walker.go`, `internal/services/processinstance/v87/service.go`, `internal/services/processinstance/v88/service.go`, and `internal/services/processinstance/v89/service.go`
- [x] T016 Align command-surface wrappers with the equivalent-breadcrumb contract in `cmd/get_processinstance.go`, `cmd/walk_processinstance.go`, `cmd/cancel_processinstance.go`, and `cmd/delete_processinstance.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel_processinstance.go
- cmd/cancel_test.go
- cmd/delete_processinstance.go
- cmd/delete_test.go
- cmd/get_processinstance.go
- cmd/get_test.go
- cmd/walk_test.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v87/service_test.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/walker/walker.go
- internal/services/processinstance/walker/walker_test.go
- specs/112-error-context-dedup/tasks.md
- specs/112-error-context-dedup/progress.md
**Learnings**:
- Raw helper errors are not normalized through `ferrors`, so helper-level ordering tests should assert breadcrumb prefixes and wrapper order without assuming a shared class prefix at that seam.
- The stable equivalent-shortening pattern for this feature is to trim wrapper labels down to stage nouns or short stage pairs such as `ancestry`, `family`, `process instance state`, `cancel validation`, and `delete wait absent`.
---
