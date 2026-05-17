# Data Model: Ops Execute Smoke Test

## SmokeTestRequest

Captures one requested `ops execute smoke-test` run.

- `commandName`: stable command path, `ops execute smoke-test`
- `dryRun`: whether the run is read-only planning
- `count`: requested process-instance creation count, positive integer defaulting to `1`
- `workers`: requested worker count for reusable process-instance creation/deletion behavior
- `failFast`: whether worker-backed operations stop scheduling on first failure
- `noWorkerLimit`: whether worker count can exceed normal limits
- `noCleanup`: whether created resources should be retained
- `autoConfirm`: convenience confirmation flag
- `automation`: non-interactive automation mode
- `noWait`: lower-level cleanup no-wait control
- `outputMode`: selected command output mode
- `reportFile`: optional audit report path
- `reportFormat`: optional report format override
- `startedAt`: command start timestamp

Validation rules:

- `count` must be positive.
- `reportFormat` must be `markdown` or `json` when provided.
- `reportFormat` requires `reportFile`.
- `dryRun` must not submit mutation steps.

## SmokeTestPlan

The read-only plan for a smoke-test run.

- `status`: workflow step status
- `camundaVersion`: configured Camunda version
- `fixture`: selected embedded smoke-test fixture
- `cleanupRequested`: inverse of `noCleanup`
- `plannedSteps`: ordered list of connectivity, deployment, run, walk, cleanup, and report steps
- `errors`: planning errors

Validation rules:

- Fixture must exist for the configured Camunda version before mutation.
- Dry-run returns a plan with mutation steps marked planned or skipped, never submitted.

## EmbeddedSmokeTestFixture

The selected BPMN fixture.

- `camundaVersion`: 8.7, 8.8, or 8.9
- `file`: embedded fixture file path
- `bpmnProcessID`: BPMN process ID, e.g. `C89_MultipleSubProcessesParentProcess`
- `available`: whether the embedded asset exists

Validation rules:

- Camunda 8.7 maps to `C87_MultipleSubProcessesParentProcess.bpmn`.
- Camunda 8.8 maps to `C88_MultipleSubProcessesParentProcess.bpmn`.
- Camunda 8.9 maps to `C89_MultipleSubProcessesParentProcess.bpmn`.

## SmokeTestDeploymentResult

Deployment step output.

- `status`: workflow step status
- `fixtureFile`: selected fixture file
- `bpmnProcessID`: deployed BPMN process ID
- `processDefinitionKey`: deployed process-definition key when available
- `processDefinitionVersion`: deployed version when available
- `tenantID`: tenant id when available
- `errors`: deployment errors

## SmokeTestRunResult

Process-instance creation output.

- `status`: workflow step status
- `requestedCount`: requested count
- `createdCount`: number of created instances
- `processInstanceKeys`: created process-instance keys
- `items`: per-instance creation status
- `errors`: run errors

Validation rules:

- Creation should prefer deployed process-definition key when available and supported.
- Output order should remain deterministic.

## SmokeTestWalkResult

Traversal output for created process instances.

- `status`: workflow step status
- `items`: per-created-instance traversal summaries
- `errors`: traversal errors

Each traversal summary should include the starting process-instance key, root/family summary counts where available, and status. It should not dump unrelated process variables or sensitive payloads.

## SmokeTestCleanupResult

Cleanup output.

- `processInstanceCleanup`: delete-plan and deletion status for created process instances
- `processDefinitionEligibility`: whether deployed definition cleanup is safe
- `processDefinitionCleanup`: process-definition deletion status
- `noCleanup`: whether cleanup was intentionally skipped
- `errors`: cleanup errors

Validation rules:

- `noCleanup` skips all cleanup mutations.
- Process-instance cleanup uses existing delete planning/deletion behavior.
- Process-definition cleanup is skipped when unrelated process instances exist for the deployed definition or BPMN process ID.

## SmokeTestAuditReport

Stable structured audit report rendered to JSON, Markdown, or command JSON output.

- `schemaVersion`
- `commandName`
- `startedAt`
- `finishedAt`
- `duration`
- `dryRun`
- `c8voltVersion`
- `camundaVersion`
- `profileIdentity`
- `tenantID`
- `fixture`
- `deployment`
- `run`
- `walk`
- `cleanupRequested`
- `noCleanup`
- `cleanup`
- `autoConfirm`
- `automation`
- `noWait`
- `errors`
- `outcome`

Final outcome values:

- `planned`
- `passed`
- `passed_cleanup_skipped`
- `partially_failed`
- `failed`

## WorkflowStepResult

Reusable per-step status entry.

- `name`: step name
- `status`: one of `planned`, `skipped`, `submitted`, `confirmed`, `confirmation_failed`, `blocked`, or `failed`
- `message`: compact human-readable summary
- `errors`: step errors

Validation rules:

- JSON output and reports must preserve complete step detail even when human output is compact.
