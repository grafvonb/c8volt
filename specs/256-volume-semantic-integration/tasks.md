# Tasks: Volume And Semantic CLI Integration Coverage

**Input**: Design documents from `/specs/256-volume-semantic-integration/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Required. This feature is an integration-test feature; story work is complete only when the new volume target or focused compile/contract check is executable.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated independently after the shared harness foundation is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel after its phase prerequisites are complete because it touches a different file or an independent scenario family
- **[Story]**: User story label from [spec.md](./spec.md)
- Every task includes exact file paths

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the target surface and documentation entry points for the opt-in volume suite.

- [x] T001 Add `integration-cli-*-volume` Make targets and shared `IT_VOLUME_TIMEOUT`/volume env plumbing in `Makefile`
- [x] T002 Document destructive volume targets, `C8VOLT_IT_VOLUME_COUNT`, `IT_GO_TEST_FLAGS=-v`, `IT_VOLUME_TIMEOUT`, baseline `IT_TIMEOUT`, and evidence locations in `integration/README.md`
- [x] T003 [P] Add volume-suite package guard and no-integration-tag harmlessness notes in `integration/cli/package_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build reusable volume harness pieces that all story-specific targets depend on.

**Critical**: No user story work should start until the volume harness, dataset model, evidence writers, and cleanliness assertions exist.

- [x] T004 Add volume target definitions, environment parsing for `C8VOLT_IT_VOLUME_COUNT`, and per-family evidence path helpers in `integration/cli/volume_harness_test.go`
- [x] T005 Add suite-owned dataset structs, marker/selector helpers, cleanup records, and dirty-cluster classification in `integration/cli/volume_data_test.go`
- [x] T006 Add volume evidence record writers for `volume-<family>.json`, `volume-data-<family>.json`, `volume-progress-<family>.json`, `volume-pipelines-<family>.json`, and `volume-ops-reports-<family>.json` in `integration/cli/volume_evidence_test.go`
- [x] T007 Add stdout cleanliness, JSON parsing, keys-only parsing, final outcome, and no-wait/submitted assertion helpers in `integration/cli/volume_assertions_test.go`
- [x] T008 Add stdin file creation, duplicate/malformed/missing key builders, and command stdin execution helpers in `integration/cli/volume_pipeline_test.go`
- [ ] T009 Add best-effort c8volt-first data seeding helpers for embedded deployment, process-instance start, variable mutation, cancellation, deletion, incident/job discovery, and proposal fallback recording in `integration/cli/volume_seed_test.go`
- [x] T010 Add ops report parsers and stable Markdown/JSON report assertion helpers in `integration/cli/volume_ops_report_test.go`
- [x] T011 Add focused compile-only validation for the new volume harness with `go test -tags=integration ./integration/cli -run '^$' -count=1` documented in `specs/256-volume-semantic-integration/quickstart.md`

**Checkpoint**: The volume harness compiles, writes empty summary/proposal evidence, and can run without invoking destructive story scenarios.

---

## Phase 3: User Story 1 - Prove Done-Is-Done Under Volume (Priority: P1) MVP

**Goal**: Operators see progress, explicit final outcomes, and observable post-condition evidence for long-running volume workflows on clean or dirty disposable clusters.

**Independent Test**: Run `make integration-cli-get-volume IT_GO_TEST_FLAGS=-v` or another single implemented volume target and confirm it creates/discovers its own data, captures progress/finality evidence, and does not rely on empty global state.

### Tests and Implementation for User Story 1

- [x] T012 [P] [US1] Implement `TestVolumeGetFamily` for paging, positive/negative filtering, totals/limits visibility, clean JSON/keys-only output, and `volume-get.json` evidence in `integration/cli/volume_get_test.go`
- [ ] T013 [P] [US1] Implement `TestVolumeWalkFamily` for multi-root walk progress, completed-root accounting, hidden-key hints, machine stdout cleanliness, and `volume-walk.json` evidence in `integration/cli/volume_walk_test.go` (partial: `integration-cli-walk-volume` is live and covers seeded multi-root walks, parent/children/flat, JSON cleanliness, enrichment flags, and listener dependency validation; true hidden-key/listener-row fixture semantics remain)
- [x] T014 [P] [US1] Implement `TestVolumeDeployEmbedRunFamily` for embedded model deploy/run volume setup, long-running finality wording, no-wait/submitted wording, and fixture proposal gaps in `integration/cli/volume_deploy_embed_run_test.go`
- [ ] T015 [US1] Extend progress assertions to record visible or durable progress facts for human and verbose modes in `integration/cli/volume_assertions_test.go`
- [x] T016 [US1] Wire `integration-cli-get-volume`, `integration-cli-walk-volume`, and `integration-cli-deploy-embed-run-volume` to `TestVolumeGetFamily`, `TestVolumeWalkFamily`, and `TestVolumeDeployEmbedRunFamily` in `Makefile`
- [x] T017 [US1] Validate the MVP target with `make integration-cli-get-volume IT_GO_TEST_FLAGS=-v` and record the command in `specs/256-volume-semantic-integration/quickstart.md`

**Checkpoint**: User Story 1 is complete when one volume family target independently proves paging/filtering or long-running progress plus explicit final outcome evidence.

---

## Phase 4: User Story 2 - Prove Critical Flag Semantics (Priority: P2)

**Goal**: Release validators can prove critical flags behave semantically under multi-record conditions instead of only proving flag acceptance.

**Independent Test**: Run a destructive family target such as `make integration-cli-update-volume IT_GO_TEST_FLAGS=-v` and verify dry-run no-mutation, confirmed post-conditions or no-wait wording, limit capping, worker accounting, fail-fast behavior, and clean machine output.

### Tests and Implementation for User Story 2

- [ ] T018 [P] [US2] Implement `TestVolumeUpdateFamily` for dry-run no-mutation, confirmed mutation post-conditions, worker/no-worker-limit accounting, fail-fast mixed outcomes, version-specific unsupported behavior, and `volume-update.json` evidence in `integration/cli/volume_update_test.go` (partial: process-instance variable update volume target is live; update job and multi-version unsupported behavior still need state/profile coverage)
- [ ] T019 [P] [US2] Implement `TestVolumeCancelFamily` for multi-process-instance dry-run, auto-confirm/force behavior, limit capping, no-wait/submitted wording, and `volume-cancel.json` evidence in `integration/cli/volume_cancel_test.go`
- [ ] T020 [P] [US2] Implement `TestVolumeDeleteFamily` for destructive delete dry-run, confirmed cleanup, stdin preview support where available, fail-fast behavior, and `volume-delete.json` evidence in `integration/cli/volume_delete_test.go`
- [ ] T021 [P] [US2] Implement `TestVolumeExpectResolveFamily` for expect limit/timeout semantics, resolve dry-run and confirmed incident handling, worker accounting, and `volume-expect-resolve.json` evidence in `integration/cli/volume_expect_resolve_test.go`
- [ ] T022 [US2] Add semantic flag coverage matrix helpers that record supported, unsupported, skipped, and proposal-backed flag cases in `integration/cli/volume_evidence_test.go`
- [ ] T023 [US2] Wire `integration-cli-update-volume`, `integration-cli-cancel-volume`, `integration-cli-delete-volume`, and `integration-cli-expect-resolve-volume` to their `TestVolume*Family` patterns in `Makefile`
- [ ] T024 [US2] Validate one destructive critical-flag target with `make integration-cli-update-volume IT_GO_TEST_FLAGS=-v` and record the command in `specs/256-volume-semantic-integration/quickstart.md`

**Checkpoint**: User Story 2 is complete when at least one destructive volume target proves dry-run safety and confirmed multi-resource semantics, and every implemented critical flag case writes explicit evidence.

---

## Phase 5: User Story 3 - Prove Pipeline And Stdin Workflows (Priority: P3)

**Goal**: Operators and scripts can pipe clean keys-only producer output into stdin-consuming commands safely in preview and confirmed workflows.

**Independent Test**: Run `make integration-cli-expect-resolve-volume IT_GO_TEST_FLAGS=-v` and inspect pipeline evidence for empty, duplicate, malformed, missing, valid, mixed, and whitespace-padded stdin cases.

### Tests and Implementation for User Story 3

- [ ] T025 [US3] Extend `TestVolumeExpectResolveFamily` with keys-only producer to stdin consumer dry-run and confirmed resolve scenarios in `integration/cli/volume_expect_resolve_test.go`
- [ ] T026 [US3] Extend `TestVolumeDeleteFamily` with keys-only producer to stdin consumer dry-run and confirmed delete scenarios where command support exists in `integration/cli/volume_delete_test.go`
- [ ] T027 [US3] Extend `TestVolumeCancelFamily` with keys-only producer to stdin consumer dry-run and confirmed cancel scenarios where command support exists in `integration/cli/volume_cancel_test.go`
- [ ] T028 [US3] Add empty stdin, duplicate key, whitespace-padded key, malformed key, missing key, valid key, and mixed-input assertions in `integration/cli/volume_pipeline_test.go`
- [ ] T029 [US3] Add pipeline proposal records for stdin-consuming command gaps and keys-only producer gaps in `integration/cli/volume_seed_test.go`
- [ ] T030 [US3] Validate pipeline behavior with `make integration-cli-expect-resolve-volume IT_GO_TEST_FLAGS=-v` and record the command in `specs/256-volume-semantic-integration/quickstart.md`

**Checkpoint**: User Story 3 is complete when at least one producer-to-consumer stdin pipeline proves keys-only cleanliness, no-hang empty input, malformed handling, dry-run safety, and confirmed mutation behavior.

---

## Phase 6: User Story 4 - Prove Ops Audit Reports (Priority: P4)

**Goal**: Maintainers can trust ops commands as audited workflows with compact output and report files that agree on discovery, planning, execution, and final outcomes.

**Independent Test**: Run one ops volume target such as `make integration-cli-ops-purge-volume IT_GO_TEST_FLAGS=-v IT_VOLUME_TIMEOUT=90m` and verify human output, JSON stdout, JSON report, Markdown report, discovery scope, statuses, accounting, notices, errors, and outcome parity.

### Tests and Implementation for User Story 4

- [ ] T031 [P] [US4] Implement `TestVolumeOpsAnalyseFamily` for user-limited discovery, page/batch reporting, dry-run report content, JSON stdout cleanliness, and `volume-ops-reports-analyse.json` evidence in `integration/cli/volume_ops_analyse_test.go`
- [ ] T032 [P] [US4] Implement `TestVolumeOpsExecuteFamily` for preview/confirmed execution reporting, step status vocabulary, mutation accounting, no-wait/submitted distinctions, and `volume-ops-reports-execute.json` evidence in `integration/cli/volume_ops_execute_test.go`
- [ ] T033 [P] [US4] Implement `TestVolumeOpsPurgeFamily` for destructive purge preview/confirmed reporting, preserve/overwrite report path behavior, retained resources, cleanup-failed records, and `volume-ops-reports-purge.json` evidence in `integration/cli/volume_ops_purge_test.go`
- [ ] T034 [P] [US4] Implement `TestVolumeOpsRepairFamily` for repair preview/confirmed reporting, worker/limit accounting, partial failures, notices/errors, and `volume-ops-reports-repair.json` evidence in `integration/cli/volume_ops_repair_test.go`
- [ ] T035 [US4] Add JSON stdout and JSON report parity checks for frozen scope, discovery completeness, final outcome, and mutation accounting in `integration/cli/volume_ops_report_test.go`
- [ ] T036 [US4] Add Markdown report section checks for command identity, workflow identity, profile context, timing, dry-run state, discovery, plan, execution, notices, errors, and final outcome in `integration/cli/volume_ops_report_test.go`
- [ ] T037 [US4] Wire `integration-cli-ops-analyse-volume`, `integration-cli-ops-execute-volume`, `integration-cli-ops-purge-volume`, and `integration-cli-ops-repair-volume` to their `TestVolumeOps*Family` patterns in `Makefile`
- [ ] T038 [US4] Validate one ops report target with `make integration-cli-ops-purge-volume IT_GO_TEST_FLAGS=-v IT_VOLUME_TIMEOUT=90m` and record the command in `specs/256-volume-semantic-integration/quickstart.md`

**Checkpoint**: User Story 4 is complete when one ops preview and one ops confirmed execution produce parseable reports with stable content and outcome parity.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Keep the suite maintainable, documented, and safe for repeated destructive reruns.

- [ ] T039 [P] Normalize all new Go files with `gofmt -w integration/cli/volume_harness_test.go integration/cli/volume_data_test.go integration/cli/volume_evidence_test.go integration/cli/volume_assertions_test.go integration/cli/volume_pipeline_test.go integration/cli/volume_seed_test.go integration/cli/volume_ops_report_test.go integration/cli/volume_get_test.go integration/cli/volume_walk_test.go integration/cli/volume_update_test.go integration/cli/volume_cancel_test.go integration/cli/volume_delete_test.go integration/cli/volume_expect_resolve_test.go integration/cli/volume_deploy_embed_run_test.go integration/cli/volume_ops_analyse_test.go integration/cli/volume_ops_execute_test.go integration/cli/volume_ops_purge_test.go integration/cli/volume_ops_repair_test.go`
- [x] T040 Run non-integration validation with `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1` and record the result in `specs/256-volume-semantic-integration/quickstart.md`
- [x] T041 Run focused integration harness compile validation with `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m` and record the result in `specs/256-volume-semantic-integration/quickstart.md`
- [ ] T042 Run at least one non-ops volume target and one ops volume target against disposable default-local profiles, then record evidence paths and outcomes in `specs/256-volume-semantic-integration/quickstart.md`
- [ ] T043 Review `proposals-command.json` and `proposals-embedded-bpmn.json` from a volume run and add any newly discovered command or embedded BPMN fixture gaps to `specs/256-volume-semantic-integration/issue-draft.md`
- [ ] T044 Confirm `integration/README.md` warns that volume targets are destructive, dirty-cluster tolerant, independent, opt-in, and separate from baseline family targets
- [ ] T045 Run `git diff --check -- Makefile integration/README.md integration/cli specs/256-volume-semantic-integration` and fix any whitespace issues in touched files
- [ ] T046 Run `GOCACHE=/tmp/c8volt-gocache make test` before commit or record the blocker in `specs/256-volume-semantic-integration/quickstart.md` and leave this task incomplete

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Phase 1 and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Phase 2 and is the MVP.
- **User Story 2 (Phase 4)**: Depends on Phase 2; can run after or alongside US1 once shared helpers exist.
- **User Story 3 (Phase 5)**: Depends on Phase 2 and benefits from US2 destructive-family files when those families are selected.
- **User Story 4 (Phase 6)**: Depends on Phase 2; can run independently from non-ops stories after shared ops report helpers exist.
- **Polish**: Depends on the desired story set being complete.

### User Story Dependencies

- **US1 (P1)**: No dependency on other stories after foundational work.
- **US2 (P2)**: No dependency on US1 after foundational work, though it may reuse seeded embedded models.
- **US3 (P3)**: Depends on at least one stdin-capable family file from US2 or existing baseline coverage.
- **US4 (P4)**: No dependency on US1-US3 after foundational ops helpers exist.

### Within Each User Story

- Add or extend helpers before scenario files that consume them.
- Create or discover suite-owned data before asserting filters, limits, progress, or reports.
- Validate dry-run no-mutation before confirmed destructive mutation.
- Write evidence before final target validation so failures remain inspectable.

---

## Parallel Execution Examples

### User Story 1

```bash
Task: "T012 [P] [US1] Implement TestVolumeGetFamily in integration/cli/volume_get_test.go"
Task: "T013 [P] [US1] Implement TestVolumeWalkFamily in integration/cli/volume_walk_test.go"
Task: "T014 [P] [US1] Implement TestVolumeDeployEmbedRunFamily in integration/cli/volume_deploy_embed_run_test.go"
```

### User Story 2

```bash
Task: "T018 [P] [US2] Implement TestVolumeUpdateFamily in integration/cli/volume_update_test.go"
Task: "T019 [P] [US2] Implement TestVolumeCancelFamily in integration/cli/volume_cancel_test.go"
Task: "T020 [P] [US2] Implement TestVolumeDeleteFamily in integration/cli/volume_delete_test.go"
Task: "T021 [P] [US2] Implement TestVolumeExpectResolveFamily in integration/cli/volume_expect_resolve_test.go"
```

### User Story 4

```bash
Task: "T031 [P] [US4] Implement TestVolumeOpsAnalyseFamily in integration/cli/volume_ops_analyse_test.go"
Task: "T032 [P] [US4] Implement TestVolumeOpsExecuteFamily in integration/cli/volume_ops_execute_test.go"
Task: "T033 [P] [US4] Implement TestVolumeOpsPurgeFamily in integration/cli/volume_ops_purge_test.go"
Task: "T034 [P] [US4] Implement TestVolumeOpsRepairFamily in integration/cli/volume_ops_repair_test.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 for `integration-cli-get-volume`.
3. Stop and validate with `make integration-cli-get-volume IT_GO_TEST_FLAGS=-v`.
4. Review `volume-get.json`, progress evidence, and proposal evidence before extending to other families.

### Incremental Delivery

1. Add foundational harness once.
2. Deliver US1 to prove done-is-done under volume.
3. Deliver US2 to prove destructive flag semantics.
4. Deliver US3 to prove stdin pipeline safety.
5. Deliver US4 to prove ops reporting and audit parity.
6. Finish polish tasks and full validation.

### Parallel Team Strategy

After Phase 2, different contributors can own independent family files while sharing only the established helper contracts. Avoid concurrent edits to `Makefile`, `integration/README.md`, and shared helper files unless coordinated.
