# CLI Contract: Resume Process Instances

## Surface and validation

```text
c8volt resume process-instance --key <key>
c8volt resume pi --key <key-a>,<key-b> --key <key-c>
c8volt resume pi --auto-confirm -
c8volt resume pi --key <key> --no-wait --auto-confirm
c8volt resume pi --key <key> --tree --dry-run
c8volt resume pi --key <key> --tree --auto-confirm --json
```

These are user-facing usage forms, not instructions to execute against a live target during implementation.

| Control | Contract |
| --- | --- |
| `--key`, `-k`, stdin `-` | Existing repeated/comma-separated and newline-separated explicit-key conventions; stable deduplication |
| `--tree` | Selected instances plus reachable descendants; no ancestor/sibling escalation; only discovered SUSPENDED eligible |
| `--dry-run` | Exact-key local preview or read-only tree planning; no approval or mutation |
| `--no-wait` | Accepted submission only; no confirmation reads; tree discovery still occurs |
| `--workers`, `-w` | Explicit value must be positive; otherwise existing automatic policy |
| `--no-worker-limit` | Existing automatic worker-limit override when explicit worker count is unset |
| `--fail-fast` | Existing stop-scheduling semantics, retaining in-flight and unattempted evidence |
| `--auto-confirm`, `--automation` | Existing unattended approval and activity restrictions |
| `--json`, `--keys-only`, quiet, verbose | Existing supported combinations and precedence |
| Backoff / `--timeout` | Parent resume verb uses existing backoff flags; default wait bound two minutes; individual requests retain timeout control |

No state/search selector, native batch, `--force`, pagination, or no-state-check flag. Register state-changing mutation metadata, full machine contract and automation support, and ordinary/JSON/keys-only output capabilities. JSON wins over keys-only, requires dry-run or unattended approval, and conflicts with verbose as suspension does. Quiet preserves explicit machine output and required errors. Explicit-key tenant behavior follows existing admin options and backend authorization.

## Frozen execution and prompts

Exact mode previews locally validated unique keys. Tree mode resolves the whole service plan before confirmation; successful zero-eligible scope returns before approval/activity. For nonempty scope, clone target keys and ask once using the expanded count:

```text
You are about to resume N process instance(s). Do you want to proceed?
```

Use the existing confirmation helper's answer suffix, default, EOF behavior, and abort classification. Write plain text to `cmd.ErrOrStderr()`, including inherited writers and redirected stdout. No mutation after decline; no rediscovery after approval. Activity starts only after approval and stops before output on all paths, with established mode suppression.

## Output payloads and outcomes

Use the existing `ResultEnvelope` unchanged. Preserve command identification, optional tenant context, detail/class, and omission semantics. See [data-model.md](../data-model.md) for payload fields.

| Case | Envelope outcome / payload |
| --- | --- |
| Exact preview | succeeded; `{items: [preview entries]}` |
| Tree preview | succeeded; `{items: [eligible preview entries], plan: ...}` |
| Confirmed execution | succeeded; actual per-target evidence, plus plan for tree |
| No-wait with accepted work and no operation error | accepted; acceptance-only evidence |
| Complete tree with no eligible work, including no-wait | succeeded; `{items: [], plan: {requestedKeys: [...], instances: [...], targetKeys: []}}` |
| Partial execution failure | established failed/invalid classification; preserve all per-key evidence and tree plan |
| Input/capability/planning failure | existing single error envelope; do not fabricate complete plan or accepted work |

Error takes precedence over accepted no-wait. Decode one envelope followed by EOF; never invoke a second error renderer after final JSON emission. Preserve original operation error/exit if writing results also fails. JSON error evidence can contain existing backend details; do not alter the schema to fix human logging. Diagnostics/activity never contaminate stdout.

Keys-only preview lists exact planned keys or eligible tree keys. Execution lists only submission-accepted keys, including accepted-but-unconfirmed work; one key and newline per item, zero bytes for no qualifying keys. Excluded, rejected, unknown, and unattempted keys are not accepted-key results.

## Human information and diagnostics

All human mutation summaries, previews, plan summaries, and verbose facts are configured INFO records through `renderHumanLine`. They obey logger level, plain/structured format, inherited destination, and activity handling; human stdout is empty. Suggested stable wording, matching suspension rhythm:

```text
dry run: resume process-instance; selected: N; eligibility not checked; no changes applied
resumption: resumed C/N process instance(s)
resumption: submitted A/N process instance(s); confirmation not requested
selected process instances: S; process instances in scope: D; suspended targets: T; excluded: E
resumption: no suspended process instances to resume; no changes applied
```

Add accepted-but-unconfirmed, failed, and unattempted counts when meaningful. Completed-after-acceptance contributes to confirmed resumption and retains its actual state in verbose/machine evidence. CANCELED/TERMINATED contributes to failure, never confirmed resumption, and receives a distinct bounded `cancellation observed` reason.

Verbose output allowlists key, submission, confirmation, last state, and plan classification; never interpolate raw machine `error` fields. Group repeated failures into one configured ERROR record with counts and bounded reasons (for example rejected, acceptance unknown, cancellation observed, timed out, not attempted). Do not print raw joined errors or HTTP details under normal/verbose/quiet. Use existing redacted DEBUG paths independently of verbose; support both flag and configured debug level. JSON failures use their existing envelope/exit path without duplicate human failure emission.

## Documentation and permissions

Document Camunda 8.10; suspended-only submission; `SUSPEND_PROCESS_INSTANCE` on `PROCESS_DEFINITION`; additional reads for confirmation and read/search permission for tree discovery even during preview. Explain independent resumption, frozen/non-atomic scope, prior independently suspended descendants, 1,000 direct children per lookup, possible later descendants outside scope, no rollback, and accepted-versus-confirmed evidence. Resumption does not promise incident-free execution, completion, or immediate subscription readiness. README/help/source metadata and regenerated docs must agree.
