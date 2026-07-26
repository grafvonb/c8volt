# Data Model: Volume And Semantic CLI Integration Coverage

## Volume Target

Represents one independently runnable family-level volume test target.

Fields:

- `name`: stable target name, such as `integration-cli-get-volume`
- `family`: command family or ops family under test
- `testPattern`: Go test name or pattern invoked by the target
- `defaultTimeout`: target timeout for normal local validation, controlled by `C8VOLT_IT_VOLUME_TIMEOUT`
- `defaultDatasetCount`: conservative default count of suite-owned records, controlled by `C8VOLT_IT_VOLUME_COUNT`
- `destructive`: whether the target may mutate cluster state
- `profiles`: selected default-local profiles used for the run

Validation rules:

- Every volume target must run independently in any order.
- Every destructive target must be clearly named and documented as destructive.
- Targets must not invoke baseline family tests unless explicitly intended.

## Volume Dataset

Represents suite-owned records created for volume scenarios.

Fields:

- `marker`: unique run marker for this dataset
- `profile`: selected c8volt profile
- `camundaVersion`: observed Camunda version
- `requestedCount`: configured dataset count
- `createdProcessDefinitionKeys`
- `createdProcessInstanceKeys`
- `createdJobKeys`
- `createdIncidentKeys`
- `createdElementInstanceKeys`
- `positiveSelectors`
- `negativeSelectors`
- `retainedResources`
- `cleanupRecords`

Validation rules:

- Dataset records must be distinguishable from unrelated dirty-cluster data.
- Positive and negative selectors must be recorded when used for filtering checks.
- Cleanup is best effort and must not hide command behavior evidence.

## Critical Flag Scenario

Represents semantic validation for one high-risk flag or flag combination.

Fields:

- `commandPath`
- `scenarioName`
- `flagsUnderTest`
- `inputMode`: explicit keys, filters, stdin, or generated data
- `expectedScope`
- `expectedMutation`
- `expectedOutputMode`
- `postConditionChecks`
- `versionBehavior`

Validation rules:

- Dry-run scenarios must include post-run no-mutation checks for suite-owned resources.
- Limit scenarios must include more matching records than the requested limit.
- Machine-output scenarios must validate stdout cleanliness.
- Version-specific scenarios must fail before unsafe mutation when unsupported.

## Pipeline Scenario

Represents a producer-to-consumer stdin workflow.

Fields:

- `producerCommand`
- `producerOutputMode`
- `consumerCommand`
- `consumerInputMode`
- `stdinPath`
- `inputShape`: empty, duplicate, malformed, missing, valid, mixed, or whitespace-padded
- `dryRun`
- `confirmedMutation`
- `expectedOutcome`

Validation rules:

- Keys-only producer output must contain only keys, one per line.
- Stdin consumers must not hang on empty input.
- Mixed input must produce actionable partial-success or fail-fast evidence.

## Progress Evidence

Represents proof that a long-running command visibly progresses and finishes clearly.

Fields:

- `scenarioName`
- `commandPath`
- `mode`: human, verbose, quiet, automation, json, keys-only, or json-log
- `stdoutPath`
- `stderrPath`
- `activityLogPath`
- `progressFacts`
- `finalOutcome`
- `elapsed`
- `machineOutputClean`

Validation rules:

- Human or verbose scenarios must record visible progress or durable progress facts.
- Machine modes must not place progress, prompts, warnings, or summaries in stdout.
- Every long-running scenario must record a final outcome or explicit submitted/no-wait state.

## Ops Report Evidence

Represents validation of one ops audit report.

Fields:

- `commandPath`
- `workflow`
- `reportPath`
- `reportFormat`
- `dryRun`
- `profileIdentity`
- `camundaVersion`
- `discoveryCompleteness`
- `stepStatuses`
- `mutationAccounting`
- `notices`
- `errors`
- `stdoutOutcome`
- `reportOutcome`

Validation rules:

- JSON reports must parse and expose stable fields.
- Markdown reports must include stable high-level sections.
- Discovery completeness must identify complete or user-limited scope.
- Step statuses must use the shared ops status vocabulary.
- JSON stdout and JSON report must agree on frozen scope and final outcome when both are produced.

## Legacy Proposal Record

Extends the baseline proposal evidence for volume-specific setup gaps. This generated-test-output pattern is deprecated by `specs/integration-test-responsibility.md`; new work should use spec-owned gap artifacts.

Fields:

- `kind`: command or embedded BPMN
- `requiredState`
- `coverageNeed`
- `fallbackUsed`
- `affectedCommands`
- `affectedVersions`
- `operatorValue`
- `volumeScenario`

Validation rules:

- Historical direct Camunda setup fallback evidence produced a command proposal.
- Historical missing embedded BPMN behavior evidence produced an embedded BPMN proposal.
- New work must not generate backlog proposal records from tests; maintain missing setup and fixture needs in spec-owned gap artifacts.
