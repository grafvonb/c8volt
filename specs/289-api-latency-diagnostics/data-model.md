# Data Model: API Latency Diagnostics

## 1. API Latency Request

Represents one validated invocation passed through the ops facade.

| Field | Type | Rules |
| --- | --- | --- |
| `mode` | enum | `read_only` or `active`; selected by the command leaf, not a user flag |
| `count` | integer | Default 20; positive; at least the computed stage minimum |
| `workers` | integer | Default 4; positive; no greater than count |
| `dryRun` | boolean | Active-only; plans and preflights but submits no mutation |
| `noCleanup` | boolean | Active-only; explicit retention acceptance |
| `tenantId` | string | Concrete effective tenant; active mode rejects all-tenants |
| `httpTimeout` | duration | Existing per-request timeout, carried for safe plan context only |
| `backoff` | value object | Existing normalized strategy, delay, retry, and timeout values used to bound visibility polling |
| `progress` | callback/channel | Optional semantic progress sink; excluded from JSON/report models |

Command-only options such as JSON, quiet, confirmation, and report destination are not service request fields unless they change service behavior. Report writing remains command-owned.

### Validation

1. `count > 0`.
2. `workers > 0` and `workers <= count`.
3. Build stage widths, then require `count >= sum(stage.workerCount)`.
4. Read-only rejects active-only flags.
5. Active requires a concrete tenant and a valid normalized backoff configuration.
6. JSON active execution is command-validated to require dry-run or implicit confirmation.

## 2. Diagnostic Plan

Immutable plan produced before measured work and, for active mode, before confirmation.

| Field | Type | Description |
| --- | --- | --- |
| `runId` | string | Active-only 128-bit random correlation identity; generated before preview |
| `mode` | enum | Read-only or active |
| `stages` | list of Load Stage Plan | Ordered deterministic worker ramp and sample allocation |
| `primarySampleLimit` | integer | Equal to requested count |
| `primarySampleAllocation` | integer | Sum of stage allocations; must equal count |
| `derivedRequestLimit` | integer | Deterministic upper bound computed for the selected mode |
| `visibilityAttemptLimit` | integer | Active-only maximum logical search attempts per created instance |
| `setupOperations` | list | Safe preflight/setup calls disclosed separately from samples |
| `fixture` | Fixture Plan | Active-only version-matched embedded fixture |
| `cleanup` | Cleanup Plan | Active-only requested behavior and capability evidence |
| `notices` | list of string | Version, retry, retention, or evidence-limit notices |
| `limitations` | list of string | Stable interpretation boundaries |

### Load Stage Plan

| Field | Type | Description |
| --- | --- | --- |
| `index` | integer | One-based stable stage number |
| `workerCount` | integer | Declared concurrency ceiling for this stage |
| `primarySamples` | integer | Sample cycles allocated to the stage; at least workerCount |
| `derivedRequestLimit` | integer | Mode-specific derived ceiling attributable to this stage |

### Stage construction

1. Start with worker width 1.
2. Append powers of two below the requested maximum.
3. Append the exact requested maximum if not already present.
4. Reserve `workerCount` samples for each stage.
5. Divide the remaining samples evenly among stages.
6. Assign any remainder from the highest-concurrency stage backward.

Default `count=20, workers=4` yields `(workers=1, samples=5)`, `(2, 6)`, `(4, 9)`.

## 3. Safe Run Context

Reportable context intentionally excludes full configuration.

| Field | Type | Description |
| --- | --- | --- |
| `commandName` | string | Exact command path |
| `schemaVersion` | string | `ops.api-latency.v1` |
| `c8voltVersion` | string | Current safe build version |
| `camundaVersion` | string | Configured/observed compatible release line |
| `profile` | string | Safe profile identity only |
| `tenant` | string | Effective tenant identity |
| `startedAt` / `finishedAt` | timestamp | UTC capture bounds |
| `duration` | duration | Total run duration |

No endpoint URL, token, authorization header, secret, raw configuration, request body, process variable, or business payload is permitted.

## 4. Measurement

Internal ephemeral record for one logical service call. Individual measurements may support aggregation but are not required in compact human output.

| Field | Type | Description |
| --- | --- | --- |
| `stageIndex` | integer | Owning stage |
| `category` | enum | `topology_read`, `process_definition_search`, `process_instance_search`, `process_definition_read`, `process_instance_read`, `fixture_deploy`, `process_instance_create`, `concurrent_read`, or `search_visibility` |
| `kind` | enum | `primary`, `derived`, `setup`, or `cleanup` |
| `startedAt` | timestamp | Local monotonic measurement start |
| `duration` | duration | End-to-end logical service-call duration, including existing retries |
| `outcome` | enum | `succeeded`, `unavailable`, `failed`, or `timed_out` |
| `classification` | enum | Safe response classification |
| `overlappedWrite` | boolean | Concurrent-read only; true only when a write was actually in flight |

### Safe response classifications

`success`, `unavailable`, `unsupported`, `not_found`, `timeout`, `backpressure`, `unhealthy_partition`, `missing_leader`, `authentication`, `connectivity`, `malformed_response`, and `request_error`.

Only bounded safe codes/details may accompany a classification. Raw `err.Error()` output and upstream bodies are not report fields.

## 5. Stage Result

Immutable aggregation for one completed or interrupted stage.

| Field | Type | Description |
| --- | --- | --- |
| `plan` | Load Stage Plan | Requested width and allocation |
| `status` | enum | `planned`, `running`, `completed`, `incomplete`, or `skipped` |
| `startedAt` / `finishedAt` | timestamp | Actual stage bounds when run |
| `actualMaxConcurrency` | integer | Observed high-water logical operations |
| `primaryAttempts` | integer | Started primary cycles |
| `derivedAttempts` | integer | Started derived logical calls |
| `categories` | ordered list of Category Summary | Stable category order |
| `classifications` | ordered counts | Safe outcome/error counts |
| `comparison` | Stage Comparison | Delta from previous stage; absent for first stage |

### Category Summary

| Field | Type | Description |
| --- | --- | --- |
| `attempts` | integer | Logical calls attempted |
| `successes` | integer | Successful logical calls |
| `errors` | integer | Failed calls excluding timeouts/unavailable |
| `timeouts` | integer | Timeout-classified calls |
| `unavailable` | integer | Unsupported, missing-key, or not-found evidence |
| `throughputPerSecond` | decimal/nullable | Successful calls divided by stage wall duration |
| `p50` / `p95` / `max` | duration/nullable | Nearest-rank values over successful durations only |

### Stage Comparison

For each category, stores latency and throughput absolute/percentage deltas relative to the preceding stage. A delta is unavailable when either required statistic is absent or the prior value is zero.

## 6. Topology Evidence

Safe structural evidence derived from the topology response.

| Field | Type | Description |
| --- | --- | --- |
| `brokerCount` | integer | Observed brokers |
| `partitionCount` | integer | Distinct partitions |
| `unhealthyPartitions` | list of integer | Partition IDs only |
| `leaderlessPartitions` | list of integer | Partition IDs only |
| `healthKnown` | boolean | False when topology omits usable partition detail |

Topology evidence contributes to findings. Version/ownership/cleanup incompatibility blocks active mutation; a health warning is disclosed and interpreted rather than automatically treated as proof of root cause.

## 7. Finding

| Field | Type | Description |
| --- | --- | --- |
| `code` | enum | Stable finding code |
| `evidence` | list of string | Safe references to measured categories, classes, and aggregates |
| `likelyArea` | enum/string | Query/secondary storage, exporter visibility, write path, gateway/connectivity/authentication, cluster pressure, partition health, or no abnormal evidence |
| `confidence` | enum | `high`, `medium`, or `low` |
| `limitation` | string | Why the finding is not definitive |
| `nextInvestigation` | string | Concrete non-mutating next step where possible |

Finding order is deterministic: explicit topology/backpressure/timeout evidence, comparative path signatures, then the no-abnormal-evidence fallback.

## 8. Active Run Ownership

| Field | Type | Description |
| --- | --- | --- |
| `runId` | string | Local correlation identity |
| `fixtureName` | string | Embedded fixture filename |
| `bpmnProcessId` | string | Informational only; never cleanup authority |
| `deploymentSubmitted` | boolean | Establishes confirmed-mutation report overwrite mode |
| `processDefinitionKey` | string/nullable | Exact returned key recorded immediately |
| `processInstanceKeys` | ordered unique list | Exact returned root keys recorded immediately |

The registry is concurrency-safe. An item enters the registry before any subsequent stage can use it. Unknown keys make the active version ineligible rather than producing an unowned mutation.

## 9. Visibility Result

| Field | Type | Description |
| --- | --- | --- |
| `processInstanceKey` | string | Exact owned key |
| `attempts` | integer | Logical exact-key search attempts |
| `attemptLimit` | integer | Previewed per-instance maximum |
| `visible` | boolean | Search observed the created key |
| `duration` | duration | Create response to first visible search observation |
| `finalClassification` | enum | `success`, `timeout`, `not_found`, or safe failure class |

## 10. Cleanup Plan and Record

### Cleanup Plan

| Field | Type | Description |
| --- | --- | --- |
| `requested` | boolean | True unless `--no-cleanup` |
| `supported` | boolean | Exact-key PI and complete PD cleanup capability |
| `intentionalRetention` | boolean | True only for explicit no-cleanup |
| `independentBudget` | duration | Existing bounded completion budget used after cancellation |
| `blockReason` | string/nullable | Pre-mutation reason when ownership or requested cleanup is unsupported |

### Cleanup Record

| Field | Type | Description |
| --- | --- | --- |
| `resourceType` | enum | `process_instance` or `process_definition` |
| `key` | string | Exact recorded ownership key |
| `status` | enum | `pending`, `submitted`, `deleted`, `retained`, `failed`, or `unknown` |
| `classification` | enum | Safe cleanup outcome/error class |
| `recoveryCommand` | string/nullable | Exact-key command only when resource may remain |

Cleanup processes process-instance roots before the process definition. Every recorded resource ends in a terminal cleanup status.

## 11. Diagnostic Result / Report Payload

One model covers both commands.

| Field | Type | Description |
| --- | --- | --- |
| `schemaVersion` | string | `ops.api-latency.v1` |
| `context` | Safe Run Context | Safe invocation context |
| `request` | API Latency Request View | Safe requested settings |
| `plan` | Diagnostic Plan | Previewed/actual bounded plan |
| `topology` | Topology Evidence | Safe control-path evidence |
| `stages` | ordered list of Stage Result | Planned and actual stage evidence |
| `findings` | ordered list of Finding | Deterministic interpretation |
| `notices` | ordered list | Operational notices |
| `limitations` | ordered list | Evidence boundaries and retry caveats |
| `ownership` | Active Run Ownership/omitted | Active-only |
| `visibility` | ordered list/omitted | Active-only |
| `cleanup` | ordered list/omitted | Active-only |
| `outcome` | enum | Final execution outcome |

### Final outcomes

- `planned`: valid dry-run with no mutation.
- `completed`: all stages completed with usable evidence; abnormal findings are allowed.
- `completed_retained`: explicit no-cleanup run completed and owned resources are intentionally retained.
- `partial`: usable evidence exists but execution or requested cleanup is incomplete.
- `failed`: preflight, execution, report, or cleanup failure prevents a complete result.
- `interrupted`: caller cancellation stopped measurement; cleanup evidence is still attached when possible.

Only `planned`, `completed`, and `completed_retained` produce a successful command exit. A requested report may serialize any outcome; stdout on a non-success exit uses the established command error envelope.

## State Transitions

### Read-only

```text
validating -> planned -> measuring -> interpreting -> completed
     |           |          |
     +-----------+----------+-> failed/interrupted
```

No transition in this state machine invokes deploy, create, cancel, or delete services.

### Active with cleanup

```text
validating -> preflight -> previewed -> confirmed -> deploying -> measuring
     |           |            |           |            |          |
     +-----------+------------+-----------+------------+----------+
                                                                  v
                                                             cleaning_up
                                                                  |
                                              +-------------------+------------------+
                                              v                                      v
                                          completed                           partial/failed
```

Active dry-run ends at `planned`. An active 8.7 run, or cleanup-enabled 8.8 run, ends at preflight failure before confirmation/mutation. Cancellation after ownership exists moves through `cleaning_up` using the independent cleanup context.
