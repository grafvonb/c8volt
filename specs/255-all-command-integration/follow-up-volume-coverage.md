# Follow-Up: Volume, Paging, Filtering, And Progress Integration Coverage

## Purpose

The all-command integration suite now proves command availability, family-level behavior, output modes, version behavior, destructive previews, confirmed mutations, and help or docs example validity. A separate follow-up should add high-volume scenario coverage where correctness depends on many records, multiple result pages, long-running operations, and mixed-state data.

This follow-up is intentionally not part of the baseline family targets. Volume tests will be slower, more destructive, and more dependent on cluster capacity.

## Goals

- Prove pagination behavior across commands that list or search resources.
- Prove totals, summaries, and count limits on commands that report aggregate results.
- Prove filtering correctness with positive and negative fixture records in the same dirty cluster.
- Prove spinner and progress output remains usable for long-running operations.
- Prove worker, batch, fail-fast, and partial-success behavior under meaningful load.
- Keep every target independently runnable and order-independent.
- Preserve default local configuration behavior: no generated config and no `--config`.

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
- Machine-readable output modes remain valid JSON or keys-only under volume.
- Any direct Camunda API setup or missing embedded BPMN need is recorded as a proposal.

## Non-Goals

- Do not fold volume checks into the baseline family targets.
- Do not require an empty cluster.
- Do not require cleanup success.
- Do not use generated or private c8volt config files.
- Do not update generated `docs/cli/*` as part of the harness follow-up.
