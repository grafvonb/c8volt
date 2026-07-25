## Iteration 1 - 2026-07-25 07:58
**Work Unit**: US1 Discover Complete Command Coverage
**Tasks Completed**:
- [x] T001: Create `integration/cli/` package with build-tagged `integration/cli/harness_test.go`
- [x] T002: Add package-level suite configuration types and default environment names in `integration/cli/harness_test.go`
- [x] T003: Add binary build-once support in `integration/cli/harness_test.go`
- [x] T004: Add subprocess command runner with stdout/stderr capture in `integration/cli/harness_test.go`
- [x] T005: Add evidence workdir initialization and path helpers in `integration/cli/harness_test.go`
- [x] T006: Add `integration/cli/testdata/.gitkeep`
- [x] T007: Update `integration/README.md` with the Go suite entry point
- [x] T008: Add command inventory structs in `integration/cli/all_commands_test.go`
- [x] T009: Add coverage manifest structs in `integration/cli/all_commands_test.go`
- [x] T010: Add initial manifest entries for all 55 command nodes in `integration/cli/all_commands_test.go`
- [x] T011: Add profile selection and default-config guardrails in `integration/cli/harness_test.go`
- [x] T012: Add profile connectivity/version gate helpers in `integration/cli/harness_test.go`
- [x] T013: Add run marker generation and JSON variable payload helpers in `integration/cli/harness_test.go`
- [x] T014: Add evidence record writer for command outputs and exit codes in `integration/cli/harness_test.go`
- [x] T015: Add proposal record writers for command and embedded BPMN gaps in `integration/cli/harness_test.go`
- [x] T016: Add dirty-cluster-safe assertion helpers in `integration/cli/harness_test.go`
- [x] T017: Add common JSON, keys-only, and human-output assertion helpers in `integration/cli/harness_test.go`
- [x] T018: Run integration package compile validation
- [x] T019: Add inventory-count test for the 55 current command nodes
- [x] T020: Add missing-path and stale-path manifest tests
- [x] T021: Add missing-flag coverage test
- [x] T022: Implement live `capabilities --json` discovery and flattening
- [x] T023: Implement manifest comparison for missing paths, stale paths, flags, aliases, and output modes
- [x] T024: Write `inventory.json` and `coverage.json` evidence
- [x] T025: Verify MVP with `TestCommandInventory`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- integration/README.md
- integration/cli/all_commands_test.go
- integration/cli/harness_test.go
- integration/cli/testdata/.gitkeep
- specs/255-all-command-integration/tasks.md
- specs/255-all-command-integration/ralph-memory.md
- specs/255-all-command-integration/progress.md
**Learnings**:
- The current capabilities inventory still contains 55 command nodes, and the explicit manifest must include the `capabilities` persistent-flag list.
---
## Iteration 2 - 2026-07-25 08:36
**Work Unit**: US2 Validate Real Local Profiles
**Tasks Completed**:
- [x] T026: Add failing test that rejects explicit generated config usage
- [x] T027: Add failing profile connectivity/version evidence test
- [x] T028: Add failing read-only smoke test for `version`, `capabilities`, `config validate`, and `config test-connection`
- [x] T029: Implement profile discovery/selection from default local config behavior
- [x] T030: Implement version gate execution for selected profiles
- [x] T031: Implement read-only smoke scenarios and evidence capture
- [x] T032: Write `profiles.json` evidence with reachable, expected, and actual version fields
- [x] T033: Verify with `go test -tags=integration ./integration/cli -run 'TestProfiles|TestReadOnlySmoke' -count=1 -timeout=10m`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- integration/cli/config_test.go
- integration/cli/harness_test.go
- specs/255-all-command-integration/tasks.md
- specs/255-all-command-integration/ralph-memory.md
- specs/255-all-command-integration/progress.md
**Learnings**:
- Run integration subprocesses from the evidence workdir to avoid repo-local config files shadowing the operator's default local config.
---
## Iteration 2 - 2026-07-25 08:44
**Work Unit**: US3 Seed And Reuse Disposable Cluster Data
**Tasks Completed**:
- [x] T034: Add embedded fixture discovery test
- [x] T035: Add deploy-and-run seeded process data test
- [x] T036: Add dirty-cluster assertion test using unrelated search results
- [x] T037: Implement `embed list` and version-matched embedded fixture selection
- [x] T038: Implement `embed deploy` seeded definition setup
- [x] T039: Implement `run process-instance` seeded instance creation with run marker variables
- [x] T040: Persist seeded process definition keys, process instance keys, and resource IDs under evidence `data/`
- [x] T041: Implement cleanup tracking without requiring cleanup success
- [x] T042: Verify with `go test -tags=integration ./integration/cli -run TestSeededData -count=1 -timeout=20m`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- integration/cli/deploy_embed_run_test.go
- integration/cli/get_test.go
- integration/cli/harness_test.go
- specs/255-all-command-integration/tasks.md
- specs/255-all-command-integration/ralph-memory.md
- specs/255-all-command-integration/progress.md
**Learnings**:
- Seeded data can use version-matched SimpleUserTask embedded fixtures and parse both shared-envelope and direct JSON command payloads.
- No-tag `go test ./integration/cli -count=1` still fails because all files are integration-tagged; T091 owns that later polish check.
---
## Iteration 3 - 2026-07-25 08:52
**Work Unit**: US4 Exercise Command Families And Flags
**Tasks Completed**:
- [x] T043: Add `get` family coverage tests
- [x] T044: Add `deploy`, `embed`, and `run` family coverage tests
- [x] T045: Add `update` family coverage tests
- [x] T046: Add `cancel` family coverage tests
- [x] T047: Add `delete` family coverage tests
- [x] T048: Add `expect` and `resolve` family coverage tests
- [x] T049: Add `walk` family coverage tests
- [x] T050: Add `ops analyse` family coverage tests
- [x] T051: Add `ops execute` family coverage tests
- [x] T052: Add `ops purge` family coverage tests
- [x] T053: Add `ops repair` family coverage tests
- [x] T054: Implement `get` command-family scenarios
- [x] T055: Implement `deploy`, `embed`, and `run` command-family scenarios
- [x] T056: Implement `update` command-family scenarios
- [x] T057: Implement `cancel` command-family scenarios
- [x] T058: Implement `delete` command-family scenarios
- [x] T059: Implement `expect` and `resolve` command-family scenarios
- [x] T060: Implement `walk` command-family scenarios
- [x] T061: Implement `ops analyse` command-family scenarios
- [x] T062: Implement `ops execute` command-family scenarios
- [x] T063: Implement `ops purge` command-family scenarios
- [x] T064: Implement `ops repair` command-family scenarios
- [x] T065: Implement family-level manifest satisfaction checks
- [x] T066: Verify family slices with targeted integration test command
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- integration/cli/all_commands_test.go
- integration/cli/cancel_test.go
- integration/cli/delete_test.go
- integration/cli/deploy_embed_run_test.go
- integration/cli/expect_resolve_test.go
- integration/cli/get_test.go
- integration/cli/harness_test.go
- integration/cli/ops_analyse_test.go
- integration/cli/ops_execute_test.go
- integration/cli/ops_purge_test.go
- integration/cli/ops_repair_test.go
- integration/cli/update_test.go
- integration/cli/walk_test.go
- specs/255-all-command-integration/tasks.md
- specs/255-all-command-integration/ralph-memory.md
- specs/255-all-command-integration/progress.md
**Learnings**:
- Family coverage can run without selected profiles by proving canonical and alias help paths plus manifest-declared flag exposure.
---
