# Contract: Slow Analysis Progress After Confirmation

## Scope

This contract defines observable CLI behavior for broad `c8volt ops analyse slow-process-instances` runs after the operator confirms the preflight prompt.

## Default Human Mode

### Preflight

- The command prints the existing slow-analysis scope and consequence lines before confirmation.
- The confirmation prompt remains the decision point before expensive discovery and timeline loading.

### Post-Confirmation Progress

- After confirmation, the command keeps high-level workflow activity visible for discovery and timeline loading.
- The command may write sparse durable progress milestones to the human progress channel when both conditions are true:
  - enough time has passed since the last durable milestone for the workflow;
  - discovery or timeline progress counters have advanced.
- Durable milestones are compact lines derived from existing progress formatters.
- Durable milestones are written away from result stdout.
- Timer-only "still working" lines are not allowed.

### Example Shape

```text
slow analysis scope: MainOrderProcess matched 6761 process instances; page size: 1000; discovery pages: 7
slow analysis is expensive: discover all matches and load runtime element timelines
Continue slow analysis for 6761 process instances? [y/N]: y
discovering process instances, page 2/7, 2000 seen, 2000 selected
loading runtime elements, 48/6761 process instance(s), 2m0s elapsed
```

Exact wording may vary with total certainty, page count certainty, elapsed timing, and phase, but it must stay compact and operator-facing.

## Verbose And Debug Modes

- Existing durable detailed progress remains available on stderr.
- New sparse default-human milestone policy must not reduce verbose or debug progress detail.
- Endpoint names, cursors, and low-level request detail remain diagnostic, not default human milestone text.

## Machine-Oriented Modes

- JSON output remains one valid result document on stdout.
- Keys-only output remains one process-instance key per line on stdout.
- Quiet mode suppresses human progress chatter according to existing quiet-mode behavior.
- Automation mode remains deterministic and does not receive human progress text on stdout.

## Service Boundary

- Internal services emit structured progress facts only.
- Command progress code owns formatting, milestone pacing, durable stderr routing, transient activity updates, and output-mode gating.

## Validation Expectations

- Shared progress tests prove elapsed-plus-progress pacing, boundary behavior, and machine-mode suppression.
- Slow-analysis command tests prove post-confirmation default human progress can produce a durable milestone without stdout leakage.
- Existing JSON, keys-only, quiet, automation, verbose, and debug progress tests continue to pass.
