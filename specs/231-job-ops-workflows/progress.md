# Ralph Progress Log

Feature: 231-job-ops-workflows
Started: 2026-05-25 15:34:07

## Codebase Patterns

- `cmd/get_job.go` currently owns keyed-only validation, full contract support, and the `--error-message-limit` JSON exclusion; list/search mode should extend this local flag grammar without bypassing the facade.
- `cmd/update_job.go` builds a pre-mutation plan by loading the current job first, renders dry-run/plan output before prompting, and relies on JSON guardrails plus `--auto-confirm`/`--automation` for unattended mutations.
- Existing retry/timeout plans should carry `MutationModeUpdate` so JSON dry-runs remain explicit as worker outcome plan modes are added later.
- `cmd/cmd_views_job.go` keeps human rows compact through flat-row helpers and sends JSON through `renderJSONPayload`; future job list and worker outcome output should reuse these view boundaries.
- `c8volt/job` remains a thin facade over `internal/services/job`: public models live in `model.go`, service/domain conversion lives in `client.go`, and errors are normalized through `ferrors.FromDomain`.
- `internal/services/job/v87` returns explicit `domain.ErrUnsupported`; v8.8/v8.9 own generated-client request construction, `common.RequirePayload`, mutation retry submission, HTTP status handling, and retry confirmation through `job/waiter`.
- Generated v8.8/v8.9 clients already expose `SearchJobsWithResponse`, `CompleteJobWithResponse`, `ThrowJobErrorWithResponse`, and `FailJobWithResponse`; generated code should not be hand-edited.
- Existing list/search patterns come from `get incident` and `get process-instance`: local flag validation first, keyed/search mutual exclusion, bounded paging, JSON collection into one document, and incremental human/keys-only rendering where supported.
- README and `docs/cli/c8volt_get_job.md`/`docs/cli/c8volt_update_job.md` currently document keyed lookup plus retry/timeout only; CLI docs are generated from command metadata via `make docs-content`.
- Foundational job search and worker outcome flags are now declared for command metadata but fail closed locally until their story implementations wire service behavior, preventing silent fallback to keyed lookup or retry updates.
- Expanding `internal/services/job.GenJobClient` requires updating versioned service test mocks for all generated job mutation methods, even before those methods are invoked.
- `get job` now advertises automation support because it already routes through `requireAutomationSupport` and has stable machine output.

---

---
## Iteration 1 - 2026-05-25 15:35:57 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Inspect current job command behavior in `cmd/get_job.go`, `cmd/update_job.go`, and `cmd/cmd_views_job.go`
- [x] T002: Inspect current job facade and domain models in `c8volt/job/model.go`, `c8volt/job/client.go`, and `internal/domain/job.go`
- [x] T003: Inspect current job service contracts and versioned implementations in `internal/services/job/api.go`, `internal/services/job/v87/`, `internal/services/job/v88/`, and `internal/services/job/v89/`
- [x] T004: Inspect generated v8.8/v8.9 job APIs in `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go`
- [x] T005: Inspect comparable search/list command patterns in `cmd/get_incident.go`, `cmd/get_processinstance.go`, and `internal/services/incident/`
- [x] T006: Inspect README and generated docs expectations in `README.md`, `docs/cli/c8volt_get_job.md`, and `docs/cli/c8volt_update_job.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/231-job-ops-workflows/tasks.md
- specs/231-job-ops-workflows/progress.md
**Learnings**:
- The next work unit is foundational and must add shared domain/facade/service types before any user story behavior starts.
- Existing job tests already cover keyed lookup, retry updates, timeout updates, and v8.7 unsupported behavior, so foundational changes should preserve those contracts while extending interfaces.
- Job search and worker outcome methods are present in generated v8.8/v8.9 clients; implementation should update mock service interfaces before adding service calls.
---

---
## Iteration 2 - 2026-05-25 15:45:39 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T007: Add job search query, worker outcome, and expanded job detail domain models
- [x] T008: Add matching facade request/result models for job search, worker outcomes, and mutation plans
- [x] T009: Extend job facade and service interfaces for search and worker outcome operations
- [x] T010: Add compile-time conformance updates for v8.7, v8.8, and v8.9 job services
- [x] T011: Extend command metadata tests for new get/update job flags and automation behavior
- [x] T012: Run targeted compile validation for foundational surfaces
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/job/api.go
- c8volt/job/client.go
- c8volt/job/client_test.go
- c8volt/job/model.go
- cmd/command_contract_test.go
- cmd/get_job.go
- cmd/update_job.go
- internal/domain/job.go
- internal/services/job/api.go
- internal/services/job/v87/contract.go
- internal/services/job/v87/service.go
- internal/services/job/v88/contract.go
- internal/services/job/v88/service.go
- internal/services/job/v88/service_test.go
- internal/services/job/v89/contract.go
- internal/services/job/v89/service.go
- internal/services/job/v89/service_test.go
- specs/231-job-ops-workflows/tasks.md
- specs/231-job-ops-workflows/progress.md
**Learnings**:
- Command metadata can expose future job search and worker outcome flags safely only when validation rejects those reserved flags before old behavior runs.
- The facade remains a thin mapper over `internal/services/job`; search and worker outcome service behavior is intentionally pending for later story slices.
- Targeted validation passed with `go test ./cmd ./c8volt/job ./internal/domain ./internal/services/job ./internal/services/job/v87 ./internal/services/job/v88 ./internal/services/job/v89`.
---

---
## Iteration 3 - 2026-05-25 15:50:48 CEST
**User Story**: User Story 1 - Preserve Keyed Job Lookup And Current Updates
**Tasks Completed**:
- [x] T013: Add or update keyed lookup regression tests in `cmd/get_job_test.go`
- [x] T014: Add or update retry update regression tests in `cmd/update_job_test.go`
- [x] T015: Add or update timeout update regression tests in `cmd/update_job_test.go`
- [x] T016: Add or update facade regression tests for existing update behavior in `c8volt/job/client_test.go`
- [x] T017: Add or update v8.8/v8.9 service regression tests for existing search-by-key and update requests
- [x] T018: Refactor keyed lookup validation so `--key` remains exact lookup in `cmd/get_job.go`
- [x] T019: Preserve existing retry/timeout update request parsing while preparing for new modes in `cmd/update_job.go`
- [x] T020: Preserve existing retry/timeout conversion and service delegation in `c8volt/job/client.go`
- [x] T021: Verify US1 with targeted tests for `cmd/`, `c8volt/job/`, `internal/services/job/v88/`, and `internal/services/job/v89/`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/job/client_test.go
- cmd/get_job_test.go
- cmd/update_job.go
- cmd/update_job_test.go
- internal/services/job/v88/service_test.go
- internal/services/job/v89/service_test.go
- specs/231-job-ops-workflows/tasks.md
- specs/231-job-ops-workflows/progress.md
**Learnings**:
- Keyed lookup requests include generated nil filter fields, so exact lookup tests should assert no additional non-nil filters rather than counting serialized keys.
- Targeted validation requires a writable Go build cache in this sandbox; `GOCACHE=/private/tmp/c8volt-go-build-cache` keeps test execution inside allowed paths.
- Targeted validation passed with `GOCACHE=/private/tmp/c8volt-go-build-cache go test ./cmd ./c8volt/job ./internal/services/job/v88 ./internal/services/job/v89`.
---
