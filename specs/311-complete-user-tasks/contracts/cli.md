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
- At most one variable source; an object is required when supplied. Reject unreadable files, malformed JSON, trailing values, arrays, scalars and outer `null`. Omission and `{}` are valid and still complete every unique key.
- Explicit `--workers` must be positive; an unset worker flag uses existing automatic selection. An explicit worker count wins over `--no-worker-limit` under existing semantics.
- `--complete --no-wait` is invalid. Variable mode retains no-wait. No `--api-diagnostics` flag is added.
- Preserve `validateUpdateUserTaskJSONConfirmation`: JSON requires dry-run, auto-confirm or automation; JSON+verbose is invalid. Preserve output and quiet/logger-level precedence.
- Configuration and explicit-key authorization use existing mechanisms. A configured discovery tenant does not restrict native keyed completion. Do not fetch task data merely to populate tenant context.

## Preview and confirmation

Freeze keys and payload before confirmation. Preview validates local input and describes intent; it does not establish task existence, eligibility, assignment, or current state. It performs zero task/variable reads, zero mutations, and no prompt. The preview's `variablesProvided` distinguishes no source from an empty object; `variableCount` reports the number of supplied names without echoing values.

Ordinary interactive execution asks once, using the command's configured/inherited stderr writer. Keep the established sentence form, for example: `You are about to complete N user task(s). Do you want to proceed?`, with existing default-answer and input behavior. Preserve prompt eligibility when stdout is redirected. Auto-confirm and automation follow existing rules without prompting. Decline/default rejection/EOF submit nothing and retain existing abort classification.

## Output contract

The completion report uses the stable fields in the data model, with ordered items and initialized arrays. The shared envelope remains unchanged. No completion fields are injected into the variable-update result schema.

| Mode/outcome | stdout | Diagnostic and control channels |
|---|---|---|
| Human preview | Compact completion preview with selected count, supplied-variable count and no-submission wording | No prompt; configured logger only for diagnostics |
| Human execution | Compact counts and explicit acceptance-without-verification wording; verbose may list safe keyed statuses | One configured-logger ERROR on failure; plain prompt on configured stderr if eligible |
| JSON preview | One succeeded envelope with phase preview and preview/not_submitted items | No prompt; no human summary |
| JSON all accepted | One accepted envelope with phase execution | No prompt; no independent completion claim |
| JSON partial failure | One failure/invalid envelope per existing classification, with complete ordered report and original machine error evidence | No second error renderer or duplicate human aggregate |
| JSON local validation failure | One existing classified error envelope; no invented task outcomes | No mutation or prompt |
| Keys-only preview | Selected keys, one per line | No prose on stdout |
| Keys-only execution | Accepted keys only, in selection order; zero bytes if none accepted | One compact configured-logger error when applicable |
| Quiet human | No informational summary/preview | Retain compact error and existing prompt eligibility |
| Quiet + JSON/keys | Preserve explicitly selected machine results | Honor existing quiet precedence for diagnostics |
| JSON + keys-only | Existing JSON precedence | One envelope, never mixed output |

Human accepted wording must say requests were accepted and confirmation was skipped, not “tasks completed.” Default output is aggregate-first and must not list each lifecycle step. Verbose details may include key, status, submission evidence and known HTTP status, never raw `Error`. Results go through the new stdout renderer, not `renderHumanLine` or the existing variable failure view.

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

After non-JSON execution failure, log one ERROR through the configured command-context logger. Include accepted/failed/interrupted/not-attempted counts as relevant and useful bounded reasons derived only from known status/submission codes, standard HTTP status names, or classified local errors. Group repeated causes; emit at most three named reason groups plus an additional-failure count. Use a stable prefix such as `user-task completion:`. Do not include task keys in the aggregate error or interpolate raw joined/transport/backend errors.

Quiet retains the error. Verbose may reveal safe outcome facts but does not enable HTTP diagnostics. DEBUG is controlled by existing flag/configuration and redaction rules, including quiet precedence. Actual transport records may describe requests and retries; no command-specific diagnostic instrumentation, synthetic exchanges, or latency report.

Machine error evidence is deliberately preserved in JSON; the normal-level prohibition on raw HTTP details applies to human results/diagnostics, not the established machine error contract. DEBUG must not change result stdout or route diagnostic records there.

## Required command execution coverage

Cover aliases; all key/payload forms; malformed late input; omitted/empty variables; no separate variable workflow; dry-run; one confirmation; decline and EOF; automation; workers and fail-fast; accepted, rejected, uncertain and unattempted outcomes; all output/quiet combinations; original exit status; exact single-envelope EOF; explicit tenant behavior; and supported versions.

Use real terminal stdin with separate stdout/stderr and configured/inherited destinations. Diagnostic cases use repeated distinctive fake backend failures and assert actual logger framing, severity and destination. DEBUG tests use both activation paths and verify redaction and actual retry/request counts for sequential and concurrent execution. Retain existing variable and read regression behavior, including their established human stream contracts.
