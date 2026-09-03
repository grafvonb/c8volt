# CLI Contract: API Latency Diagnostics

This feature adds two leaves to the existing ops command tree. It reuses the shared command, automation, progress, tenant, error-envelope, and report contracts; this document defines only the command-specific behavior and payload.

## Commands

```bash
c8volt ops analyse api-latency [flags]
c8volt ops execute api-latency-test [flags]
```

`ops analyse` and `ops execute` remain grouping commands.

## Capability Metadata

| Command | Mutation | Contract | Automation | All tenants |
| --- | --- | --- | --- | --- |
| `ops analyse api-latency` | Read-only | Full | Full | Existing analysis tenant behavior |
| `ops execute api-latency-test` | State-changing | Full | Full | Rejected; requires a concrete destination tenant |

Both commands support human one-line/section output and JSON. Both explicitly reject inherited `--keys-only` as invalid for this diagnostic.

## Command Flags

### Shared by both leaves

| Flag | Type | Default | Behavior |
| --- | --- | --- | --- |
| `--count`, `-n` | positive integer | `20` | Total primary sample-cycle budget across all stages |
| `--workers`, `-w` | positive integer | `4` | Maximum closed-loop worker count and final stage width |
| `--report-file` | path | empty | Writes the shared ops report payload |
| `--report-format` | `markdown` or `json` | inferred | Explicit report format; requires `--report-file` |

The command rejects:

- count or workers less than 1;
- workers greater than count;
- count below the sum of deterministic stage widths;
- invalid report format/dependency/path under shared ops validation;
- keys-only output.

For example, workers 4 requires at least 7 samples for stages 1, 2, and 4. The default plan allocates 20 samples as 5, 6, and 9.

### Active-only

| Flag | Type | Default | Behavior |
| --- | --- | --- | --- |
| `--dry-run` | boolean | `false` | Performs request, fixture, version, capability, and plan validation without mutation |
| `--no-cleanup` | boolean | `false` | Explicitly retains exact run-owned resources |

The command also inherits established `--auto-confirm` and `--automation` behavior. It adds no fail-fast, no-worker-limit, no-wait, custom fixture, key, rate, duration, or whole-run deadline flag.

## Inherited Behavior

- `--timeout` remains the HTTP timeout for each request. It does not become a run deadline.
- Visibility polling uses the normalized ops backoff configuration already inherited under `ops`; its logical attempt ceiling is computed and displayed.
- Profile, tenant, quiet, verbose, debug, JSON, automation, confirmation, and error-code behavior remain unchanged.
- The active command rejects all-tenants because deployment, creation, ownership, and cleanup require one concrete tenant.

## Stage Contract

Stage worker widths are deterministic:

```text
1, powers of two below the maximum, exact requested maximum
```

Examples:

| Workers | Stages |
| --- | --- |
| 1 | 1 |
| 3 | 1, 2, 3 |
| 4 | 1, 2, 4 |
| 5 | 1, 2, 4, 5 |

Each worker is closed-loop: it waits for and records one logical call before beginning its next call. Actual high-water concurrency must not exceed the stage width or requested worker ceiling.

The plan reserves at least one sample per worker slot in every stage, distributes the remaining count evenly, and assigns any remainder from the final stage backward. Preview, JSON, and reports show both planned and actual allocations.

## Read-only Operation

### Measured paths

Each primary cycle measures:

1. cluster topology/control read;
2. lightweight process-definition search;
3. lightweight process-instance search.

Each search may produce one derived direct keyed read using only a key returned by that measured search. Missing keys, disappearing resources, and version-unsupported keyed reads are `unavailable` evidence rather than successful measurements. Camunda 8.7 process-instance keyed read is unavailable; the rest of read-only analysis remains supported.

The maximum derived logical-call bound is `2 * count`.

### Safety

The service path for this command must make zero deploy, create, cancel, or delete calls on success, failure, timeout, and cancellation paths. Every result states that read-only evidence cannot prove write health, exporter health, end-to-end process execution, capacity, or overall cluster health.

## Active Operation

### Preflight and version matrix

Preflight reads configuration, connectivity/topology, configured and observed version compatibility, embedded fixture availability, ownership-key support, and requested cleanup capability before any mutation.

| Camunda | Active result |
| --- | --- |
| 8.7 | Unsupported before mutation, including with `--no-cleanup`, because exact created keys cannot be guaranteed |
| 8.8 | Cleanup-enabled request blocked before mutation; explicit `--no-cleanup` may proceed |
| 8.9 | Cleanup-enabled and no-cleanup execution supported |
| 8.10 | Cleanup-enabled and no-cleanup execution supported |

A dry-run applies the same capability checks. A blocked dry-run reports the plan and block reason through the requested report, then returns nonzero; it does not imply that execution is ready.

### Preview

Before confirmation the human preview includes:

- unique run ID;
- safe profile and concrete tenant;
- configured/observed Camunda version;
- version-matched embedded `SimpleUserTask` fixture;
- stage widths and primary allocation;
- primary sample limit and maximum derived logical calls;
- visibility attempt ceiling/backoff budget;
- maximum workers;
- cleanup or intentional-retention behavior;
- version/capability and bounded-evidence notices.

Every non-dry execution prompts unless established implicit confirmation applies. `--no-cleanup` does not bypass confirmation. When JSON is selected, active execution requires `--dry-run`, `--auto-confirm`, or `--automation` so stdout remains one JSON document.

### Mutation and measurement

1. Generate the run ID locally before preview.
2. Deploy the matching embedded `SimpleUserTask` fixture.
3. Record the returned process-definition key immediately.
4. Create no more than count process instances by exact deployed definition key, measuring response latency without waiting for exporter visibility.
5. Record each returned process-instance key immediately.
6. Within the same worker ceiling, issue one lightweight read per primary cycle and record whether it actually overlapped an in-flight write.
7. Poll exact-key search visibility with the previewed attempt ceiling and backoff budget.
8. Interpret stage evidence.
9. Unless explicitly retained, clean exact recorded PI keys before the exact PD key.

The maximum derived logical-call bound is:

```text
count * (1 concurrent-read probe + visibility attempt ceiling)
```

Fixture deployment, preflight, and cleanup are displayed separately and do not consume primary samples.

### Ownership and cleanup

The local run ID is for correlation only. Exact returned keys are the only cleanup authority. The workflow never deletes by BPMN process ID, tenant-wide discovery, or later search results.

Cleanup is attempted after successful measurement, stage failure, caller cancellation, or request timeout using an independent bounded context. Every recorded resource ends as deleted, retained, failed, or unknown. Unknown/failed remainder entries include exact-key recovery guidance.

Representative recovery commands:

```bash
c8volt delete process-instance --key <key> --force --auto-confirm
c8volt delete process-definition --key <key> --auto-confirm
```

The process-definition command is supplied only on versions that support complete definition-history deletion. Explicit no-cleanup yields `completed_retained`; requested cleanup failure yields a non-success outcome.

## Human Output Contract

Default output is compact and scan-friendly:

1. context and operator-readable scope with the tested load range;
2. request-error and timeout totals;
3. the slowest read path or active create latency expressed as "95% completed within", plus median load effect when comparable;
4. plain-language findings followed by the next investigation;
5. active-only eventual search visibility and cleanup summary;
6. final outcome, including the number of actionable findings;
7. `report: written <path>` when applicable.

Default output omits stage allocation, theoretical request ceilings, statistical abbreviations, run metadata, notices, limitations, endpoints, cursors, raw errors, request bodies, per-key lifecycle chatter, and individual samples. Retained or unresolved cleanup resources and their recovery commands remain visible because they require operator action. `--verbose` adds the full plan, stage metrics, metadata, notices, and limitations. Dry-run continues to display the complete plan before mutation. Quiet suppresses successful progress and retains failures. Automation and JSON remain free of human progress.

## JSON Output Contract

Completed JSON stdout uses the shared success envelope with a payload shaped as `ops.api-latency.v1`:

```json
{
  "outcome": "succeeded",
  "result": {
    "schemaVersion": "ops.api-latency.v1",
    "context": {},
    "request": {},
    "plan": {},
    "topology": {},
    "stages": [],
    "findings": [],
    "notices": [],
    "limitations": [],
    "outcome": "completed"
  }
}
```

Active-only `ownership`, `visibility`, and `cleanup` fields are omitted for read-only results. Maps are avoided where stable ordered lists are needed. Durations and timestamps follow existing ops JSON conventions.

On an invalid or incomplete run, stdout uses the existing error envelope and nonzero exit. It does not add a new partial-result envelope. A requested raw report preserves available partial evidence.

## Shared Report Contract

The feature calls existing shared helpers without changing their semantics:

- `--report-format` requires `--report-file`.
- Explicit `markdown`/`json` overrides inference.
- `.json` infers JSON.
- `.md`, `.markdown`, no extension, and unknown extensions infer Markdown.
- Read-only, dry-run, and unconfirmed/pre-mutation planning preserve existing files.
- Confirmed mutation uses the existing overwrite policy; final write mode reflects whether mutation was actually submitted.
- Files use mode `0600`; missing parent directories are errors.
- Reports are attempted from partial results before the command returns an execution/cleanup error.
- Destination or rendering failures return nonzero.

Markdown contains the same logical information as JSON in compact sections. The raw JSON report is the result payload, not the stdout command envelope.

## Statistics and Findings

For every stage/category:

- attempts, successes, errors, timeouts, and unavailable count;
- successful throughput per second;
- nearest-rank p50, p95, and maximum over successful durations;
- absolute/percentage latency and throughput deltas from the previous stage when defined;
- ordered safe classification counts.

Completed stages with request failures or abnormal latency still produce a successful command exit when usable findings exist. Findings may identify backpressure, topology health/leadership, query-only degradation, broad read degradation, write degradation, delayed visibility, combined degradation, or no abnormal evidence. Each finding contains evidence, likely area, confidence, limitation, and next investigation; none claims a definitive infrastructure root cause.

## Error and Exit Behavior

| Condition | Exit | Result behavior |
| --- | --- | --- |
| Valid dry-run | Success | `planned`, zero mutation |
| All stages complete with usable evidence | Success | `completed`, even with abnormal observations |
| Explicit no-cleanup run completes | Success | `completed_retained`, retained exact keys listed |
| Invalid flags or keys-only | Invalid-arguments error | Established error envelope |
| Active 8.7, cleanup-enabled 8.8, incompatible version, missing fixture/key capability | Nonzero local precondition | No mutation; partial report if requested |
| Authentication/connectivity failure before usable plan | Nonzero mapped error | No mutation |
| Interrupted/incomplete stage | Nonzero | Cleanup attempted if needed; partial report if requested |
| Requested cleanup incomplete | Nonzero | Every remaining exact key and recovery command reported |
| Report destination/render/write failure | Nonzero | Established error behavior |

## Output Safety

Human output, JSON, progress, diagnostics produced by this feature, and reports must exclude access tokens, authorization headers, client secrets, raw configuration secrets, process variables, business payloads, raw response bodies, and unbounded upstream error strings. Only safe profile/tenant/version context, exact owned resource keys, bounded classifications, and sanitized fixed guidance are reportable.

## Examples

```bash
# Safest production diagnostic
c8volt ops analyse api-latency

# Smaller valid ramp: stages 1 and 2 require at least three samples
c8volt ops analyse api-latency --count 6 --workers 2

# JSON plus a Markdown report (format inferred)
c8volt ops analyse api-latency --json --report-file api-latency.md

# Validate an active plan without mutation
c8volt ops execute api-latency-test --dry-run

# Confirmed cleanup-enabled active run on 8.9/8.10
c8volt ops execute api-latency-test --auto-confirm --report-file api-latency-test.json

# Explicit retained run on 8.8+
c8volt ops execute api-latency-test --no-cleanup --auto-confirm --report-file retained.md

# Unattended JSON execution
c8volt ops execute api-latency-test --automation --json --report-file api-latency-test.json --report-format json
```
