# Ralph Progress Log

Feature: 231-job-ops-workflows
Started: 2026-05-25 15:34:07

## Codebase Patterns

- `cmd/get_job.go` currently owns keyed-only validation, full contract support, and the `--error-message-limit` JSON exclusion; list/search mode should extend this local flag grammar without bypassing the facade.
- `cmd/update_job.go` builds a pre-mutation plan by loading the current job first, renders dry-run/plan output before prompting, and relies on JSON guardrails plus `--auto-confirm`/`--automation` for unattended mutations.
- `cmd/cmd_views_job.go` keeps human rows compact through flat-row helpers and sends JSON through `renderJSONPayload`; future job list and worker outcome output should reuse these view boundaries.
- `c8volt/job` remains a thin facade over `internal/services/job`: public models live in `model.go`, service/domain conversion lives in `client.go`, and errors are normalized through `ferrors.FromDomain`.
- `internal/services/job/v87` returns explicit `domain.ErrUnsupported`; v8.8/v8.9 own generated-client request construction, `common.RequirePayload`, mutation retry submission, HTTP status handling, and retry confirmation through `job/waiter`.
- Generated v8.8/v8.9 clients already expose `SearchJobsWithResponse`, `CompleteJobWithResponse`, `ThrowJobErrorWithResponse`, and `FailJobWithResponse`; generated code should not be hand-edited.
- Existing list/search patterns come from `get incident` and `get process-instance`: local flag validation first, keyed/search mutual exclusion, bounded paging, JSON collection into one document, and incremental human/keys-only rendering where supported.
- README and `docs/cli/c8volt_get_job.md`/`docs/cli/c8volt_update_job.md` currently document keyed lookup plus retry/timeout only; CLI docs are generated from command metadata via `make docs-content`.

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
