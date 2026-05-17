# Data Model: Ops Purge All Process Definitions

## AllProcessDefinitionsPurgeRequest

Represents one requested purge run.

Fields:

- `CommandName`: stable command name, `ops purge all-process-definitions`
- `DryRun`: whether mutation is disabled
- `AutoConfirm`: whether destructive confirmation is pre-approved for interactive/scripted use
- `Automation`: whether non-interactive automation mode is enabled
- `OutputMode`: selected command output mode
- `Selection`: supported process-definition lookup filters
- `Workers`: delete worker count
- `FailFast`: whether delete execution should stop scheduling after failures
- `NoWorkerLimit`: whether worker count limits are disabled
- `NoWait`: whether delete confirmation waiting is skipped
- `Force`: whether active affected process instances may be canceled before delete
- `ReportFile`: optional report output path
- `ReportFormat`: optional requested report format
- `DiscoveredCandidateProcessDefinitionKeys`: optional frozen candidate keys for execution after a prior planning step
- `StartedAt`: command start time

Validation:

- Selection flags are process-definition filters: key, BPMN process ID, process-definition version, version tag, and latest-only scope.
- Display-only process-definition flags are not part of this request.
- `DryRun` bypasses destructive confirmation and mutation.
- `Automation` is allowed only when the command contract declares full automation support.
- Frozen candidate keys, when supplied by command orchestration, must come from the discovery result rather than a second process-definition search.

## ProcessDefinitionDiscoveryResult

Represents the immutable process-definition discovery result.

Fields:

- `Status`: shared step status
- `Filters`: normalized process-definition filters
- `CandidateProcessDefinitionKeys`: unique process-definition keys discovered through `get pd`-equivalent selection
- `CandidateProcessDefinitions`: optional display metadata such as key, BPMN process ID, version, version tag, and latest flag when available
- `DuplicateCandidateProcessDefinitionKeys`: process-definition keys seen more than once across matching results
- `CandidateProcessDefinitionCount`: number of unique candidate process definitions
- `LatestOnly`: whether `--latest` narrowed the candidate scope
- `Notices`: semantic notices from discovery and filtering
- `Errors`: discovery errors, if any

Validation:

- Candidate count must equal the number of unique `CandidateProcessDefinitionKeys`.
- Downstream delete preflight must use only `CandidateProcessDefinitionKeys`.
- No-target discovery must submit no delete request.
- `LatestOnly` must be represented in human, JSON, and report output when true.

## AllProcessDefinitionsPurgeDeletePlan

Represents the validated delete preflight for frozen candidate process-definition keys.

Fields:

- `Status`: shared step status
- `CandidateProcessDefinitionKeys`: frozen candidate keys from discovery
- `Items`: existing process-definition delete-plan items
- `DuplicateCandidateProcessDefinitionKeys`: duplicate candidates removed before planning
- `AffectedProcessInstanceCount`: total process-instance impact from delete preflight
- `ActiveProcessInstanceCount`: active process-instance impact that may require `--force`
- `RequiresConfirmation`: whether mutation requires confirmation
- `RequiresForce`: whether active process-instance impact blocks mutation without `--force`
- `Errors`: validation errors, if any

Validation:

- Must be built before mutation.
- Must be serializable for human, JSON, and report output.
- Must preserve existing process-definition delete preflight behavior, including active-instance checks and force rules.

## AllProcessDefinitionsPurgeDeletionResult

Represents mutation submission and confirmation results.

Fields:

- `Status`: shared step status
- `SubmittedProcessDefinitionKeys`: keys submitted to deletion
- `Items`: per-key delete reports from the existing resource delete path
- `Submitted`: whether a delete, cancellation, or history cleanup request was sent
- `Confirmed`: whether the requested deletion outcome was verified
- `NoWait`: whether confirmation waiting was skipped
- `Errors`: deletion or confirmation errors, if any

Validation:

- `Submitted` must be false for dry-run and no-target flows.
- Submitted keys must come from `AllProcessDefinitionsPurgeDeletePlan.CandidateProcessDefinitionKeys`.
- Result items must correspond to the planned deletion scope.

## AllProcessDefinitionsPurgeReport

Stable audit model rendered to human output, JSON output, and optional report files.

Fields:

- `SchemaVersion`
- `CommandName`
- `StartedAt`
- `FinishedAt`
- `Duration`
- `DryRun`
- `C8voltVersion`
- `CamundaVersion`
- `ProfileIdentity`
- `TenantID`
- `SelectionFilters`
- `Discovery`
- `DeletePlan`
- `Deletion`
- `AutoConfirm`
- `Automation`
- `NoWait`
- `Force`
- `FailFast`
- `NoWorkerLimit`
- `Errors`
- `Notices`
- `Outcome`: `planned`, `deleted`, `partially_failed`, or `failed`

Validation:

- Must be serializable as deterministic JSON.
- Markdown and JSON reports must render from this model.
- Requested report files should be written when the workflow plans, succeeds, or fails after discovery/report data exists.
- Sensitive data and unrelated process variables must not be included.

## AllProcessDefinitionsPurgeWorkflowNotice

Represents semantic workflow notices for compact human output and complete machine/report output.

Fields:

- `Code`: stable notice code
- `Severity`: informational, warning, or error level
- `Message`: human-readable summary
- `Details`: structured optional context for JSON/report output

Validation:

- Expected conditions should not be rendered as misleading warnings in compact human output.
- JSON and report output should retain complete notice details.
