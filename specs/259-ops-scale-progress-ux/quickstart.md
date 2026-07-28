# Quickstart: Ops-Scale Preflight And Progress UX

## Prerequisites

- Work on branch `259-ops-scale-progress-ux`.
- Read `specs/259-ops-scale-progress-ux/spec.md`, `plan.md`, `data-model.md`, and `contracts/cli-progress-contract.md`.
- For Ralph-driven implementation, include `--implementation-context specs/ralph-implementation-rules.md`.

## Targeted Validation Order

1. Run focused tests for shared activity behavior:

   ```bash
   GOCACHE=/tmp/c8volt-gocache go test ./toolx/logging ./testx/activitysink -count=1
   ```

2. Run focused process-instance and ops service tests while building first proof behavior:

   ```bash
   GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processinstance ./internal/services/ops -run 'Progress|Preflight|SlowProcess|SearchProcessInstances' -count=1
   ```

3. Run focused facade tests for affected public packages:

   ```bash
   GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process ./c8volt/ops -run 'Progress|Preflight|SlowProcess|SearchProcessInstances' -count=1
   ```

4. Run focused command tests for output-mode safety and the proof command:

   ```bash
   GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'SlowProcess|Progress|Activity|OutputMode|KeysOnly|JSON|Automation' -count=1
   ```

5. Run the full repository validation before merge:

   ```bash
   make test
   ```

## Proof Scenario: Slow Process Analysis

Use fake multi-page process-instance search fixtures and runtime element fixtures.

Expected outcomes:

- `ops analyse slow-process-instances -b MainOrderProcess` emits preflight before loading all timelines.
- Discovery progress shows page progress and seen count.
- Runtime element loading progress shows exact `done/total` after process-instance scope is frozen.
- The first discovery page used for preflight is reused during discovery.
- The command does not require `--debug` for visible progress in default human interactive mode.
- Explicit `--key` analysis remains concise and skips broad discovery preflight.

## Output Safety Scenarios

Run equivalent large fake-volume command tests in all supported modes.

Expected outcomes:

- JSON stdout parses as one valid JSON document.
- Keys-only stdout contains only keys, one per line.
- Quiet mode suppresses progress chatter.
- Automation mode remains non-interactive and deterministic.
- Progress and preflight text appear only on allowed human stderr/activity paths.

## Preflight Certainty Scenarios

Use fixtures with exact totals, lower-bound totals, and missing totals.

Expected outcomes:

- Exact totals render without `+`.
- Lower-bound totals render with `+` or explicit lower-bound wording.
- Unknown totals explain why exact scope is unavailable or expensive.
- Page count appears only when safely known or estimable.
- Broad destructive workflows include consequence text before confirmation.

## ETA Scenarios

Use controlled timing fixtures or an injectable clock during tests.

Expected outcomes:

- No ETA appears before the minimum sample threshold.
- ETA appears only after enough completed work and only with a known or frozen total.
- ETA wording is approximate.
- Elapsed time and done/total counters remain correct when ETA is absent.

## Documentation Validation

After command behavior changes:

```bash
make docs-content
```

Expected outcomes:

- Generated CLI docs mention preflight/progress behavior for covered commands.
- Help text explains `--batch-size` versus `--limit`.
- README examples or operational notes match shipped behavior.
