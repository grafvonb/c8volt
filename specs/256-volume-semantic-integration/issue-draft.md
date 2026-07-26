# Add volume and semantic CLI integration coverage

## Summary

Add a second integration coverage layer for c8volt that proves the `done is done` product promise under larger datasets and long-running workflows. The baseline all-command suite proves command coverage and representative behavior; this follow-up should prove deeper paging, filtering, critical flag, stdin pipeline, visible progress, and ops audit-report semantics.

## Why

c8volt is positioned as operator-grade Camunda 8 control for people and pipelines. For long-running and destructive workflows, correctness is not only "the command accepted the flag." Operators need visible progress, final outcomes, clean machine output, and audit reports they can trust.

## Scope

- Volume and paging scenarios for read/search and ops discovery commands.
- Critical flag semantics for dry-run, workers, no-worker-limit, limit, fail-fast, no-wait, force, auto-confirm, JSON, keys-only, and report output.
- Stdin and pipeline scenarios using clean keys-only output.
- Visible progress and finality checks for long-running commands.
- Consistent operator information across related commands.
- Ops report content checks for preview and confirmed execution.
- Proposal evidence for missing c8volt setup commands or embedded BPMN fixtures.

## Acceptance

- At least one scenario proves paging or limits with more records than a single backend page.
- At least one scenario proves filtering with positive and negative suite-owned records.
- At least one destructive scenario proves dry-run safety and confirmed mutation over multiple resources.
- At least one stdin pipeline scenario proves keys-only producer output can feed a stdin consumer safely.
- At least one long-running human-mode scenario captures visible progress and final outcome.
- Machine-readable output remains parseable and uncontaminated by progress or prompt text.
- Ops preview and confirmed scenarios write valid audit reports with discovery, plan, execution, outcome, notices, errors, and accounting.
- Every target can run independently in any order against clean or dirty disposable clusters.

## Spec

Local Spec Kit feature: `specs/256-volume-semantic-integration/spec.md`
