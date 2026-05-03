# Tasks: Improve Cluster Info Output And Version Command

**Input**: Design documents from `/specs/159-cluster-info-output/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-command-contract.md, quickstart.md

**Tests**: Required by the feature specification for topology tree output, topology JSON output, version-only output, broker version output, flat license output, license JSON output, `licence` alias behavior, and legacy command removal.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare shared cluster rendering and test fixtures without changing command behavior yet.

- [x] T001 [P] Review existing cluster command fixtures and update shared topology/license fixture helpers in `cmd/get_test.go`
- [x] T002 [P] Add cluster command output contract notes to `specs/159-cluster-info-output/contracts/cli-command-contract.md`
- [x] T003 [P] Create cluster view helper file scaffold in `cmd/cmd_views_cluster.go`
- [x] T004 [P] Confirm generated docs removal path for `docs/cli/c8volt_get_cluster-topology.md` in `specs/159-cluster-info-output/research.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add reusable formatting helpers and remove the obsolete direct command surface that affects all stories.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T005 Add sorted broker and sorted partition helper functions in `cmd/cmd_views_cluster.go`
- [x] T006 Add shared cluster summary formatting helper in `cmd/cmd_views_cluster.go`
- [x] T007 Remove direct `getClusterTopologyCmd` registration and aliases from `cmd/get_cluster_topology.go`
- [x] T008 Update `TestGetHelp` to stop expecting `cluster-topology` in `cmd/get_test.go`
- [x] T009 Remove or replace legacy topology command success/help/alias tests in `cmd/get_test.go`
- [x] T010 Add command-not-found regression test for `get cluster-topology` in `cmd/get_test.go`
- [x] T011 Update capability/discovery expectations to omit `get cluster-topology` in `cmd/command_contract_test.go` and `cmd/root_test.go`

**Checkpoint**: Legacy direct topology command is removed from command routing, tests, and discovery expectations.

---

## Phase 3: User Story 1 - Read Cluster Topology As A Tree (Priority: P1) MVP

**Goal**: `c8volt get cluster topology` renders sorted human-readable cluster topology as a tree by default.

**Independent Test**: Run `c8volt get cluster topology` against unsorted fixture data and verify cluster summary, sorted broker rows, sorted partition rows, and tree connectors.

### Tests for User Story 1

- [x] T012 [P] [US1] Add command test for default topology tree output with sorted brokers and partitions in `cmd/get_test.go`
- [x] T013 [P] [US1] Add command test for topology tree output with zero brokers or empty partitions in `cmd/get_test.go`
- [x] T014 [P] [US1] Add helper-level test for cluster topology tree rendering in `cmd/get_test.go`

### Implementation for User Story 1

- [x] T015 [US1] Implement topology tree renderer in `cmd/cmd_views_cluster.go`
- [x] T016 [US1] Wire `runGetClusterTopology` to render topology tree when `pickMode()` is not JSON in `cmd/get_cluster_topology.go`
- [x] T017 [US1] Update `get cluster topology` help text and examples for human default and `--json` in `cmd/get_cluster_topology.go`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Preserve Machine-Readable Cluster Responses (Priority: P2)

**Goal**: `--json` on topology and license commands continues to return structured result-envelope JSON.

**Independent Test**: Run topology and license commands with `--json` and verify output remains structured JSON without human tree or flat text.

### Tests for User Story 2

- [ ] T018 [P] [US2] Add or update topology `--json` command test in `cmd/get_test.go`
- [ ] T019 [P] [US2] Add or update license `--json` command test in `cmd/get_test.go`
- [ ] T020 [P] [US2] Add assertion that JSON output excludes tree connector and flat license lines in `cmd/get_test.go`

### Implementation for User Story 2

- [ ] T021 [US2] Preserve `renderJSONPayload` path for topology when `pickMode()` is JSON in `cmd/get_cluster_topology.go`
- [ ] T022 [US2] Preserve `renderJSONPayload` path for license when `pickMode()` is JSON in `cmd/get_cluster_license.go`
- [ ] T023 [US2] Confirm output mode metadata keeps JSON as machine-preferred for topology and license in `cmd/get_cluster_topology.go` and `cmd/get_cluster_license.go`

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Check Gateway And Broker Versions (Priority: P3)

**Goal**: `c8volt get cluster version` returns gateway-only output by default and broker-inclusive output with `--with-brokers`.

**Independent Test**: Run version command tests against unsorted topology fixture data and verify default gateway-only output plus sorted broker output with `--with-brokers`.

### Tests for User Story 3

- [ ] T024 [P] [US3] Add `get cluster version --help` command test in `cmd/get_test.go`
- [ ] T025 [P] [US3] Add default gateway-only version output test in `cmd/get_test.go`
- [ ] T026 [P] [US3] Add `--with-brokers` version output test with sorted brokers in `cmd/get_test.go`
- [ ] T027 [P] [US3] Add version command failure-path test reusing topology failure behavior in `cmd/get_test.go`

### Implementation for User Story 3

- [ ] T028 [US3] Add `getClusterVersionCmd` with `--with-brokers` flag in `cmd/get_cluster_version.go`
- [ ] T029 [US3] Implement cluster version renderer in `cmd/cmd_views_cluster.go`
- [ ] T030 [US3] Register version command under `getClusterCmd` and set command metadata in `cmd/get_cluster_version.go`
- [ ] T031 [US3] Update `get cluster` parent help examples to include version in `cmd/get_cluster.go`

**Checkpoint**: User Stories 1, 2, and 3 are independently functional.

---

## Phase 6: User Story 4 - Read Cluster License As Flat Information (Priority: P4)

**Goal**: `c8volt get cluster license` renders concise flat human-readable fields by default while `licence` reaches the same behavior.

**Independent Test**: Run license command tests against required-only and optional-field fixture data, then verify flat output uses domain field names and `licence` behaves like `license`.

### Tests for User Story 4

- [ ] T032 [P] [US4] Update required-field license command test to expect flat output in `cmd/get_test.go`
- [ ] T033 [P] [US4] Update optional-field license command test to expect flat output in `cmd/get_test.go`
- [ ] T034 [P] [US4] Add `licence` alias behavior test in `cmd/get_test.go`
- [ ] T035 [P] [US4] Add license `--json` alias test for `licence --json` in `cmd/get_test.go`

### Implementation for User Story 4

- [ ] T036 [US4] Implement flat license renderer in `cmd/cmd_views_cluster.go`
- [ ] T037 [US4] Wire `runGetClusterLicense` to render flat output when `pickMode()` is not JSON in `cmd/get_cluster_license.go`
- [ ] T038 [US4] Add `licence` alias to the license command in `cmd/get_cluster_license.go`
- [ ] T039 [US4] Update license help text and examples for flat default, `--json`, and alias behavior in `cmd/get_cluster_license.go`

**Checkpoint**: All user stories are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated artifacts, formatting, and validation across all cluster command changes.

- [ ] T040 [P] Update README cluster examples and command tree references in `README.md`
- [ ] T041 [P] Update docs homepage cluster examples and command tree references in `docs/index.md`
- [ ] T042 Regenerate CLI reference docs with `make docs-content`
- [ ] T043 Remove generated legacy topology doc page `docs/cli/c8volt_get_cluster-topology.md` if docs generation no longer creates it
- [ ] T044 [P] Run `gofmt` on changed Go files in `cmd/`
- [ ] T045 Run targeted validation with `GOCACHE=/tmp/c8volt-gocache go test ./cmd -count=1`
- [ ] T046 Run full repository validation with `make test`
- [ ] T047 Confirm `specs/159-cluster-info-output/quickstart.md` scenarios match final command behavior

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories because it removes legacy command routing and establishes shared sorting helpers.
- **User Story 1 (Phase 3)**: Depends on Foundational and is the MVP.
- **User Story 2 (Phase 4)**: Can run after Foundational and must remain green while US1 and US4 change defaults.
- **User Story 3 (Phase 5)**: Depends on shared topology sorting helpers from Foundational.
- **User Story 4 (Phase 6)**: Depends on shared cluster view helper structure from Foundational.
- **Polish (Phase 7)**: Depends on all desired user stories.

### User Story Dependencies

- **User Story 1 (P1)**: No dependency on later stories after Foundational.
- **User Story 2 (P2)**: No dependency on later stories after Foundational; should be checked after US1 and US4 because those stories alter human defaults.
- **User Story 3 (P3)**: Uses topology retrieval and sorting helpers but is independently testable.
- **User Story 4 (P4)**: Uses license retrieval and is independently testable.

### Parallel Opportunities

- Setup tasks T001-T004 can run in parallel.
- Command removal tests T008-T011 can be prepared in parallel after T007 scope is clear.
- Topology tests T012-T014 can run in parallel.
- JSON preservation tests T018-T020 can run in parallel.
- Version tests T024-T027 can run in parallel.
- License tests T032-T035 can run in parallel.
- README and docs source updates T040-T041 can run in parallel after command metadata stabilizes.

---

## Parallel Example: User Story 1

```text
Task: "Add command test for default topology tree output with sorted brokers and partitions in cmd/get_test.go"
Task: "Add command test for topology tree output with zero brokers or empty partitions in cmd/get_test.go"
Task: "Add helper-level test for cluster topology tree rendering in cmd/get_test.go"
```

---

## Parallel Example: User Story 3

```text
Task: "Add get cluster version --help command test in cmd/get_test.go"
Task: "Add default gateway-only version output test in cmd/get_test.go"
Task: "Add --with-brokers version output test with sorted brokers in cmd/get_test.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1 only.
3. Validate with targeted `go test ./cmd -run 'TestGetClusterTopology|TestGetHelp' -count=1`.
4. Stop if needed with a usable `get cluster topology` tree default and removed legacy direct path.

### Incremental Delivery

1. Remove the legacy direct command and establish shared helpers.
2. Deliver topology tree default.
3. Preserve and re-check JSON output.
4. Add cluster version.
5. Add flat license output and `licence` alias.
6. Refresh docs and run full validation.

### Final Validation

1. `GOCACHE=/tmp/c8volt-gocache go test ./cmd -count=1`
2. `make docs-content`
3. `make test`
