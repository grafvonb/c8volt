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
- [x] T031: Classify BPMN-error coverage as dry-run-covered with confirmed mutation blocked by a missing embedded catchable BPMN-error fixture
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
- Existing embedded BPMN files declare error codes but do not provide a catchable BPMN error path for confirmed `update job --throw-bpmn-error`; dry-run-covered plus skipped-prerequisite confirmed mutation is the honest current coverage.
---
## Iteration 4 - 2026-07-25 22:20
**Work Unit**: Correct integration test responsibility boundary in specs
**Tasks Completed**:
- [x] T037: Add repository-level integration test responsibility rules in `specs/integration-test-responsibility.md`
- [x] T038: Add feature-local real-state gap tracking in `specs/257-c89-real-state-integration/gaps.md`
- [x] T044: Mark inherited 255 and 256 generated proposal JSON as deprecated for future integration work
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- specs/integration-test-responsibility.md
- specs/255-all-command-integration/contracts/evidence-reporting.md
- specs/255-all-command-integration/quickstart.md
- specs/255-all-command-integration/research.md
- specs/255-all-command-integration/spec.md
- specs/256-volume-semantic-integration/contracts/evidence-progress-reporting.md
- specs/256-volume-semantic-integration/data-model.md
- specs/256-volume-semantic-integration/quickstart.md
- specs/256-volume-semantic-integration/research.md
- specs/256-volume-semantic-integration/spec.md
- specs/257-c89-real-state-integration/contracts/real-state-evidence.md
- specs/257-c89-real-state-integration/contracts/real-state-targets.md
- specs/257-c89-real-state-integration/coverage-matrix.md
- specs/257-c89-real-state-integration/data-model.md
- specs/257-c89-real-state-integration/gaps.md
- specs/257-c89-real-state-integration/plan.md
- specs/257-c89-real-state-integration/quickstart.md
- specs/257-c89-real-state-integration/ralph-memory.md
- specs/257-c89-real-state-integration/research.md
- specs/257-c89-real-state-integration/spec.md
- specs/257-c89-real-state-integration/tasks.md
- specs/257-c89-real-state-integration/progress.md
**Validation**:
- `rg -n "MUST record.*proposal|must record proposal|proposal evidence|proposal reports|proposal-backed|real-state-proposals|TestRealStateProposal|aggregate proposal|Record proposal evidence|proposal-recorded fallback" specs/255-all-command-integration specs/256-volume-semantic-integration specs/257-c89-real-state-integration specs/integration-test-responsibility.md`
- `git diff --check -- specs/255-all-command-integration specs/256-volume-semantic-integration specs/257-c89-real-state-integration specs/integration-test-responsibility.md`
**Learnings**:
- The earlier proposal evidence pattern crossed the integration-test responsibility boundary.
- Runtime tests should report observed execution truth, including skipped prerequisites and dry-run coverage.
- Missing c8volt setup commands and embedded BPMN assets now belong in spec-owned gap artifacts, starting with `gaps.md`.
---
## Iteration 5 - 2026-07-25 23:45
**Work Unit**: Remove real-state runtime proposal evidence
**Tasks Completed**:
- [x] T039: Remove proposal fields from real-state fixture evidence and add skipped-prerequisite/dry-run-covered runtime fields
- [x] T040: Replace real-state proposal helper tests with spec-owned gap artifact validation
- [x] T041: Rename reserved real-state proposal target to `integration-cli-real-state-gaps`
- [x] T042: Convert BPMN-error evidence to dry-run-covered plus skipped-prerequisite confirmed mutation
- [x] T043: Remove proposal writing from real-state jobs, incidents, and listeners
- [x] T045: Validate the correction against local and live C89 targets
**Tasks Remaining in Work Unit**: 0
**Commit**: Pending
**Files Changed**:
- Makefile
- integration/cli/harness_test.go
- integration/cli/real_state_harness_test.go
- integration/cli/real_state_data_test.go
- integration/cli/real_state_jobs_test.go
- integration/cli/real_state_incidents_test.go
- integration/cli/real_state_listeners_test.go
- integration/cli/real_state_bpmn_error_test.go
- integration/cli/real_state_gap_validation_test.go
- integration/cli/real_state_proposals_test.go
- specs/257-c89-real-state-integration/quickstart.md
- specs/257-c89-real-state-integration/tasks.md
- specs/257-c89-real-state-integration/ralph-memory.md
- specs/257-c89-real-state-integration/progress.md
**Validation**:
- `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run 'TestRealStateTargetCatalog|TestRealStateC89ProfileClassification|TestRealStateEvidenceWritersEmitArrays|TestRealStateMachineOutputAssertions|TestRealStateGapArtifactDocumentsCurrentPrerequisites' -count=1 -timeout=5m`
- `make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v`
- `make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v`
- `make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v`
- `make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v`
**Learnings**:
- Real-state runtime reports now list only real-state family, data, progress, and ops evidence files for the corrected families.
- BPMN-error dry-run remains useful real integration evidence because it uses a real suite-owned job and verifies unchanged job state afterward.
- Confirmed BPMN-error mutation should not be attempted until the missing catchable BPMN fixture and setup path from `gaps.md` exist.
---
## Iteration 6 - 2026-07-25
**Work Unit**: User Story 3 first retention/destructive slice
**Tasks Completed**:
- [x] T046: Add `TestRealStateRetentionFamily` in `integration/cli/real_state_retention_test.go`
- [x] T047: Add `TestRealStateDestructiveFamily` in `integration/cli/real_state_destructive_test.go`
- [x] T048: Wire `integration-cli-real-state-retention` and `integration-cli-real-state-destructive` in `Makefile`
- [x] T049: Implement deterministic fresh completed process-instance setup for retention scenarios
**Tasks Remaining in Work Unit**: 0
**Commit**: Pending
**Files Changed**:
- Makefile
- integration/README.md
- integration/cli/real_state_retention_test.go
- integration/cli/real_state_destructive_test.go
- specs/257-c89-real-state-integration/coverage-matrix.md
- specs/257-c89-real-state-integration/quickstart.md
- specs/257-c89-real-state-integration/tasks.md
- specs/257-c89-real-state-integration/ralph-memory.md
- specs/257-c89-real-state-integration/progress.md
**Validation**:
- `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m`
- `GOCACHE=/tmp/c8volt-gocache go test ./... -count=1`
- `make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m`
- `make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m`
**Learnings**:
- `C89_NoOpCompletion.bpmn` is enough to create completed suite-owned instances, and `--retention-days 0` produces a real non-empty retention candidate set without waiting for aged data.
- Cancel/delete can already be proven on real active suite-owned process instances with explicit-key dry-runs and confirmed post-state checks.
---
## Iteration 7 - 2026-07-26
**Work Unit**: Confirmed real-state retention deletion
**Tasks Completed**:
- [x] T050: Add `ops execute retention-policy` confirmed execution assertions with deleted and absent post-state evidence
**Tasks Remaining in Work Unit**: 0
**Commit**: This work-unit commit
**Files Changed**:
- integration/cli/real_state_retention_test.go
- specs/257-c89-real-state-integration/coverage-matrix.md
- specs/257-c89-real-state-integration/gaps.md
- specs/257-c89-real-state-integration/quickstart.md
- specs/257-c89-real-state-integration/ralph-memory.md
- specs/257-c89-real-state-integration/tasks.md
- specs/257-c89-real-state-integration/progress.md
**Validation**:
- `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m`
- `make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m`
**Learnings**:
- Confirmed retention can be tested without explicit-key selection by validating the JSON report's frozen seed, root, and affected keys, then verifying the reported affected key is absent.
---
## Iteration 8 - 2026-07-26
**Work Unit**: Real-state destructive incident purge and repair slice
**Tasks Completed**:
- [x] T051: Implement incident-selected real purge candidate setup and track remaining process-definition/orphan prerequisites in `gaps.md`
- [x] T052: Add dry-run and confirmed real-state assertions for cancel, delete, resolve submission, ops repair, and incident-selected purge
- [x] T055: Extend real-state ops report parity to ops repair and incident-selected purge
- [x] T056: Update `gaps.md` for remaining purge, resolve, and mixed-failure prerequisites
- [x] T057: Update `coverage-matrix.md` for current destructive and retention statuses
- [x] T058: Validate User Story 3 with the documented retention and destructive live targets
**Tasks Remaining in Work Unit**: 0
**Commit**: Pending
**Files Changed**:
- c8volt/incident/convert.go
- c8volt/incident/client_test.go
- internal/services/incident/v88/incidents.go
- internal/services/incident/v88/incidents_test.go
- internal/services/incident/v89/convert.go
- internal/services/incident/v89/incidents.go
- internal/services/incident/v89/incidents_test.go
- integration/cli/real_state_destructive_test.go
- specs/257-c89-real-state-integration/coverage-matrix.md
- specs/257-c89-real-state-integration/gaps.md
- specs/257-c89-real-state-integration/quickstart.md
- specs/257-c89-real-state-integration/ralph-memory.md
- specs/257-c89-real-state-integration/tasks.md
- specs/257-c89-real-state-integration/progress.md
**Validation**:
- `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/incident ./internal/services/incident/v88 ./internal/services/incident/v89 -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m`
- `make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m`
- `make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m`
**Learnings**:
- Dirty-cluster coverage exposed an incident-key selection bug: public `incident.Filter.Keys` was not mapped into the domain filter, and adapter local filtering did not guard explicit incident keys.
- `ops purge process-instances-with-incidents --inc-key` now freezes the suite-owned incident and process instance before deletion, preventing unrelated dirty data from satisfying the test.
- The current `C89_SimpleUserTaskWithIncident.bpmn` incident is self-recreating after a resolution-only command; durable clearing is correctly proven through `ops repair` after changing `hasIncident`.
---
## Iteration 9 - 2026-07-26
**Work Unit**: Real-state mixed-target and fail-fast destructive coverage
**Tasks Completed**:
- [x] T053: Implement mixed valid, missing, malformed, stale, and already-mutated target sets in `integration/cli/real_state_destructive_test.go`
- [x] T054: Add fail-fast, partial-failure accounting, and machine-output cleanliness assertions for destructive mixed scenarios
**Tasks Remaining in Work Unit**: 0
**Commit**: Pending
**Files Changed**:
- cmd/cmd_views_contract.go
- cmd/resolve_incident.go
- cmd/resolve_incident_test.go
- integration/cli/real_state_destructive_test.go
- specs/257-c89-real-state-integration/coverage-matrix.md
- specs/257-c89-real-state-integration/quickstart.md
- specs/257-c89-real-state-integration/ralph-memory.md
- specs/257-c89-real-state-integration/tasks.md
- specs/257-c89-real-state-integration/progress.md
**Validation**:
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestResolveIncidentCommand' -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./integration/cli -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test -tags=integration ./integration/cli -run '^$' -count=1 -timeout=5m`
- `make integration-cli-real-state-destructive IT_REAL_STATE_TIMEOUT=90m`
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./integration/cli -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./... -count=1`
- `make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m` (captured in `/tmp/c8volt-257-destructive-verbose.log`)
**Learnings**:
- Mixed process-instance cancel/delete paths correctly fail missing or malformed targets before mutation, so the suite asserts retained valid-key state rather than expecting partial mutation.
- `resolve incident --no-wait` gives real partial accounting for one submitted incident and one valid-shaped missing incident.
- The real partial path exposed duplicate JSON envelopes on failure; `resolve incident` now exits with the mapped failure code after rendering the partial result envelope once.
---
