# CLI Contract: Late Business-ID Assignment

## Process assignment

```text
c8volt update pi --key <key> [--key <key> ...] --set-bid <value> [existing mutation controls]
c8volt update pi - --set-bid <value> [existing mutation controls]
```

`--set-bid` selects assignment mode before variable payload parsing. One opaque exact value applies to every distinct explicit process key. Read all stdin and validate all keys before mutation. Do not use BID discovery, expand a family or redirect a child. Existing explicit-key authorization applies.

Reject `--bid`, `--vars`, or `--vars-file` with assignment. Existing key/search exclusivity and search-control restrictions remain in force; assignment does not gain selectors from `--limit` or `--batch-size`. Preserve worker/fail-fast options, confirmation, automation, dry-run and no-wait.

Backend acceptance requires an active root instance without BID and uniqueness disabled. Repeating identical BID is a conflict. Preview explains requested value and scope but cannot promise acceptance. Normal execution confirms accepted assignment by exact-instance observation; no-wait labels observation skipped. No automatic replay after an ambiguous response.

## Atomic completion

```text
c8volt update job --key <key> --complete --set-bid <value> [--vars <object>]
c8volt update job --key <keys> [--key <keys> ...] --complete --set-bid <value>
c8volt update job - --complete --set-bid <value>
c8volt update job <existing AND selectors> --complete --set-bid <value>
```

Selectors choose jobs; assignment is payload, never a selector by itself. Keyless `--complete --retries 0 --set-bid <value>` keeps zero as an exact predicate. Keyed completion still rejects retries. Reject assignment outside completion, including retry-update, timeout-update, technical failure and BPMN error.

Selection is tenant-scoped for search, uses current batch/limit rules, completes before approval, and deduplicates by job key. Explicit keys retain existing authorization. Every selected distinct job carries BID and variables in one request. Two jobs in one process remain two targets; a later conflict remains a per-job failure. No fallback without BID or separate assignment mutation is permitted.

Scalar input retains scalar results. Repeated/comma/stdin/selector collections retain bulk results even if one job remains. Existing listener-variable restrictions stay in effect. Bulk job completion continues rejecting keys-only; this feature does not add that mode.

## Common value/version rules

Camunda 8.10 and later supported adapters accept the feature; 8.8/8.9 reject assignment before mutation. Explicit empty/blank, invalid UTF-8, and values exceeding 256 UTF-16 units fail validation. Accepted strings retain spaces, commas, case and Unicode exactly. Omission retains established behavior, including absent completion businessId payload.

## Results, prompts, and errors

- Optional requestedBusinessId is intent; existing businessId fields remain observed facts. Do not backfill snapshots.
- Report accepted, confirmed, rejected, not-submitted and unknown evidence accurately. Job COMPLETED alone cannot prove assignment from this request.
- Dry-run submits nothing. Completed empty stdin/discovery returns successful mode-appropriate empty output without confirmation, submission or pending claims, including no-wait. Failed/incomplete discovery submits nothing.
- Human summaries, previews and verbose outcomes use configured INFO logging, suppressed at WARN and by quiet. Errors retain one compact configured ERROR under quiet. Raw HTTP details remain redacted DEBUG, independently enabled by debug flag or config.
- Machine results use stdout; prompts remain plain configured stderr with existing eligibility, wording/input behavior, EOF and abort rules. Quiet preserves explicitly requested machine results.
- Shared JSON emits one envelope followed by EOF, preserving partial results, original error classification and established null/empty conventions. No second error rendering follows.
- Preserve each supported keys-only mode's current key meaning and zero bytes when empty. Preserve rejection where unsupported; do not invent new key output.

## Documentation examples (descriptive only)

```bash
c8volt update pi --key 123 --set-bid 'ORDER-123' --dry-run
c8volt update pi --key 123 --set-bid 'ORDER-123'
c8volt get pi --bid 'ORDER-123'
c8volt update job --key 456 --complete --set-bid 'ORDER-123' --vars '{"approved":true}'
```

Direct assignment and combined completion are alternative first assignments. Executing the former successfully makes the latter conflict, even with identical BID. These are usage examples, not implementation validation commands; use isolated fixtures in [quickstart](../quickstart.md).
