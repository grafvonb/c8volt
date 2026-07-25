---
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
