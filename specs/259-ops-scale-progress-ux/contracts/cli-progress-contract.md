# CLI Progress Contract: Ops-Scale Preflight And Progress UX

## Scope

This contract defines the user-visible behavior for c8volt preflight and progress reporting. It applies to commands that can discover, enrich, plan, analyze, repair, cancel, delete, or otherwise process high-volume Camunda resources.

## Output Channel Rules

| Mode | stdout | stderr/activity | Prompting | Required behavior |
|------|--------|-----------------|-----------|-------------------|
| Default human | Final human result only | Transient activity allowed; compact durable scope/progress allowed when useful | Interactive prompts allowed | Show preflight for broad high-volume work and exact progress after frozen scope |
| Verbose human | Final human result only | Durable progress lines and activity allowed | Interactive prompts allowed | Include page progress, certainty wording, and next-step detail |
| Debug | Final selected output plus debug logging according to existing rules | Debug logs may include endpoint/request detail | Existing prompting rules | Keep low-level HTTP traces in debug, not default human progress |
| JSON | One valid JSON document | Activity/progress may occur only outside stdout and only when allowed by existing terminal rules | No extra human prompts unless command contract already allows confirmation | Never write progress to stdout |
| Keys-only | One key per line and nothing else | Activity/progress may occur only outside stdout and only when allowed by existing terminal rules | No extra human prompts that corrupt key streams | Never write progress to stdout |
| Quiet | Required result or errors only according to existing command rules | Suppress progress chatter | Existing required safety prompts only | No non-error progress lines |
| Automation | Deterministic machine output/report only | No interactive progress requirements | Non-interactive | Scope must be auditable through existing structured fields or reports |

## Preflight Contract

Preflight appears before expensive batch processing begins for broad high-volume selectors.

Required human content:

- Core resource type.
- Best available count and certainty label: exact, lower bound, estimated, or unknown.
- Page-size and page-count context when paging applies and page count is known or safely estimable.
- Consequence summary for the next expensive or destructive work.
- Confirmation prompt when interactive human mode requires it.

Example shapes:

```text
preflight: MainOrderProcess matches 10,000+ process instance(s); page size 1000; discovery will require at least 10 page(s)
preflight: slow analysis will discover all matches and load runtime element timelines for each selected process instance. Continue?
```

```text
preflight: exact count unavailable because orphan detection requires parent checks; first page has 1000 child candidate(s)
```

Rules:

- A reused first discovery page must not be fetched again only for preflight.
- Lower-bound totals must use `+` or explicit lower-bound wording.
- Unknown totals must explain why better scope is unavailable or expensive.
- Explicit small key sets should skip broad preflight unless the command can expand into a large affected set.

## Page Progress Contract

Page progress appears during discovery when paging is involved.

Required human content when available:

- Phase name.
- Current page and known or estimated total pages.
- Seen resource count.
- Selected or frozen count if it differs from seen count.
- More-matches or limit status when relevant.

Example shapes:

```text
discovering process instances: page 4/10, 3,812/10,000+ seen
discovering incidents: page 2, 1,400 seen, exact total unavailable
```

Rules:

- Page counts derived from lower-bound totals must be labeled at least/estimated.
- Progress must distinguish candidate discovery from frozen scope when local filters, dependency checks, or eligibility checks apply.
- Limit stops must be labeled user-limited.

## Frozen-Scope Progress Contract

Once the final set of core resources is frozen, progress uses exact counters.

Example shapes:

```text
analysis: loading runtime elements 48/800 process instance(s)
delete: roots 12/250 done, affected process instances 1,480
repair: incidents 73/600 checked
```

Rules:

- `done/total` totals must be exact.
- Phase names must be operator-facing, not endpoint-facing.
- Final command output must not contradict frozen progress counts.

## ETA Contract

ETA is optional and approximate.

Example shape:

```text
analysis: loading runtime elements 48/800 process instance(s), 6.0%, 2m10s elapsed, ~34m remaining
```

Rules:

- ETA must be omitted before enough samples exist.
- ETA must be omitted when total is unknown.
- ETA must use approximate wording such as `~` or `approx`.
- Elapsed time may appear before ETA if the phase is long enough.

## First Proof Workflow Contract

For `ops analyse slow-process-instances` process-definition search mode:

Required phases:

- `preflight`
- `discovering process instances`
- `freezing candidate scope`
- `loading runtime elements`
- `loading listener jobs` when listeners are requested
- `analysis complete`

Required behavior:

- Broad `--bpmn-process-id` or `--pd-key` search shows preflight before full discovery/enrichment.
- Discovery progress includes page progress and count certainty.
- Element enrichment progress uses exact frozen process-instance counters.
- JSON and keys-only stdout stay clean.
- Explicit-key mode keeps concise behavior and does not force broad search preflight.

## Documentation Contract

Help and generated docs for covered commands must explain:

- Whether the command can process many resources.
- What preflight means.
- Count certainty labels.
- How `--batch-size` differs from `--limit`.
- Which modes suppress or redirect progress.
- What confirmation means for frozen scope and destructive work.
