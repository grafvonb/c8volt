# Quickstart: API Latency Diagnostics

## Implementation Context

Work on branch `289-api-latency-diagnostics` and read these artifacts before implementation:

1. [spec.md](spec.md)
2. [plan.md](plan.md)
3. [research.md](research.md)
4. [data-model.md](data-model.md)
5. [contracts/api-latency-cli.md](contracts/api-latency-cli.md)
6. `specs/ralph-implementation-rules.md` when using Ralph

Do not create a new report system, worker system, fixture, generated client, or versioned latency adapter. Extend the existing ops layers and shared helpers.

## Recommended Build Order

1. Add version-neutral request, plan, measurement, result, finding, ownership, and cleanup models in `internal/domain` with deterministic planning/statistics tests.
2. Add shared stage planning, safe classification, aggregation, and findings in `internal/services/ops`.
3. Add the strictly read-only service workflow and prove zero mutation calls in every tested terminal path.
4. Add active preflight, the existing `SimpleUserTask` deployment/create path, exact-key ownership, bounded visibility/read work, and independent cleanup.
5. Extend the existing `c8volt/ops` facade with thin conversions and `ferrors` handling.
6. Add the two Cobra leaves, capability metadata, validation, preview/confirmation, progress, renderers, and exact shared report-file behavior.
7. Extend command inventory, ops volume suites, README/ops guides, and generated CLI docs.

Keep a distinct command lifecycle in its focused file. If shared latency progress/mode behavior reaches the repository's three-declaration threshold, move it to a focused `cmd/ops_api_latency_progress.go` rather than expanding a command file.

## Local Contract Scenarios

### 1. Safe read-only default

```bash
c8volt ops analyse api-latency
```

Expected:

- stages 1, 2, and 4 with sample allocations 5, 6, and 9;
- no deployment, creation, cancellation, or deletion;
- compact stage metrics and findings;
- read-only limitations shown;
- success exit when all stages complete, including when measurements contain abnormal evidence.

### 2. Read-only JSON and report

```bash
c8volt ops analyse api-latency \
  --json \
  --report-file api-latency.json
```

Expected:

- stdout is one shared success-envelope JSON document;
- report file is raw `ops.api-latency.v1` JSON;
- no progress text on stdout;
- report and stdout exclude raw response bodies and secrets.

### 3. Invalid stage budget

```bash
c8volt ops analyse api-latency --count 4 --workers 4
```

Expected: invalid-arguments failure explaining that stages 1, 2, and 4 require at least 7 primary samples. No client work begins.

### 4. Active dry-run

```bash
c8volt ops execute api-latency-test --dry-run
```

Expected:

- complete preflight and preview;
- run ID, fixture, version capability, stages, count, worker ceiling, derived bound, tenant, and cleanup plan shown;
- zero mutation calls;
- a blocked version/capability returns nonzero without mutation rather than claiming readiness.

### 5. Cleanup-enabled active run

Use a disposable Camunda 8.9 or 8.10 environment:

```bash
c8volt ops execute api-latency-test \
  --count 20 \
  --workers 4 \
  --auto-confirm \
  --report-file api-latency-test.md
```

Expected:

- one existing version-matched `SimpleUserTask` fixture deployment;
- no more than 20 created PI roots;
- actual concurrency never above 4;
- exact PD/PI keys recorded immediately;
- create, overlapping-read, and visibility evidence separated;
- exact-key cleanup attempted for every owned resource;
- successful cleanup leaves no run-owned resource behind.

### 6. Explicit retained 8.8 run

Only on a disposable environment where retention is acceptable:

```bash
c8volt ops execute api-latency-test \
  --no-cleanup \
  --auto-confirm \
  --report-file retained-api-latency.md
```

Expected on Camunda 8.8+:

- preview prominently states retention;
- confirmation still applies;
- final outcome is `completed_retained` when measurements finish;
- every retained exact key is listed distinctly from cleanup failure.

Camunda 8.7 must fail before mutation even with `--no-cleanup` because exact created ownership keys are unavailable.

### 7. Interrupted cleanup

With a stub or disposable cleanup-capable cluster, cancel after at least one key is recorded.

Expected:

- measurement scheduling stops;
- cleanup receives an independent bounded context;
- only recorded keys are targeted;
- a partial report is written when requested;
- every possible remainder has exact-key recovery guidance;
- command exits nonzero because execution was incomplete.

## Focused Validation

### Pure planning, statistics, classification, and service behavior

```bash
go test ./internal/services/ops -run 'APILatency' -count=1
```

Cover:

- stage sequences for workers 1, power-of-two, and non-power-of-two maxima;
- minimum count and deterministic allocation;
- primary/derived ceilings and observed high-water concurrency;
- nearest-rank p50/p95/max, throughput, and zero-baseline deltas;
- safe backpressure/timeout/topology classifications;
- finding precedence, confidence, and no unsupported root-cause wording;
- read-only zero mutation on success, error, timeout, and cancellation;
- active 8.7/8.8/8.9/8.10 capability behavior;
- ownership recorded before subsequent work;
- cleanup after success, error, request timeout, and cancellation;
- retained versus failed cleanup;
- no raw error bodies, variables, or secrets in result fields.

Use injected clock/sleep seams for exact timing tests. Use channel gates rather than sleeps for concurrency tests.

### Public facade

```bash
go test ./c8volt/ops -run 'APILatency' -count=1
```

Cover request/options/result/progress conversion, defensive collection copying, partial-result conversion, and `ferrors` mapping.

### Commands, output, reports, contracts, and exits

```bash
go test ./cmd -run 'APILatency|OpsWorkflowReport|CommandCapability|CapabilityDocument' -count=1
```

Cover:

- exact defaults and `--count/-n`, `--workers/-w` aliases;
- invalid count/worker combinations and keys-only rejection;
- read-only/state-changing metadata, full contracts, automation, and active all-tenants rejection;
- dry-run, preview, confirmation, no-cleanup confirmation, and JSON mutation guardrails;
- compact human output and one-document JSON with no progress leakage;
- Markdown/JSON inference, explicit format override, existing-file policy, and partial report preservation;
- completed-abnormal success exit versus incomplete/cleanup/report nonzero exits;
- secret-marker absence in output and reports;
- deterministic progress modes and no per-key default chatter.

Add a subprocess exit-code regression for a completed run containing abnormal measurements.

### Existing integration and docs surfaces

```bash
go test ./integration/cli -count=1
make docs-content
git diff --check
```

Update the expected command inventory from 55 to 57 and add both leaves to the existing ops analyse/execute family expectations. Extend the existing volume suites rather than adding new make targets. Update command help/examples first, then regenerate `docs/cli`; do not hand-edit generated CLI pages.

### Full repository validation

```bash
make test
```

This is the required final race-enabled repository suite.

## Disposable-Cluster Validation

After unit/contract tests pass:

```bash
C8VOLT_IT_AUTOMATION=1 make integration-cli-ops-analyse-volume C8VOLT_IT_GO_TEST_FLAGS=-v
C8VOLT_IT_AUTOMATION=1 make integration-cli-ops-execute-volume C8VOLT_IT_GO_TEST_FLAGS=-v
```

Use existing selected-version profiles. The execute suite should run dry-run on selected versions and a confirmed cleanup run only on a disposable cleanup-capable 8.9/8.10 profile. Do not run retained 8.8 tests against a shared environment.

Assert structural evidence, request bounds, ownership keys, report parity, and cleanup post-state. Do not assert real-cluster millisecond thresholds.

## Documentation Checklist

- Add both commands to `README.md` ops discovery/examples where appropriate.
- Add both commands to `docs/ops/index.md`.
- Add `docs/ops/analyse-api-latency.md` with read-only guarantees and limits.
- Add `docs/ops/execute-api-latency-test.md` with mutation, version, confirmation, retention, and cleanup guidance.
- Regenerate `docs/cli` with `make docs-content`.
- Verify examples through `integration/cli/examples_test.go`.
