# Data Model: Ops Execute Retention Policy

## RetentionPolicyRequest

Represents one requested retention cleanup run.

Fields:

- `CommandName`: stable command name, `ops execute retention-policy`
- `RetentionDays`: required non-negative age threshold
- `DerivedEndDateBoundary`: computed boundary when available
- `DryRun`: whether mutation is disabled
- `AutoConfirm`: whether destructive confirmation is pre-approved for interactive/scripted use
- `Automation`: whether non-interactive automation mode is enabled
- `OutputMode`: selected command output mode
- `Selection`: compatible process-instance discovery filters
- `ExecutionControls`: worker, fail-fast, wait, state-check, and force controls that preserve existing delete semantics
- `ReportFile`: optional report output path
- `ReportFormat`: optional requested report format
- `StartedAt`: command start time

Validation:

- `RetentionDays` is required and must be a non-negative integer.
- Selection must always include the retention age semantics.
- Selection must not include explicit process-instance keys.
- `DryRun` bypasses destructive confirmation and mutation.
- `Automation` is allowed only when the command contract declares full automation support.

## RetentionDiscoveryResult

Represents the immutable retention seed set discovered at command start.

Fields:

- `Status`: shared step status
- `RetentionDays`: requested retention age
- `DerivedEndDateBoundary`: computed boundary when available
- `Filters`: normalized selected filters
- `SeedKeys`: discovered retention seed process-instance keys
- `Count`: number of discovered seed keys
- `Notices`: semantic notices from search or filtering
- `Errors`: discovery errors, if any

Validation:

- `Count` must equal the number of unique `SeedKeys`.
- Downstream delete planning must use only `SeedKeys` from this result.
- Process instances without `endDate` are excluded by existing process-instance search behavior.

## RetentionDeletePlan

Represents the validated delete plan for discovered retention seed keys.

Fields:

- `Status`: shared step status
- `SeedKeys`: immutable discovered seed set
- `ResolvedRootKeys`: root keys selected by existing delete planning
- `AffectedKeys`: process-instance family keys affected by descendant traversal
- `DuplicateKeys`: duplicate keys removed by planning
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

## RetentionDeletionResult

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
- Submitted keys must come from `RetentionDeletePlan.ResolvedRootKeys`.
- Result items must correspond to the planned deletion scope.

## RetentionAuditReport

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
- `RetentionDays`
- `DerivedEndDateBoundary`
- `SelectionFilters`
- `Discovery`
- `DeletePlan`
- `Deletion`
- `AutoConfirm`
- `Automation`
- `NoWait`
- `NoStateCheck`
- `Force`
- `Errors`
- `Outcome`: `planned`, `deleted`, `partially_failed`, or `failed`

Validation:

- Must be serializable as deterministic JSON.
- Markdown and JSON reports must render from this model.
- Requested report files should be written when cleanup succeeds, plans, or fails after discovery.
- Sensitive data and unrelated process variables must not be included.

## RetentionWorkflowNotice

Represents semantic workflow notices for compact human output and complete machine/report output.

Fields:

- `Code`: stable notice code
- `Severity`: informational, warning, or error level
- `Message`: human-readable summary
- `Details`: structured optional context for JSON/report output

Validation:

- Expected conditions should not be rendered as misleading warnings in compact human output.
- JSON and report output should retain complete notice details.
