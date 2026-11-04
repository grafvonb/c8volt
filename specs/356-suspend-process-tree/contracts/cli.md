# CLI Contract: Process-Instance Tree Suspension

**Feature**: [spec.md](../spec.md) | **Data model**: [data-model.md](../data-model.md) | **Service contract**: [facade-service.md](facade-service.md)

## Grammar and unchanged defaults

```text
c8volt suspend process-instance --key <key>[,<key>...] [--tree] [existing suspension flags]
c8volt suspend pi --key <key> --tree --dry-run
c8volt suspend pi --key <key> --tree --auto-confirm --no-wait --json
printf '%s\n' "$PARENT" "$CHILD" | c8volt suspend pi --tree --auto-confirm -
```

Add boolean `--tree` with default false, no shorthand. Suggested action-focused usage: `include each selected instance and its descendants; suspend active instances only`. Update metadata/tests without adding unrelated flags or output modes. Existing repeated/comma-separated flags and stdin `-` validation remain authoritative. No `--force`, root escalation, arbitrary state selector, or search selector is introduced.

Without --tree, preserve the existing runner, request counts, local preview, exact-key JSON shape, logs, prompt, rejection semantics, and spinner. JSON still requires dry-run/auto-confirm/automation and rejects verbose. Effective result precedence is JSON, then keys-only, then human; quiet remains orthogonal.

Keep help focused on scope, eligibility, permissions, and operational limits. Do not describe stdout/stderr/logger implementation details in Short/Long help; `TestAllCommandHelpFocusesOnActions` is a required regression gate. Routing belongs in this contract and README.

## Tree flow and file ownership

- `cmd/suspend_processinstance.go`: existing validation, --tree registration, metadata, and dispatch only.
- `cmd/suspend_processinstance_tree.go`: plan, preview/no-op decision, confirmation, existing collection call, activity lifetime, error/exit dispatch.
- `cmd/cmd_views_processinstance_suspend_tree.go`: typed tree payload, human scope messages, preview/no-op/results, and machine formatting. Reuse existing suspension formatters instead of changing shared view machinery.

Sequence:

1. Validate input/output modes normally. Call `PlanProcessInstanceSuspension` with existing explicit-input options; the service guards capability and performs discovery. Any error is handled before no-op or approval, through the established command error helper with bounded human wording.
2. Preserve the successful plan and copy TargetKeys. Derive all scope counts only from this frozen plan.
3. If dry-run, render a succeeded preview and stop, even with no-wait. Preview items are a view projection of eligible keys using existing preview statuses; no mutation call is needed.
4. If actual execution has zero eligible targets, render a succeeded no-op and stop, even with no-wait. No prompt, execution activity, or facade mutation call occurs.
5. For remaining actual runs, show the single compact plan line below when human INFO output is enabled, then invoke the existing confirmation helper once with the unique active count. Freeze both target keys and displayed scope before approval.
6. After approval, call existing `SuspendProcessInstances` with the copied keys and ordinary explicit-input execution options. Do not pass forced planner dry-run options. There is no second tree discovery or new mutation executor.
7. Close execution activity before rendering. Render one result, emit one safe aggregate non-JSON failure if needed, and retain the original operation error for exit handling.

## Confirmation and activity

Reuse the current question with N equal to len(plan.TargetKeys):

```text
You are about to suspend N process instance(s). Do you want to proceed?
```

The shared formatter owns final line breaks, `[y/N]: `, default-no, answers, and EOF. Use configured stderr as plain text with no logger prefixes. Real terminal stdin and redirected stdout retain existing eligibility. Dry-run, invalid input, unsupported version, failed planning, zero eligible targets, and implicit approval modes do not prompt. Decline submits no mutation.

The approved call uses the same local `indicatorEnabled` / `startCommandActivity` wrapper and `suspending N process instance(s)` message as ordinary suspension. Its lifetime includes all worker execution, observation, and backoff; deferred cleanup ends it before final output or unwinding. Preserve shared no-indicator/quiet/JSON/keys/automation/structured-logging/terminal restrictions. Do not add a planning spinner, progress counters, completion callbacks, or global pool/activity changes.

## Human messages

Use configured INFO helpers for all informational messages. Never print human summaries directly. Respect logger WARN/ERROR filtering, quiet, activity coordination, configured/inherited destination, and plain/structured formats. Do not put logger severity or timestamps into message strings.

The compact pre-confirmation message is:

```text
plan: suspend A process instance(s); S selected; D in scope; U already suspended; E excluded
```

Dry-run uses:

```text
dry run: suspend process-instance tree
selected process instances: S; process instances in scope: D; active targets: A; already suspended: U; excluded: E
outcome: preview; no changes applied
```

Final actual output preserves the frozen scope line above and reuses `formatProcessInstanceSuspensionSummary` for actual mutation results. If A is zero, the actual outcome instead is:

```text
suspension: no active process instances to suspend; no changes applied
```

S is the unique input-seed count, D the unique discovered count, A the planned active-target count, U the already-suspended count, and E the excluded count; D = A + U + E. Existing accepted, confirmed, failed, and not-attempted counts apply only to execution items, so an existing SUSPENDED observation never increases new acceptance/confirmation counts.

Verbose preview/final scope details may list each planned key, observed state, and allowlisted classification. Actual outcome rows reuse `formatProcessInstanceSuspensionOutcome` for eligible execution facts. Do not print per-key scope or lifecycle rows by default, concatenate arbitrary backend errors, or duplicate the same execution row through an activity reporter.

## Tree-only JSON payload

Keep the shared envelope. Tree mode alone uses this typed payload shape:

```json
{
  "items": [],
  "plan": {
    "requestedKeys": ["2251799813685249"],
    "instances": [
      {"key": "2251799813685249", "observedState": "SUSPENDED", "classification": "already_suspended"}
    ],
    "targetKeys": []
  }
}
```

This is an actual no-op example: outcome is succeeded, even with no-wait. No item is invented for the already-suspended member. All successful plan arrays and the items array are non-null; exact-key payloads do not acquire a plan field.

- Preview: succeeded envelope, one existing preview item per TargetKey; no accepted or observed-confirmation claim.
- Actual no-op: succeeded envelope with `items: []` plus the frozen plan. No accepted/pending claim.
- Actual waited success: succeeded envelope with existing per-key evidence plus the frozen plan.
- Actual no-wait with accepted submissions and no error: accepted envelope with acceptance-only items plus plan.
- Actual failure: select the existing failure envelope before considering no-wait and retain items plus plan, original class, and machine error evidence. No second envelope or duplicate human failure record.
- Planning/input/capability failure: one existing error envelope, no executable partial plan. Never render an empty succeeded plan after an error.

No shared schema or existing exact-key field changes are required. Rendering must not issue discovery or mutation requests.

## Keys-only and quiet

Preview emits only planned eligible keys, in order. Actual execution emits only accepted keys, including accepted-but-unconfirmed failures; retain the failure exit. Never output already-suspended, excluded, rejected, uncertain, or unsubmitted keys. An empty eligible scope emits zero bytes, including no blank line. Quiet suppresses informational scope/outcome logs without suppressing explicit JSON/keys or required failure diagnostics.

## Failure routing

Trace service cause -> facade error conversion -> one renderer -> configured safe aggregate diagnostic when non-JSON -> original exit code. Reuse `formatProcessInstanceSuspensionFailure` for mutation failures. Discovery/preflight errors receive a bounded command message while retaining wrapped causes for machine evidence and error classification.

After partial results, use `exitAfterRenderedResult`, not a second error renderer. Machine-writer failures get a compact configured ERROR and retain an existing operation error's exit; a partially written envelope is never followed by another envelope. Normal and verbose logs exclude URLs, HTTP methods, raw bodies/cursors, and joined transport chains. Existing redacted DEBUG diagnostics remain enabled independently by --debug or configured debug level, subject to existing quiet precedence.

## Documentation and acceptance checks

Update README and source help/metadata, then run `make docs-content` to regenerate CLI docs. Document additional discovery permissions, the retained up-to-1,000-direct-children lookup bound, independent subtree behavior, frozen/non-atomic scope, and possible omission of future descendants. Do not hand-edit generated docs.

Execution tests must cover mixed/nested/overlapping subtrees; exact-key zero-discovery controls; fail-before-mutation; no-active actual versus preview/no-wait output; post-approval hierarchy/state races; partial evidence; logger destination/format/level/redaction; one JSON envelope with EOF; keys zero bytes; real PTY prompt/no-prompt cases; and activity lifetime/suppression. See [quickstart.md](../quickstart.md).
