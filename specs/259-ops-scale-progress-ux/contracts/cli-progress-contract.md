# CLI Progress Contract: Ops-Scale Preflight And Progress UX

## Scope

This contract defines the user-visible behavior for c8volt preflight and progress reporting. It applies to commands that can discover, enrich, plan, analyze, repair, cancel, delete, or otherwise process high-volume Camunda resources.

## Output Channel Rules

| Mode | stdout | stderr/activity | Prompting | Required behavior |
|------|--------|-----------------|-----------|-------------------|
| Default human | Command result data only | Transient activity allowed; compact durable scope/progress allowed when useful | Interactive prompts allowed | Show command-specific scope checks for broad high-volume work and exact progress after frozen scope |
| Verbose human | Command result data only | Durable progress lines and activity allowed | Interactive prompts allowed | Include page progress, certainty wording, and next-step detail |
| Debug | Final selected output plus debug logging according to existing rules | Debug logs may include endpoint/request detail | Existing prompting rules | Keep low-level HTTP traces in debug, not default human progress |
| JSON | One valid JSON document | Activity/progress may occur only outside stdout and only when allowed by existing terminal rules | No extra human prompts unless command contract already allows confirmation | Never write progress to stdout |
| Keys-only | One key per line and nothing else | Activity/progress may occur only outside stdout and only when allowed by existing terminal rules | No extra human prompts that corrupt key streams | Never write progress to stdout |
| Quiet | Required result or errors only according to existing command rules | Suppress progress chatter | Existing required safety prompts only | No non-error progress lines |
| Automation | Deterministic machine output/report only | No interactive progress requirements | Non-interactive | Scope must be auditable through existing structured fields or reports |

## Preflight Contract

Preflight appears before expensive batch processing begins for broad high-volume selectors. Human wording must be command-specific; do not print the generic prefix `preflight:`.

Required human content:

- Core resource type.
- Best available count and certainty label: exact, lower bound, estimated, or unknown.
- Page-size and page-count context when paging applies and page count is known or safely estimable.
- Consequence summary for the next expensive or destructive work.
- Confirmation prompt when interactive human mode requires it.

Example shapes:

```text
slow analysis scope: MainOrderProcess matched at least 10000 process instances; page size: 1000; discovery pages: at least 10
slow analysis is expensive: discover all matches and load runtime element timelines
Continue slow analysis for at least 10000 process instances?
```

```text
orphan purge scope: an unknown number of process instances matched; page size: 1000
orphan purge dry run: will check parent existence and validate delete impact only; no changes will be applied
```

Rules:

- A reused first discovery page must not be fetched again only for preflight.
- Lower-bound totals must use explicit lower-bound wording such as `at least 10000 process instances`.
- Unknown totals must explain why better scope is unavailable or expensive.
- Explicit small key sets should skip broad preflight unless the command can expand into a large affected set.
- A zero exact scope must say `matched no <resources>` and must not invent one discovery page.
- Dry-run preflight must not say `will delete`, `will repair`, or `destructive`; it must say what will be planned or validated and that no changes will be applied.
- Actual destructive or state-changing paths may use warning-level wording before confirmation, but only when mutation can really happen in that run.

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

## Human Report Contract

Ops workflow human reports are operator metadata, not data streams. They must use the same activity-aware human renderer as scope checks and warnings, matching commands such as `delete process-definition`.

Examples:

```text
incident purge dry run: planning only; no changes will be applied
selection filters: {state=active}
candidate incidents: 4
candidate process instances: 4
delete preview: 4 candidate incidents, 4 candidate process instances, 6 affected process instances across 4 roots would be deleted
dependency expansion: 2 additional process instances due to dependencies
non-final affected process instances: 6; use --force to allow deletion
outcome: planned; no changes applied; elapsed: <1s
```

Rules:

- Do not mix logger-prefixed operational lines with plain operational lines in one ops workflow.
- `WARN` is reserved for destructive mutation, irreversible deletion, state-changing repair, expensive read-only analysis requiring confirmation, safety blockers, and unexpected partial outcomes.
- Dry-run reports normally use `INFO`; only real safety blockers or surprising partial results should be warnings.
- Query/list command rows remain stdout result data; ops workflow reports use the human renderer.

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
