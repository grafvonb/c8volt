# CLI Contract: Complete User Tasks

**Issue**: #311 | **Specification**: [spec.md](../spec.md) | **Payload model**: [data-model.md](../data-model.md)

## Grammar and modes

```text
c8volt update user-task [-] --key <key>[,<key>...] [--key <key>...] --complete
    [--vars <JSON-object> | --vars-file <path>]
    [--dry-run] [--workers <positive-int> | -w <positive-int>]
    [--no-worker-limit] [--fail-fast]
    [existing confirmation, automation, tenant, timeout, output and logging flags]
```

Keys may come entirely from stdin with positional `-`; `--key` is not required when stdin supplies a valid selection. Preserve aliases `user-tasks`, `ut`, and `uts`, and key shorthand `-k`. Without `--complete`, preserve the existing variable-update grammar and required variable source. An explicit `--complete=false` selects existing variable mode.

Examples for documentation (do not execute these against a real environment during implementation):

```sh
c8volt update user-task --key 2251799813685249 --complete
c8volt update ut --key 2251799813685249 --key 2251799813685250 --complete --auto-confirm
c8volt update ut --key 2251799813685249,2251799813685250 --complete --vars '{"approved":true}' --dry-run
c8volt update ut --key 2251799813685249 --complete --vars-file ./vars.json --dry-run
c8volt update ut --key 2251799813685249,2251799813685250 --complete --workers 1 --auto-confirm --debug
printf '%s\n' "$TASK_KEY_A" "$TASK_KEY_B" | c8volt update ut - --complete --auto-confirm
c8volt update ut --key 2251799813685249 --vars '{"approved":true}'
c8volt update ut --key 2251799813685249 --vars-file ./vars.json --no-wait --auto-confirm
```

## Validation and selection

- Consume flag values in supplied order, then stdin lines. Ignore blank stdin lines; validate every nonempty input token using the existing 16-digit user-task key rule. Reject empty flag tokens rather than silently discarding them.
- Deduplicate in first-input order. Validate all keys and payload content before any completion request, including invalid input late in stdin.
- Empty final selection is invalid, including empty stdin alone. Valid flag keys plus empty stdin is valid. No input triggers discovery or a successful completion no-op.
- At most one variable source; an object is required when supplied. Reject unreadable files, malformed JSON, trailing values, arrays, scalars and outer `null`. Omission and `{}` are valid and still complete every eligible unique key.
- Explicit `--workers` must be positive; an unset worker flag uses existing automatic selection. An explicit worker count wins over `--no-worker-limit` under existing semantics.
- `--complete --no-wait` is invalid. Variable mode retains no-wait to skip its additional verification. Completion has no additional client waiter; no-wait cannot bypass the native request or blocking listener processing. No `--api-diagnostics` flag is added.
- Preserve `validateUpdateUserTaskJSONConfirmation`: JSON requires dry-run, auto-confirm or automation; JSON+verbose is invalid. Preserve output and quiet/logger-level precedence.
- Configuration and explicit-key authorization use existing mechanisms. A configured discovery tenant does not restrict native keyed completion. Do not fetch task data merely to populate tenant context.

## Preview and confirmation

Freeze keys and payload before confirmation. Preview validates local input and describes intent; it does not establish task existence, eligibility, assignment, or current state. It performs zero task/variable reads, zero mutations, and no prompt. The preview's `variablesProvided` distinguishes no source from an empty object; `variableCount` reports the number of supplied names without echoing values.

Ordinary interactive execution asks once for the eligible count after preflight, and never prompts if no submissions remain, using the command's configured/inherited stderr writer. For one selected key use `You are about to update user-task <key>. Do you want to proceed?`; for bulk input use `You are about to complete N user tasks. Do you want to proceed?` (singular `task` when only one is eligible), with existing default-answer and input behavior. Preserve prompt eligibility when stdout is redirected. Auto-confirm and automation follow existing rules without prompting. Decline/default rejection/EOF submit nothing and retain existing abort classification.

## Preflight checks

After local validation, read each unique task once before confirmation; do not retry the read or recheck after approval. A matching `COMPLETED` observation produces a conflict without POST or variable application. Preserve the existing schema: `status: failed`, `submission: not_submitted`, safe `error`, omitted `statusCode`, incremented failed count, and the existing conflict exit classification. With fail-fast, any such conflict stops all submissions; other keys remain not_attempted. Otherwise they continue.

Warn through the configured command logger before the prompt when eligible tasks have no assignee. This is advisory, never implicit assignment. Failed/malformed/mismatched reads produce one aggregate warning that the check was unavailable; they do not imply completion or unassignment and the backend decides whether completion is permitted. Caller cancellation stops preparation. The successful read is a snapshot, not a lock or guarantee of future eligibility.

Warnings contain bounded counts, no raw read errors or per-key transport details. They use configured/inherited diagnostic destinations and existing quiet suppression, including JSON/keys modes; stdout remains JSON/keys-only results only. DEBUG retains existing redaction. Dry-run remains local with no checks or warnings.

## Output contract

The completion report uses the stable fields in the data model, with ordered items and initialized arrays. The shared envelope remains unchanged. No completion fields are injected into the variable-update result schema.

| Mode/outcome | stdout | Diagnostic and control channels |
|---|---|---|
| Human preview | Empty | INFO intent with the single key or selected count, optional `variables: submit`, and no-submission wording; no prompt |
| Human execution | Empty | INFO single-key success or compact batch/failure counts and verbose rows; one ERROR on failure; plain prompt on configured stderr if eligible |
| JSON preview | One succeeded envelope with phase preview and preview/not_submitted items | No prompt; no human summary |
| JSON all accepted | One accepted envelope with phase execution | No prompt; no independent completion claim |
| JSON partial failure | One failure/invalid envelope per existing classification, with complete ordered report and original machine error evidence | No second error renderer or duplicate human aggregate |
| JSON local validation failure | One existing classified error envelope; no invented task outcomes | No mutation or prompt |
| Keys-only preview | Selected keys, one per line | No prose on stdout |
| Keys-only execution | Accepted keys only, in selection order; zero bytes if none accepted | One compact configured-logger error when applicable |
| Quiet human | Empty | Retain compact error and existing prompt eligibility |
| Quiet + JSON/keys | Preserve explicitly selected machine results | Honor existing quiet precedence for diagnostics |
| JSON + keys-only | Existing JSON precedence | One envelope, never mixed output |

A single success says `updated user-task <key>: completion confirmed`. Batch and failure results say `update user-task: N requested; ...`, use `completed` for successful responses, and omit zero counters. There is no generic response-received or polling suffix. A successful native response confirms completion, including blocking task listeners; do not imply independent read-back, downstream completion, or success for uncertain/failed/unattempted items. Do not say `confirmation skipped`, which confuses native completion evidence with an optional client waiter. Keep all machine fields, status/submission identifiers, counts, envelope outcomes, keys-only selection, and exit behavior unchanged. Default batch output is aggregate-first and must not list each lifecycle step. Verbose details may include key, status, submission evidence and known HTTP status, never raw `Error`. Failed/not_submitted items say `task is already completed; completion request not sent`. The aggregate uses `N tasks are already completed; their completion requests were not sent` for multiple conflicts. Append `; supplied variables were not applied` only when variablesProvided is true, including a supplied `{}`. Plans and previews omit variables when absent and add `variables: submit` when supplied, including `{}`; values are never echoed. These human changes leave all machine counts, item error strings, fields and conflict exits unchanged. Human previews, success/failure summaries and verbose rows use `renderHumanLine` and the configured INFO logger, including structured formatting and configured/inherited destinations. Configured WARN/ERROR levels and quiet suppress INFO; quiet retains ERROR. JSON/keys rendering remains unchanged and never gains INFO records. Do not create another logger or print a synthetic INFO prefix.

Preview item illustration (inside the shared envelope payload):

```json
{
  "operation": "complete",
  "phase": "preview",
  "requestedKeys": ["2251799813685249"],
  "requestedCount": 1,
  "variablesProvided": false,
  "variableCount": 0,
  "items": [{"key": "2251799813685249", "status": "preview", "submission": "not_submitted"}],
  "counts": {"preview": 1, "accepted": 0, "failed": 0, "timedOut": 0, "canceled": 0, "notAttempted": 0}
}
```

All-accepted execution uses `renderAcceptedResult` explicitly. Do not infer acceptance from no-wait, since that flag is rejected here. Failure uses `resultEnvelopeForError` with the report attached, renders once, and uses `exitAfterRenderedResult` with the original converted error. Preserve existing no-error-code overrides; do not invent exit numbers. Validation/initialization failures use existing command/bootstrap helpers before any completion result is rendered.

## Aggregate diagnostics

After non-JSON execution failure, log one ERROR through the configured command-context logger. Lead with useful bounded reasons derived only from known status/submission codes, standard HTTP status names, or classified local errors. Do not repeat the result summary counters in the diagnostic. Group repeated causes with nonzero counts; emit at most three named reason groups plus an additional-failure count. Use a stable prefix such as `user-task completion:`. Do not include task keys in the aggregate error or interpolate raw joined/transport/backend errors.

Quiet retains the error. Verbose may reveal safe outcome facts but does not enable HTTP diagnostics. DEBUG is controlled by existing flag/configuration and redaction rules, including quiet precedence. Actual transport records may describe requests and retries; no command-specific diagnostic instrumentation, synthetic exchanges, or latency report.

Machine error evidence is deliberately preserved in JSON; the normal-level prohibition on raw HTTP details applies to human results/diagnostics, not the established machine error contract. DEBUG must not change result stdout or route diagnostic records there.

## Required command execution coverage

Cover aliases; all key/payload forms; malformed late input; omitted/empty variables; no separate variable workflow; dry-run; one confirmation; decline and EOF; automation; workers and fail-fast; accepted, rejected, uncertain and unattempted outcomes; all output/quiet combinations; original exit status; exact single-envelope EOF; explicit tenant behavior; and supported versions.

Use real terminal stdin with separate stdout/stderr and configured/inherited destinations. Diagnostic cases use repeated distinctive fake backend failures and assert actual logger framing, severity and destination. DEBUG tests use both activation paths and verify redaction and actual retry/request counts for sequential and concurrent execution. Retain existing variable and read regression behavior, including their established human stream contracts.

### Completed-task human example

Without a variable payload:

```text
INFO update user-task: 1 requested; 1 failed
ERROR user-task completion: task is already completed; completion request not sent
```

The ERROR uses the configured logger, including its usual timestamp/format. When variables were supplied, append `; supplied variables were not applied`. Conflict exit code remains 6 under ordinary exit-code settings. JSON retains the full original envelope, zero counters and machine error evidence; keys-only still emits zero bytes when nothing was accepted.

## Ordinary human approval rhythm

Scope and plan use the configured INFO logger, warnings use WARN, confirmation is plain configured stderr, and human results use the same configured INFO logger; JSON and keys-only results use stdout. With logger timestamps omitted for readability:

```text
INFO selection scope: explicit resource keys; tenant filter not applied
INFO plan: update user-task 2251799813685249: completion: submit
WARN user-task completion: task has no assignee
You are about to update user-task 2251799813685249.
Do you want to proceed? [y/N]: y
INFO updated user-task 2251799813685249: completion confirmed
```

The warning appears only when observed. For bulk input, the plan uses `plan: update user-task completion: N requested; M eligible; C already completed`, omitting zero counts. Fail-fast with a mixed selection replaces eligibility with `fail-fast: no completion requests will be sent`; it does not prompt. All-completed selections skip the plan and prompt. No affected tenant is inferred or fetched solely for display. Auto-confirm and automation skip the approval plan; quiet and machine output suppress INFO. Dry-run shows known selection scope and an INFO preview such as `dry run: update user-task <key>: completion: submit; no requests submitted`, with no backend reads or prompt. These paths preserve existing warning suppression and result-mode precedence.
