# Data Model: User-Task Completion

**Issue**: #311 | **Design**: [plan.md](plan.md)

All operation data is in memory. Domain types live in `internal/domain/usertask_completion.go`; equivalent public types belong to `c8volt/task`. Generated Camunda types remain confined to version adapters. Existing variable-update models and JSON tags are unchanged.

## Completion input

The public operation accepts `keys typex.Keys`, `variables map[string]any`, `wantedWorkers int`, and existing facade options. The CLI keeps a frozen local request for preview and confirmation; the service independently validates and copies it for callers that bypass the CLI.

| Value | Meaning and invariants |
|---|---|
| Keys | Nonempty selection; exactly 16 decimal digits per key under existing user-task conventions; stable first-occurrence uniqueness; flags precede stdin |
| Variables | Nil means omitted; nonnil empty map means supplied `{}`; nested valid JSON values retain their types; one object shared by every completion |
| Wanted workers | Zero means automatic at the service boundary; positive values bound concurrency; negative invalid. CLI explicitly supplied zero is invalid |
| Options | Existing fail-fast, no-worker-limit, dry-run, verbose, and explicit-key authorization options; no-wait rejected |

Validate the entire input and JSON serializability before any native call. Snapshot nested variables through existing JSON-copy helpers; workers read the snapshot without mutation. No task existence, tenant identity, current variables, or assignment state is inferred from a local preview.

## UserTaskCompletionResult

| Field | Public JSON | Type | Meaning |
|---|---|---|---|
| Key | `key` | string | Selected task key; always present |
| Status | `status` | UserTaskCompletionStatus | Local execution result, defined below |
| Submission | `submission` | UserTaskCompletionSubmission | Backend acceptance evidence, independently defined below |
| StatusCode | `statusCode,omitempty` | int | Last observed HTTP status, omitted when none; not proof of an earlier attempt's outcome |
| Error | `error,omitempty` | string | Existing machine-readable failure evidence; never interpolate into human output |

Domain fields have the same meaning. Native adapters populate these facts before facade error conversion, since that conversion may no longer expose every original typed error cause.

### Execution statuses

| Value | Meaning |
|---|---|
| `preview` | Valid local selection, no submission requested |
| `accepted` | Backend confirmed completion through the successful native response; no independent task-state read-back |
| `failed` | Operation failed for a reason other than timeout/caller cancellation; submission evidence may be rejected or unknown |
| `timed_out` | An attempted operation timed out; backend acceptance may remain unknown |
| `canceled` | An attempted operation was canceled by the caller; backend acceptance may remain unknown |
| `not_attempted` | Native operation did not begin, including fail-fast or pre-submission cancellation |

### Submission evidence

| Value | Meaning |
|---|---|
| `not_submitted` | No native submission attempt; valid for preview/not-attempted |
| `accepted` | A documented successful completion response was observed |
| `rejected` | Observed rejection with no earlier ambiguous attempt |
| `unknown` | At least one attempted submission lacks conclusive acceptance/rejection evidence |

An item with status accepted must have submission accepted. Preview and not-attempted must have submission not_submitted. Timed-out/canceled items must not imply rejection solely from their execution status. A final rejection after an earlier transport ambiguity retains submission unknown. A later 204 establishes acceptance even if an earlier attempt was ambiguous.

## UserTaskCompletionResults

`Items []UserTaskCompletionResult` has public tag `items` and is initialized to an empty array when no item evidence exists. For a valid executed or dry-run selection, it contains exactly one item per unique key in selection order. Do not sort by completion time or drop unattempted slots. Return partial results alongside a nonnil error.

## Command completion report

A completion-specific typed view payload wraps the public item results without changing the shared envelope or existing variable-update payload:

| JSON field | Type | Rule |
|---|---|---|
| `operation` | string | Always `complete` |
| `phase` | string | `preview` or `execution` |
| `requestedKeys` | string array | Frozen unique selection, always an initialized array |
| `requestedCount` | integer | Length of requestedKeys |
| `variablesProvided` | boolean | Distinguishes omitted source from `{}` |
| `variableCount` | integer | Count of top-level supplied variable names; no variable values echoed |
| `items` | outcome array | Initialized array in requested order |
| `counts` | object | Integer fields `preview`, `accepted`, `failed`, `timedOut`, `canceled`, `notAttempted`, including zero values |

For complete preview/execution reports, the six status counts sum to requestedCount and correspond exactly to items. Preview items are all preview/not_submitted; their count is not presented as accepted or attempted work. A local validation failure may use the existing envelope without a completion report because no valid frozen selection exists.

The shared envelope retains existing `outcome`, `class`, `command`, optional `tenantContext`, `payload`, and `detail`. All-accepted execution uses outcome accepted. Preview uses succeeded. Failure uses the existing classification and exit-code policy with the complete execution report attached when available.

## Lifecycle

1. Parse all keys and optional variables; reject invalid input without native requests.
2. Freeze validated unique selection. Dry-run renders preview and ends; ordinary execution confirms once when required.
3. Service snapshots and validates inputs, initializes all result slots as not_attempted/not_submitted, then schedules bounded work.
4. A worker checks scheduling stop/caller cancellation before starting the native operation. Once started, the original caller context governs it.
5. Adapter records each actual attempt, last response and ambiguity, including evidence lost by the retry helper's zero return on interrupted backoff.
6. Accepted work becomes accepted/accepted. Rejected or uncertain work receives the accurate failure status/evidence. Fail-fast stops new scheduling; it does not overwrite existing accepted results or roll back submitted work.
7. Return ordered results plus aggregate error, render once, and exit using existing error classification.

The existing machine model retains accepted/accepted for a successful completion response; the human wording clarification introduces no completed/confirmed enum or new fields. No database schema, persistent plan, task-state polling, or reconciliation model is introduced.

## Tenant and error facts

Use existing explicit-key tenant context from configuration; it is authorization context, not a filter. Native no-content responses provide no actual task tenant, so resolved tenant IDs must not be synthesized from configuration. Do not add reads to populate them.

Normal/verbose failure reasons come only from known statuses, submission enums, known HTTP status labels, and aggregate class. Machine `error` evidence and original aggregate errors remain separate. Unknown backend text must never become a human reason label.
