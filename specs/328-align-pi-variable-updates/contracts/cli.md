# CLI Contract: `c8volt update pi`

## Invocation and preparation

Retain existing aliases, explicit/repeated/stdin key input, variable payload syntax, flags, and validation. This command has no process-instance selector mode. Missing keys and empty stdin remain errors before mutation. JSON continues to require an existing noninteractive approval route or dry-run, and JSON with verbose remains invalid. Preserve existing option precedence and automation behavior.

The command requests one complete service plan, adapts it into the existing preview, handles dry-run/no-op, obtains approval if eligible, executes that same plan, and renders outcomes. It must not own paging, comparison, polling, retry, or worker loops. Tenant evidence travels with the plan; rendering adds no reads.

## Preview and successful no-op

Retain existing preview fields:

`operation`, `requestedKeys`, `requestedCount`, `updateCount`, `variableAddCount`, `variableChangeCount`, `variableUnchangedCount`, `variableUntouchedCount`, `processInstances,omitempty`, `mutationSubmitted`.

Per-instance fields remain `processInstanceKey`, `additions`, `changes`, `unchangedRequested`, and `untouched`. Their existing nil-array serialization as `null` remains; do not normalize them to empty arrays merely for this change. Preserve planned value/change fields and existing `apiTruncated,omitempty` behavior. `mutationSubmitted` is false for all previews and no-ops.

| Mode | Dry-run | All-unchanged execution |
| --- | --- | --- |
| Human | Existing compact/verbose preview and `pi:<key>` root labels | Existing no-change wording; no approval prompt |
| JSON | One existing successful envelope containing preview | One existing successful envelope containing no-op preview |
| Keys-only | Selected keys, including unchanged selections, one per line | Zero stdout bytes |
| Quiet | Existing informational suppression | Existing informational suppression |
| Quiet + JSON/keys | Explicit machine output preserved | JSON preserved; no-key output remains zero bytes |

JSON takes precedence over keys-only when both are enabled. Empty internal scopes are safe, but missing command targets are not converted into no-op success. A selected instance with no root variables still needs additions when values are requested.

## Executed results and errors

Preserve existing result fields described in [data-model.md](../data-model.md). Serialize accepted/no-wait writes as `submitted`, confirmed values as `confirmed`, failures as the existing mutation/confirmation failure statuses. Add `unchanged` and `skipped` status values for honest mixed-collection reporting without adding public fields.

Keys-only execution emits unique accepted mutation keys, including accepted writes whose later verification fails; no key is emitted for unchanged, skipped, or rejected work. Partial errors retain successful and failed/skipped entries. Emit one failure envelope with available results and preserve existing nonzero exit/error classification; do not append a second generic envelope after rendering the partial result.

Default human output stays compact and follows existing stream routing. Existing human logger summaries remain on stderr; stdout is uncontaminated selected machine output. Functional detail stays behind verbose and low-level HTTP diagnostics behind DEBUG independently.

## Approval versus verification

Approval is the existing operator prompt, routed to configured or inherited `cmd.ErrOrStderr()` as plain text. Verification is a backend read workflow owned by the service. `--no-wait` skips verification only. Preserve prompt wording, eligible input rules with redirected stdout, default answers, EOF, decline, auto-confirm, and automation behavior.

Dry-run and completed no-op plans never prompt or write. Aborts and errors remain distinct from no-op success. A one-page immediately-visible real update performs one planning read and one verification read, with no third confirmation read. Multi-page results and normal polling may require additional reads within those workflows.

## Documentation acceptance

Update command source help/examples/metadata and README to describe complete planning, selective writes, local root scope, no-op, no-wait, and added outcome values. Regenerate CLI docs with `make docs-content`; do not hand-edit generated files. Do not advertise new flags, selection modes, or changed UT behavior.
