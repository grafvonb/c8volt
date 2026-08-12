# Tasks: Camunda 8.10 Support

**Input**: Design documents from `/specs/273-camunda-v810-support/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: The specification explicitly requires automated version, generation, factory, adapter, command, boundary, regression, and documentation verification. Test tasks therefore precede their corresponding implementation tasks.

**Organization**: Tasks are grouped by user story so each operator/maintainer outcome can be implemented and validated as an incremental slice. All commands and paths are repository-relative.

## Phase 1: Setup (Shared Test Infrastructure)

**Purpose**: Establish isolated, reusable validation seams before changing version or generated-client behavior.

- [x] T001 Create the temporary-worktree, checksum, fake-tool, and assertion harness for V810 generation tests in `api/tests/v810_generation_test.sh`
- [x] T002 [P] Create the repository import/source-boundary test scaffold for V810 adapters and command/facade layering in `internal/services/v810_source_boundary_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Produce the pinned unified V810 client safely. This phase blocks all user stories because runtime identity must point at a reproducible, isolated contract.

**Critical**: No user-story implementation begins until the generated client compiles and protected v8.7-v8.9 trees remain unchanged.

- [x] T003 [P] Add failing target, tag, commit, output-escape, missing-tool, mutation-no-op, and atomic-publication cases to `api/tests/v810_generation_test.sh`
- [x] T004 [P] Add failing provenance schema, ordered-transformation, hash, nondeterministic-field, and second-run checks in `api/tests/v810_provenance_test.py`
- [x] T005 Add backward-compatible `--target v810` parsing and dispatch while preserving no-target behavior in `api/refresh-clients.sh`
- [x] T006 Implement temporary fetch, peeled-commit verification, bundling, ordered mutations, symbol checks, protected-tree fingerprints, provenance creation, and atomic publication in `api/generate-v810-client.sh`
- [x] T007 Generate and check in the pinned `8.10.0-alpha4` artifacts at `internal/clients/camunda/v810/camunda/client.gen.go` and `internal/clients/camunda/v810/camunda/provenance.json`
- [x] T008 Add generated-client compile and required-symbol contract tests in `internal/clients/camunda/v810/camunda/client_test.go`
- [x] T009 Run the foundational generation guards and resolve failures only in `api/refresh-clients.sh`, `api/generate-v810-client.sh`, `api/tests/`, and `internal/clients/camunda/v810/camunda/`

**Checkpoint**: The canonical V810 generation command succeeds twice with no second-run diff, while protected client trees retain their recorded content.

---

## Phase 3: User Story 1 - Select Camunda 8.10 Normally (Priority: P1) 🎯

**Goal**: Operators select one canonical `8.10` identity through existing configuration, see honest prerelease-baseline disclosure, match an 8.10 gateway, and retain the V88 default.

**Independent Test**: Configure `8.10`, `810`, `v810`, and `v8.10` (including whitespace/case variants), reject source-tag aliases, verify `8.10.0-alpha4` gateway compatibility and mismatch/unrecognizable outcomes, omit configuration to retain V88, and inspect human/JSON version output.

### Tests for User Story 1

- [x] T010 [US1] Add V810 alias, canonical string, supported/implemented-set staging, source-tag rejection, and unchanged-default tests in `toolx/version_test.go` and `config/app_test.go`
- [x] T011 [P] [US1] Add gateway `8.10`, patch, alpha4, different-minor, empty, and malformed comparison cases in `cmd/config_test.go`
- [ ] T012 [P] [US1] Add human/JSON baseline disclosure, root help, supported-version, and bootstrap behavior tests in `cmd/version_test.go`, `cmd/bootstrap_errors_test.go`, and `cmd/get_test.go`

### Implementation for User Story 1

- [x] T013 [US1] Add `toolx.V810`, canonical/alias normalization, string rendering, and supported-version membership while retaining `CurrentCamundaVersion = V88` in `toolx/version.go`
- [x] T014 [US1] Implement explicit match/mismatch/unrecognizable release-line comparison and preserve diagnostic rendering in `cmd/config_diagnostics.go` and `cmd/config_test_connection.go`
- [ ] T015 [US1] Add one V810 active-baseline metadata model sourced from the pinned client provenance in `toolx/camunda_baseline.go` and `toolx/camunda_baseline_test.go`
- [ ] T016 [US1] Add compact baseline disclosure and additive JSON fields to version/root output in `cmd/version.go` and `cmd/root.go`
- [ ] T017 [US1] Run and fix the US1 focused suites for `toolx/version_test.go`, `toolx/camunda_baseline_test.go`, `config/app_test.go`, `cmd/config_test.go`, `cmd/version_test.go`, `cmd/bootstrap_errors_test.go`, and `cmd/get_test.go`

**Checkpoint**: User Story 1 is independently demonstrable through configuration and version diagnostics; it does not claim implemented V810 workflows until User Story 2 completes.

---

## Phase 4: User Story 2 - Run Existing Workflows Natively on 8.10 (Priority: P1)

**Goal**: Construct the complete c8volt client with native V810 behavior for all eleven service families while preserving version-neutral workflows, output, confirmation, and error contracts.

**Independent Test**: Construct `c8volt.New` for V810, select all eleven V810 factories, run representative read/mutation fake-server cases, verify unsupported operations stop before mutation, and pass source-boundary scans rejecting older and removed clients.

### Tests for User Story 2

- [ ] T018 [P] [US2] Add V810 factory, interface, conversion, success, transport, and malformed-payload cases for batch operations and cluster in `internal/services/batchoperation/factory_test.go`, `internal/services/batchoperation/v810/`, `internal/services/cluster/factory_test.go`, and `internal/services/cluster/v810/`
- [ ] T019 [P] [US2] Add V810 factory, interface, paging, success, error, and mutation-confirmation cases for elements and incidents in `internal/services/element/factory_test.go`, `internal/services/element/v810/`, `internal/services/incident/factory_test.go`, and `internal/services/incident/v810/`
- [ ] T020 [P] [US2] Add V810 factory, filter-shape, success, error, and mutation-confirmation cases for jobs and process definitions in `internal/services/job/factory_test.go`, `internal/services/job/v810/`, `internal/services/processdefinition/factory_test.go`, and `internal/services/processdefinition/v810/`
- [ ] T021 [P] [US2] Add V810 factory, nested-variable selection, paging/walk/wait, value conversion, success, and error cases for process instances and variables in `internal/services/processinstance/factory_test.go`, `internal/services/processinstance/v810/`, `internal/services/variable/factory_test.go`, and `internal/services/variable/v810/`
- [ ] T022 [P] [US2] Add V810 factory, deployment visibility, tenant conversion, success, error, and confirmation cases for resources and tenants in `internal/services/resource/factory_test.go`, `internal/services/resource/v810/`, `internal/services/tenant/factory_test.go`, and `internal/services/tenant/v810/`
- [ ] T023 [P] [US2] Add V810 unified user-task factory/behavior tests that reject any Tasklist fallback and prove explicit unavailable/not-found outcomes in `internal/services/usertask/factory_test.go` and `internal/services/usertask/v810/`
- [ ] T024 [P] [US2] Add version-neutral incident state/error-type normalization tests that do not import generated enums in `internal/services/incidentfilter/incidentfilter_test.go` and extend the rejection rules in `internal/services/v810_source_boundary_test.go`
- [ ] T025 [P] [US2] Add named full-process-definition-history capability tests for V87/V88 rejection and V89/V810 acceptance before discovery/mutation in `toolx/camunda_capabilities_test.go`, `cmd/delete_processdefinition_test.go`, `cmd/ops_purge_all_processdefinitions_test.go`, and `internal/services/ops/all_process_definitions_purge_test.go`

### Implementation for User Story 2

- [ ] T026 [P] [US2] Implement native V810 batch-operation and cluster adapters plus explicit factory cases in `internal/services/batchoperation/v810/`, `internal/services/batchoperation/factory.go`, `internal/services/cluster/v810/`, and `internal/services/cluster/factory.go`
- [ ] T027 [P] [US2] Implement native V810 element and incident adapters plus explicit factory cases in `internal/services/element/v810/`, `internal/services/element/factory.go`, `internal/services/incident/v810/`, and `internal/services/incident/factory.go`
- [ ] T028 [P] [US2] Implement native V810 job and process-definition adapters plus explicit factory cases in `internal/services/job/v810/`, `internal/services/job/factory.go`, `internal/services/processdefinition/v810/`, and `internal/services/processdefinition/factory.go`
- [ ] T029 [P] [US2] Implement native V810 process-instance and variable adapters, nested V810 variable construction, and factory cases in `internal/services/processinstance/v810/`, `internal/services/processinstance/factory.go`, `internal/services/variable/v810/`, and `internal/services/variable/factory.go`
- [ ] T030 [P] [US2] Implement native V810 resource and tenant adapters plus explicit factory cases in `internal/services/resource/v810/`, `internal/services/resource/factory.go`, `internal/services/tenant/v810/`, and `internal/services/tenant/factory.go`
- [ ] T031 [P] [US2] Implement the unified-client-only V810 user-task adapter and explicit factory case without Operate/Tasklist/Admin dependencies in `internal/services/usertask/v810/` and `internal/services/usertask/factory.go`
- [ ] T032 [US2] Replace the generated v89 enum dependency with version-neutral canonical incident filter values in `internal/services/incidentfilter/incidentfilter.go`
- [ ] T033 [US2] Implement the named full-process-definition-history capability and consume it in both mutation gates in `toolx/camunda_capabilities.go`, `cmd/delete_processdefinition.go`, and `internal/services/ops/all_process_definitions_purge.go`
- [ ] T034 [US2] Add failing full V810 client-construction and CLI bootstrap tests across all factories in `c8volt/client_test.go` and `cmd/bootstrap_errors_test.go`
- [ ] T035 [US2] Finalize complete-client wiring and publish V810 in the implemented-version set only after all eleven adapters construct in `c8volt/client.go` and `toolx/version.go`
- [ ] T036 [US2] Add representative command fake-server coverage for supported reads, confirmed mutations, unsupported-before-mutation errors, and stable human/JSON/keys-only/prompt/activity behavior in `cmd/get_test.go`, `cmd/delete_test.go`, `cmd/update_test.go`, and `cmd/run_test.go`
- [ ] T037 [US2] Update V810-aware capability and unsupported-version descriptions without adding command-local backend mechanics in `cmd/get_processinstance.go`, `cmd/get_element.go`, `cmd/get_job.go`, `cmd/get_processdefinition.go`, `cmd/update.go`, `cmd/update_job.go`, `cmd/update_processinstance.go`, `cmd/delete_processdefinition.go`, and `cmd/ops_purge_all_processdefinitions.go`
- [ ] T038 [US2] Run the eleven service-suite commands from `specs/273-camunda-v810-support/quickstart.md`, plus `c8volt/client_test.go` and `internal/services/v810_source_boundary_test.go`, resolving failures only in the named owning V810/service/factory/command paths

**Checkpoint**: User Story 2 provides a complete native V810 client with operational proof and no older/removed runtime fallback.

---

## Phase 5: User Story 3 - Preserve Stable Version Behavior (Priority: P1)

**Goal**: Prove V810 is additive: V87-V89 clients and behavior, the V88 default, and the real-cluster integration harness remain unchanged.

**Independent Test**: Compare protected client and integration baselines, run all stable-version factory/command regression tests, and verify V810 production fixtures reuse C89 content only through an explicit mapping.

### Tests for User Story 3

- [ ] T039 [P] [US3] Add explicit V810-to-C89 production fixture mapping and stable-version selection tests in `toolx/fixture_compatibility_test.go`, `cmd/embed_test.go`, and `internal/services/ops/smoke_test_test.go`
- [ ] T040 [P] [US3] Extend current-default and V87-V89 selection regressions in `internal/services/batchoperation/factory_test.go`, `internal/services/cluster/factory_test.go`, `internal/services/element/factory_test.go`, `internal/services/incident/factory_test.go`, `internal/services/job/factory_test.go`, `internal/services/processdefinition/factory_test.go`, `internal/services/processinstance/factory_test.go`, `internal/services/resource/factory_test.go`, `internal/services/tenant/factory_test.go`, `internal/services/usertask/factory_test.go`, and `internal/services/variable/factory_test.go`
- [ ] T041 [P] [US3] Add repository allowlist assertions for unchanged v87-v89 generated trees and zero V810 integration assets in `api/tests/v810_repository_boundary_test.sh`

### Implementation for User Story 3

- [ ] T042 [US3] Implement the named V810-to-C89 production fixture mapping and update embedded/smoke consumers in `toolx/fixture_compatibility.go`, `cmd/embed_files.go`, and `internal/services/ops/smoke_test_service.go`
- [ ] T043 [US3] Run stable V87-V89 package/command regressions and repository boundary checks, resolving any regression only in V810/shared compatibility files identified by `api/tests/v810_repository_boundary_test.sh`

**Checkpoint**: Stable behavior and protected artifacts match their pre-feature baseline, and `integration/` has no V810 changes.

---

## Phase 6: User Story 4 - Update and Audit One 8.10 Baseline (Priority: P2)

**Goal**: Let maintainers reproduce and replace one active V810 source baseline in place without new version identities, packages, or factory wiring.

**Independent Test**: Reproduce alpha4 from provenance, run generation twice deterministically, simulate a permitted later 8.10 tag through the same target, and verify paths/identity remain V810 while protected trees stay unchanged.

### Tests for User Story 4

- [ ] T044 [P] [US4] Add in-place alpha-to-later-prerelease/final transition, deterministic rerun, rollback-on-failure, and identity/path invariance cases in `api/tests/v810_generation_test.sh` and `api/tests/v810_provenance_test.py`
- [ ] T045 [P] [US4] Add baseline tag/status update and version-output invariance tests in `toolx/camunda_baseline_test.go` and `cmd/version_test.go`

### Implementation for User Story 4

- [ ] T046 [US4] Harden target validation, deterministic provenance replacement, and atomic rollback for later 8.10 baselines without alpha/RC package naming in `api/generate-v810-client.sh`
- [ ] T047 [US4] Document the exact initial reproduction command, provenance fields, update-in-place procedure, failure guarantees, and protected paths in `api/README.md`
- [ ] T048 [US4] Execute the alpha4 reproduction twice and the update-transition fixture cases, then resolve only generator/provenance/doc discrepancies in `api/generate-v810-client.sh`, `internal/clients/camunda/v810/camunda/provenance.json`, and `api/README.md`

**Checkpoint**: A maintainer can audit and replace the active baseline without changing `V810`, `v810`, factories, or protected stable artifacts.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Align all authored/generated documentation, format touched code, and execute the repository delivery gate.

- [ ] T049 Update supported-version wording, aliases, unchanged default, prerelease baseline, and in-place update model in `README.md`, `cmd/root.go`, `cmd/version.go`, `cmd/get_processinstance.go`, `cmd/get_element.go`, `cmd/get_job.go`, `cmd/get_processdefinition.go`, `cmd/update.go`, `cmd/update_job.go`, `cmd/update_processinstance.go`, `cmd/delete_processdefinition.go`, `cmd/ops_purge_all_processdefinitions.go`, `docsgen/main.go`, and `docsgen/main_test.go`
- [ ] T050 Regenerate `docs/cli/` and `docs/index.md` with the `Makefile` `docs-content` target and verify no generated CLI documentation was hand-edited
- [ ] T051 Run `gofmt` on all touched Go files and execute the focused validation sequence documented in `specs/273-camunda-v810-support/quickstart.md`
- [ ] T052 Run the race-enabled repository gate from the `Makefile` `test` target and resolve all failures in their owning production/test files
- [ ] T053 Verify `git diff --check`, deterministic V810 regeneration, unchanged `internal/clients/camunda/v87`, `v88`, `v89`, and zero feature changes under `integration/` using `api/tests/v810_repository_boundary_test.sh`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Starts immediately.
- **Foundational (Phase 2)**: Depends on Setup; blocks all stories.
- **US1 (Phase 3)**: Depends on Foundational; establishes the configured identity and diagnostics.
- **US2 (Phase 4)**: Depends on Foundational and US1 identity; makes the configured identity operationally honest.
- **US3 (Phase 5)**: Depends on Foundational; can run alongside US1/US2 once protected baselines exist, but its final regression run follows US2.
- **US4 (Phase 6)**: Depends on Foundational generation; transition hardening can proceed alongside adapter work, but final proof follows US1 output metadata.
- **Polish (Phase 7)**: Depends on all user stories.

### User Story Dependency Graph

```text
Setup -> Foundational -> US1 -> US2 ─┐
                     ├──────> US3 ──┼─> Polish
                     └──────> US4 ──┘
```

### Within Each User Story

- Write the story's contract/behavior tests before implementation.
- Implement package-local adapters/conversions before factories and full-client wiring.
- Keep service-family pairs independent until the full-client checkpoint.
- Run the story checkpoint before moving its dependent tasks forward.

---

## Parallel Execution Examples

### User Story 1

After T010 defines expected normalization/default behavior, T011 can cover gateway diagnostics while T012 covers version/help rendering in separate command test files. T014 and T015 can then proceed independently before T016 integrates output.

### User Story 2

After US1 and the generated client are ready, dispatch service-family work in parallel:

```text
T018 -> T026  batch operations + cluster
T019 -> T027  elements + incidents
T020 -> T028  jobs + process definitions
T021 -> T029  process instances + variables
T022 -> T030  resources + tenants
T023 -> T031  user tasks
T024 -> T032  version-neutral incident filters
T025 -> T033  shared capability predicate
```

Converge at T034-T038 for full-client and command proof.

### User Story 3

T039 fixture compatibility, T040 stable factory regressions, and T041 repository boundary guards touch separate concerns and can run concurrently before T042-T043.

### User Story 4

T044 generation/provenance transitions and T045 version-output invariance can run concurrently before T046-T048.

---

## Implementation Strategy

### MVP First

The minimum honest operator slice is **US1 + US2**:

1. Complete Setup and Foundational generation.
2. Deliver canonical V810 selection/disclosure (US1).
3. Deliver the complete native V810 client and existing workflows (US2).
4. Validate both checkpoints before broad documentation polish.

US1 alone is independently testable but is not a shippable compatibility claim because selecting 8.10 without native service construction would mislead operators.

### Incremental Delivery

1. **Generation foundation**: reproducible pinned client with protected-tree proof.
2. **Identity slice**: aliases, default, gateway match, human/JSON disclosure.
3. **Operational slice**: eleven adapters, factories, capability gates, command proof.
4. **Regression slice**: stable versions, fixtures, and integration boundary.
5. **Maintenance slice**: deterministic in-place baseline updates.
6. **Release gate**: docs generation, formatting, focused checks, `make test`, final boundary audit.

### Parallel Team Strategy

Once the foundational client exists, service-family pairs can be assigned independently because each owns its `v810` package and factory tests. Keep `toolx` version/capability work, command UX, generation/provenance, and stable-boundary validation as separate workstreams, then converge only for full-client construction, docs, and repository-wide validation.
