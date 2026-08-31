# Contract: Semantic Progress and Output

## Service callback contract

1. Services emit wording-free structured facts through the existing optional progress callback.
2. A completion fact is emitted immediately after each executed item reaches its configured boundary.
3. The fact contains a stable identity where available, a disposition of `submitted`, `confirmed`, or `failed`, optional failure detail, and an optional trustworthy affected delta.
4. No service event contains a rendered command sentence or decides whether the user sees output.
5. Concurrent callbacks are allowed. Services do not change worker count, scheduling, fail-fast, retry, wait, result order, or backend request behavior to produce progress.
6. Fail-fast items that were not scheduled produce no completion fact.

## Command reporter contract

1. The command starts one workflow-priority activity for real work. Destructive discovery/planning activity stops before the prompt, and a fresh scope starts after confirmation.
2. Every completion updates exact transient aggregate state.
3. Default human information is completion-driven: the first completion at least 10 seconds after scope start or the previous informational line emits one compact aggregate line.
4. Idle time alone never emits a line.
5. A failure emits immediately and activates durable progress.
6. `Finish` is idempotent and emits one aggregate flush only when durable progress is active and the aggregate changed since the last durable line.
7. A clean run finishing before 10 seconds emits no durable progress line.
8. Verbose/debug emits one identity-and-outcome line per completion and does not emit paced aggregate information.
9. Durable output uses the activity-aware diagnostic writer so an active spinner is cleared and restored without interleaving.

## Lifecycle wording contract

| Fact disposition | Command meaning |
| --- | --- |
| `submitted` | Request was accepted and the command did not wait for operational confirmation. |
| `confirmed` | The command's established wait/proof contract completed; command-family wording may say canceled, deleted, repaired, deployed, started, or satisfied. |
| `failed` | The configured completion boundary was not reached; warning identifies the item and preserves the cumulative failed count. |

The command family supplies nouns and verbs. Services never embed operator prose in the disposition.

## Affected-count contract

- Affected counts are non-negative deltas, not estimates.
- Pointer-to-zero is a trustworthy zero; nil is unavailable.
- The scope must declare that every contributor can provide a trustworthy delta before affected output is eligible.
- If any contributor is unavailable, the reporter omits the affected aggregate for the entire scope.
- Overlapping cleanup plans must be deduplicated before contributing deltas; otherwise affected output is omitted.

## Output-mode contract

| Mode | Activity | Paced info | Per-item info | Failure warning | Stream |
| --- | --- | --- | --- | --- | --- |
| Default human | yes | yes | no | yes | diagnostic stderr |
| Verbose/debug | yes | no | yes | yes | diagnostic stderr |
| Quiet | no | no | no | yes | direct activity-aware stderr path |
| Automation | no | no | no | no | none; structured result/report remains authoritative |
| JSON | no | no | no | no | none; stdout JSON is unchanged |
| Keys-only | no | no | no | no | none; stdout remains one key per line |

No progress text may be written to stdout.

## Compatibility contract

- No new CLI flag or configuration key.
- No change to final human rows, JSON envelopes, keys-only output, audit report schema, result ordering, exit codes, confirmation, dry-run, force, fail-fast, no-wait, worker, retry, or wait semantics.
- Existing timer-driven or stage logs remain only for callers without structured progress; a command must not emit duplicate legacy and semantic streams.
- Discovery page progress stays labeled as discovery and is never counted as completed mutation work.

## Required verification examples

- Fake clock: completions at 9.9s and 10s produce zero then one informational line.
- Rapid completions after a milestone do not emit another informational line before 10 seconds.
- Failure at any time emits immediately; subsequent unreported completion is included in one final flush.
- Concurrent out-of-order completion preserves exact monotonic counters under `-race`.
- No-wait accepted work renders submitted wording; waited work uses its confirmed command verb.
- Direct, stdin, and search-selected equivalent scopes produce the same aggregate semantics.
- JSON and keys-only stdout fixtures remain parseable and contain no progress.
- Quiet shows only failures; automation shows no human progress, including failures.
