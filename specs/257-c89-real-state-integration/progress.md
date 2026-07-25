---
## Iteration 1 - 2026-07-25 16:44
**Work Unit**: Setup and foundational real-state scaffolding
**Tasks Completed**:
- [x] T001: Add `IT_REAL_STATE_TIMEOUT` and real-state Make target placeholders in `Makefile`
- [x] T002: Add real-state suite overview, destructive warning, and target list in `integration/README.md`
- [x] T003: Add real-state target catalog scaffolding in `integration/cli/real_state_harness_test.go`
- [x] T004: Add real-state evidence structs and report writer scaffolding in `integration/cli/real_state_data_test.go`
- [x] T005: Add target-catalog validation for all planned real-state targets in `integration/cli/real_state_harness_test.go`
- [x] T006: Update runnable validation command examples in `specs/257-c89-real-state-integration/quickstart.md`
- [x] T007: Implement Camunda 8.9 profile selection and skip/classification helpers in `integration/cli/real_state_harness_test.go`
- [x] T008: Implement real-state family, data, progress, ops, proposal report writers, and reusable JSON/keys-only stdout cleanliness assertions in `integration/cli/real_state_data_test.go`
- [x] T009: Implement suite-owned marker, resource-key, and dirty-cluster containment helpers in `integration/cli/real_state_data_test.go`
- [x] T010: Implement reusable before-state and after-state command query helpers in `integration/cli/real_state_data_test.go`
- [x] T011: Implement embedded fixture deployment and process-instance start wrappers that reuse existing helpers in `integration/cli/real_state_data_test.go`
- [x] T012: Implement proposal fallback helpers for real-state command and embedded BPMN gaps in `integration/cli/real_state_proposals_test.go`
- [x] T013: Add compile-only and helper validation checks for the real-state scaffolding in `integration/cli/real_state_harness_test.go`
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- Makefile
- integration/README.md
- integration/cli/harness_test.go
- integration/cli/real_state_harness_test.go
- integration/cli/real_state_data_test.go
- integration/cli/real_state_proposals_test.go
- specs/257-c89-real-state-integration/quickstart.md
- specs/257-c89-real-state-integration/tasks.md
- specs/257-c89-real-state-integration/ralph-memory.md
- specs/257-c89-real-state-integration/progress.md
**Learnings**:
- Reserved real-state Make targets must fail clearly until their family tests exist, preventing false green runs from unmatched `go test -run` patterns.
- Existing volume and seeded-data helpers provide the right scaffolding pattern; no separate integration framework is needed.
---
## Iteration 2 - 2026-07-25 20:35
**Work Unit**: User Story 1 real job and incident state
**Tasks Completed**:
- [x] T014: Add `TestRealStateJobsFamily` in `integration/cli/real_state_jobs_test.go`
- [x] T015: Add `TestRealStateIncidentsFamily` in `integration/cli/real_state_incidents_test.go`
- [x] T016: Wire `integration-cli-real-state-jobs` and `integration-cli-real-state-incidents` in `Makefile`
- [x] T017: Implement service-task job fixture setup through c8volt embed deploy/run commands
- [x] T018: Add non-empty `get job` JSON, stdout cleanliness, and human output assertions
- [x] T019: Add `update job` dry-run, retries, timeout dry-run, fail, no-wait, and JSON cleanliness scenarios
- [x] T020: Implement job-backed incident setup and active incident discovery
- [x] T021: Add related-job checks for incident discovery and ops repair dry-run
- [x] T022: Record a C89 command proposal for the remaining activated-job timeout setup gap
- [x] T023: Keep embedded BPMN proposal helpers wired for missing job or incident fixture behavior
- [x] T024: Update coverage matrix job and incident rows
- [x] T025: Validate jobs and incidents targets and document expected behavior in quickstart
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- Makefile
- integration/cli/real_state_jobs_test.go
- integration/cli/real_state_incidents_test.go
- specs/257-c89-real-state-integration/coverage-matrix.md
- specs/257-c89-real-state-integration/quickstart.md
- specs/257-c89-real-state-integration/tasks.md
- specs/257-c89-real-state-integration/ralph-memory.md
- specs/257-c89-real-state-integration/progress.md
**Validation**:
- `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m`
- `make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v`
- `make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v`
**Learnings**:
- `C89_SimpleServiceTask.bpmn` creates suite-owned `CREATED` execution-listener jobs that are enough for real `get job`, retry mutation, no-wait mutation, and failure/incident setup.
- Camunda rejects confirmed timeout updates for `CREATED` jobs because timeout update requires an active/activated job; the suite records this as a command proposal and covers timeout as a dry-run plan until c8volt can create activated jobs.
- Failing a suite-owned job with retries `0` creates a real active incident with a related `jobKey`, giving stronger repair evidence than the older incident-only fixture path.
---
## Iteration 3 - 2026-07-25 21:50
**Work Unit**: User Story 2 listener and BPMN error real-state coverage
**Tasks Completed**:
- [x] T026: Add `TestRealStateListenersFamily` in `integration/cli/real_state_listeners_test.go`
- [x] T027: Add `TestRealStateBPMNErrorFamily` in `integration/cli/real_state_bpmn_error_test.go`
- [x] T028: Wire listener and BPMN-error real-state Make targets
- [x] T029: Reuse `C89_SimpleServiceTask.bpmn` for live listener fixture setup
- [x] T030: Add non-empty listener evidence for `get element`, `walk process-instance`, and `ops analyse slow-process-instances`
- [x] T031: Classify BPMN-error coverage as proposal-backed because no embedded catchable BPMN-error fixture exists
- [x] T032: Add `update job --throw-bpmn-error --dry-run` JSON cleanliness and unchanged job-state verification
- [x] T033: Record BPMN-error command setup proposal
- [x] T034: Record BPMN-error embedded BPMN proposal without modifying existing embedded models
- [x] T035: Update listener and BPMN-error matrix rows
- [x] T036: Validate listener and BPMN-error real-state targets and document behavior in quickstart
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- Makefile
- integration/cli/real_state_jobs_test.go
- integration/cli/real_state_listeners_test.go
- integration/cli/real_state_bpmn_error_test.go
- specs/257-c89-real-state-integration/coverage-matrix.md
- specs/257-c89-real-state-integration/quickstart.md
- specs/257-c89-real-state-integration/tasks.md
- specs/257-c89-real-state-integration/ralph-memory.md
- specs/257-c89-real-state-integration/progress.md
**Validation**:
- `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m`
- `make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v`
- `make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v`
**Learnings**:
- The existing service-task fixture is enough for live execution-listener enrichment; no listener proposal is needed for execution listeners.
- Traversal commands intentionally reject `--automation`, so JSON listener traversal must be run with `--json` alone.
- Existing embedded BPMN files declare error codes but do not provide a catchable BPMN error path for confirmed `update job --throw-bpmn-error`; proposal-backed dry-run is the honest current coverage.
---
