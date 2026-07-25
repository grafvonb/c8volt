# Follow-Up: Volume, Paging, Filtering, And Progress Integration Coverage

## Purpose

The all-command integration suite now proves command availability, family-level behavior, output modes, version behavior, destructive previews, confirmed mutations, and help or docs example validity. A separate follow-up should add high-volume scenario coverage where correctness depends on many records, multiple result pages, long-running operations, and mixed-state data.

This follow-up is intentionally not part of the baseline family targets. Volume tests will be slower, more destructive, and more dependent on cluster capacity.

## Product North Star

The follow-up must test the product promise, not only flag plumbing.

c8volt's slogan is `done is done`: if an action needs retries, waiting, tree traversal, state checks, cleanup, or deterministic machine output before it is truly finished, the CLI should do that work for the operator. Volume tests should therefore prove that long-running and multi-step commands visibly progress, finish with an observable outcome, and report enough context for humans, scripts, and ops audits to trust the result.

Deep-scan findings from the current codebase:

- README positions c8volt as "Operator-grade Camunda 8 control for people and pipelines" and emphasizes preview, execute, wait, verify, and finish cleanly.
- Paged reads document `--batch-size` as backend page size only and `--limit` as the explicit cap across all pages.
- The command contract requires JSON, keys-only, quiet, and automation output to stay free of prompts and progress text.
- Root activity indicators are disabled for quiet, JSON log format, and automation, and are written away from stdout.
- Process-instance bulk operations emit durable progress such as completed roots, affected counts, and slow in-flight work.
- Ops workflows use shared report vocabulary: planned, skipped, not_applicable, submitted, confirmed, confirmation_failed, blocked, and failed.
- Ops compact output already has a recognizable grammar: selection filters, candidate counts, discovery status, preview or plan, submitted deletion or repair status, report location, outcome, and elapsed time.
- User-limited ops discovery must be visible in compact human output; complete discovery details are verbose-only.

## Goals

- Prove pagination behavior across commands that list or search resources.
- Prove totals, summaries, and count limits on commands that report aggregate results.
- Prove filtering correctness with positive and negative fixture records in the same dirty cluster.
- Prove spinner and progress output remains usable for long-running operations.
- Prove worker, batch, fail-fast, and partial-success behavior under meaningful load.
- Keep every target independently runnable and order-independent.
- Preserve default local configuration behavior: no generated config and no `--config`.
- Prove the `done is done` contract for long-running operations: visible progress, final outcome, elapsed time, and observable post-condition evidence.

## Proposed Make Targets

Keep these separate from the baseline family targets:

- `integration-cli-get-volume`
- `integration-cli-walk-volume`
- `integration-cli-update-volume`
- `integration-cli-cancel-volume`
- `integration-cli-delete-volume`
- `integration-cli-expect-resolve-volume`
- `integration-cli-deploy-embed-run-volume`
- `integration-cli-ops-analyse-volume`
- `integration-cli-ops-execute-volume`
- `integration-cli-ops-purge-volume`
- `integration-cli-ops-repair-volume`

If this proves too many targets, add only the families with real volume assertions first:

- `integration-cli-get-volume`
- `integration-cli-update-volume`
- `integration-cli-delete-volume`
- `integration-cli-ops-volume`

## Data Strategy

- Each target must seed the data it needs in its own test process.
- Seeded records must use the run marker so assertions can distinguish suite-owned data from unrelated dirty-cluster data.
- Tests must tolerate pre-existing data and leftovers from prior runs.
- Cleanup is best-effort only; evidence must classify data as seeded, pre-existing, mutated, retained, or cleanup-failed.
- Dataset sizes should be configurable, for example `C8VOLT_IT_VOLUME_SIZE`, with a conservative default.
- Volume tests should avoid exact global counts; assert bounded, marker-scoped, or contains-at-least behavior instead.

## Candidate Scenario Classes

- Paging: create more matching records than one request page and assert the command follows or honors pagination.
- Limit: create many records and assert `--limit` constrains the result.
- Filtering: create records with different BPMN IDs, variables, states, tenants, and timestamps, then assert filters include and exclude correctly.
- Totals: assert summary counts match the suite-owned dataset where the command reports totals.
- Progress: run commands with enough work to show spinner/progress behavior, then assert output remains stable and machine-readable modes stay clean.
- Workers: exercise `--workers`, `--no-worker-limit`, and `--fail-fast` with enough keys to prove batching behavior.
- Mixed outcomes: combine valid, already-mutated, missing, and unsupported keys to prove partial failure reporting.

## Critical Flag Semantics

The baseline suite proves critical flags are exposed, accepted, and exercised through representative command-family behavior. The volume follow-up must prove their semantics under enough data to catch wiring, batching, and reporting bugs.

- `--dry-run`: assert no suite-owned resource is mutated, deleted, cancelled, resolved, purged, or repaired after preview execution, including multi-resource selections.
- `--workers`: compare single-worker and multi-worker runs over enough keys to prove the flag is honored without requiring timing-sensitive assertions.
- `--no-worker-limit`: exercise a large selected set and verify the command accepts the unbounded mode while still reporting stable evidence.
- `--limit`: create more matching records than the limit and assert returned keys, rows, JSON items, and totals are capped according to the command contract.
- `--fail-fast`: combine valid and invalid keys or filters and assert execution stops early where the command promises fail-fast behavior.
- `--no-wait`: assert commands return before completion polling while still recording submitted mutation identifiers where available.
- `--force`: assert destructive commands reject risky selections without force and accept the same suite-owned selection with force.
- `--auto-confirm`: assert unattended destructive execution works only where confirmation would otherwise be required.
- `--json`: assert JSON remains parseable and structurally stable under volume, including empty, partial, and large result sets.
- `--keys-only`: assert output contains only keys, one per line, with no spinner, summary, warning, or progress text.
- `--report-json`: assert report files are written, parseable, scenario-specific, and reflect affected suite-owned resources.
- `--report-md`: assert Markdown reports are written, non-empty, scenario-specific, and include the expected high-level sections without relying on brittle prose.

## Pipeline And Stdin Semantics

Bulk operator workflows depend on clean `--keys-only` output and reliable stdin consumption. The volume follow-up must treat pipelines as first-class scenarios instead of only documenting them as blocked help examples.

- `get ... --keys-only | destructive ... --stdin --dry-run`: assert keys-only output can be piped directly into preview commands without parsing errors or accidental mutation.
- `get ... --keys-only | destructive ... --stdin --auto-confirm`: assert confirmed stdin mutations work against suite-owned resources and record affected keys.
- Empty stdin: assert each stdin-capable command returns a clear no-input outcome without hanging.
- Duplicate keys: assert repeated stdin keys are either deduplicated or reported consistently according to the command contract.
- Invalid keys: assert malformed and well-formed missing keys produce actionable failures.
- Mixed valid and invalid keys: assert partial-success reporting and `--fail-fast` behavior are correct.
- Whitespace handling: assert blank lines, trailing newlines, and surrounding spaces are handled consistently.
- Machine-readable pipelines: assert `--json` and `--keys-only` modes do not leak spinner, progress, warning, or prompt text into stdout.
- Destructive examples: assert any help or generated docs pipeline example that mutates state has source warning context.

## Progress And Completion Semantics

Long-running commands must provide enough visibility to keep operators confident without breaking automation.

- Human interactive mode: assert progress or activity is visible for long-running create, cancel, delete, repair, purge, retention, smoke-test, paging-total, and wait flows.
- Non-interactive modes: assert `--automation`, `--quiet`, `--json`, `--keys-only`, and JSON log formats suppress transient indicators and keep stdout parseable.
- Durable progress: assert verbose or log output includes stable progress facts, such as page number, counted or frozen candidates, completed roots, affected resources, and slow in-flight work where applicable.
- Finality: assert every long-running scenario ends with an explicit final line or report outcome, including elapsed time where ops workflows already expose it.
- Post-condition proof: assert confirmed mutations verify observable outcomes when the command contract promises verification, or explicitly report accepted/submitted/no-wait when verification is intentionally skipped.
- Stall visibility: for scenarios that can be made slow without flaking, assert the operator sees an unchanged or slow-work progress message instead of silence.
- Width safety: assert activity/progress messages do not corrupt subsequent normal output and remain bounded enough for narrow terminals.

## Consistent Operator Information

Volume tests should compare the information grammar across related commands so equivalent workflows do not drift in wording or evidence.

- Selection: human output and reports include filters or selectors used to choose the target set.
- Scope: output distinguishes candidate, frozen, affected, skipped, duplicate, missing, and retained resources where those categories exist.
- Limits: limited discovery is visible in compact output as user-limited with limit, pages, and batch size; complete discovery is available in verbose output or machine/report fields.
- Preview: dry-run output uses clear preview language and states that no changes were applied.
- Execution: confirmed mutation output states what was submitted, changed, skipped, failed, or retained.
- Hidden details: compact output gives `use --verbose` guidance when it suppresses key lists or retained resources.
- Timing: ops outcomes include elapsed time consistently.
- Errors: partial and failed workflows use actionable status, error, and suggestion fields where the shared result envelope supports them.

## Ops Reporting Semantics

Ops commands are the highest expression of `done is done`; volume coverage should treat audit reports as product output, not incidental files.

- Report existence: assert `--report-file` writes a scenario-specific file and respects preserve-versus-overwrite behavior.
- JSON reports: assert schema version, command name, workflow name, started/finished timestamps, duration, dry-run, c8volt version, Camunda version, profile identity, outcome, steps, notices, and errors where supported.
- Markdown reports: assert stable high-level sections are present, including selection, discovery, plan, execution, outcome, notices, errors, and hidden or retained resources where relevant.
- Discovery completeness: assert reports identify complete versus user-limited discovery and include limit, page count, batch size, candidates seen, and candidates frozen where available.
- Step vocabulary: assert statuses come from the shared ops vocabulary and that dry-run uses planned/skipped rather than submitted/confirmed mutation statuses.
- Mutation accounting: assert reports reconcile selected, affected, attempted, skipped, successful, failed, retained, and cleanup-failed resources.
- Machine parity: assert JSON stdout and JSON report describe the same frozen scope and final outcome.

## Command And Fixture Gaps To Track

The current baseline suite records blocked/actionable evidence for targets that are hard to create through c8volt alone. The volume follow-up should turn those into explicit product or fixture proposals when needed:

- Commands to create or reliably discover suite-owned job targets.
- Commands to create or reliably discover suite-owned incident targets.
- Commands to create or reliably discover suite-owned element-instance and user-task targets.
- Commands to create or reliably discover numeric resource IDs across supported versions.
- Commands or fixtures to create tenant-specific test data.
- Embedded BPMN models with listeners, incidents, timers, variable shapes, and long-running states.

## Acceptance Criteria

- Each volume target can run independently in any order against a clean or dirty disposable cluster.
- Each target writes dedicated evidence distinct from baseline `behavior-<family>.json` files.
- At least one scenario proves pagination with more records than a single page.
- At least one scenario proves filtering with both positive and negative suite-owned records.
- At least one destructive volume scenario proves dry-run and confirmed mutation over multiple resources.
- At least one ops volume scenario proves report generation over multiple resources.
- Critical flags have semantic assertions, not only help or command-acceptance evidence.
- Stdin and pipeline scenarios prove clean `--keys-only` producer output and reliable stdin consumer behavior.
- Long-running scenarios prove visible human progress and clean machine output.
- Ops scenarios prove consistent compact output and audit report contents across preview and confirmed execution.
- Machine-readable output modes remain valid JSON or keys-only under volume.
- Any direct Camunda API setup or missing embedded BPMN need is recorded as a proposal.

## Non-Goals

- Do not fold volume checks into the baseline family targets.
- Do not require an empty cluster.
- Do not require cleanup success.
- Do not use generated or private c8volt config files.
- Do not update generated `docs/cli/*` as part of the harness follow-up.
