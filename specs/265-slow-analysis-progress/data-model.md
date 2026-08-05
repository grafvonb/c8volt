# Data Model: Slow Analysis Progress After Confirmation

## Durable Milestone

Represents one compact human progress line written to the allowed durable progress channel.

**Fields**:

- `text`: formatted operator-facing progress line, such as discovery page progress or frozen-scope runtime element progress.
- `phase`: workflow phase represented by the line, such as `discovering process instances`, `loading runtime elements`, or `loading listener jobs`.
- `emittedAt`: time the milestone was written.
- `sourceKind`: progress event kind used to create the line: page, frozen scope, or ETA when allowed.

**Rules**:

- Must not be written to stdout.
- Must not be emitted for JSON, keys-only, quiet, or automation modes.
- Must be compact and free of endpoint names, cursors, and per-resource debug detail.

## Milestone Pacing State

Command-owned state that decides whether a formatted progress event should become a default human durable milestone.

**Fields**:

- `lastMilestoneAt`: time of the last durable default-human milestone for this workflow.
- `lastProgressSignature`: normalized progress identity used to detect forward movement.
- `minimumElapsed`: named threshold that must pass before another durable default-human milestone may be written.
- `clock`: time source for deterministic tests.

**Rules**:

- A default human milestone is allowed only when enough time has elapsed and the new progress signature proves forward movement.
- The first post-confirmation milestone may be emitted only after the elapsed threshold is met and work has advanced.
- The policy must be reusable from shared command progress code.
- Thresholds must use named constants rather than inline numeric literals.

## Progress Signature

Comparable summary of the progress counters that prove forward movement.

**Fields**:

- `kind`: progress event kind.
- `phase`: current operator-facing phase.
- `currentPage`: page number for discovery progress.
- `seen`: total seen resources for discovery progress.
- `selected`: total selected resources for discovery progress.
- `done`: completed frozen-scope items.
- `total`: total frozen-scope items.
- `completedSamples`: ETA sample count when ETA events are considered.

**Rules**:

- Page progress advances when page, seen, or selected counters increase.
- Frozen-scope progress advances when `done` increases for the same phase and total.
- ETA-only events should not create duplicate milestones unless they also represent forward progress not already covered by frozen-scope counters.

## Progress Channel

Existing command-mode contract that records where progress may be emitted.

**Relevant modes**:

- `human`: transient activity allowed; sparse durable milestones allowed on stderr when pacing permits.
- `verbose`: transient activity and durable detailed progress allowed on stderr.
- `debug`: transient activity and durable detailed progress allowed on stderr, alongside diagnostic behavior.
- `json`: progress suppressed from stdout and durable human output.
- `keys-only`: progress suppressed from stdout and durable human output.
- `quiet`: progress suppressed except existing required prompts or errors.
- `automation`: deterministic behavior; human progress suppressed from stdout/stderr.

## Slow Analysis Workflow

The operator-level workflow for `ops analyse slow-process-instances` in broad process-definition search mode.

**Phases**:

- `preflight`: report scope and ask for confirmation when required.
- `discovering process instances`: continue page discovery after confirmation.
- `loading runtime elements`: load timeline data for each frozen process instance.
- `loading listener jobs`: load listener job details when requested.

**Rules**:

- Explicit key mode bypasses broad discovery preflight and does not need default human milestones for this feature.
- Broad search mode must keep workflow activity visible after confirmation.
- Services continue to emit structured events and do not decide human milestone pacing.
