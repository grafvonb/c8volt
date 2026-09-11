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
