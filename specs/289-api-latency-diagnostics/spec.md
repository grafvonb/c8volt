# Feature Specification: API Latency Diagnostics

**Feature Branch**: `289-api-latency-diagnostics`

**Created**: 2026-09-01

**Status**: Draft

**Input**: GitHub issue [#289: feat(ops): add read-only API latency analysis and active latency test](https://github.com/grafvonb/c8volt/issues/289)

## Clarifications

### Session 2026-09-01

- Q: What should the active test do when preflight determines that complete cleanup is unsupported? → A: Abort before mutation; allow an explicit `--no-cleanup` rerun.
- Q: When all planned stages finish but measurements contain latency anomalies or request errors, should the command still return a successful process exit? → A: Yes; completed diagnostics with usable findings exit successfully.
- Q: How should read-only analysis obtain resource keys for direct keyed-read measurements without expanding the issue's limited flag set? → A: Automatically reuse keys returned by measured searches.
- Q: What should `--count` represent when a primary measurement triggers follow-up reads or search-visibility checks? → A: Count primary samples; bound and disclose derived follow-ups.
- Q: Should the new latency commands follow the existing shared `ops` report contract, including `--report-format`? → A: Reuse the existing shared `ops` report contract exactly; introduce no new report behavior.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Diagnose Read Latency Safely (Priority: P1)

As an operator investigating slow or timed-out Camunda requests, I want to run a strictly read-only latency analysis so that I can compare cluster control reads with query reads without changing cluster state.

**Why this priority**: This is the safest diagnostic path and provides immediate value in production environments where mutation is unacceptable.

**Independent Test**: Run `c8volt ops analyse api-latency` against an environment with observable control-path and query-path behavior, verify that no Camunda resources are created or changed, and verify that the result compares the measured paths and states the limits of read-only evidence.

**Acceptance Scenarios**:

1. **Given** a reachable compatible Camunda environment, **When** an operator runs the read-only analysis with default settings, **Then** the command measures representative control and query reads, performs no mutation, and reports per-stage results and findings.
2. **Given** control reads remain responsive while search reads slow down, **When** the analysis completes, **Then** the findings identify query-path or secondary-storage pressure as the likely affected area and recommend a focused next investigation.
3. **Given** both control and query reads slow down, **When** the analysis completes, **Then** the findings identify a broader connectivity, gateway, authentication, client, or cluster-pressure area without asserting a definitive root cause.
4. **Given** a measured search returns resource keys supported by the configured Camunda version, **When** the read-only analysis continues, **Then** it reuses those keys for direct keyed reads measured separately from search reads without requiring additional key-input flags.

---

### User Story 2 - Exercise End-to-End Latency Under Bounded Load (Priority: P2)

As an operator who needs evidence beyond read-only checks, I want to run a controlled active latency test so that I can compare write response, read response, and search visibility behavior under a small and predictable load.

**Why this priority**: Active testing distinguishes write-path and exporter visibility problems that read-only analysis cannot observe, but it carries mutation risk and therefore follows the safe analysis capability.

**Independent Test**: Run `c8volt ops execute api-latency-test` with a small count and worker limit, confirm the preview, and verify that the command deploys only its owned fixture, creates no more than the requested number of process instances, measures write and visibility latency, reads while writes are active, and produces a final analysis.

**Acceptance Scenarios**:

1. **Given** a reachable compatible environment with complete cleanup support, **When** an operator confirms the active test, **Then** the command creates a uniquely identified run, executes the displayed bounded stage plan without exceeding the primary-sample or derived-request limits, measures write response and search visibility, and cleans up its resources.
2. **Given** writes respond quickly but created resources become searchable only after a delay, **When** the test completes, **Then** the findings identify exporter or secondary-storage delay as the likely affected area.
3. **Given** the environment returns Camunda backpressure responses, **When** the test completes, **Then** those responses are classified distinctly from timeouts and other errors.
4. **Given** the operator selects `--dry-run`, **When** the command completes validation and displays the plan, **Then** no fixture is deployed and no process instance is created.

---

### User Story 3 - Preserve Ownership and Recover Safely (Priority: P2)

As an operator authorizing an active diagnostic, I want every created resource to be attributable to one run and cleanup to be attempted reliably so that the diagnostic cannot remove unrelated resources or silently abandon its own resources.

**Why this priority**: Safe ownership and recovery are required for the active test to be acceptable in operational environments.

**Independent Test**: Interrupt or fail an active run after resources are created, then verify that cleanup uses only the recorded keys for that run and that any remaining resources produce a partial or unsuccessful outcome with explicit recovery instructions.

**Acceptance Scenarios**:

1. **Given** an active test ends successfully, partially fails, times out, or is interrupted, **When** cleanup is enabled, **Then** cleanup is attempted for every recorded run-owned resource and no unrecorded resource is targeted.
2. **Given** the configured Camunda version cannot completely clean up the planned resources, **When** the operator requests a cleanup-enabled active test, **Then** the limitation is disclosed and the command aborts before mutation with guidance to rerun explicitly with `--no-cleanup` if intentional retention is acceptable.
3. **Given** cleanup cannot remove one or more recorded resources, **When** the command finishes, **Then** the outcome is partial or unsuccessful and the report lists the remaining keys and safe manual recovery commands.
4. **Given** the operator explicitly selects `--no-cleanup`, **When** the preview is shown, **Then** intentional retention is prominent and the final report distinguishes retained resources from cleanup failures.

---

### User Story 4 - Share Reproducible Diagnostic Evidence (Priority: P3)

As an operator collaborating with platform or support teams, I want compact human output and a safe structured report so that findings can be reviewed without exposing credentials or business data.

**Why this priority**: Reproducible reports improve handoff and comparison, while the primary diagnostic value remains available through the first three stories.

**Independent Test**: Run either operation with human output, JSON output, and each supported report format; verify that each contains the required measurements, context, findings, limitations, and final outcome while excluding protected values.

**Acceptance Scenarios**:

1. **Given** either latency operation completes, **When** human output is selected, **Then** the operator receives a compact, scan-friendly summary of stages, findings, limitations, and outcome.
2. **Given** JSON output is selected, **When** the command runs, **Then** standard output contains exactly one valid document and no progress text.
3. **Given** a Markdown or JSON report is requested, **When** the command completes or partially completes, **Then** the report captures available results, notices, cleanup state where applicable, and the final outcome.
4. **Given** credentials, process variables, business payloads, or secrets are available in the execution context, **When** output and reports are produced, **Then** none of those protected values are disclosed.
5. **Given** all planned stages complete and produce usable findings that include abnormal latency or request errors, **When** the command finishes, **Then** it returns a successful process exit and records the abnormal evidence in its outcome and findings.

### Edge Cases

- `--count` is zero, negative, or smaller than the requested worker limit.
- `--workers` is zero, negative, one, or not a power of two.
- The primary-sample budget cannot be divided evenly across stages or measurement categories, or a category requires more derived requests than other categories.
- An HTTP request times out during preflight, a measurement stage, visibility waiting, report generation, or cleanup, or the bounded visibility-polling budget is exhausted.
- Connectivity or authentication succeeds during preflight but fails during a later stage.
- A response is successful but lacks the data needed for the intended measurement.
- A measured search returns no reusable key, or a returned key is unsupported by the configured Camunda version or no longer exists when the direct read occurs.
- A process instance is accepted but never becomes visible to search before the configured visibility-polling budget is exhausted.
- A worker receives a mixture of successes, Camunda errors, HTTP errors, client deadlines, and unexpected failures.
- Partitions are unhealthy or lack leaders before or during an operation.
- The operator interrupts the active test while writes are in flight.
- Fixture deployment succeeds but one or more process-instance creations fail.
- Cleanup succeeds for some resource types but is unsupported or fails for others.
- The requested report path is unwritable, has an invalid explicit format, or conflicts with the shared `ops` overwrite policy.
- `--no-cleanup`, `--dry-run`, confirmation, automation, quiet, and structured-output behavior interact in the same invocation.
- The bounded sample contains no abnormal evidence; the result must avoid implying that overall health has been proven.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The product MUST provide two clearly separated operations: `c8volt ops analyse api-latency` for read-only analysis and `c8volt ops execute api-latency-test` for active testing.
- **FR-002**: The read-only operation MUST perform no Camunda mutation under default, custom, failure, timeout, or interruption paths.
- **FR-003**: The read-only operation MUST measure representative cluster control reads and lightweight process-definition and process-instance search reads.
- **FR-004**: The read-only operation MUST reuse keys returned by its own measured searches to measure direct keyed reads separately when the configured Camunda version supports those reads; it MUST add no key-input flags, and unavailable keyed measurements MUST be reported as unavailable rather than treated as successful evidence.
- **FR-005**: The read-only result MUST compare control-path latency with query-path latency and distinguish query-specific degradation from degradation affecting all measured reads.
- **FR-006**: Every read-only result MUST state that it cannot prove write-path health, exporter health, end-to-end process execution health, or overall cluster health.
- **FR-007**: Before any active mutation, the active operation MUST validate configuration, connectivity, version compatibility, and whether the planned resources are eligible for complete cleanup.
- **FR-008**: Every active run MUST have a unique ownership identity that is visible in the preview and final report.
- **FR-009**: The active operation MUST deploy a compatible, c8volt-owned diagnostic fixture and MUST NOT accept arbitrary user-provided mutation workloads.
- **FR-010**: The active operation MUST create no more process instances than the requested `--count` primary-sample budget, measure write-response latency, measure time until created instances become visible to search, and measure read latency while controlled writes are active.
- **FR-011**: Cleanup MUST be enabled by default for the active operation; retaining test resources MUST require the explicit `--no-cleanup` option.
- **FR-012**: The active operation MUST record the identity and key of every resource it creates before that resource can be considered eligible for cleanup.
- **FR-013**: Cleanup MUST target only resources recorded as owned by the current run and MUST NOT use tenant-wide or process-identifier-only discovery as cleanup authority.
- **FR-014**: When cleanup is enabled, it MUST be attempted after success, partial failure, request timeout, visibility-budget exhaustion, or interruption, with an independent bounded opportunity after measurement work stops.
- **FR-015**: Version-specific cleanup limitations MUST be detected before mutation and disclosed in the preview; when complete cleanup is unsupported and cleanup is enabled, the command MUST abort before mutation and explain that the operator may explicitly rerun with `--no-cleanup` to accept intentional retention.
- **FR-016**: Incomplete cleanup MUST produce a partial or unsuccessful outcome and MUST report every known remaining resource key with a safe manual recovery command.
- **FR-017**: Intentional retention through `--no-cleanup` MUST be reported distinctly from failed cleanup and MUST list the retained run-owned resources.
- **FR-018**: Before active execution, the command MUST preview fixture deployments, process-instance count, maximum workers, target tenant, cleanup behavior, unique run identity, and known version-specific cleanup limitations.
- **FR-019**: Active execution MUST require confirmation unless established automatic-confirmation or automation behavior applies.
- **FR-020**: `--dry-run` MUST validate the complete active request and display the complete execution and cleanup plan without deploying resources or creating process instances.
- **FR-021**: Both operations MUST default to `--count 20` and the established `--workers 4` (`-w 4`) option, reject non-positive values, reject a worker limit that exceeds the primary-sample count, and reject a primary-sample budget too small to exercise every displayed stage at its declared concurrency.
- **FR-022**: `--count` MUST define the maximum number of primary samples for a run; required follow-up reads and search-visibility checks MUST remain outside that count but within a deterministic upper bound, and the preview and report MUST show the primary-sample allocation and maximum derived-request count across stages and measurement categories.
- **FR-023**: Both operations MUST use bounded, closed-loop workers in which each worker waits for a response and records its result before sending its next request.
- **FR-024**: Both operations MUST use a deterministic stage sequence that begins with one worker, increases toward the requested maximum without exceeding it, and includes the requested maximum as the final stage.
- **FR-025**: Each stage MUST report attempted and successful operations, errors, timeouts, throughput, p50, p95, and maximum latency, plus latency and throughput changes relative to the preceding stage when one exists.
- **FR-026**: Responses MUST be classified sufficiently to distinguish Camunda backpressure, request timeouts, unhealthy or leaderless partitions, query-only degradation, write degradation, delayed search visibility, combined read/write degradation, and samples with no abnormal evidence.
- **FR-027**: Each finding MUST identify observed evidence, a likely affected area, confidence, and a recommended next investigation; findings MUST NOT claim a definitive infrastructure root cause when the available evidence is insufficient.
- **FR-028**: Human output MUST remain compact and scan-friendly, and existing quiet, verbose, debug, automation, confirmation, profile, tenant, and timeout behaviors MUST remain consistent with their established contracts.
- **FR-029**: JSON standard output MUST remain exactly one valid document without progress text.
- **FR-030**: Completed-result JSON output and report files MUST include a schema version, command and mutation mode, capture times, c8volt and configured Camunda versions, safe profile and tenant context, requested and actual stage configuration, stage measurements, response and error classifications, findings and confidence, notices, limitations, and final outcome. On an incomplete run, standard output MUST retain the established error envelope while a requested report preserves the available partial evidence.
- **FR-031**: Completed active-test JSON output and active-test reports MUST additionally include run ownership, created resource keys, search visibility measurements, cleanup eligibility, cleanup attempts, cleanup results, and remaining resources. On a non-success exit, any requested report MUST preserve the available active-test evidence while standard output retains the established error envelope.
- **FR-032**: A requested report MUST reuse the existing shared `ops` report contract without introducing new behavior, including Markdown and JSON rendering, extension-based format inference, optional `--report-format`, path validation, overwrite policy, partial-result preservation, and file writing; destination or format failures MUST be reported through that contract.
- **FR-033**: Standard output, diagnostics, and report files MUST NOT contain access tokens, authorization headers, client secrets, process variable values, business payloads, or unredacted configuration secrets.
- **FR-034**: Command help and user-facing documentation MUST describe the safety distinction, defaults, supported options, confirmation behavior, cleanup behavior, shared report behavior, limitations, and representative examples for both operations.
- **FR-035**: A diagnostic that completes all planned stages and produces usable findings MUST return a successful process exit even when those findings include abnormal latency or request errors; invalid requests, incomplete execution, report failure, and incomplete requested cleanup MUST return a non-success process exit.

### Key Entities

- **Diagnostic Run**: One invocation, identified by command, mutation mode, start and end times, primary-sample limit, derived-request limit, target context, actual stage plan, notices, limitations, and final outcome.
- **Load Stage**: A deterministic portion of a run with a worker count, allocated primary-sample budget, timing window, attempts, successes, failures, throughput, latency distribution, and comparison with the preceding stage.
- **Measurement**: One timed control read, query read, keyed read, write response, concurrent read, or search-visibility observation, including its category, outcome, duration, and safe error classification.
- **Finding**: A compact interpretation that connects observed evidence to a likely affected area, confidence level, limitation, and recommended next investigation.
- **Owned Resource**: A fixture deployment, process instance, or other resource created by an active run, associated with that run's unique ownership identity and recorded key.
- **Cleanup Record**: The eligibility, attempt, outcome, and recovery guidance for each owned resource, distinguishing successful cleanup, failed cleanup, and intentional retention.
- **Diagnostic Report**: A human-readable or structured record of run context, measurements, findings, ownership, cleanup, limitations, and final outcome with protected data excluded.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Across automated safety tests, 100% of read-only analyses and active-test dry runs perform zero Camunda mutations.
- **SC-002**: Across bounded-load tests, 100% of runs attempt no more than the requested primary-sample count, never exceed the previewed derived-request bound, and never exceed the requested concurrent worker limit.
- **SC-003**: For every completed stage, 100% of reports contain attempts, successes, errors, timeouts, throughput, p50, p95, and maximum latency; every stage after the first also contains a comparison with its predecessor.
- **SC-004**: In predefined diagnostic scenarios, at least 90% of findings identify the expected affected area and next investigation without making an unsupported definitive root-cause claim.
- **SC-005**: For active tests with cleanup enabled, 100% of created resources receive a recorded ownership key and a cleanup attempt after successful, failed, timed-out, and interrupted runs.
- **SC-006**: Across cleanup-failure tests, 100% of remaining known resources appear in the final report with safe manual recovery guidance and a partial or unsuccessful outcome.
- **SC-007**: Across output-safety tests, zero credentials, authorization data, process variable values, business payloads, or unredacted secrets appear in standard output or report files.
- **SC-008**: At least 90% of operators in acceptance testing can select the safe operation, interpret the highest-confidence finding, and identify the next recommended investigation on their first attempt using command help and the generated report.
- **SC-009**: For both operations, the final summary or partial report becomes available within 5 seconds after the last measurement or cleanup attempt, excluding delays imposed by the selected report destination.

## Assumptions

- The primary users are Camunda operators with an existing valid c8volt profile and permission to perform the selected operation in the target tenant.
- `--count` is a total run-level primary-sample budget, not a per-worker or per-stage count; derived follow-up requests are excluded from that count but have a deterministic upper bound, and both allocations are disclosed before execution.
- When `--workers` is not a power of two, the deterministic ramp still begins at one and ends exactly at the requested maximum without exceeding it.
- The existing global `--timeout` remains the timeout for each HTTP request. The finite primary-sample allocation and existing bounded ops backoff configuration bound diagnostic and visibility work structurally, while cleanup receives a separate bounded opportunity when cleanup is enabled.
- Automatic-confirmation and automation modes retain their established repository behavior; this feature adds no parallel confirmation mechanism.
- An explicitly requested `--no-cleanup` run may complete successfully with retained resources when retention is fully disclosed and recorded; inability to perform requested cleanup remains partial or unsuccessful.
- Report format selection, explicit format override, path validation, overwrite handling, and file writing reuse the existing shared `ops` report behavior exactly.
- Direct keyed reads reuse keys returned by measured searches and are included only when a reusable key is available and the configured Camunda version supports the lookup.
- Read-only analysis supports all configured Camunda versions, with unsupported keyed process-instance reads reported as unavailable. Active testing aborts before mutation on Camunda 8.7 because the available create/deploy responses cannot supply the exact ownership keys required for safe cleanup; Camunda 8.8 requires explicit `--no-cleanup`, while 8.9 and 8.10 support cleanup-enabled execution.
- A sample with no abnormal evidence means only that no issue was observed during that bounded run; it is not proof of overall health or capacity.

## Dependencies

- A configured and reachable Camunda environment is required for measured execution.
- Active testing depends on a version-compatible c8volt-owned diagnostic fixture and on preflight knowledge of version-specific cleanup capabilities.
- The feature depends on established c8volt profile, tenant, timeout, output, confirmation, automation, and shared `ops` report behavior remaining available.

## Out of Scope

- Continuous monitoring, watch mode, alerting, or historical trend storage.
- Capacity benchmarking, stress-to-failure testing, or unbounded request-rate generation.
- Prometheus, OpenTelemetry, direct search-store access, direct database access, Kubernetes inspection, log collection, or support-bundle generation.
- Automatic remediation or definitive infrastructure root-cause attribution.
- Arbitrary user-provided mutation workloads.
