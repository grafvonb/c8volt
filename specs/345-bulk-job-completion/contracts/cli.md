# CLI Contract: Bulk Job Completion

This extends `c8volt update job --complete`. All examples describe the planned implementation, not currently available functionality on v4.3.4.

## Selection grammar

```text
c8volt update job --key KEY --complete [existing options]
c8volt update job --key KEY[,KEY...] [--key KEY...] --complete [execution options]
c8volt update job - [--key KEY[,KEY...] ...] --complete [execution options]
c8volt update job <job predicates> --complete [search controls] [execution options]
```

The only permitted positional argument is one `-`, meaning read newline-separated keys from stdin. Read all input first; ignore blank lines, trim whitespace, reject malformed/nonpositive/out-of-range decimal keys, and deduplicate valid keys. Preserve string representation through public APIs and JSON; do not pass 64-bit keys through floating point. Normalize equivalent numeric spellings consistently with existing key validation.

One scalar key with no stdin/selectors uses the legacy single-key contract. A repeated or comma collection form remains bulk after deduplication. Explicit flags and stdin may combine; flags precede stdin in frozen input order. Explicit input and search predicates cannot combine. `--key ''` and empty comma entries are invalid; deliberate empty stdin is a successful empty batch.

| Predicate | Meaning |
|---|---|
| `--state` | Existing normalized job state |
| `--type` | Job type, e.g. `removeTaskData` |
| `--pi-key` | Owning process-instance key |
| `--element-instance-key` | Owning element-instance key |
| `--element-id` | BPMN element ID |
| `--worker` | Searchable worker attribute; does not prove current ownership |
| `--retries` | Exact retries predicate in selector mode, including explicit zero |
| `--kind` | Existing normalized job kind, including `TASK_LISTENER` |
| `--listener-event-type` | Existing normalized listener event, including `CANCELING` |

Predicates combine with AND and retain `get job` validation. `--retries 0 --complete` with no keys is a valid selector. Keyed `--complete --retries` remains invalid. Reject no-selector completion, explicit/filter mixing, unsupported values and negative limits before reads/mutations as appropriate. `--total` is not supported here.

## Controls and compatibility

- Search mode uses `--batch-size` for page size and `--limit` for unique matching targets; omission/zero limit means no user cap. Neither flag selects jobs. Reject search-only controls for explicit completion to avoid silently truncating supplied keys.
- Root tenant/configuration resolution remains authoritative. Named discovery tenant is enforced server-side and rechecked locally. Empty/all-tenants uses accessible scope. Explicit keys retain backend authorization and report actual tenant context.
- Use existing workers, fail-fast, no-worker-limit, no-wait, auto-confirm, automation, quiet, verbose, JSON and backoff controls where applicable. No-worker-limit removes the ordinary helper cap but still uses a finite effective worker count.
- New collections/predicates are accepted only with completion. Preserve ordinary retries, timeout, failure and BPMN-error parsing, including single-key limitations.
- Existing variables parsing applies. Nonempty variables on a batch containing task listeners fail preparation; variables are never silently dropped.
- `--dry-run` performs discovery/validation only. It must never submit completion, update retries, resolve incidents, cancel processes or delete history.
- Existing mutation eligibility restrictions for JSON/noninteractive operation remain. Automation/auto-confirm are prompt-free. Eligible interactive execution asks once after complete preparation, through configured stderr. Paging does not ask additional questions.
- Capability metadata must stop marking `--key` unconditionally required, document alternative selectors, and advertise repeated key support. Keep unrelated command metadata stable.

## Human and structured output

Default preview shows workflow, normalized filters or explicit count, tenant scope, selected count, limit when reached and the listener consequence. Verbose preview lists candidate context using existing job row conventions. JSON preview contains all candidates. A real run is aggregate-first; verbose output may list final per-key outcomes. Low-level HTTP details remain DEBUG-only.

Human previews, summaries and verbose rows use `renderHumanLine` and the command-context INFO logger, honoring its configured format, destination and level. They must not use `renderOutputLine` to bypass logging. Plain interactive prompts remain on configured stderr, and JSON results remain exclusively on stdout.

Human failures log the result summary first, then emit exactly one ERROR through the configured command logger. The error uses bounded outcome reasons and HTTP status labels, groups repeated causes, and shows at most three distinct reasons plus an aggregate remainder. Preparation failures use the normalized error class; aborts and interruptions remain explicit. Raw joined errors, URLs and response bodies must not appear in normal or verbose diagnostic output. Quiet retains the compact error. Existing redacted HTTP diagnostics remain available at DEBUG through either flag or configuration. JSON keeps its original error evidence and partial payload in one envelope, without a duplicate human ERROR.

Required listener explanation: completing a listener job acknowledges its work without executing its business logic. Do not describe this as guaranteed process cancellation.

Bulk JSON uses exactly one existing `ResultEnvelope` with `command: "update job"`, existing outcome/class/detail/tenant fields and a typed bulk payload. Use existing naming and omission conventions; do not introduce a second envelope schema. The payload follows [data-model.md](../data-model.md).

| Result | Envelope/exit behavior |
|---|---|
| Dry-run | Successful preview; candidate context, zero submissions |
| Complete empty selection | Succeeded, zero counts and empty collection payload; no accepted claim, even with no-wait |
| All accepted or observed completed | Succeeded normally; accepted envelope with no-wait only if mutations were actually accepted; observation remains explicit |
| Only already-completed no-ops | Succeeded; no mutation accepted/pending claim |
| Mixed/unsatisfied outcomes | One failed envelope with full partial payload and normalized error; nonzero exit |
| Caller interruption | Full available report, unstarted rows, nonzero exit |
| Declined confirmation | Aborted result; zero submissions, not successful completion |
| Invalid input/preparation failure | Existing invalid/failed classification; no writes; incomplete observations never masquerade as executable scope |

Retain existing explicit exit-code suppression behavior. Render partial results once, then exit without invoking a second JSON error renderer. Quiet suppresses informational human output, not requested JSON. No keys-only mode is added. The bulk collection encoding is fixed in the data model: preview `candidates: []`, execution `outcomes: []` when empty, with the inapplicable collection omitted and all count fields present. Do not change shared envelope or legacy payload semantics.

## Examples

```sh
c8volt update job --kind TASK_LISTENER --listener-event-type canceling \
  --state created --type removeTaskData --complete --dry-run --verbose

c8volt update job --kind TASK_LISTENER --listener-event-type CANCELING \
  --state CREATED --type removeTaskData --limit 1000 --batch-size 100 \
  --complete --auto-confirm --json

c8volt update job --key 2251799870796844,2251799870796848 \
  --key 2251799870796849 --complete --dry-run

c8volt update job - --complete --auto-confirm --json < reviewed-job-keys.txt
```

A dry-run invocation and a later selector execution rediscover independently; they are not a persisted frozen plan. For an exact reviewed set, save and execute explicit keys. Within each execution invocation, preparation/confirmation/mutation use one frozen set. New sequential listeners require a separate invocation.
