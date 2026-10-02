# Tasks: Simplify Supported Camunda Versions

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [support contract](contracts/support-policy.md), [quickstart.md](quickstart.md)

**Issue**: [#331](https://github.com/grafvonb/c8volt/issues/331)

**Tests**: Required by FR-012. Every work unit includes implementation, fixtures/doubles, formatting and relevant checks. No separate test-only or bookkeeping tasks.

## Format and Path Conventions

Tasks use `- [ ] Tnnn [USn] description`. Paths are relative to the repository root. For family paths, `<area>` means batchoperation, cluster, element, incident, job, processdefinition, processinstance, resource, tenant, usertask or variable. A retired subtree is the release-specific directory identified by issue #331; enumerate its exact tracked files before deletion. The feature's documents intentionally avoid retaining obsolete support identifiers.

## Phase 1: Setup

No initialization task: the repository, issue branch and design already exist. Each work unit reads the feature artifacts and `specs/ralph-implementation-rules.md`, preserves unrelated changes and records actual evidence in `specs/331-simplify-version-support/progress.md` as part of its delivery.

## Phase 2: Foundational

No standalone infrastructure task. The coupled version-selection change is owned by T001. The explicit user instruction authorizes replacing old version inventories in guidance; all architecture, operator UX and validation rules remain applicable.

## Phase 3: User Story 1 — Operate with retained versions (P1)

**Goal**: Runtime accepts the retained set and preserves its observable behavior.

**Independent test**: Supported selection/default, generic invalid-version bootstrap with zero remote calls/prompts, 8.8 Operate deletion, 8.8/8.9 Tasklist fallback, retained tenant/output/confirmation regressions.

- [ ] T001 [US1] Complete the compiling runtime cutover in `toolx/version.go`, `toolx/fixture_compatibility.go`, `config/app.go`, `cmd/root_config.go`, `cmd/get_processinstance_validation.go`, its get/cancel/delete/repair callers, `internal/services/ops/slow_process_analysis.go`, all eleven `internal/services/<area>/factory.go` files and applicable `api.go` assertions: remove the retired public identity/aliases, adapter directories and dedicated adapter tests; update every remaining constant reference and invalidated test in `toolx/`, `config/`, `cmd/`, `c8volt/diagnostics_test.go`, `internal/services/common/tenant_context_test.go` and ops/factory tests in the same unit. Enforce data-model constraints “Allowed canonical values: 8.8, 8.9, 8.10”, “Default: 8.9” and empty discovery versus concrete creation tenant semantics. Remove retired command cases and subprocess entrypoints without deleting retained table rows. Keep generic invalid identity validation for flags/environment/files/profiles/library construction, preserving precedence, error classes and zero HTTP calls/prompts; use existing terminal harness where confirmation eligibility is relevant. Replace obsolete Operate diagnostics exercise with 8.8 deletion and preserve retained Tasklist transport checks. Update version-sensitive help in `cmd/root.go`, `cmd/get_processdefinition.go`, `cmd/get_processinstance.go`, `cmd/get_usertask.go`, `cmd/update_usertask.go`, related metadata tests, `README.md`, `docs/ops/analyse-slow-process-instances.md`, `AGENTS.md` and `specs/ralph-implementation-rules.md`; run `make docs-content` and relevant docsgen tests in this unit. Format changed Go files; run the runtime/configuration commands from `quickstart.md`, retained component tests, and package compilation across `./...`; resolve all failures before completion. Do not retain a deprecated symbol or add a dedicated retired-release rejection suite.

**Checkpoint**: The unsupported runtime is unreachable and its adapter code is gone; retained workflows and public support messages agree. Unreferenced generated output/assets are removed in US2.

## Phase 4: User Story 2 — Maintain necessary support only (P2)

**Goal**: Remove dead compatibility code, generation targets, fixtures and integration exceptions without reducing supported behavior.

**Independent test**: Shared traversal regressions, deterministic generation tests and retained output checks, tagged integration compilation, complete executable regression suite/build.

- [ ] T002 [US2] Simplify `internal/services/processinstance/dryrun.go` and `internal/services/processinstance/dryrun_test.go` by deleting the exclusive legacy marker interface, dispatch, key-only planner, tenant-evidence helper and test double; trace and remove any newly orphaned exclusive helpers across `internal/services/` and `cmd/`. Preserve normal traversal/walker paths, unknown tenant evidence from incomplete supported responses, missing ancestors, exact/lower-bound/indeterminate totals and mutation planning. In `cmd/cmd_deploy_run_data.go`, `cmd/cmd_deploy_run_data_test.go`, `cmd/deploy_test.go`, `cmd/embed_test.go` and `cmd/deploy_processdefinition_progress.go`, retain generic deployment identifier/BPMN fallback and guards while removing obsolete release-specific cases/comments left after T001. Format touched Go files and run targeted dry-run, page-planning, deploy/embed and affected command tests from `quickstart.md`; preserve sparse pages, abort/error behavior, exact empty output and real-terminal stream-routing coverage where touched.

- [ ] T003 [US2] Delete the retired output subtree under `internal/clients/camunda/` and its generation commands/file-list entries in `api/3-generate-clients-from-fetched-specs.sh` together; remove retired mock targets from `.mockery.yml` and protection entries from `api/generate-v810-client.sh` and `api/tests/v810_repository_boundary_test.sh`, updating `api/README.md`. Preserve 8.8 Operate, 8.8/8.9 Tasklist, retained client provenance and unrelated unused clients. Review tracked inputs under `api/camunda/` for actual obsolete support content without deleting supported endpoint definitions or corrupting numeric provenance. Add focused mocked coverage beside `api/tests/v810_generation_test.sh` for the full generator's target inventory if existing tests cannot prove non-recreation, asserting allowed current outputs rather than keeping retired identifiers. Run shell syntax checks, generation/provenance checks from `quickstart.md`, and compile the retained service/client consumers. Run the repository-boundary check only in a clean relevant tree or reviewed temporary snapshot; record its cleanliness precondition without weakening the guard.

- [ ] T004 [US2] Remove retired fixture files under `embedded/processdefinitions/` and the dedicated retired suite under `testx/`; update `testx/camunda_fake_server.go`, remaining fixture/smoke/embed tests, and affected integration profiles and scenarios in `integration/cli/harness_test.go`, `integration/cli/behavioral_scenarios_test.go`, `integration/cli/deploy_embed_run_test.go`, `integration/cli/get_usertask_test.go`, `integration/cli/get_usertask_variables_test.go` and `integration/cli/volume_get_usertask_variables_test.go`. Remove exclusive integration helpers and adjust `integration/README.md` and affected `integration/assets/` guidance; preserve retained version gates, restrictions and scenario coverage, without adding experimental-version integration support. Respect `integration/AGENTS.md`. Format changed Go files, run targeted fixture/command/ops tests, and compile tagged tests using `go test -tags=integration ./integration/cli ./testx/integration88 -run '^$' -count=1`. After targeted checks pass, run `make test` and `make build` for the completed executable change; document actual coverage and any live-cluster gap in `specs/331-simplify-version-support/progress.md`. Ignored local configs are untouched.

**Checkpoint**: Runtime, shared compatibility, generated targets and executable validation assets satisfy the removal contract. Full race-enabled tests and build have passed or explicit blockers remain uncompleted.

## Phase 5: User Story 3 — Read a consistent support policy (P2)

**Goal**: No obsolete support instruction remains in tracked documentation or project context.

**Independent test**: Tracked path/content audit, support-policy consistency, changed-link and formatting checks; generated documentation checked through its owner.

- [ ] T005 [US3] Complete the tracked support-content cleanup across `specs/`, `tasks/`, `AGENTS.md`, `README.md`, `docs/`, `api/README.md`, tracked configuration/examples and other matches discovered by `git ls-files` plus `rg`, including this feature's own artifacts. Remove obsolete requirements, examples and caveats while preserving unrelated planning history and context; do not delete entire historical artifacts merely to eliminate one match. Review ignored Spec Kit memory for obsolete active guidance without deleting unrelated memory; it is outside the tracked acceptance count. Audit path names as well as content and classify every remaining numeric/dependency/provenance match rather than suppressing whole directories. Keep user-facing material on the retained support policy without migration/deprecation prose. Regenerate changed CLI/homepage material through `make docs-content`, verify support/default/maturity statements, check local links and `git diff --check`, and record final FR/SC evidence in `specs/331-simplify-version-support/progress.md`. Carry forward T004's successful executable validation; if audit repairs executable files, run their relevant checks and broader checks warranted by those changes. Run the clean-tree repository boundary check under the preconditions in `quickstart.md` once cleanup commits are present. Do not mark completion with unresolved support references or validation failures.

## Final Phase: Polish and Cross-Cutting Concerns

Final audit and evidence are included in T005; no standalone routine-review task. Full runtime validation belongs to T004 because it is the last planned executable work. Subsequent code fixes invalidate only the relevant evidence as described above. No further feature, optimization or dependency cleanup is authorized by these tasks.

## Dependencies and Execution Order

```text
T001 (US1) -> T002 (US2) -> T003 (US2) -> T004 (US2) -> T005 (US3)
```

T001 must remove all compile-time references together. T002 removes unreachable shared mechanics. T003 removes generated output only once no adapter needs it. T004 closes the executable inventory and runs broad validation. T005 closes historical support content and the tracked audit. The story acceptance checks are independently assessable, but the implementation order is deliberately sequential.

## Parallel Opportunities and Examples

No tasks are marked `[P]`: deletion scopes, metadata, fixtures and final evidence share dependencies. Do not create extra tasks solely for parallel execution.

- **US1**: After edits, independent read-only review of version selection and retained transport coverage can run concurrently; mutations and test-helper updates remain in T001.
- **US2**: Generation fixture inspection and traversal review can be separate read-only checks, but complete T002–T004 in order so final validation sees the combined executable result.
- **US3**: Documentation inventories for `specs/` and `tasks/` can be inspected independently; consolidate edits and the final audit in T005 to avoid contradictory guidance.

## Consolidation Review

Five tasks: US1 has one, US2 has three, US3 has one. Setup/context/branch/progress responsibilities are embedded, not extra tasks. Runtime wiring, public-identity removal, adapter tests and command metadata are deliberately combined to avoid uncompilable or misleading intermediate states. Shared traversal, generation and integration assets remain separate because each has a distinct verification method and a practical review boundary. User-facing support docs ship with T001; T005 is remaining historical/context cleanup, not delayed runtime documentation. No coverage is removed solely to reduce task count.

## Requirement Coverage

| Requirements | Tasks |
| --- | --- |
| FR-001, FR-002, FR-003 | T001 |
| FR-004 | T001, T002, T003 |
| FR-005 | T001, T002, T004 |
| FR-006 | T003 |
| FR-007, FR-008 | T001, T002, T004 |
| FR-009, FR-010 | T001, T003, T004, T005 |
| FR-011 | T005 |
| FR-012 | Validation within T001–T005, broad executable checks in T004 |

## Implementation Strategy

T001 is the smallest useful runtime milestone, but it does not complete issue #331. Finish all five units before claiming complete removal. Ralph executes one permitted story/work unit at a time, includes relevant tests and progress in the same commit, and uses Conventional Commits with issue #331. No implementation or optional Ralph loop is started by generating this task list.
