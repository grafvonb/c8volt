# Ralph Progress Log

Feature: 110-camunda-v89-support
Started: 2026-04-17 08:42:48

## Codebase Patterns

- `c8volt/client.go` is the single top-level wiring seam; keep version selection in the service factories and facade wiring instead of branching in commands.
- Factory regression tests under `internal/services/*/factory_test.go` prove supported-version routing by asserting concrete service types and `services.ErrUnknownAPIVersion` behavior.
- Feature research is the right place to record support-boundary inventory and generated-client capability checks before implementation starts.
- When the repository needs to advertise a broader supported-version contract before every factory implements it, keep separate helper surfaces for contract support versus runtime-implemented versions so factory error text stays truthful during incremental rollout.
- When top-level client support lags behind advertised version support, lock the behavior with a `c8volt.New(...)` regression test so the first failing factory remains the only version gate and commands do not grow their own branching.
- When a new versioned service family is not wired yet, add constructor-only `v89` package scaffolds plus interface-level `api.go` assertions first; this preserves package shape and generated-client contracts without falsely claiming runtime support.

---

## Iteration 1 - 2026-04-17 09:08 CEST
**User Story**: Phase 1: Setup
**Tasks Completed**:
- [x] T001: Inventory the repository-wide `v8.8` command families and current `v8.9` support boundary
- [x] T002: Confirm the generated `v89` Camunda client endpoints needed for native cluster, process-definition, process-instance, and resource support
- [x] T003: Confirm existing factory and regression-test seams for `cluster`, `processdefinition`, `processinstance`, and `resource`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/110-camunda-v89-support/research.md
- specs/110-camunda-v89-support/tasks.md
- specs/110-camunda-v89-support/progress.md
**Learnings**:
- The current repository state intentionally stops real runtime support at `v8.8` even though `toolx` already normalizes `8.9`.
- The generated `v89` Camunda client already exposes the main process-instance endpoints needed for a native final path, so Phase 2 can treat `processinstance` as an implementation problem rather than a client-gap problem.
- Existing factory tests and `c8volt/client.go` provide the repository-native seams for adding `v89` without introducing command-local version branching.
---

## Iteration 2 - 2026-04-17 08:49 CEST
**User Story**: Partial progress on Phase 2: Foundational
**Tasks Completed**:
- [x] T004: Update the shared supported-version source of truth for `v8.9`
**Tasks Remaining in Story**: 3
**Commit**: No commit - partial progress
**Files Changed**:
- toolx/version.go
- toolx/version_test.go
- cmd/version_test.go
- internal/services/cluster/factory.go
- internal/services/cluster/factory_test.go
- internal/services/processdefinition/factory.go
- internal/services/processdefinition/factory_test.go
- internal/services/processinstance/factory.go
- internal/services/processinstance/factory_test.go
- internal/services/resource/factory.go
- internal/services/resource/factory_test.go
- specs/110-camunda-v89-support/tasks.md
- specs/110-camunda-v89-support/progress.md
**Learnings**:
- `cmd/version.go` can inherit the broader support contract without direct code changes as long as `CurrentBuildInfo()` continues to source its version list from `toolx`.
- Factory unsupported-version messages currently double as operator guidance, so they must stay tied to runtime-implemented versions until native `v89` constructors land.
- `c8volt/ferrors/errors.go` did not need a code change for this slice because its unsupported-capability normalization stays correct once factory messages remain honest.
---

## Iteration 3 - 2026-04-17 08:53 CEST
**User Story**: Partial progress on Phase 2: Foundational
**Tasks Completed**:
- [x] T005: Extend shared factory coverage and top-level client wiring expectations for `v8.9`
**Tasks Remaining in Story**: 2
**Commit**: No commit - partial progress
**Files Changed**:
- internal/services/cluster/factory_test.go
- internal/services/processdefinition/factory_test.go
- internal/services/processinstance/factory_test.go
- internal/services/resource/factory_test.go
- c8volt/client_test.go
- specs/110-camunda-v89-support/tasks.md
- specs/110-camunda-v89-support/progress.md
**Learnings**:
- Each service factory now has an explicit regression proving `v8.9` is advertised but still rejected at the runtime-constructor layer until native services land.
- A single top-level `c8volt.New(...)` regression is enough to prove centralized wiring still fails through the shared service-factory boundary rather than through command-local checks.
- Focused validation for this slice passes with `go test ./internal/services/cluster ./internal/services/processdefinition ./internal/services/processinstance ./internal/services/resource ./c8volt ./toolx ./cmd -count=1`.
---

## Iteration 4 - 2026-04-17 09:33 CEST
**User Story**: Phase 2: Foundational
**Tasks Completed**:
- [x] T006: Create the base `v89` service package scaffolds and shared contract assertions
- [x] T007: Add the foundational version-support contract and release-gate notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- internal/services/cluster/api.go
- internal/services/cluster/v89/contract.go
- internal/services/cluster/v89/service.go
- internal/services/processdefinition/api.go
- internal/services/processdefinition/v89/contract.go
- internal/services/processdefinition/v89/service.go
- internal/services/processinstance/api.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/service.go
- internal/services/resource/api.go
- internal/services/resource/v89/contract.go
- internal/services/resource/v89/service.go
- specs/110-camunda-v89-support/contracts/v89-support.md
- specs/110-camunda-v89-support/quickstart.md
- specs/110-camunda-v89-support/tasks.md
- specs/110-camunda-v89-support/progress.md
**Learnings**:
- Phase 2 can reserve the final `v89` package shape by asserting the version-local interfaces in shared `api.go` files before the concrete services implement the full shared API.
- Constructor-only `v89` scaffolds are enough to validate dependency wiring, logger/client setup, and version-local generated-client contracts without prematurely routing factory selection to unfinished code.
- Focused validation for the completed foundational phase passes with `go test ./internal/services/cluster ./internal/services/processdefinition ./internal/services/processinstance ./internal/services/resource ./c8volt ./toolx ./cmd -count=1`.
---
