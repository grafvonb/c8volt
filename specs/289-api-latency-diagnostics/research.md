# Research: API Latency Diagnostics

## Decision: Extend the existing ops command, facade, and service path

**Rationale**: Nearby operations already keep Cobra concerns in `cmd`, expose thin public models through `c8volt/ops`, and place workflow mechanics in `internal/services/ops`. The existing ops service is already constructed with cluster, process-definition, process-instance, resource, and version dependencies needed by both latency modes.

**Alternatives considered**:

- Call generated Camunda clients from the command. Rejected because it violates repository layering and would bypass shared error and retry behavior.
- Shell out to existing c8volt commands. Rejected because it would make output, cancellation, ownership, and error aggregation unreliable.
- Add a separate latency package or versioned latency adapters. Rejected because the required differences are already represented by existing version-neutral services and capability interfaces.

## Decision: Reuse the shared ops report contract exactly

**Rationale**: `cmd/ops_contract.go` and `cmd/ops_report.go` already define report flags, Markdown/JSON format selection, path validation, overwrite safety, partial-result preservation, file permissions, and rendering helpers. The new feature supplies a command-specific payload to that machinery; it does not create a second report framework.

**Exact behavior retained**:

- `--report-format` accepts `markdown` or `json` and requires `--report-file`.
- An explicit valid format wins; `.json` infers JSON; `.md`, `.markdown`, no extension, and unknown extensions infer Markdown.
- Report files use the established `0600` mode and parent directories are not created implicitly.
- Read-only, dry-run, and pre-mutation failures preserve an existing destination. Active execution may use the existing confirmed-mutation overwrite mode only after deployment was submitted or a process-instance key was recorded.
- A requested report is attempted from available partial service evidence before an incomplete execution, report failure, or cleanup failure is returned.
- Raw report JSON remains unwrapped; JSON on stdout remains wrapped in the shared command result envelope.

**Alternatives considered**:

- Add a report-only package, new format flag, or separate report schema contract. Rejected as unnecessary and inconsistent.
- Assume every report is Markdown. Rejected because the established ops contract supports Markdown and JSON.

## Decision: Use established flag names and output guardrails

**Rationale**: Nearby ops and bulk commands standardize `--count/-n` and `--workers/-w`; command-contract tests reserve `-w` for workers. Both latency leaves therefore default to `--count 20 --workers 4`. `--keys-only` is explicitly rejected because a latency diagnostic is not a key-list command.

The active JSON path requires one of `--dry-run`, `--auto-confirm`, or `--automation`. This reuses the existing state-changing JSON guardrail so an interactive prompt cannot corrupt the one-document stdout contract. Every non-dry active run, including `--no-cleanup`, shows the preview and uses established confirmation through `shouldImplicitlyConfirm` and `confirmCmdOrAbortFn`.

**Alternatives considered**:

- Add `--max-workers`. Rejected because it duplicates the established worker flag contract.
- Allow interactive confirmation with `--json`. Rejected because the existing prompt writes to stdout.
- Advertise inherited keys-only output. Rejected because it would provide no meaningful diagnostic result.

## Decision: Treat `--timeout` as an HTTP request timeout

**Rationale**: The root flag configures `config.HTTP.Timeout` and the shared HTTP client. It is not a whole-workflow deadline. The implementation must measure the latency operators actually experience through existing service calls without silently redefining an inherited flag.

The workflow is structurally finite: primary samples, derived operations, and workers are bounded. Search-visibility polling uses normalized `config.App.Backoff` values. Planning simulates the configured delay sequence and applies the stricter of the configured retry ceiling and backoff timeout to calculate a finite maximum number of logical visibility checks before mutation. Caller cancellation still stops measurement work.

Each recorded logical measurement includes time spent in existing service/client retries. Physical HTTP attempts may therefore exceed the logical diagnostic-operation bound, and reports disclose that topology/keyed GET paths can retry while POST search paths generally do not.

**Alternatives considered**:

- Reinterpret `--timeout` as the full diagnostic deadline. Rejected because it would change a shared CLI contract.
- Add another public deadline or polling flag. Rejected because existing backoff configuration already provides bounded polling semantics.
- Instrument generated clients or disable retries for comparable samples. Rejected because diagnostics should observe normal c8volt behavior.

## Decision: Use a deterministic closed-loop stage plan

**Rationale**: `toolx/pool` already provides bounded workers that wait for one operation before accepting the next. Stage widths are `1, 2, 4, ...` and include the requested maximum exactly, without exceeding it. For example, workers 5 produces stages `1, 2, 4, 5`.

To exercise every declared width, `count` must be at least the sum of the stage widths. Validation rejects smaller budgets. Planning first reserves one sample per worker slot in each stage, then distributes the remainder evenly across stages; any remainder is assigned from the highest-concurrency stage backward. Thus the default count 20/workers 4 plan is stages `1, 2, 4` with allocations `5, 6, 9`.

Workers execute closed-loop sample cycles. A cycle never begins its next call until the previous call completed and was recorded. The requested worker value is the ceiling for all diagnostic work, including derived calls.

**Alternatives considered**:

- Build a new load generator or rate-based scheduler. Rejected because the feature is bounded diagnostics, not load testing.
- Accept count values that cannot fill every stage. Rejected because the displayed maximum concurrency would not actually be exercised.
- Allocate all remaining samples to the final stage. Rejected because even distribution yields more comparable stage evidence with little complexity.

## Decision: Define one sample cycle per mode and disclose derived bounds

**Rationale**: `count` is a run-level budget of sample cycles rather than raw transport attempts.

- A read-only primary cycle measures cluster topology, process-definition search, and process-instance search. It may derive at most one direct process-definition read and one direct process-instance read from keys returned by those searches. Its maximum derived bound is therefore `2 * count`; unsupported or missing keys produce `unavailable`, not fabricated success.
- An active primary cycle creates one process instance. Within the same worker ceiling it performs one lightweight read while other stage writes may still be active and polls for exact-key search visibility using the precomputed backoff attempt ceiling. The maximum derived bound is `count * (1 + visibility_attempt_ceiling)`. The one fixture deployment and cleanup operations are setup/recovery operations reported separately, not primary samples.

The active result records whether a read actually overlapped an in-flight write; a non-overlapping observation is not labeled concurrent. All actual primary/derived counts and ceilings are included in preview and report data.

**Alternatives considered**:

- Count every internal retry as a sample. Rejected because retry attempts are transport mechanics hidden by established services.
- Add user-supplied keys or visibility flags. Rejected by the feature scope and existing clarification.
- Run a separate read pool alongside the write pool. Rejected because it could exceed the declared worker ceiling.

## Decision: Reuse measured search keys and existing read primitives

**Rationale**: Read-only analysis can obtain process-definition and process-instance keys from its own measured searches and immediately perform version-supported keyed reads. This avoids new key flags and keeps the test representative of a normal search-to-detail workflow. Process-instance keyed reads are reported unavailable on Camunda 8.7; process-definition keyed reads remain available.

Missing results, unsupported reads, and resources that disappear between search and read are reported as unavailable evidence rather than execution errors when the stage otherwise completes.

## Decision: Reuse the version-matched SimpleUserTask fixture

**Rationale**: `C88_SimpleUserTask.bpmn`, `C89_SimpleUserTask.bpmn`, and `C810_SimpleUserTask.bpmn` are already embedded, active, small, and free of business variables. They create less incidental work than the multi-subprocess smoke-test fixture and require no new artifact or deployment dependency graph.

The active run generates a local 128-bit random identity with the standard library for preview/report correlation. Cleanup authority comes only from exact process-definition and process-instance keys returned by Camunda, never the local run ID, BPMN ID, tenant-wide discovery, or a later search.

**Alternatives considered**:

- Add a latency-specific BPMN fixture. Rejected because the existing fixture already supplies the required stable wait state.
- Reuse the multi-subprocess smoke fixture. Rejected because it creates unrelated activity that would distort a lightweight latency diagnostic.
- Put the run ID into process variables. Rejected because variables are unnecessary business payload and could leak into diagnostics.

## Decision: Gate active versions before mutation

**Rationale**: Camunda 8.7 adapters expose unknown deploy/create keys, so the workflow cannot satisfy exact ownership recording even with retention. Active execution therefore aborts before mutation on 8.7. Camunda 8.8 returns ownership keys but cannot delete complete process-definition history, so cleanup-enabled execution aborts before mutation and explicit `--no-cleanup` is allowed. Camunda 8.9 and 8.10 expose keys and support complete cleanup.

Read-only analysis remains available on all configured versions and reports version-unsupported keyed measurements as unavailable.

**Alternatives considered**:

- Permit active 8.7 with BPMN-ID discovery. Rejected because discovery is not safe cleanup authority.
- Treat 8.8 definition retention as successful cleanup. Rejected because cleanup would be incomplete.

## Decision: Keep cleanup exact-key, independent, and bounded

**Rationale**: The active service records each returned key immediately, cleans process instances before the deployed process definition, and never targets unrecorded resources. A deferred cleanup path runs after success, stage error, timeout, or cancellation. It derives a fresh context with `context.WithoutCancel` and the existing `poller.DefaultCompletionTimeout`; shared HTTP per-request timeouts still apply.

The active command installs signal-aware cancellation only for its execution window so an operator interrupt reaches the service instead of immediately terminating the process. No global signal framework is added. If cleanup cannot remove a recorded resource, the outcome is partial/failed and the report lists the exact key plus an exact-key manual recovery command. Explicit `--no-cleanup` is a successful retained state, not a cleanup failure.

**Alternatives considered**:

- Reuse the canceled measurement context. Rejected because cleanup could not start after interruption.
- Search by BPMN ID during cleanup. Rejected because it could select unrelated resources.
- Change process-wide root execution for all commands. Rejected as broader than this feature.

## Decision: Aggregate safe logical measurements and deterministic findings

**Rationale**: Each logical measurement records category, stage, duration, success/unavailable/error outcome, and a safe classification. It never stores raw response bodies, URLs with credentials, request payloads, variables, or arbitrary upstream error strings.

Safe classes include success, unavailable, timeout, backpressure, unhealthy partition, missing leader, authentication, connectivity, not found, and other sanitized failure. Backpressure uses repository error identity for rate limiting and a bounded `RESOURCE_EXHAUSTED` check for the known 503 form. Topology evidence records only broker/partition health and leadership facts.

Stage latency p50/p95/max uses successful samples and deterministic nearest-rank percentiles. Throughput is successful logical operations divided by stage wall duration. Error and timeout counts are separate. Deltas compare each stage with its predecessor and remain unavailable when the denominator is zero.

Findings use deterministic, tested evidence rules: explicit topology/backpressure/timeout evidence takes precedence, comparative read/write/visibility signatures follow, and a bounded sample with no signal yields `no_abnormal_evidence`. Relative degradation rules require adequate samples and a material ratio; they never define a universal healthy latency threshold. Every finding includes evidence, likely affected area, confidence, limitation, and next investigation.

**Alternatives considered**:

- Serialize raw errors for troubleshooting. Rejected because existing HTTP errors can contain response bodies.
- Declare health from fixed millisecond thresholds. Rejected because environments differ and the feature is diagnostic rather than an SLO monitor.
- Compute percentiles over failures. Rejected because failed durations and successful response latency answer different questions.

## Decision: Preserve established success and failure envelopes

**Rationale**: A run that completes every planned stage with usable evidence returns a successful process exit even when observations include timeouts, backpressure, or latency anomalies. Invalid input, incomplete execution, report failure, and requested-cleanup failure return nonzero through established command errors.

Completed JSON uses the shared success envelope. On nonzero exits, stdout retains the established error envelope; if `--report-file` was requested, the raw report preserves available partial diagnostic and cleanup evidence. Extending the shared envelope with a partial payload is intentionally out of scope because it would create a new cross-command contract.

## Decision: Extend existing validation and documentation surfaces

**Rationale**: Unit tests cover planning, statistics, classifications, findings, worker ceilings, version gates, ownership, cancellation cleanup, output secrecy, and facade conversions. Command tests cover flags, mutation metadata, automation, confirmation, JSON purity, reports, exits, and progress. Existing command inventory and ops analyse/execute volume suites gain the two leaves; no new integration target is needed.

Help metadata is the source for regenerated `docs/cli`. README and `docs/ops/index.md` gain discoverability, with one focused guide per command. Real-cluster checks validate structure, ownership, bounds, and cleanup state rather than fragile millisecond thresholds.
