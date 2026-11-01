# Contract: Complete Resource Views

**Scope**: [specification](../spec.md), FR-008–FR-017.
**Model rules**: [data-model.md](../data-model.md).

## Shared rules

Optional `businessId` is a field of the resource observation. JSON omits nil and preserves any backend-provided string, including empty. Existing outer envelopes, payloads, enrichment sections, null/array semantics, and field names are unchanged. A process instance's current ID never overwrites a job, task, child instance, or listener ID.

Human rows show a compact `bid:<value>` when present, beside the process name in process-instance rows. Leave letters, digits, and `-_.@/:` unquoted; quote empty values and values containing other characters. Use existing row conventions, escaping control characters so an opaque value cannot add lines or corrupt columns. JSON preserves the actual string. Saved Markdown must escape table delimiters, line breaks, and markup appropriately without changing the underlying value. Do not truncate or normalize the stored model field.

This is a resource-view addition, not a new ID lookup or enrichment operation. Key/status-only results, local exact-key previews, and fallback objects such as `ProcessInstance{Key: rootKey}` keep ID absent. Totals stay counts. Keys-only stays one key per line, or zero bytes for no keys. Keep empty results and quiet/machine precedence unchanged.

## Required source and acceptance inventory

Each row requires present, absent, and different-from-related-resource values in human and JSON output where supported. Exercise command paths in addition to pure renderers for changed stream/error behavior.

| Surface | Primary source locations and copy sites | Acceptance focus |
| --- | --- | --- |
| Creation | `cmd/run_processinstance.go`; move dedicated result/flat-row formatting to new `cmd/cmd_views_processinstance_run.go` | Both selectors; response vs observed ID; no-wait; partial result envelope; configured human logger |
| PI single/list/age | `cmd/cmd_views_processinstance.go`: `flatRowPIWithTimezone`, age wrappers | Optional field survives ordinary and age metadata JSON; no unrelated field changes |
| PI variables/incidents | `cmd/cmd_views_processinstance_vars.go`, `cmd/cmd_views_processinstance_incidents.go` | Embedded PI retains ID; existing section shapes remain |
| PI elements/activity | `cmd/cmd_views_processinstance_activity.go`: custom `MarshalJSON`, merge and traversal conversion | Combined variants retain PI and independent listener IDs; unrequested listeners omitted, requested-empty listeners unchanged |
| Walk | `cmd/cmd_views_walk.go`, `cmd/cmd_views_walk_incidents.go` | Family, parent/child, flat/tree, enriched output; parent and child IDs may differ |
| Jobs | `cmd/cmd_views_job.go`, `c8volt/job/client.go` | Single/list rows and observed update preview data |
| Bulk completion candidates | `internal/services/job/completion/{prepare.go,execution_plan.go}`, `c8volt/job/convert.go`, `cmd/cmd_views_job.go:jobCompletionCandidateRow` | Candidate reconstruction and frozen plans preserve independent job ID; this adds no job-completion assignment input |
| User tasks | `cmd/cmd_views_usertask.go`, `cmd/cmd_views_usertask_vars.go` | Ordinary and variable-enriched output preserve task ID |
| Runtime listeners | `cmd/cmd_views_listener.go` and adapters in activity, element, and slow-analysis views | Source job ID survives every listener adapter; no parent fallback |
| Slow process analysis | `internal/services/ops/slow_process_analysis.go`, `c8volt/ops/convert.go`, `cmd/cmd_views_ops_slow_process_analysis.go`, `cmd/cmd_views_ops_slow_process_analysis_machine.go` | Root reconstruction, summary, full timeline, process and listener IDs |
| PI mutation previews | `cmd/cmd_views_processinstance_dryrun.go`, `cmd/cmd_views_processinstance_update.go`, suspension/resumption tree views, `cmd/delete_processinstance_selector.go:aggregateDeleteSearchPlan` | Bespoke selected-final-state and requires-cancel entries carry observed IDs; exact-key-only preview remains read-free |
| Operations previews/reports | `c8volt/ops/convert.go` duplicated PI converter; `cmd/cmd_views_ops_execute_retention_policy.go`, orphan/incident/all-definition purge and repair view files; `cmd/ops_report*.go` | Retention, purge, repair observed data, JSON reports, and custom saved Markdown lists such as `retentionProcessInstanceItems` |
| Definition deletion impact | `internal/services/processdefinition/delete.go`, process facade impact conversion, `cmd/delete_processdefinition.go` and existing impact renderers | Observed impacted instances retain IDs without adding discovery |

## Required adversarial fixtures

Use a parent process ID `new`, child/job/task/listener IDs `old`, and nil values in the same fixture. Add a backend-provided empty string to mapping/JSON tests; it is preserved even though explicit-empty creation input is invalid. Add a quoted/comma-containing value and control/Markdown delimiters to rendering tests. Verify round trips and pointer isolation for public input and frozen completion plans.

Successful empty-scope execution must render exactly the existing successful empty payload under JSON and zero bytes under keys-only. Cover normal/dry-run and applicable automation, auto-confirm, quiet, and no-wait combinations. Do not infer a successful no-op from an intermediate empty page or an error. Existing sparse-page, explicit-key, abort, and error behavior remains.

## Stream and diagnostic acceptance

Use command execution with separate stdout/stderr and configured/inherited logger destinations. Assert INFO routing of mutation, dry-run, and verbose results in both plain and structured formats; quiet and WARN suppress those records. ERROR remains visible under quiet. Query rows stay on stdout and prompts plain stderr.

Include repeated multi-resource backend failures containing a distinctive URL, method, body, and resource key. At normal and verbose-only levels, both streams must exclude raw HTTP evidence while the configured logger emits one compact aggregate ERROR. Test both `--debug` and configured debug level independently of verbose, verify redaction, and keep stdout diagnostic-free. Respect current incompatible-flag precedence rather than inventing new supported combinations.

For shared-contract JSON, decode one envelope and require EOF; preserve partial payload and original exit status. Assertions must fail if a logger call is replaced with a direct print, if raw aggregate errors leak, or if an error is emitted twice. Use real terminal stdin for affected interactive routing and prompt-free empty-scope paths, retaining configured/inherited stderr and redirected stdout behavior.
