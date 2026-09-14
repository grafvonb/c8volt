# CLI Logging Contract: Process-Instance Delete and Cancel

Applies to existing process-instance delete/cancel commands and aliases. There are no new flags or result schemas. Existing tenant logging from #303 and HTTP exchange diagnostics from #305 remain authoritative.

## 1. Output ownership and admission

| Output | Owner | Admission / destination |
| --- | --- | --- |
| State observation | Shared process-instance waiter | Effective logger admits DEBUG; configured/inherited stderr. |
| HTTP diagnostic | Existing exchange observer | Existing DEBUG/quiet filtering and stderr; unchanged `api #...` grammar, fields, and redaction. |
| Workflow explanation | Existing selected service transition messages | Actual verbose enabled, permitted human mode, INFO admitted, existing quiet/automation restrictions. |
| Per-tree failure warning | Existing semantic completion renderer, with process-instance display projection | Existing warning/mode policy; no policy redesign. |
| Compact final human error | Command error view boundary | Existing human error admission and stderr; exit based on original error. |
| Full causal chain | Command error boundary | Once at admitted DEBUG per final aggregate workflow failure. |
| Result or error envelope | Existing command result view | stdout; original schemas, omission rules, text and class retained. |
| Interactive prompt | Existing explicit-writer prompt helper | Plain text on configured/inherited stderr; eligibility and input behavior unchanged. |

DEBUG without verbose does not enable the new workflow explanations. Verbose without DEBUG does not enable HTTP/state diagnostics. Configured DEBUG behaves like flag-enabled DEBUG. Existing quiet filtering wins. Machine result formatting and diagnostic log formatting are independent.

## 2. State observations

Use a stable, compact message with these ordered fields; logger timestamp/level/source remain logger-owned:

```text
pi state observation: key=<key> attempt=<n> state=<observed-state> elapsed=<duration> [next_delay=<duration>]
pi state observation: key=<key> attempt=<n> lookup_error=<safe-detail> elapsed=<duration>
```

The brackets denote an optional field, not literal output. Use existing ASCII duration conventions. Include exactly one observed-state or lookup-error outcome. ABSENT remains the established waiter interpretation of recognized not-found responses. Never include a full nested causal chain in the lookup-error field; the final DEBUG failure record owns that chain.

`elapsed` is measured at lookup completion. A pending nonterminal observation may be emitted when the existing sleep finishes or is interrupted, so `next_delay` is included only on an actual continuation. Terminal success, failed lookup, max attempts, deadline, and canceled waits omit it. Starting with an already canceled context emits no observation because no lookup occurred.

For N successful cached-auth checks each using one exchange: N observation records + N HTTP records. A 36-check fixture yields exactly 72 records in the polling scope. Initial token work and phase records are counted separately. Retries/multiple exchanges retain one HTTP record each and do not justify extra state observations. No nested checking/fetching/state-result or routine token cache lookup/hit records remain in the polling scope.

## 3. Workflow explanations

Each applicable transition has one owner and appears once per actual phase/root. Wording must communicate these facts:

| Transition | Required content |
| --- | --- |
| Cancellation required | Deletion conflicted and requires cancellation before deletion can proceed. |
| Root escalation | Requested child and affected root; why cancellation targets the root. |
| Cancellation submitted | Root/key and accepted submission; no claim of terminal state. |
| Wait entry | Known scope, desired states, effective timeout and configured backoff/attempt policy. |
| Deletion resumed | Cancellation confirmation completed and deletion is resuming for the relevant target. |

Wait policy information is emitted once at wait entry, not once per key observation. Distinct family and single-key confirmation waits remain distinguishable. Repeated transitions for different roots are legitimate, not duplicate narration.

## 4. Failure presentation

For the reported path, the combined normal transcript must make all of these facts available:

- The failed phase is cancellation confirmation.
- The affected root and known scope are identified.
- The wait budget/timeout is identified, distinguishing a shorter parent deadline where applicable.
- Last observed root/child states are shown when available and described as last observed.
- Initial child deletion was attempted and conflicted.
- Root cancellation was submitted; its final outcome is unconfirmed.
- Resumed deletion was not reached.

The tree warning carries concise operational facts; the final human command error carries a compact aggregate. Neither repeats the full nested chain. Example shape, with illustrative keys/durations:

```text
WARN delete tree 100: cancellation confirmation timed out after 120s; last observed 100=ACTIVE, 101=ACTIVE; child 101 deletion conflicted; root cancellation submitted, outcome unconfirmed; resumed deletion not reached
ERROR delete process instances: cancellation confirmation timed out for root 100 (1 tree); cancellation submitted, outcome unconfirmed
```

Actual messages retain established surrounding warning/command conventions. Multiple-root details use deterministic key ordering and no invented states for unseen keys. At DEBUG, one final full-chain record is added; concise HTTP error evidence and observation outcomes are separate technical records, not repeated full-chain dumps.

Machine error envelopes continue to use original normalized error text/classification. The DEBUG copy on stderr does not authorize changing JSON detail or report/status fields. Quiet and no-err-codes keep their existing behavior.

## 5. Compatibility invariants

- No added requests, discovery, mutations, sleeps, retries, or changed cancellation/deletion ordering.
- Existing family and single-key checks remain; successful submission is never described as rollback or definitive cancellation failure solely because confirmation timed out.
- Direct lookups retain useful diagnostics; v87 retains search-based tenant-safe state lookup and unsupported direct-get behavior.
- JSON contains exactly one applicable envelope/value followed by EOF, with existing null/empty/omission rules.
- Keys-only contains only result keys, one per line; empty scopes produce zero bytes.
- Dry-run remains a preview; no-wait retains submission semantics; completed empty scope submits no mutations and prompts for none.
- Real-terminal stdin with redirected stdout keeps prompts on configured/inherited stderr; confirmation text, defaults, EOF, auto-confirm, automation, and abort behavior remain unchanged.
- Tenant severity, text, ordering and suppression remain unchanged; auth and HTTP redaction still exclude credentials and payloads.

## 6. Required acceptance matrix

Use the full command execution path for the reported timeout and a successful confirmation/resumption companion. Exercise default, verbose, DEBUG, verbose+DEBUG, configured DEBUG, quiet, automation, JSON, keys-only, and supported combinations. Include quiet+JSON, quiet+keys, and machine output with DEBUG. Retain dry-run, no-wait, empty/sparse discovery, explicit keys, abort, and error regressions.

Run shared waiter and adapter tests for v87/v88/v89/v810, with absent, lookup-failure, deadline, max-attempt, and concurrent-root cases. Test redaction using seeded credentials, not production secrets. Real-terminal tests must run on a supported platform; skipped terminal tests are reported rather than counted as validation.
