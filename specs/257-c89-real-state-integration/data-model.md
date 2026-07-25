# Data Model: C89 Real-State Semantic Integration Coverage

## Real-State Target

Represents one independently runnable Camunda 8.9 real-state test target.

Fields:

- `name`: stable Make target name, such as `integration-cli-real-state-jobs`
- `testPattern`: Go test name or pattern invoked by the target
- `topic`: jobs, incidents, listeners, BPMN errors, retention, destructive commands, or proposals
- `destructive`: whether the target may mutate the selected cluster
- `defaultTimeout`: Make-level timeout for the target
- `profiles`: selected profiles from the default local c8volt config
- `requiredState`: runtime state the target must create or discover

Validation rules:

- Every target must run independently in any order.
- Every target must tolerate clean and dirty clusters.
- Targets must not require a private config file or explicit `--config`.
- Destructive targets must document that they may mutate selected disposable clusters.

## Real-State Fixture

Represents process models and runtime data used to create observable Camunda state.

Fields:

- `fixtureKind`: embedded BPMN, c8volt command setup, direct Camunda API setup, or discovered pre-existing state
- `bpmnProcessID`
- `processDefinitionKeys`
- `processInstanceKeys`
- `elementInstanceKeys`
- `jobKeys`
- `incidentKeys`
- `listenerJobKeys`
- `marker`
- `camundaVersion`

Validation rules:

- Existing embedded BPMN fixtures must be preferred before new setup paths.
- Direct Camunda API setup must produce a command proposal.
- Missing embedded behavior must produce an embedded BPMN proposal.
- Fixture records must identify suite-owned data when assertions depend on ownership.

## State Evidence

Represents observable proof collected before and after a command.

Fields:

- `scenarioName`
- `commandPath`
- `profile`
- `camundaVersion`
- `beforeKeys`
- `afterKeys`
- `beforeState`
- `afterState`
- `stdoutPath`
- `stderrPath`
- `reportPath`
- `outcome`
- `classification`

Validation rules:

- Mutation scenarios must include before-state and after-state evidence where the command claims completion.
- No-wait or accepted scenarios must say confirmation was intentionally skipped.
- Machine-readable output must stay parseable and free of human progress text.
- Human output must include enough final outcome wording for an operator to know what happened.

## Mixed Target Set

Represents input used to prove partial failure and fail-fast behavior.

Fields:

- `validKeys`
- `missingKeys`
- `malformedInputs`
- `staleKeys`
- `alreadyMutatedKeys`
- `stdinPath`
- `failFast`
- `expectedAttempted`
- `expectedSkipped`
- `expectedFailed`
- `expectedCompleted`

Validation rules:

- Mixed target scenarios must include at least one valid target and one invalid or stale target.
- Fail-fast scenarios must prove where execution stopped or how stopping was reported.
- Non-fail-fast scenarios must prove partial accounting remains actionable.

## Gap Proposal

Represents missing command setup or missing embedded BPMN behavior.

Fields:

- `kind`: command or embedded BPMN
- `requiredState`
- `coverageNeed`
- `fallbackUsed`
- `affectedCommands`
- `affectedVersions`
- `operatorValue`
- `realStateTopic`

Validation rules:

- Proposal records must be written when live real-state coverage is blocked.
- Aggregate proposal evidence must include all per-family proposal records.
- Proposal records must not be treated as live coverage.

## Coverage Matrix Row

Represents current status for one real-state coverage topic.

Fields:

- `topic`
- `currentEvidenceLevel`
- `targetRealStateProof`
- `firstFollowUp`
- `status`: live-covered, partially live-covered, no-match only, proposal-backed, or not yet started

Validation rules:

- Every priority topic from the spec must have a row.
- Rows must be updated when a topic moves from proposal-backed or no-match-only to live-covered.
- Version-sensitive rows must keep Camunda 8.9 scope explicit.
