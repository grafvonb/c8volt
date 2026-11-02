# Ralph Progress Log

Feature: 362-late-business-id-assignment
Started: 2026-11-02 15:21:10

---


---
## Iteration 1 - 2026-11-02 15:28
**Work Unit**: US1 T001 — completed process assignment service and public plan/execute API; partial US1 delivery
**Tasks Completed**:
- [x] T001: Implement assignment validation, copied exact-key plans, one-shot versioned submission, bounded observation, public mappings, isolated tests and library documentation.
**Tasks Remaining in Work Unit**: T002 remains in US1; T003/T004 remain in US2.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/process/api.go
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- c8volt/resource/client_test.go
- cmd/process_api_stub_test.go
- internal/domain/business_id.go
- internal/domain/business_id_test.go
- internal/services/processinstance/api.go
- internal/services/processinstance/dryrun_test.go
- internal/services/processinstance/v810/contract.go
- internal/services/processinstance/v810/service_test.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v89/contract.go
- specs/362-late-business-id-assignment/tasks.md
- c8volt/process/business_id_assignment.go
- c8volt/process/business_id_assignment_test.go
- internal/domain/process_instance_business_id_assignment.go
- internal/services/processinstance/business_id_assignment.go
- internal/services/processinstance/business_id_assignment_test.go
- internal/services/processinstance/v810/business_id_assignment.go
- internal/services/processinstance/v810/business_id_assignment_test.go
- internal/services/processinstance/v88/business_id_assignment.go
- internal/services/processinstance/v89/business_id_assignment.go
- internal/services/processinstance/waiter/business_id_assignment.go
- specs/362-late-business-id-assignment/progress.md
- specs/362-late-business-id-assignment/ralph-memory.md
**Learnings**:
- Shared validator tests first failed as expected before implementation, then passed including creation whitespace compatibility and UTF-16 boundary cases.
- Passed: env GOFLAGS= go test ./internal/domain -run TestValidateBusinessID -count=1.
- Passed: env GOFLAGS= go test ./internal/services/processinstance/... ./c8volt/process -run BusinessIDAssignment -race -count=1 (nine selected service/facade test functions with subtests).
- Passed affected-package regressions: env GOFLAGS= go test ./internal/domain ./internal/services/processinstance/... ./c8volt/process ./c8volt/resource -count=1.
- Passed repository-wide compilation: env GOFLAGS= go test ./... -run '^$' -count=1; initial failures exposed expanded-interface doubles, repaired in this unit.
- Passed gofmt and git diff --check. No generated clients or command behavior changed; CLI documentation generation and command-output validation belong to T002.
- Reviewed failure path: native classified evidence -> ordered service results/original cause -> copied facade result plus ferrors conversion. New err.Error() usage is machine evidence only; no added direct diagnostic prints or normal-level raw-error logging.
- All tests used isolated dependencies, with GOFLAGS cleared and integration-tagged suites excluded. Full make test is reserved for T004. Real-engine rules/timing remain unverified.
---

---
## Iteration 2 - 2026-11-02 15:45
**Work Unit**: US1 T002 — completed exact-key process business-ID assignment CLI and acceptance contract
**Tasks Completed**:
- [x] T002: Deliver update pi --set-bid lifecycle, views, isolated command/terminal/logging tests, source metadata, README and generated CLI docs.
**Tasks Remaining in Work Unit**: 0; US1 is complete. T003/T004 remain in US2.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- cmd/cmd_views_processinstance_assignment.go
- cmd/command_contract_test.go
- cmd/update_processinstance.go
- cmd/update_processinstance_assignment.go
- cmd/update_processinstance_assignment_logging_test.go
- cmd/update_processinstance_assignment_test.go
- docs/cli/c8volt_update.md
- docs/cli/c8volt_update_process-instance.md
- docs/cli/command-tree.md
- docs/index.md
- specs/362-late-business-id-assignment/progress.md
- specs/362-late-business-id-assignment/ralph-memory.md
- specs/362-late-business-id-assignment/tasks.md
**Learnings**:
- First command assertion failed on the absent flag before implementation; the completed workflow validates full input and submits once per distinct exact key with copied approval scope/value.
- Passed targeted command validation: env GOFLAGS= go test ./cmd -run 'Test.*(BusinessIDAssignment|UpdateProcessInstance)' -count=1. Updated capability/help expectations for assignment and clarified variable-only payload requirements.
- Passed final targeted race validation: env GOFLAGS= GORACE=atexit_sleep_ms=0 go test ./cmd -run 'Test.*(BusinessIDAssignment|UpdateProcessInstance)' -race -count=1.
- Passed full CLI package race suite: env GOFLAGS= GORACE=atexit_sleep_ms=0 go test ./cmd -race -count=1 (565.222s). Final help wording and strengthened logging assertions also passed the targeted race rerun.
- Covered normal confirmation, no-wait acceptance, accepted-unconfirmed failure, repeated rejection, lost response without replay, partial JSON, fail-fast not-submitted slots, frozen approval, all-input validation, old adapters, empty modes/automation, keys semantics and real-terminal confirmation/decline/EOF.
- Configured/inherited plain-time and JSON logger tests capture streams separately, check INFO suppression at WARN/quiet, one bounded ERROR, DEBUG flag/config visibility and redaction, and one JSON envelope followed by EOF with preserved failure class/evidence.
- Negative mutation checks proved the failure assertions reject a direct stderr print and raw aggregate error logging. Both mutations were restored before final targeted race validation.
- Reviewed service -> facade -> view -> configured logger -> exit: original failures and partial results remain machine evidence, no cancellation formatter is used, no second envelope follows. New direct writes are selected-format keys only; human previews, summaries and verbose rows use configured INFO.
- Passed gofmt, env GOFLAGS= make docs-content and git diff --check. Docs generation also synced prior library documentation into docs/index.md.
- Validation used isolated fakes/subprocesses/terminals with no live-environment tests. Real-engine eligibility, uniqueness, durability and timing remain unverified. Repository-wide make test remains T004's final delivery gate.
---

---
## Iteration 3 - 2026-11-02 15:53
**Work Unit**: US2 T003 — completed atomic business-ID completion service and public API evidence
**Tasks Completed**:
- [x] T003: Deliver copied scalar/bulk assignment payloads, typed one-shot ports, version/value guards, scalar approval, independent evidence, isolated tests and public API documentation.
**Tasks Remaining in Work Unit**: T003 complete; T004 remains in US2 for CLI delivery and final repository-wide validation.
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/job/client.go
- c8volt/job/completion_assignment_test.go
- c8volt/job/completion_plan.go
- c8volt/job/convert.go
- c8volt/job/model.go
- internal/domain/job.go
- internal/domain/job_completion.go
- internal/services/job/completion/assignment_test.go
- internal/services/job/completion/business_id_test.go
- internal/services/job/completion/execute.go
- internal/services/job/completion/execute_test.go
- internal/services/job/completion/execution_plan.go
- internal/services/job/completion/prepare.go
- internal/services/job/completion/reconcile.go
- internal/services/job/completion/reconcile_test.go
- internal/services/job/v810/completion.go
- internal/services/job/v810/completion_assignment.go
- internal/services/job/v810/completion_assignment_test.go
- internal/services/job/v810/completion_payload_test.go
- internal/services/job/v810/completion_test.go
- internal/services/job/v810/service.go
- internal/services/job/v88/completion.go
- internal/services/job/v88/completion_assignment_test.go
- internal/services/job/v88/completion_test.go
- internal/services/job/v88/service.go
- internal/services/job/v89/completion.go
- internal/services/job/v89/completion_assignment_test.go
- internal/services/job/v89/completion_test.go
- internal/services/job/v89/service.go
- specs/362-late-business-id-assignment/progress.md
- specs/362-late-business-id-assignment/ralph-memory.md
- specs/362-late-business-id-assignment/tasks.md
**Learnings**:
- The first assignment evidence test failed to compile on missing payload/evidence fields, then passed after implementation. Shared reconciliation now preserves rejected/unknown assignment and original error when later reads report COMPLETED; ordinary completion behavior stays intact.
- Added 11 assignment test functions across shared coordinator, facade and adapters, including exact atomic wire/Unicode boundaries, omission regressions, duplicate keys/two jobs in one process, immutable preparation/execution/scalar approval, old-version/invalid/non-completion guards, listener restrictions, already-completed jobs, fail-fast slots, explicit conflicts and one-shot lost/malformed responses.
- Passed targeted service/facade selection: env GOFLAGS= go test ./internal/services/job/... ./c8volt/job -run 'Completion|WorkerOutcome' -count=1.
- Passed full affected-package race suites: env GOFLAGS= GORACE=atexit_sleep_ms=0 go test ./internal/domain ./internal/services/job/... ./c8volt/job -race -count=1. Final public guard changes additionally passed the full c8volt/job race suite; added listener validation passed TestCompletionAssignmentValidation under race.
- Passed repository-wide compilation: env GOFLAGS= go test ./... -run '^$' -count=1.
- Passed affected CLI regressions: env GOFLAGS= go test ./cmd -run 'Test.*(UpdateJob|BulkCompletion)' -count=1 (22.775s).
- Passed gofmt and git diff --check. README describes library delivery and source compatibility; no command behavior or generated clients changed. CLI docs regeneration and make test remain T004 obligations.
- Failure review: native classifier -> coordinator preserves original failure plus independent observation -> scalar/bulk facade maps optional evidence and ferrors. New err.Error() assignments are machine evidence only; no added human print/logger boundary. T004 owns exposing this evidence safely through command output.
- All validation used isolated fakes with GOFLAGS cleared, never live services. Real-engine eligibility, uniqueness, durability and timing remain unverified.
---

---
## Iteration 4 - 2026-11-02 16:11
**Work Unit**: US2 T004 — complete atomic business-ID assignment CLI delivery and final isolated validation
**Tasks Completed**:
- [x] T004: Deliver scalar/collection/stdin/selector --complete --set-bid, frozen approval, truthful evidence, output/terminal coverage, documentation and full race validation.
**Tasks Remaining in Work Unit**: 0; no feature tasks remain
**Commit**: This work-unit commit
**Files Changed**:
- README.md
- c8volt/job/completion_plan.go
- cmd/cmd_views_job.go
- cmd/update_job.go
- cmd/update_job_assignment_logging_test.go
- cmd/update_job_assignment_test.go
- cmd/update_job_batch.go
- cmd/update_job_batch_acceptance_test.go
- cmd/update_job_batch_logging_test.go
- cmd/update_job_batch_terminal_test.go
- cmd/update_job_outcome.go
- cmd/update_job_plan.go
- cmd/update_job_request.go
- cmd/update_job_test.go
- docs/cli/c8volt_update_job.md
- docs/index.md
- specs/362-late-business-id-assignment/progress.md
- specs/362-late-business-id-assignment/quickstart.md
- specs/362-late-business-id-assignment/ralph-memory.md
- specs/362-late-business-id-assignment/tasks.md
**Learnings**:
- Shared service completion already owned one-shot execution; CLI work copies intent, freezes scalar approval, and preserves errors plus results through one envelope or bounded configured logging. No backend lifecycle or generated-client changes were needed.
- Eight new command test functions and expanded paging/logger/terminal matrices cover atomic wire payloads, separate observed identity, scalar/bulk provenance, nonempty stdin, two jobs in one process, conflicts, completed targets, unknown response then COMPLETED, empty discovery/stdin, unsupported versions, flag edits and prompt/exit contracts.
- Initial tests failed on the absent flag. Final targeted command selection passed in 24.992s; the last input-guard test passed separately in 1.013s. Targeted job facade tests passed in 0.667s.
- Full env GOFLAGS= make test passed with race detection: 87 packages passed, 19 had no tests, cmd completed in 560.531s. Test target/inherited settings were checked; tagged live suites were excluded. No real engine was contacted, and eligibility/uniqueness/durability/timing remain unverified.
- Negative mutations proved the command logging assertions reject direct stderr diagnostics, raw transport logging and stdout summaries. Production code was restored before final validation.
- Failure path reviewed: native evidence -> coordinator -> facade error conversion -> one scalar/bulk view -> configured logger -> original exit classification. No new raw-error human output boundary remains.
- Command cohesion gate passed: the base file adds flag/validation/dispatch only; existing focused request/outcome/plan/batch files own their lifecycle concerns and final views own rendering.
- README/help and generated CLI/index docs updated; gofmt, git diff --check and feature local Markdown links passed. Current Handoff is terminal; no follow-up work was added.
---
