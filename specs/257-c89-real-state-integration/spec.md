# Feature Specification: C89 Real-State Semantic Integration Coverage

**Feature Branch**: `257-c89-real-state-integration`

**Created**: 2026-07-25

**Status**: Draft

**Input**: User description: "Implement the next integration-test foundation for c8volt against Camunda 8.9. Use the existing all-command and volume suites as the foundation, but close gaps where tests currently rely on mocks, stubs, artificial no-match data, or proposal records. Focus on real jobs, incidents with jobs, listener jobs, BPMN error job paths, deterministic retention candidates, real purge and delete semantics, partial failure and fail-fast behavior, and explicit gap tracking. Keep future Camunda minor releases in mind without expanding this feature beyond the current 8.9 foundation."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Prove Real Job And Incident State (Priority: P1)

Maintainers can run focused integration targets against the default local c8volt Camunda 8.9 profile and prove that job, incident, and related-job command behavior works against real cluster state, not only against mocks, empty results, or command-accepted flags.

**Why this priority**: Jobs and incidents are the core operational state behind repair, retry, BPMN error, and long-running workflow confidence. If this layer is shallow, "done is done" remains unproven for the most important recovery paths.

**Independent Test**: Run the job and incident real-state slice on a clean or dirty disposable Camunda 8.9 cluster, seed or discover suite-owned targets, then verify non-empty job and incident rows plus post-mutation evidence for every confirmed mutation.

**Acceptance Scenarios**:

1. **Given** a clean or dirty Camunda 8.9 cluster selected by the default local c8volt configuration, **When** the job real-state slice runs, **Then** it creates or discovers suite-owned active jobs and verifies `get job` plus job update paths against non-empty real rows.
2. **Given** suite-owned incidents with related jobs, **When** repair, retry, fail, timeout, and BPMN error scenarios run, **Then** the suite verifies observable post-state or records an explicit unsupported or skipped-prerequisite runtime outcome.
3. **Given** unrelated dirty cluster data, **When** assertions count jobs or incidents, **Then** they scope evidence to suite-owned identifiers and do not depend on exact global cluster counts.

---

### User Story 2 - Prove Listener And BPMN Error Workflows (Priority: P2)

Release validators can prove listener-related flags and BPMN error job behavior with real deployed process models rather than only checking flag wiring or leaving the need as tribal knowledge.

**Why this priority**: Listener jobs and BPMN error paths are version-sensitive, workflow-specific features. They need explicit real-state evidence before operators can trust update, walk, and repair behavior around them.

**Independent Test**: Run the listener and BPMN error slice with embedded models when available. If c8volt lacks the command or embedded model needed to create the required state, the test records a skipped-prerequisite or dry-run-covered runtime outcome and the missing capability is maintained in the feature-local gap artifact.

**Acceptance Scenarios**:

1. **Given** an embedded listener-capable process model exists, **When** the listener slice runs, **Then** it produces non-empty listener job or element evidence for commands using listener-related flags.
2. **Given** a BPMN error-capable job exists, **When** `update job --throw-bpmn-error` runs, **Then** the suite verifies the command outcome and the process state produced by the BPMN error path.
3. **Given** no suitable c8volt command or embedded model exists, **When** the slice reaches that setup need, **Then** it records a precise skipped-prerequisite reason and the missing command or embedded BPMN fixture is documented in `gaps.md`.

---

### User Story 3 - Prove Destructive Ops Semantics On Real Candidates (Priority: P3)

Operators can run ops purge, ops repair, ops execute, delete, cancel, and expect-resolve real-state slices and see that destructive or corrective commands report truthful candidates, previews, confirmations, partial outcomes, and cleanup results.

**Why this priority**: These commands are explicitly destructive and often used during incidents. They must be validated with real candidates, real failures, and real dirty-cluster leftovers so reports and exit behavior stay trustworthy.

**Independent Test**: Run one destructive real-state slice that creates deterministic suite-owned candidates, executes preview and confirmed paths, and verifies deletion, retention, repair, resolve, or purge evidence after the command returns.

**Acceptance Scenarios**:

1. **Given** suite-owned purge or delete candidates, **When** dry-run mode runs, **Then** none of those candidates are mutated and the preview report identifies the planned scope.
2. **Given** suite-owned purge, retention, delete, cancel, repair, or resolve candidates, **When** confirmed execution runs, **Then** the suite verifies the resulting cluster state or explicitly records an accepted, no-wait, retained, cleanup-failed, unsupported, or skipped state.
3. **Given** mixed valid, missing, malformed, stale, and already-mutated targets, **When** fail-fast or partial-failure scenarios run, **Then** stdout, reports, and exit behavior explain which targets were attempted, stopped, skipped, failed, or completed.

---

### User Story 4 - Keep Real-State Gaps Visible And Extensible (Priority: P4)

Maintainers can inspect a single real-state coverage matrix and feature-local gap artifact to understand what is live-covered, what remains blocked by missing setup or fixture prerequisites, and what must be revisited for later Camunda minor releases.

**Why this priority**: c8volt needs a suite that can grow as Camunda 8.9 stabilizes and later minor releases arrive. Missing setup capabilities should become visible product work, not tribal memory.

**Independent Test**: Run the matrix and gap-artifact validation slice and confirm every known missing command setup or embedded BPMN fixture is represented in spec-owned artifacts with affected commands, versions, blocked runtime proof, and operator value.

**Acceptance Scenarios**:

1. **Given** existing command-family setup gaps, **When** the gap-artifact validation runs, **Then** all current family gaps, including ops repair gaps, appear in spec-owned gap artifacts.
2. **Given** a new real-state gap is discovered, **When** the suite cannot create the state through c8volt commands or embedded models, **Then** the runtime test records a precise skipped-prerequisite or dry-run outcome and the implementation updates `gaps.md` with a concrete required state and affected command list.
3. **Given** a future Camunda minor release adds or changes behavior, **When** maintainers extend the suite, **Then** the coverage matrix identifies the current 8.9 foundation without blocking future version-specific additions.

### Edge Cases

- The selected Camunda 8.9 cluster is completely clean before the run.
- The selected cluster is already dirty with unrelated process instances, incidents, jobs, deployments, reports, and previous suite leftovers.
- A previous suite run left active, completed, cancelled, cleanup-failed, retained, or partially repaired resources.
- A required state cannot be produced by an existing c8volt command and must be reported as a skipped prerequisite, not converted into test-generated backlog output.
- A required process behavior cannot be produced by an existing embedded BPMN model and must be tracked in `gaps.md`.
- A command is supported on Camunda 8.9 but unsupported or behaviorally different on another minor version.
- A target disappears, completes, or changes state between discovery and mutation.
- A real destructive command partially succeeds before hitting a malformed, missing, stale, or unsupported target.
- Human output contains progress and warnings while JSON, keys-only, and report output must remain machine-safe.
- Evidence exists for no-match behavior but not yet for non-empty real candidates.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The integration suite MUST add a Camunda 8.9 real-state coverage layer that is separate from baseline all-command and volume targets.
- **FR-002**: Real-state targets MUST use the default local c8volt configuration and MUST NOT create private config files or pass custom config paths.
- **FR-003**: Real-state targets MUST be safe to run independently and in any order against clean or dirty disposable clusters.
- **FR-004**: The suite MAY destructively mutate, delete, cancel, repair, purge, or update suite-owned and selected cluster data when required by the scenario.
- **FR-005**: Each real-state scenario MUST create or discover enough real cluster data to prove its command behavior without relying on mocks, stubs, or empty no-match evidence as the only pass condition.
- **FR-006**: Test data creation MUST prefer existing c8volt commands before direct Camunda APIs.
- **FR-007**: Any direct Camunda API setup MUST be reflected in spec-owned gap tracking when it indicates missing c8volt setup capability; runtime tests MUST record only the execution facts and setup path used.
- **FR-008**: Process definitions used by real-state tests MUST prefer existing embedded c8volt models.
- **FR-009**: Any missing embedded process behavior, including listener, BPMN error, repair, retention, or partial-failure fixtures, MUST be recorded in `gaps.md`, not generated by integration test execution.
- **FR-010**: The suite MUST prove non-empty real job coverage for read, update, repair, retry, timeout, failure, and BPMN error paths where Camunda 8.9 supports them.
- **FR-011**: The suite MUST prove incidents with related jobs where command behavior depends on job repair or retry semantics.
- **FR-012**: The suite MUST prove listener-related command behavior with real listener state or record the missing setup as skipped-prerequisite runtime evidence and maintain the gap in `gaps.md`.
- **FR-013**: The suite MUST prove deterministic retention, purge, delete, cancel, resolve, repair, and cleanup candidates with real post-state verification or explicit no-wait, accepted, retained, cleanup-failed, skipped, unsupported, or dry-run-covered outcomes.
- **FR-014**: The suite MUST prove partial failure and fail-fast behavior with mixed valid, missing, malformed, stale, and already-mutated targets where commands support those modes.
- **FR-015**: Human output evidence MUST verify visible progress, final outcome wording, warnings for destructive examples, and consistent operator vocabulary across related commands.
- **FR-016**: Machine-readable stdout evidence MUST verify that JSON and keys-only outputs are parseable and free of progress, prompt, warning, and human-summary contamination.
- **FR-017**: Ops report evidence MUST verify discovery scope, plan, execution, accounting, notices, errors, dry-run state, no-wait state, and outcome parity with stdout.
- **FR-018**: Spec-owned gap artifacts MUST include all known command-family and embedded BPMN setup gaps from the baseline, volume, and real-state suites that still block deeper live coverage.
- **FR-019**: The suite MUST maintain a feature-local coverage matrix showing each priority topic, current runtime evidence level, target real-state proof, and remaining prerequisite gaps.
- **FR-020**: Camunda version handling MUST focus on the 8.9 foundation while keeping affected-version fields explicit enough to extend for future minor releases.

### Key Entities

- **Real-State Scenario**: A focused integration case that proves command behavior using observable Camunda cluster state.
- **Suite-Owned Resource**: A deployment, process instance, job, incident, report, or cleanup candidate created or tagged by the integration run so assertions can ignore unrelated dirty data.
- **State Evidence**: Captured command output, report content, resource keys, and follow-up query results proving the target state before and after a command.
- **Destructive Candidate**: A suite-owned or deliberately selected cluster resource that a scenario may mutate, delete, cancel, resolve, repair, or purge.
- **Prerequisite Gap**: A spec-owned record for missing command setup capability or missing embedded BPMN behavior that blocks deeper live coverage.
- **Coverage Matrix**: A feature-local planning and evidence-control artifact listing current coverage status, target real-state proof, and follow-up work by topic.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every topic listed in the feature coverage matrix has an explicit status of live-covered, partially live-covered, dry-run-covered, skipped-prerequisite, no-match only, or not yet started.
- **SC-002**: At least one Camunda 8.9 real-state job scenario proves non-empty `get job` results scoped to suite-owned data.
- **SC-003**: At least one Camunda 8.9 job mutation scenario verifies observable post-state after `update job`.
- **SC-004**: At least one incident scenario proves active incidents with related job evidence or records the missing setup as skipped-prerequisite runtime evidence with a matching `gaps.md` entry.
- **SC-005**: Listener and BPMN error topics are either live-covered on Camunda 8.9 or represented by skipped-prerequisite runtime evidence with affected commands and fixture needs documented in `gaps.md`.
- **SC-006**: At least one destructive ops or command scenario proves both dry-run non-mutation and confirmed post-state on real candidates.
- **SC-007**: At least one mixed-target scenario proves partial failure or fail-fast reporting with real command execution.
- **SC-008**: The feature-local gap artifact includes ops repair gaps and all other known setup gaps that still block deeper live coverage.
- **SC-009**: Real-state targets can run independently in any order against a clean or dirty Camunda 8.9 cluster.
- **SC-010**: No real-state scenario passes solely because mocked data, stubbed services, or empty global no-match results satisfied the assertion.

## Assumptions

- The selected local Camunda 8.9 cluster is disposable for integration testing.
- Existing 255 all-command targets remain the baseline for command and flag breadth.
- Existing 256 volume targets remain the baseline for paging, limits, progress, reports, and pipeline semantics.
- This feature deepens real cluster state coverage for 8.9 first and records explicit extension points for later Camunda minor releases.
- Some desired scenarios may require new c8volt commands or new embedded BPMN models before they can be fully live-covered.
