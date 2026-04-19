# Ralph Progress Log

Feature: 110-camunda-v89-support
Started: 2026-04-17 08:42:48

## Codebase Patterns

- User-facing version-support wording should be authored in `cmd/root.go` and `README.md`, then propagated through `make docs-content`; generated docs are verification output, not the source of truth.
- `c8volt/client.go` is the single top-level wiring seam; keep version selection in the service factories and facade wiring instead of branching in commands.
- Factory regression tests under `internal/services/*/factory_test.go` prove supported-version routing by asserting concrete service types and `services.ErrUnknownAPIVersion` behavior.
- When `toolx.CurrentCamundaVersion` stays on an older default during a new-version rollout, add explicit factory regressions that keep the default pinned to the current runtime (`v8.8` here) so broader support claims do not silently change default selection behavior.
- Feature research is the right place to record support-boundary inventory and generated-client capability checks before implementation starts.
- When the repository needs to advertise a broader supported-version contract before every factory implements it, keep separate helper surfaces for contract support versus runtime-implemented versions so factory error text stays truthful during incremental rollout.
- When top-level client support lags behind advertised version support, lock the behavior with a `c8volt.New(...)` regression test so the first failing factory remains the only version gate and commands do not grow their own branching.
- When a new versioned service family is not wired yet, add constructor-only `v89` package scaffolds plus interface-level `api.go` assertions first; this preserves package shape and generated-client contracts without falsely claiming runtime support.
- Camunda `v89` search endpoints can diverge from their generated typed aliases; when the generated request/response types flatten away required `filter`/`sort` or `items` fields, stay on the generated client boundary by using its raw-body `WithBodyWithResponse` method and decode the real JSON envelope locally in the service.
- Resource `v89` deploy/get/delete behavior can mirror `v88` directly, including deployment confirmation polling through the version-local process-definition client, because the generated Camunda `v89` resource and definition endpoints retain the same response contracts.
- Process-instance `v89` search has the same generated alias gap as process-definition `v89`: typed search bodies lose `filter` and `sort`, and typed `JSON200` omits `items`, so the service needs a version-local raw-body request/response wrapper while still staying on the generated `v89` Camunda client boundary.
- Camunda `v89` single-instance deletion uses the generated `/process-instances/{key}/deletion` POST operation rather than the older `DELETE /v1/process-instances/{key}` seam, so command proof tests should assert the generated-client endpoint shape directly.
- In-process command tests that touch `run` flags must reset the shared `flagRunPI*` globals alongside the process-instance search flags, otherwise later tests can inherit stale selector state and fail with false mutually-exclusive-flag errors.
- Deploy command tests that set `StringSlice` file flags should prefer the existing subprocess helper pattern over the generic in-process root executor, because Cobra can round-trip the shared default as a literal `[]` when the tree is reset in-process.

---

## Iteration 8 - 2026-04-17 09:41 CEST
**User Story**: User Story 1 - Run Existing Commands on v8.9
**Tasks Completed**:
- [x] T011: Add native `v89` service tests for process-instance create/get/search/cancel/delete/wait behavior
- [x] T012: Add explicit `v8.9` command execution tests for cluster, process-definition/resource, and process-instance command families
- [x] T016: Wire the new `v89` services through the shared client and process/resource facades
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/client_test.go
- cmd/bootstrap_errors_test.go
- cmd/delete_test.go
- cmd/deploy_test.go
- cmd/get_processinstance_test.go
- cmd/get_test.go
- cmd/run_test.go
- cmd/walk_test.go
- specs/110-camunda-v89-support/tasks.md
- specs/110-camunda-v89-support/progress.md
**Learnings**:
- Native `v89` command proof is easiest to keep honest by asserting the exact generated Camunda endpoint seams per family, especially the search-backed process-instance lookups and the `/deletion` process-instance delete operation.
- `c8volt.New(...)` plus one process facade call and one resource facade call is enough to prove the shared client wiring now reaches the `v89` runtime path without reintroducing command-local version branching.
- User Story 1 now passes the required validation bar with `go test ./internal/services/processinstance/... -count=1`, `go test ./c8volt ./cmd -count=1`, and `make test`.
---

## Iteration 10 - 2026-04-17 09:52 CEST
**User Story**: User Story 3 - Make v8.9 Support Verifiable and Explicit
**Tasks Completed**:
- [x] T023: Add doc-facing regression coverage for updated supported-version output
- [x] T024: Add or refresh final verification notes and quickstart validation guidance
- [x] T025: Update user-facing version-support guidance
- [x] T026: Regenerate CLI reference output for the updated help text and sync homepage content from README.md
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- README.md
- cmd/get_test.go
- cmd/root.go
- cmd/version_test.go
- docs/cli/c8volt.md
- docs/index.md
- specs/110-camunda-v89-support/plan.md
- specs/110-camunda-v89-support/progress.md
- specs/110-camunda-v89-support/quickstart.md
- specs/110-camunda-v89-support/tasks.md
**Learnings**:
- Root help and `README.md` are the authored version-support sources; `make docs-content` is the required sync step that carries the same wording into `docs/index.md` and generated CLI reference pages.
- Doc-facing regression stays cheap and stable when it pins the rendered root help and JSON `version` output instead of asserting generated markdown directly.
- This story closes cleanly only after the wording changes survive both `go test ./cmd -count=1` and the repository gate `make test`, because user-facing support claims now live in tested command metadata as well as generated docs.
---

## Iteration 9 - 2026-04-17 10:25 CEST
**User Story**: User Story 2 - Keep Version Selection Predictable
**Tasks Completed**:
- [x] T017: Add regression coverage for supported-version selection and preserved `v8.7`/`v8.8` behavior
- [x] T018: Add bootstrap and config regression coverage for `v8.9` support messaging and unsupported-version failures
- [x] T019: Add process-instance-specific regression coverage that proves final native `v8.9` paths stay on the `v89` Camunda client boundary and any temporary fallback stays documented/non-final
- [x] T020: Update root command version messaging and supported-version behavior for `v8.9`
- [x] T021: Preserve older-version behavior while extending version-aware helpers and error normalization for `v8.9`
- [x] T022: Finalize documented transition-only fallback rules and removal conditions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/ferrors/errors.go
- c8volt/ferrors/errors_test.go
- cmd/bootstrap_errors_test.go
- cmd/root.go
- config/app.go
- config/app_test.go
- internal/services/cluster/factory_test.go
- internal/services/processdefinition/factory_test.go
- internal/services/processinstance/factory_test.go
- internal/services/processinstance/v89/service_test.go
- internal/services/resource/factory_test.go
- specs/110-camunda-v89-support/contracts/v89-support.md
- specs/110-camunda-v89-support/plan.md
- specs/110-camunda-v89-support/progress.md
- specs/110-camunda-v89-support/research.md
- specs/110-camunda-v89-support/tasks.md
- toolx/version.go
- toolx/version_test.go
**Learnings**:
- Once native `v89` implementations exist for every versioned service family, `ImplementedCamundaVersions()` and runtime unsupported-version error text must move forward with them; leaving the old list behind makes bootstrap and factory behavior look partially unsupported even when routing is complete.
- Preserved-version regression is clearer when each factory also proves the unchanged default runtime path explicitly; `toolx.CurrentCamundaVersion` staying on `v8.8` is a separate promise from `v8.9` being supported.
- The `processinstance/v89` client-boundary rule is easiest to keep honest with the version-local `GenProcessInstanceClientCamunda` interface plus tests that exercise lookup/search behavior without any fallback-specific client seam.
---

## Iteration 7 - 2026-04-17 09:27 CEST
**User Story**: Partial progress on User Story 1 - Run Existing Commands on v8.9
**Tasks Completed**:
- [x] T015: Implement the native `v89` process-instance service and factory selection
**Tasks Remaining in Story**: 3
**Commit**: No commit - partial progress
**Files Changed**:
- c8volt/client_test.go
- internal/services/processinstance/factory.go
- internal/services/processinstance/factory_test.go
- internal/services/processinstance/v89/bulk.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/convert.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/v89/service_test.go
- specs/110-camunda-v89-support/tasks.md
- specs/110-camunda-v89-support/progress.md
**Learnings**:
- Final native `v89` process-instance behavior can keep the tenant-safe lookup model by staying search-backed for key/state reads, while cancel/delete/walker/waiter compose on top of that shared lookup seam.
- Camunda `v89` delete semantics can stay repository-compatible without the old Operate dependency: recursive child deletion, conflict-on-active handling, optional cancel-first retry, and absent-state waiting all work on the `v89` Camunda boundary.
- Focused validation for this slice passes with `go test ./c8volt ./internal/services/processinstance ./internal/services/processinstance/... -count=1`.
---

## Iteration 6 - 2026-04-17 09:52 CEST
**User Story**: Partial progress on User Story 1 - Run Existing Commands on v8.9
**Tasks Completed**:
- [x] T010: Add native `v89` service tests for resource deploy/get/delete behavior
- [x] T014: Implement native `v89` resource service plus factory selection
**Tasks Remaining in Story**: 4
**Commit**: No commit - partial progress
**Files Changed**:
- internal/services/resource/factory.go
- internal/services/resource/factory_test.go
- internal/services/resource/v89/contract.go
- internal/services/resource/v89/convert.go
- internal/services/resource/v89/service.go
- internal/services/resource/v89/service_test.go
- specs/110-camunda-v89-support/tasks.md
- specs/110-camunda-v89-support/progress.md
**Learnings**:
- The generated Camunda `v89` resource endpoints preserve the same deploy/get/delete response contracts as `v88`, so the native service can reuse the existing repository patterns without fallback.
- Deployment confirmation for resources should stay aligned with the `v89` process-definition client rather than reaching across versions, which keeps the final native path on the required `v89` Camunda client boundary.
- Focused validation for this slice passes with `go test ./internal/services/resource/... -count=1`.
---

## Iteration 5 - 2026-04-17 09:10 CEST
**User Story**: Partial progress on User Story 1 - Run Existing Commands on v8.9
**Tasks Completed**:
- [x] T008: Add native `v89` service tests for cluster topology/license behavior
- [x] T009: Add native `v89` service tests for process-definition search/get/XML/statistics behavior
- [x] T013: Implement native `v89` cluster and process-definition services plus factory selection
**Tasks Remaining in Story**: 6
**Commit**: No commit - partial progress
**Files Changed**:
- internal/services/cluster/factory.go
- internal/services/cluster/factory_test.go
- internal/services/cluster/v89/convert.go
- internal/services/cluster/v89/service.go
- internal/services/cluster/v89/service_test.go
- internal/services/processdefinition/factory.go
- internal/services/processdefinition/factory_test.go
- internal/services/processdefinition/v89/contract.go
- internal/services/processdefinition/v89/convert.go
- internal/services/processdefinition/v89/service.go
- internal/services/processdefinition/v89/service_test.go
- specs/110-camunda-v89-support/tasks.md
- specs/110-camunda-v89-support/progress.md
**Learnings**:
- Native `v89` cluster support mirrors `v88` directly, so the main work there is converter parity plus factory routing and regression proof.
- `processdefinition/v89` cannot reuse the `v88` typed search request path because the generated `v89` aliases drop the search `filter`/`sort` fields and the typed search response omits `items`; local JSON envelopes on top of the generated raw-body method preserve the repository contract without leaving the `v89` client boundary.
- Focused validation for this slice passes with `go test ./internal/services/cluster/... ./internal/services/processdefinition/... -count=1`.

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

## Iteration 11 - 2026-04-17 13:47 CEST
**User Story**: Polish & Cross-Cutting Concerns
**Tasks Completed**:
- [x] T027: Refresh implementation notes and final support-boundary records
- [x] T028: Run focused `v8.9` validation
- [x] T029: Run documentation regeneration and repository validation
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- docs/index.md
- specs/110-camunda-v89-support/contracts/v89-support.md
- specs/110-camunda-v89-support/data-model.md
- specs/110-camunda-v89-support/progress.md
- specs/110-camunda-v89-support/research.md
- specs/110-camunda-v89-support/tasks.md
**Learnings**:
- The final polish pass is mostly artifact alignment and proof: once the runtime and docs stories are complete, the remaining work is to replace rollout-era wording in feature records with the final accepted `v8.9` support boundary.
- `make docs-content` refreshed `docs/index.md` build metadata from the current dirty tree even without new README edits, so generated doc output still belongs in the final validation commit when that gate is part of the work unit.
- The closing verification bar for this feature passes with the focused `v8.9` suites, `make docs`, `make docs-content`, and repository-wide `make test`.
---
