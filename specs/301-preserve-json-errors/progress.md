# Ralph Progress Log

Feature: 301-preserve-json-errors
Started: 2026-09-11 15:45:51

---

---
## Iteration 1 - 2026-09-11 15:52
**Work Unit**: US1 Actionable Validation Errors
**Tasks Completed**:
- [x] T001: Confirm feature branch, active feature resolution, and audited correction/exclusion inventory
- [x] T002: Verify toolchain and test fixtures; record the focused baseline
- [x] T003: Add independent shared subprocess error-envelope assertions
- [x] T004: Cover delete search validation in JSON and human modes with both exit policies
- [x] T005: Cover both stdin errors through all 12 full-contract callers and both exit policies
- [x] T006: Route delete search validation through the shared command error handler
- [x] T007: Supply actual command context to stdin validation and update all callers atomically
- [x] T008: Format and validate the complete US1 matrix
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- AGENTS.md
- cmd/cmd_error_envelope_test.go
- cmd/cmd_stdin_error_envelope_test.go
- cmd/delete_processinstance_error_envelope_test.go
- cmd/cmd_cli.go
- cmd/delete_processinstance.go
- cmd/get_incident.go
- cmd/get_processinstance.go
- cmd/expect_processinstance.go
- cmd/update_processinstance.go
- cmd/cancel_processinstance.go
- cmd/resolve_processinstance.go
- cmd/delete_processdefinition.go
- cmd/resolve_incident.go
- cmd/ops_analyse_slow_process_instances.go
- cmd/ops_repair_incident.go
- cmd/ops_repair_processinstance.go
- specs/301-preserve-json-errors/
**Learnings**:
- The independent subprocess matrix reproduced the missing-envelope defect before the two narrow dispatch changes and now proves immediate termination even when the process status is suppressed.
---
---
## Iteration 2 - 2026-09-11 16:14
**Work Unit**: US2 Structured Runtime Failures
**Tasks Completed**:
- [x] T009: Add the cluster runtime and malformed-response error-envelope matrix
- [x] T010: Add process-definition retrieval, selector, search, and XML validation regressions
- [x] T011: Extract the command-local embedded-list execution seam
- [x] T012: Add deterministic embedded-list failure and empty-scope regressions
- [x] T013: Route cluster runtime failures through the shared command error handler
- [x] T014: Route eligible process-definition failures through the shared command error handler
- [x] T015: Route embedded listing and no-matching-files failures through the shared command error handler
- [x] T016: Format and validate the complete US2 matrix and existing regressions
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- cmd/get_cluster_error_envelope_test.go
- cmd/get_processdefinition_error_envelope_test.go
- cmd/embed_list_error_envelope_test.go
- cmd/get_cluster_topology.go
- cmd/get_cluster_version.go
- cmd/get_cluster_license.go
- cmd/get_processdefinition.go
- cmd/embed_list.go
- specs/301-preserve-json-errors/tasks.md
- specs/301-preserve-json-errors/ralph-memory.md
- specs/301-preserve-json-errors/quickstart.md
- specs/301-preserve-json-errors/progress.md
**Learnings**:
- Existing read retries can write transient informational context to stderr in JSON mode; the regressions exclude only duplicate final error diagnostics while preserving that behavior.
---
---
## Iteration 3 - 2026-09-11 16:26
**Work Unit**: US3 Preserve Existing Workflows (partial: output-mode compatibility)
**Tasks Completed**:
- [x] T017: Add corrected validation/runtime output-mode and automation compatibility coverage
**Tasks Remaining in Work Unit**: T018-T021
**Commit**: This work-unit commit
**Files Changed**:
- cmd/cmd_error_envelope_modes_test.go
- specs/301-preserve-json-errors/tasks.md
- specs/301-preserve-json-errors/ralph-memory.md
- specs/301-preserve-json-errors/progress.md
**Learnings**:
- JSON retains precedence over quiet and keys-only, while non-JSON modes keep zero-byte stdout and the existing stderr diagnostic under both exit policies; unsupported automation still rejects before stdin validation.
---
