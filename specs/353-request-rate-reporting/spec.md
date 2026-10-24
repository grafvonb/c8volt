# Feature Specification: Optional Request Pacing and DEBUG Rate Reporting

**Feature Branch**: `353-request-rate-reporting`

**Created**: 2026-10-24

**Status**: Draft

**Input**: [GitHub issue #353](https://github.com/grafvonb/c8volt/issues/353), "feat(http): add optional request pacing and configurable debug rate reporting", and the agreed conversation scope: one optional speed control and configurable DEBUG reporting without changes to worker concurrency or mutation workflows.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Pace a Large Command Without Changing Its Scope (Priority: P1)

As a Camunda operator processing many resources, I want to choose how quickly c8volt starts API requests so I can reduce pressure on the environment while keeping the existing selection, preflight checks, confirmations, and mutation ordering.

**Why this priority**: Optional pacing provides a practical mitigation for excessive traffic without introducing another mutation workflow.

**Independent Test**: Execute representative cancel, delete, and purge commands against isolated fakes that record request starts. Compare paced and unpaced runs with the same inputs, outcomes, and worker settings.

**Acceptance Scenarios**:

1. **Given** a positive request rate and concurrent workers, **When** discovery, preflight, mutations, confirmation reads, or application-level retries start requests, **Then** they share one invocation-wide allowance with a burst allowance of one request.
2. **Given** an omitted or explicitly zero rate, **When** requests are ready, **Then** this feature introduces no pacing wait, including with DEBUG reporting enabled.
3. **Given** a configured rate, **When** the operator explicitly supplies `--request-rate`, **Then** the flag wins, including zero overriding a positive configured rate.
4. **Given** process-definition deletion requiring instance cleanup, **When** pacing is enabled, **Then** the same approved cleanup follows the existing required order without a new resource limit, batching stage, or confirmation question.
5. **Given** a request waiting for permission, **When** cancellation or an existing deadline stops it before dispatch, **Then** it is not sent or counted as started, and its failure identifies local waiting rather than backend rejection or accepted submission.
6. **Given** a dry run, an empty scope, or no-wait execution, **When** pacing is enabled, **Then** dry runs do not mutate, empty scopes do not prompt or mutate, and no-wait does not gain a confirmation phase.

---

### User Story 2 - Observe Traffic at a Chosen Interval (Priority: P2)

As an operator investigating traffic, I want periodic DEBUG summaries of the actual request rate, even with throttling disabled, so I can choose a suitable rate without adding noise to ordinary command output.

**Why this priority**: Measurement lets operators tune pacing using observed traffic, while configurable reporting accommodates different operating needs.

**Independent Test**: Use isolated fakes with known request counts and controlled timing. Capture results and configured diagnostics separately while varying the rate, report interval, and effective logging level.

**Acceptance Scenarios**:

1. **Given** rate zero, an effective DEBUG logger, and a positive interval, **When** the interval elapses, **Then** a compact summary reports the request attempts actually started in that window, elapsed duration, and measured requests per second without pacing traffic.
2. **Given** positive pacing and reporting settings, **When** summaries are emitted, **Then** measurement reflects actual starts, including dispatched retries, rather than the configured rate, queued requests, selected resources, or completed requests.
3. **Given** no report interval setting, **Then** the default is `5s`; **Given** `10s`, **Then** that cadence is used; **Given** `0s`, **Then** rate reporting is disabled independently of pacing.
4. **Given** DEBUG enabled by `--debug` or configured log level, **When** summaries are emitted, **Then** they use the configured logger, destination, and format, preserve existing quiet/flag precedence, and never enter result stdout. Verbose alone does not enable them.
5. **Given** invalid or negative rate/interval values, **When** configuration is validated, **Then** the command fails through the established configuration-error path before remote work.
6. **Given** completion, failure, or interruption, **When** the command ends, **Then** periodic reporting stops without delaying exit until the next interval.

### Edge Cases

- Explicit rate zero overrides a positive configured rate; an omitted flag does not erase configuration.
- Positive fractional rates such as `0.5` are valid. Non-finite numbers, negative numbers, and malformed rates are invalid.
- Negative or malformed report durations are invalid; `0s` disables reporting only.
- Idle time does not accumulate permission for a burst larger than one request.
- A dispatched attempt counts even if it fails or is rejected. Cancellation before dispatch contributes zero.
- Empty reporting windows have a measured rate of zero. Use actual elapsed window duration and avoid division by zero.
- Separate invocations have separate budgets and counts; independent processes are not coordinated.
- Low rates and many workers can consume existing deadlines while waiting locally. No deadline is silently extended or local failure presented as a remote response.
- Quiet, JSON, keys-only, and redirected output retain existing precedence and stream contracts, including one JSON envelope and available partial results where supported.
- A command shorter than its reporting interval may exit without a periodic summary; reporting never keeps it running.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Expose `--request-rate` and `http.request_rate` as the maximum average application-level Camunda API request starts per second for one invocation. Default to `0`, meaning no throttling.
- **FR-002**: Accept finite nonnegative rates, including fractional values. Reject invalid or negative rates before remote work. Explicit CLI values, including zero, MUST override configuration through established configuration resolution.
- **FR-003**: A positive rate MUST be shared across discovery, preflight, mutations, confirmation reads, and application-level retries, including concurrent workers and nested service work. Permit a burst of only one request, without separate allowances per worker, resource, or phase.
- **FR-004**: Rate zero MUST introduce no limiter-induced waiting. Measurement and its reporting interval MUST NOT introduce rate-based waiting or additional remote requests.
- **FR-005**: Rate waiting MUST respect existing cancellation and deadlines without extending them. A pre-dispatch failure MUST remain identifiable as local waiting and not sent, rather than backend rejection or acceptance. Pacing MUST NOT introduce additional retries or change existing retry eligibility/budgets.
- **FR-006**: Count attempts when they actually start, after any pacing wait. Count every dispatched application-level retry separately; exclude queued work and resources merely selected for mutation.
- **FR-007**: Expose `http.request_rate_report_interval`, default `5s`. Positive durations control periodic reporting; `0s` disables it. Invalid or negative durations MUST fail configuration validation before remote work.
- **FR-008**: Reporting MUST work with pacing enabled or disabled, and MUST emit only when the effective configured logger admits DEBUG. Use its configured destination and active format and preserve existing quiet/flag precedence. Verbose alone MUST NOT enable reporting.
- **FR-009**: Each summary MUST include the window's count of actual request starts, actual elapsed duration, and measured requests per second. Window counts MUST be consistent across concurrent callers without double-counting or losing starts at boundaries. Empty windows report zero traffic.
- **FR-010**: Add compact window summaries, not new per-request logs. Summaries MUST omit resource keys, URLs, request/response bodies, credentials, and raw transport errors. Preserve diagnostic redaction and keep stdout exclusively for selected-format results.
- **FR-011**: Counting/reporting MUST stay lightweight: zero additional remote calls, deliberate reporting-related waits in request execution, or added per-request output. Disabling both pacing and reporting leaves no periodic reporting activity. Reporting MUST stop with the invocation without waiting for the next interval.
- **FR-012**: Preserve selected scopes, dry-run behavior, preflight checks, confirmation questions, cancellation/deletion ordering, no-wait semantics, worker settings, and existing error/exit classifications. Do not add batching stages, resource-selection limits, or nested-concurrency changes.
- **FR-013**: Preserve human, JSON, keys-only, and quiet output contracts, including successful empty results, available partial results, original cancellation/timeout classifications, and exactly one shared JSON envelope where supported. New diagnostics MUST use the configured logger without duplicate emission or stdout contamination.
- **FR-014**: Update user-facing help, configuration guidance, README examples, and affected generated CLI documentation to explain defaults, overrides, invocation-wide scope, request starts versus concurrency, and existing deadlines. Do not promise elimination of all backpressure or GOAWAY failures.
- **FR-015**: Validate with isolated fakes: concurrent pacing, zero-rate pass-through, retries, interval control, accurate counting, cancellation/deadlines before dispatch, and representative cancel/delete/purge execution. Capture stdout and configured diagnostic destinations separately; cover DEBUG from flags and configuration, verbose/quiet precedence, and plain/structured formats. No live-environment validation is an implementation completion gate.

### Key Entities

- **Request Rate Setting**: Invocation-wide speed allowance in request starts per second; zero disables pacing and an explicit CLI value overrides configuration.
- **Reporting Interval**: Duration between DEBUG summaries; zero disables reporting without affecting pacing.
- **Traffic Window**: A reporting period with actual elapsed duration, dispatched-attempt count, and measured average rate.
- **Request Attempt**: One application-level Camunda API attempt, counted at dispatch whether it later succeeds or fails; a request waiting for permission has not started.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Under continuously available fake work, request starts obey the shared rate with at most one request of burst allowance regardless of worker count. For rate R over interval T seconds, starts do not exceed 1 + R × T, subject only to documented timing precision in the isolated check.
- **SC-002**: At rate zero, enabling DEBUG reporting or changing its positive interval adds zero pacing waits and zero remote calls compared with the same operation without reporting.
- **SC-003**: Across controlled windows, each dispatched application-level attempt is counted exactly once, retries included; requests cancelled before dispatch contribute zero. Rates equal counts divided by actual elapsed duration to displayed precision.
- **SC-004**: Reporting follows both default and custom positive intervals. Interval zero and non-DEBUG effective logging produce zero rate summaries. Command exit does not wait for a reporting interval.
- **SC-005**: Representative cancel, process-instance delete, process-definition delete, and purge cases preserve targets, existing ordering constraints, confirmations, result formats, and exit classifications with pacing/reporting enabled, including dry-run, empty-scope, no-wait, failure, and interruption cases.
- **SC-006**: All invalid configuration cases fail before remote work. All tested pre-dispatch cancellations/deadline failures send zero affected requests and identify local waiting rather than backend rejection.

## Assumptions

- Issue #353 and the agreed conversation scope are authoritative. The earlier nested-concurrency proposal was explicitly excluded and is not part of this feature.
- Settings cover Camunda API traffic generated by the command's existing shared request path. Independent identity-provider traffic and transport-internal replays not exposed as application-level attempts are outside this contract.
- Operators choose a suitable rate. Ten requests per second is an example, not a universally safe throughput guarantee.
- The report interval is configuration-only here; no extra interval CLI flag, burst setting, or concurrency setting is introduced.
- Existing request/workflow deadlines remain authoritative and include applicable local waiting. Existing worker and timeout settings remain operator choices; this feature does not automatically tune them.
- Reporting is periodic and ends with the command. This issue does not introduce a mandatory final partial-window summary.
- Reuse established diagnostic formatting and configuration/profile precedence. No new command result schema or persistence format is required for traffic summaries.
- Adaptive throttling, broader mutation retry/reconciliation policies, checkpoint/resume, native batch migration, unrelated pagination/partial-result repairs, and live-environment testing are out of scope.
