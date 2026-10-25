# CLI Contract: Element Scope Variables

The command spellings below describe the intended feature. They are documentation examples, not live validation instructions. Use only isolated fixtures in [quickstart.md](../quickstart.md) to validate implementation.

## Inspection

```text
c8volt get element --key <element-key> --with-vars
c8volt get ei --pi-key <process-instance-key> --with-vars --limit 10
c8volt --json get element --key <element-key> --with-vars --with-listeners
c8volt --keys-only get element --pi-key <process-instance-key> --with-vars
c8volt get element --pi-key <process-instance-key> --total --with-vars
```

Add only `--with-vars`. Preserve existing exact-key/search grammar, aliases, filters, bounds, and conflicts. Keys-only and total suppress variable retrieval; they do not bypass listener or other validation. Variable-only search retains ordinary page continuation prompts, sparse-page continuation and early stops. Combined listener/variable search retains collected listener behavior. Each element's local variables are complete independently of the element selection limit.

Human output retains existing element rows and listener formatting. Add deterministic local-variable child rows using shared value formatting and truncation indicators, with correct tree connectors when listeners are also present. Empty local variables leave the element visible without inventing a variable. No new display-control flag is required. Empty selected collections retain established `found: 0` human wording; quiet suppresses informational summaries.

JSON keeps the existing single-element or collection payload inside the shared envelope, adding optional `variables` to each selected element. No new outer item wrapper is introduced. Without performed enrichment omit `variables`; after a successful empty local read emit `variables: []`. Preserve received value strings, `apiTruncated`, and existing listener fields. Empty collection shape/null-versus-empty semantics remain those of the current get renderer.

Keys-only emits selected keys once each, one per line, including elements with empty local collections; no selection means zero bytes. Numeric total behavior remains unchanged. Search discovery honors configured tenant filtering, including the correction described in [research](../research.md); exact keys bypass that discovery filter while retaining backend authorization.

## Update input and lifecycle

```text
c8volt update element --key <element-key> --vars '{"approved":true}' --dry-run
c8volt update ei --key <key-a>,<key-b> --key <key-a> --vars-file ./vars.json --auto-confirm
c8volt update element - --vars '{"status":null}' --auto-confirm
c8volt --json update element --key <element-key> --vars '{"approved":true}' --auto-confirm --no-wait
```

Register `update element`, alias `ei`, as a mutation command with the shared output/automation contract. Accept repeated/comma-separated `--key`/`-k` and newline-separated keys from stdin when positional `-` is present. Preserve merge order and deduplicate first occurrence. No search selectors, discovery fallback, BPMN-ID interpretation, or implicit called-process traversal are added.

Require exactly one of `--vars` or `--vars-file`, containing a non-null JSON object. Validate source conflicts, key syntax, absent/empty selections, malformed JSON, wrong top-level types, and unreadable files before remote work. Keep established variable-update value semantics. `{}` is valid after authoritative planning; a null property sets null.

Preserve the existing flags and inherited policies for `--dry-run`, `--workers`/`-w`, `--no-worker-limit`, `--fail-fast`, `--no-wait`, auto-confirm, automation, timeout/backoff, and invocation-wide request rate. JSON mutation requires dry-run, auto-confirm, or automation; JSON with verbose is rejected under existing variable-update guardrails. Use existing effective mode precedence rather than adding a second resolver.

The command completes all identity/authorization/local-variable planning before any approval prompt or write. Preview destinations use `element:<key>`. Dry-run and completed no-op plans return without mutation prompts, writes, or confirmation reads. An empty local scope plus requested names is an addition plan. Decline/EOF, missing targets, and incomplete discovery retain their existing error/abort behavior and never become no-ops.

After approval, execute the frozen plan once. One logical local write per changed unique element carries only additions/changes, subject to existing retries. Normal execution verifies every requested value at that scope; no-wait reports accepted submissions without verification. No replanning, parent redirection, resubmission after confirmation failure, or rollback is implied.

## Update payloads

The shared envelope stays unchanged. New command payloads use stable structs with camelCase fields.

| Path | Payload contract |
|------|------------------|
| Dry-run and authoritative no-op | Preview with `operation: "update"`, `requestedKeys`, `requestedCount`, `updateCount`, four variable classification counts, `elements`, and `mutationSubmitted: false` |
| Preview element | `elementKey`, `processInstanceKey`, `scopeKey`, optional known `tenantId`, and `additions`, `changes`, `unchangedRequested`, `untouched` arrays |
| Executed success or partial failure | `items` array of the ordered outcomes defined in [data-model.md](../data-model.md) |

Initialize all preview classification arrays to `[]`. Valid update input always selects at least one element, so preview contains those entries even for `{}`. Outcome items cover all unique planned keys, including unchanged and skipped entries. Planning failures use the ordinary shared error contract and do not manufacture a complete plan or execution report. Carry known tenant context through existing envelope/context support without presentation-only requests.

Envelope selection is explicit: dry-run/no-op always `succeeded`; successful fully verified execution is `succeeded`; successful no-wait execution with actual acceptance is `accepted`. A failed execution uses the original error-derived outcome/class and includes partial results regardless of no-wait. Accepted writes whose confirmation never started after interruption retain accepted evidence and `confirmationStatus: "not_started"`; they do not make an ordinary interrupted execution successful. Never select accepted solely from the flag on an empty/no-op path.

## Mode and stream matrix

| Mode | Preview/no-op | Executed update | Failure handling |
|------|---------------|-----------------|------------------|
| Human | Compact counts and exact destination details under established single-target/verbose preview conventions | Compact aggregate counts; functional per-element lifecycle detail only under verbose | One bounded grouped ERROR through the configured command logger; never raw item/aggregate errors |
| JSON | One succeeded envelope containing preview | One envelope containing ordered outcomes | Same single envelope retains partial outcomes and original error class; no second renderer or extra diagnostic emission |
| Keys-only | Changed destination keys for dry-run; zero bytes for no-op | Only mutation-accepted keys, even after confirmation failure | Retain accepted stdout keys and nonzero exit; one compact configured stderr error |
| Quiet | Suppress human information, not explicit JSON/keys | Preserve explicit machine results | Retain compact required human ERROR when not in JSON |

Classify output before implementation: selected-format results use stdout result renderers; progress/diagnostic messages use the configured logger; interactive control text uses plain configured/inherited stderr. Quiet suppression must be mode-aware even for empty discovery or early return. Human summaries must not leak raw request/error details through a result renderer. Use existing safe formatting patterns without copying nearby direct-print or raw-error violations.

Ordinary human update summaries distinguish counts of accepted, confirmed, unchanged, failed, and skipped work without claiming all selected work was updated. Preview/no-op states explicitly indicate no changes applied/no confirmation required. Safe verbose details may contain the intended resource key and allowlisted lifecycle status, but never HTTP request paths, methods, bodies, cursors, raw response status strings, or wrapped errors.

For failed batches, group repeated causes into a bounded number of safe reasons with nonzero failure counts. Derive reasons from known error classes/enums or standard status labels. Never pass raw `Error`, `Message`, joined errors, transport errors, or arbitrary backend strings into human logging. Quiet retains this ERROR. Existing machine evidence remains available; do not modify existing shared schemas to solve human formatting.

## Failure and prompt ownership

Execution errors travel service → facade conversion with partial results → mode-aware result rendering → one configured human aggregate ERROR when applicable → original classified exit. `exitAfterRenderedResult` does not log; the new batch runner must explicitly log once for non-JSON failures. After rendering the shared JSON envelope, do not invoke another error renderer. Planning/input failures use established safe command/bootstrap helpers, with bounded human reasons.

Prompt helpers receive `cmd.ErrOrStderr()` explicitly, including inherited destinations. Prompt text has no logger prefix. Retain existing wording patterns, default answer, input/EOF handling, auto-confirm/automation, and caller-specific decline behavior. Eligibility depends on terminal stdin as before; redirected stdout does not disable an otherwise eligible prompt. Empty/no-op/dry-run scopes never prompt.

DEBUG diagnostics remain in the existing redacted path, enabled by either `--debug` or configured DEBUG independently of verbose and subject to current flag precedence. They never write to stdout or trigger extra requests. Normal/verbose/quiet human diagnostics expose no raw HTTP details on either stream.

## Compatibility and documentation

Preserve existing PI root updates, UT effective-scope updates, task completion, listeners, output precedence and error classifications. Configured 8.7 continues to fail in bootstrap with local-precondition exit 1 before remote work; do not promise a new JSON bootstrap envelope. Support 8.8/8.9/8.10 through version adapters.

Update README, get/update parent discovery/help, command metadata, examples, and automation descriptions with local-scope boundaries, modes, tenant correction, no-op, no-wait, and partial/uncertain outcomes. Regenerate CLI documentation using `make docs-content`; do not edit generated `docs/cli/*` by hand.
