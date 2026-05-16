# Data Model: Ops Purge Process Instances With Incidents

## IncidentPurgeRequest

Represents one requested incident-based purge run.

Fields:

- `CommandName`: stable command name, `ops purge process-instances-with-incidents`
- `DryRun`: whether mutation is disabled
- `AutoConfirm`: whether destructive confirmation is pre-approved for interactive/scripted use
- `Automation`: whether non-interactive automation mode is enabled
- `OutputMode`: selected command output mode
- `Selection`: supported incident lookup filters
- `BatchSize`: incident search batch size
- `Limit`: maximum number of matching incidents considered before candidate process-instance dedupe
- `Workers`: delete worker count
- `FailFast`: whether delete execution should stop scheduling after failures
- `NoWorkerLimit`: whether worker count limits are disabled
- `NoWait`: whether delete confirmation waiting is skipped
- `Force`: whether non-final affected instances may be canceled before delete
- `ReportFile`: optional report output path
- `ReportFormat`: optional requested report format
- `DiscoveredCandidateProcessInstanceKeys`: optional frozen candidate keys for execution after a prior planning step
- `StartedAt`: command start time

Validation:

- Selection flags are incident filters; `--key` maps to incident key.
- Display-only incident flags are not part of this request.
- `DryRun` bypasses destructive confirmation and mutation.
- `Automation` is allowed only when the command contract declares full automation support.
- Frozen candidate keys, when supplied by command orchestration, must come from the discovery result rather than a second incident search.

## IncidentDiscoveryResult

Represents the immutable incident discovery result.

Fields:

- `Status`: shared step status
- `Filters`: normalized incident filters
- `IncidentKeys`: matching incident keys
- `CandidateProcessInstanceKeys`: unique process-instance keys extracted from matching incidents
- `DuplicateCandidateProcessInstanceKeys`: process-instance keys seen more than once across matching incidents
- `SkippedIncidents`: matching incidents that cannot produce a usable process-instance candidate
- `IncidentCount`: number of candidate incidents considered
- `CandidateProcessInstanceCount`: number of unique candidate process instances
- `Notices`: semantic notices from incident discovery and filtering
- `Errors`: discovery errors, if any

Validation:

- `IncidentCount` must respect `Limit` and incident search order.
- `CandidateProcessInstanceCount` must equal the number of unique `CandidateProcessInstanceKeys`.
- Downstream delete planning must use only `CandidateProcessInstanceKeys`.
- Missing process-instance keys must not create delete requests.

## IncidentPurgeDeletePlan

Represents the validated delete plan for frozen candidate process-instance keys.

Fields:

- `Status`: shared step status
- `CandidateProcessInstanceKeys`: frozen candidate keys from discovery
- `ResolvedRootKeys`: root keys selected by existing delete planning
- `AffectedKeys`: process-instance family keys affected by descendant traversal
- `DuplicateCandidateProcessInstanceKeys`: duplicate candidates removed before planning
- `FinalStateItems`: selected or affected items already in final states
- `NonFinalAffectedItems`: affected items that require cancellation or block mutation
- `MissingAncestors`: missing ancestor details from traversal
- `TraversalWarnings`: partial traversal and related warnings
- `RequiresConfirmation`: whether mutation requires confirmation
- `Errors`: validation errors, if any

Validation:

- Must be built before mutation.
- Must be serializable for human, JSON, and report output.
- Must preserve existing process-instance delete planning behavior, including force and state-check rules.

## IncidentPurgeDeletionResult

Represents mutation submission and confirmation results.

Fields:

- `Status`: shared step status
- `SubmittedRootKeys`: keys submitted to deletion
- `Items`: per-key or per-batch delete reports
- `Submitted`: whether a delete or cancellation request was sent
- `Confirmed`: whether the requested deletion outcome was verified
- `NoWait`: whether confirmation waiting was skipped
- `Errors`: deletion or confirmation errors, if any

Validation:

- `Submitted` must be false for dry-run and no-target flows.
- Submitted keys must come from `IncidentPurgeDeletePlan.ResolvedRootKeys`.
- Result items must correspond to the planned deletion scope.

## IncidentPurgeReport

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

## IncidentPurgeWorkflowNotice

Represents semantic workflow notices for compact human output and complete machine/report output.

Fields:

- `Code`: stable notice code
- `Severity`: informational, warning, or error level
- `Message`: human-readable summary
- `Details`: structured optional context for JSON/report output

Validation:

- Expected conditions should not be rendered as misleading warnings in compact human output.
- JSON and report output should retain complete notice details.
