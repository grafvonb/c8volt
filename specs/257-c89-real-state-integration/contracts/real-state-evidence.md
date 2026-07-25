# Contract: Real-State Evidence

## Evidence Files

Real-state targets must write evidence under the integration workdir.

Required file classes:

- real-state family report, such as `real-state-jobs.json`
- real-state data report, such as `real-state-data-jobs.json`
- real-state progress or output report where human visibility matters
- real-state ops report evidence when an ops command is involved
- `proposals-command.json`
- `proposals-embedded-bpmn.json`

Evidence files must stay outside `docs/`.

## Scenario Evidence

Each scenario record must identify:

- scenario name
- command path
- selected profile
- observed Camunda version
- covered flags
- output mode
- resource keys used for assertions
- before-state and after-state evidence when a mutation claims completion
- outcome classification
- failure class when validation fails

## Outcome Classifications

Supported outcome classifications:

- `live-covered`
- `partially-live-covered`
- `planned`
- `submitted`
- `no-wait`
- `retained`
- `cleanup-failed`
- `skipped`
- `unsupported`
- `proposal-backed`
- `failed`

The suite must not mark a topic `live-covered` when only mocks, stubs, help output, or empty no-match results were observed.

## Proposal Evidence

Command setup proposals must include:

- required state
- affected commands
- affected versions
- fallback used
- operator value

Embedded BPMN proposals must include:

- missing process behavior
- affected commands
- affected versions
- operator value

Aggregate proposal reports must include all per-family proposal records, including ops repair gaps.

## Output Cleanliness

Machine-readable stdout must remain parseable:

- JSON mode contains only JSON on stdout
- keys-only mode contains only keys, one per line
- prompts, warnings, progress, and human summaries must not leak into machine stdout

Human output must remain operator-useful:

- destructive examples and scenarios must carry visible danger wording where user-facing examples are involved
- long-running commands must show progress or durable progress facts
- final output must say what happened or why confirmation was skipped
