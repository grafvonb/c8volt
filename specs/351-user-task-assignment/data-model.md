# Data Model: User-Task Assignment and Unassignment

**Feature**: [spec.md](spec.md) | **Design**: [plan.md](plan.md)

No stored schema or migration is introduced. Models below are in-memory inputs, domain/public outcomes and command reports.

## 1. Frozen assignment intent

Command-local `userTaskAssignmentInput` contains:

| Field | Type / values | Invariant |
| --- | --- | --- |
| Operation | `assign` or `unassign` | Exactly one active new mode |
| Keys | Ordered task-key collection | Nonempty; exactly 16 ASCII digits each; unique first-input order |
| Assignee | Opaque string | Nonblank for assign; absent for unassign; preserve nonblank supplied bytes |
| AllowOverride | Boolean | False by default; true only from explicit permission; false for unassign |
| WantedWorkers | Integer | CLI omitted = automatic; explicit CLI values positive |

Capture flag-presence evidence before constructing intent: empty assignment is invalid, false boolean selectors are inactive, supplied variable flags are incompatible even empty, and override flag presence is forbidden outside assignment. No identity resolution or backend eligibility fact belongs in this input.

The command snapshots local intent before approval; the service copies the incoming key slice and evaluates options before scheduling so caller mutation cannot alter submitted keys or policy mid-run. Strings are immutable. The public prepare/execute plan exposes ready keys, blocked and unchanged counts, and replacement identities while retaining frozen service inputs and observations privately. It is not a persisted approval token.

## 2. Facade and call options

Add `UserTaskAssignmentOverride bool` to existing `foptions.FacadeCfg` and `services.CallCfg`, with matching `WithUserTaskAssignmentOverride()` option functions and explicit facade-to-call mapping. Default false is preserved to the adapter, which sends a nonnil pointer even when false. Do not reuse `Force` or auto-confirm as permission.

Reuse `DryRun`, `FailFast`, `NoWorkerLimit`, `Verbose`, `SuppressWorkflowDetailLogs`, `IgnoreTenant` and existing context controls. Reject `NoWait` in both new operations. Unassignment rejects enabled assignment override. Public `wantedWorkers == 0` requests automatic selection and negative values are invalid.

## 3. Internal assignment-family outcome

Introduce `domain.UserTaskAssignmentResult` and `UserTaskAssignmentResults`, with assignment-specific status/submission enums. Both assign and unassign share this family; completion types remain unchanged.

| Field | Domain type | Meaning |
| --- | --- | --- |
| Key | string | Selected unique task key, always present |
| Status | `UserTaskAssignmentStatus` | Local operation outcome |
| Submission | `UserTaskAssignmentSubmission` | Evidence about backend acceptance |
| StatusCode | int | Last observed submission response code; zero when none |
| Error | string | Existing machine-readable error evidence; not human display text |
| Items (collection) | ordered slice of outcomes | One item per unique validated key; initialized empty for validation failures |

Statuses: `preview`, `unchanged`, `accepted`, `failed`, `timed_out`, `canceled`, `not_attempted`.

Submission values: `not_submitted`, `accepted`, `rejected`, `unknown`.

Initialize all valid selected keys to `not_attempted/not_submitted`. Maintain a private started marker per scheduling slot so absent pool results cannot erase or manufacture outcomes. Dry-run converts all items to `preview/not_submitted` without backend calls. Results keep input order regardless of response order.

## 4. State and evidence transitions

| Event | Local status | Submission | Consequence |
| --- | --- | --- | --- |
| Local preview | preview | not_submitted | No backend calls |
| Completed ownership lookup matches requested assignee | unchanged | not_submitted | No mutation or confirmation; no error or status code |
| Not scheduled due to fail-fast or pre-start cancellation | not_attempted | not_submitted | Never describe as attempted or rolled back |
| Valid 204, no overriding call error | accepted | accepted | Backend-confirmed operation; no read-back claim |
| Explicit rejection without earlier ambiguity | failed | rejected | Preserve class, status and machine evidence |
| Missing/malformed response, generic server/proxy failure or transport ambiguity | failed, timed_out or canceled as classified | unknown | Do not infer rejection |
| Newly attempted request returns 504 or a network timeout | timed_out | unknown | A new ambiguous attempt overrides any earlier rejection evidence |
| Deadline while backing off after a definite rejection, with no new attempt | timed_out | rejected unless an earlier attempt was ambiguous | Preserve known evidence without inventing a new submission |
| Caller cancellation during an in-flight attempt | canceled | unknown when the attempt has no conclusive response | Do not inherit certainty from an earlier rejected attempt |
| Caller cancellation during backoff, with no new attempt | canceled | Evidence retained | Rejected stays rejected only if no attempt was ambiguous |
| Ambiguous attempt then rejection | failed or classified terminal status | unknown | Later rejection does not erase earlier ambiguity |
| Ambiguous attempt then documented success | accepted | accepted | Successful evidence supersedes uncertainty |

A nonnil response, nil call error, or undocumented 2xx does not establish acceptance. A call error overrides a partial success response. Preserve last observed status across subsequent transport failure; statusCode is evidence, not proof of the final assignment state. Existing retry policy determines whether another attempt occurs; these transitions do not broaden retryability.

Cancellation before any mutation leaves every item not attempted and preserves the caller's error. Parent cancellation after some successes can fail the invocation while retaining successful item evidence. A renderer must never fill missing items as successful or infer an already-completed conflict from failed/not-submitted combinations.

## 5. Public result

`task.UserTaskAssignmentResult`, its status/submission types and collection mirror the neutral domain facts without exposing internal/generated types. Convert results even when error is nonnil, then use `ferrors.FromDomain` for error conversion.

JSON fields on public items follow established conventions:

| Field | Serialization |
| --- | --- |
| `key`, `status`, `submission` | Required |
| `statusCode` | Omit when zero |
| `error` | Omit when empty; retain machine failure evidence |
| `items` | Initialized array, never null |

Existing completion public schemas and conversion behavior are untouched.

## 6. Command report

`userTaskAssignmentReport` is a view model in `cmd/cmd_views_usertask_assignment.go`:

| JSON field | Presence | Meaning |
| --- | --- | --- |
| `operation` | Required | `assign` or `unassign` |
| `phase` | Required | `preview` or `execution` |
| `requestedKeys` | Required array | Frozen unique selection |
| `requestedCount` | Required integer | Number of requested unique keys |
| `assignee` | Assign only | Supplied identity |
| `allowOverride` | Assign only, including false | Explicit reassignment policy; use presence-capable representation |
| `items` | Required array | Ordered public outcomes |
| `counts` | Required object | Counts for every known local status |

Count fields always present: `preview`, `accepted`, `failed`, `timedOut`, `canceled`, `notAttempted`. An additive `unchanged` count is present only when nonzero. The sum of all counts equals requestedCount for complete execution/preview reports. Do not include variable/completion fields in assignment reports. Pre-input or validation failure uses the existing error envelope without an invented report.

The report carries operation and intent once; items need not duplicate assignee/action. Machine evidence and counts are stable independently of compact human wording or zero-count suppression. The shared result envelope and tenant context remain unchanged; see [CLI contract](contracts/cli-user-task-assignment.md).

## 7. Internal private submission evidence

A small private record in `internal/services/common` may hold attempt-started, last status, ambiguity, accepted response and terminal error while the existing retry helper runs. It is not a facade type or a new shared envelope. Concrete completion/assignment wrappers map it to their established result types and action-specific labels. Its extraction must preserve completion semantics exactly and is covered by completion regressions plus the new assignment cases.

## Reassignment diagnostic evidence

`ReassignmentDenied` is non-serialized domain/facade metadata derived either from a successful ownership lookup showing another assignee (failed/not_submitted, no HTTP status) or from a definite failed/rejected HTTP 409 response with Camunda's already-assigned claim reason for the selected key. It never replaces the original error or submission evidence. The command combines it with single-key assignment and disabled override to select the INFO hint; no machine fields or outcome enums change.
