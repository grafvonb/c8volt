# Tasks: Simplify Supported Camunda Versions

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [support contract](contracts/support-policy.md), [quickstart.md](quickstart.md)

**Issue**: [#331](https://github.com/grafvonb/c8volt/issues/331)

## Execution Rules

Three cohesive work units, one per user story. Ralph permits multiple tasks within one story but cannot start another story in the same iteration; fewer checkboxes alone do not guarantee fewer iterations. Do not stop early merely because a substep is finished, and do not force completion if validation fails.

Paths are repository-relative. `<area>` expands to batchoperation, cluster, element, incident, job, processdefinition, processinstance, resource, tenant, usertask and variable. Retired subtrees are identified by the source issue and tracked inventory, without repeating obsolete support identifiers in these artifacts.

Every unit includes required tests, fixtures/doubles, formatting and coordinated Ralph progress/memory updates. Reuse recorded findings and passing checks until relevant changes invalidate them. No setup, branch-check, context-reading, repeated whole-repository scan, test-only or bookkeeping tasks. Generic validation already covered by existing tests must not be rebuilt as a new matrix.

## Setup and Foundation

Already available: issue branch, specification, design and validation guide. Each unit reads the current Ralph handoff and relevant artifacts. The explicit removal instruction supersedes obsolete version inventories in guidance; architecture, UX and validation rules remain binding. No separate foundational work is required.

## User Story 1 — Operate with Retained Versions (P1)

- [x] T001 [US1] Remove the retired runtime and its exclusive shared behavior atomically across `toolx/version.go`, `config/app.go`, `cmd/get_processinstance_validation.go`, `internal/services/<area>/`, `internal/services/processinstance/dryrun.go`, affected tests and support metadata; deliver the verified runtime contract described below.

**Implementation scope**:

- Remove the public version identity/aliases, supported/implemented list entries and fixture mappings in `toolx/version.go` and `toolx/fixture_compatibility.go`. Enforce canonical values `8.8`, `8.9`, `8.10` and default `8.9` from the data model.
- Delete all eleven retired adapter subtrees and dedicated tests; update `factory.go`, applicable `api.go` assertions and `factory_test.go` together. Remove every remaining compile-time reference and invalidated expectation in `toolx/`, `config/`, `cmd/`, `c8volt/`, `internal/services/common/` and `internal/services/ops/`.
- Remove exclusive tenant defaults in `config/app.go` and `cmd/root_config.go`, search guards and their get/cancel/delete/repair callers, and the retired guard in `internal/services/ops/slow_process_analysis.go`. Preserve explicit tenant precedence, empty discovery scope and concrete creation tenant behavior.
- Delete the legacy dry-run marker, dispatch, key-only planner, exclusive tenant-evidence helper and test double in `internal/services/processinstance/dryrun.go` and its test file now that their sole production caller is gone. Keep normal traversal, unknown metadata, sparse pages and partial outcomes.
- Remove retired command cases, subprocess helpers and stale comments in `cmd/deploy_test.go`, `cmd/embed_test.go`, `cmd/deploy_processdefinition_progress.go` and other affected tests. Keep the generic deployment identifier/BPMN fallback in `cmd/cmd_deploy_run_data.go` and its independent tests; no redesign is required.
- In `c8volt/diagnostics_test.go`, replace the retired Operate search exercise with supported 8.8 deletion. Preserve Tasklist fallback checks for 8.8/8.9, including tenant/identity validation and primary success/failure routing.
- Update support messaging in `cmd/root.go`, `cmd/get_processdefinition.go`, `cmd/get_processinstance.go`, `cmd/get_usertask.go`, `cmd/update_usertask.go`, their metadata tests, `README.md` and `docs/ops/analyse-slow-process-instances.md`. Update current version inventories in `AGENTS.md` and `specs/ralph-implementation-rules.md`; regenerate documentation once through `make docs-content`.

**Validation and done criteria**:

Run the T001 checks in `quickstart.md` once after related edits and formatting. Existing generic unsupported-version coverage must prove rejection before requests/prompts across the existing configuration entry points; fill demonstrated gaps only, without a retired-release rejection suite or exhaustive new flag permutations. Retain affected output/empty-scope/terminal tests rather than reimplementing them. Every adapter/constant reference and broken expectation must be resolved before marking this task complete. Unreferenced generated outputs and fixture assets can remain for T002.

## User Story 2 — Maintain Necessary Support Only (P2)

- [x] T002 [US2] Remove the remaining retired generated output, generation targets and validation assets together in `internal/clients/camunda/`, `api/`, `.mockery.yml`, `embedded/processdefinitions/`, `testx/` and `integration/`; validate the final executable repository once as described below.

**Implementation scope**:

- Delete the retired client subtree alongside commands/file-list entries in `api/3-generate-clients-from-fetched-specs.sh`, mockery entries, and protection-list entries in `api/generate-v810-client.sh` and `api/tests/v810_repository_boundary_test.sh`. Update `api/README.md`.
- Preserve retained clients, Operate and Tasklist authentication/transport, and unrelated unused clients. Inspect generation inputs for actual obsolete support content without deleting supported endpoints or altering unrelated numeric provenance.
- Prove removed output cannot be recreated. Use existing generation tests and target-list review first; add a narrow mocked test only if they leave a real gap. Do not build a new generation framework, refresh upstream repositories, or regenerate retained clients merely to delete an output target.
- Delete retired fixtures and the dedicated integration suite under `testx/`. Update `testx/camunda_fake_server.go`, affected smoke/embed/fixture tests, and the current profile/scenario files under `integration/cli/`: `harness_test.go`, `behavioral_scenarios_test.go`, `deploy_embed_run_test.go`, `get_usertask_test.go`, `get_usertask_variables_test.go`, `volume_get_usertask_variables_test.go`. Remove exclusive helpers and adjust `integration/README.md` and relevant `integration/assets/` prose in the same unit.
- Preserve retained profile gates/restrictions and respect `integration/AGENTS.md`. Do not add experimental-version integration coverage or touch ignored local configuration.

**Validation and done criteria**:

Run T002 generation checks and integration compilation from `quickstart.md`; use compile-only commands that do not execute `TestMain`. Then run `make test` and `make build` once for the completed executable change. The full suite covers fixture/command regression paths already exercised by T001; do not prepend another broad duplicate command sweep. Record actual results and live-cluster gaps. Run the Git-cleanliness boundary check in a temporary reviewed snapshot before completion, using the recipe in `quickstart.md`; do not block on committing the work before validation or weaken the guard. This unit has no remaining retired executable assets or generator targets.

## User Story 3 — Read a Consistent Support Policy (P2)

- [x] T003 [US3] Remove remaining obsolete support content in `specs/`, `tasks/`, `AGENTS.md`, `README.md`, `docs/`, `api/README.md` and tracked configuration/examples, then complete the final tracked-path/content audit and record evidence in `specs/331-simplify-version-support/progress.md`.

**Implementation scope**:

- Build one candidate inventory from tracked paths/content and new unstaged files; reuse it throughout this unit. Historical specification content is substantial: the review found 419 candidate files under `specs/` and six under `tasks/`. These are search hits, not an instruction to rewrite every document.
- Apply narrowly scoped replacements/removals and inspect their diffs. Preserve unrelated requirements and historical context. Do not reread/rewrite entire specifications, reopen completed features, create per-document tasks or delete whole historical artifacts just to remove one sentence.
- Include this feature's artifacts and all tracked source/input paths in the final audit. Classify numeric/dependency/provenance matches individually; do not exclude historical plans or API inputs wholesale. Remove actual support claims and artifacts, not unrelated numeric data. Ignored Spec Kit memory may need a focused guidance correction but is outside the tracked acceptance count.
- Keep resulting user-facing text on the retained support set, default and existing maturity labels. No retirement migration/deprecation page. CLI/homepage output already regenerated in T001 needs regeneration only if its source changes here.

**Validation and done criteria**:

Run T003 document/link/audit checks in `quickstart.md`. Reuse T002's executable evidence: no runtime tests, build or generation suite solely because prose changed. If final review finds a substantive code omission, fix it in its owner and rerun only invalidated checks, including full validation when the change warrants it. No unresolved support reference, broken changed link, or relevant failing check may remain.

## Dependencies and Validation Ownership

```text
T001 / US1: runtime + shared compatibility + current user docs + focused checks
  -> T002 / US2: generation + fixtures/integration + one full executable check
  -> T003 / US3: remaining support content + final audit + document checks
```

There are no `[P]` tasks or parallel-agent requirements. Each result is independently reviewable, while execution is sequential. T001 is a useful runtime milestone; all three are required to complete the issue.

## Consolidation Review

The previous five tasks revisited deploy/embed tests, repeated generator/consumer checks and separated small dead-code removal from deletion of its sole caller. Those boundaries did not provide enough value for their overhead.

- Move the former shared dry-run cleanup into T001: it belongs to the runtime deletion and removes one follow-up source/test pass.
- Combine generated-output/tooling and fixture/integration cleanup in T002: these are mostly bounded deletion/list updates, and their combined result permits one full validation run.
- Keep T003 separate: historical support cleanup spans hundreds of candidate documents and has a different validation method. Combining it with runtime work would make the unit impractical.
- Keep compilation-coupled identity, adapters, factories and affected tests together. A shorter checklist must not leave broken intermediate revisions.
- All FRs and meaningful regression coverage remain. Three tasks are an intended work-unit structure, not a promise that every unit fits one model turn or a fixed execution-time estimate.

## Requirement Coverage

| Requirements | Tasks |
| --- | --- |
| FR-001, FR-002, FR-003 | T001 |
| FR-004 | T001, T002 |
| FR-005 | T001, T002 |
| FR-006 | T002 |
| FR-007, FR-008 | T001, T002 |
| FR-009, FR-010 | T001, T002, T003 |
| FR-011 | T003 |
| FR-012 | Targeted checks in T001; generation and full executable checks in T002; final audit in T003 |

## Ralph Handoff

Complete one permitted story/work unit at a time. Store actual evidence, remaining scope and reusable discoveries in the existing coordinated progress/memory records. Do not repeat successful validation just to commit, and do not add a review-only follow-up task when the current unit can resolve the finding. Use Conventional Commits referencing #331 when the Ralph workflow reaches its commit step. This planning revision does not start implementation.
