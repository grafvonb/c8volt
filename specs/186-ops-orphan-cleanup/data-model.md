# Data Model: Ops Purge Orphan Process Instances

## OrphanPurgeRequest

Represents one requested cleanup run.

Fields:

- `CommandName`: stable command name, `ops purge orphan-process-instances`
- `DryRun`: whether mutation is disabled
- `AutoConfirm`: whether destructive confirmation is pre-approved
- `Automation`: whether non-interactive automation mode is enabled
- `OutputMode`: selected command output mode
- `Selection`: process-instance filters compatible with orphan-child discovery
- `ReportFile`: optional report output path
- `ReportFormat`: optional requested report format
- `StartedAt`: command start time

Validation:

- `Automation` with mutation requires `AutoConfirm`.
- `DryRun` bypasses destructive confirmation.
- Selection must always include orphan-child-only semantics.
- Selection must not include explicit keys or filters that convert the workflow into general process-instance deletion.

## OrphanDiscoveryResult

Represents the immutable orphan set discovered at command start.

Fields:

- `Status`: shared step status
- `Filters`: normalized selected filters
- `Keys`: discovered process-instance keys
- `Count`: number of discovered keys
- `Errors`: discovery errors, if any

Validation:

- `Count` must equal the number of unique `Keys`.
- Downstream deletion must use only `Keys` from this result.

## DeletionPlan

Represents the validated plan for deleting the discovered keys.

Fields:

- `Status`: shared step status
- `RequestedKeys`: immutable discovered key set
- `AffectedKeys`: keys affected after existing process-instance delete planning expands dependencies
- `RootKeys`: root keys submitted to existing deletion behavior when applicable
- `RequiresConfirmation`: whether mutation requires confirmation
- `DryRunPreview`: existing delete plan preview data where available
- `Errors`: validation errors, if any

Validation:

- Must be built before mutation.
- Must fail before mutation if destructive execution is requested in automation without auto-confirm.
- Must preserve existing delete planning behavior, including force/state checks.

## DeletionResult

Represents mutation submission and confirmation results.

Fields:

- `Status`: shared step status
- `Items`: per-key or per-batch delete reports
- `Errors`: deletion or confirmation errors, if any
- `Submitted`: whether a delete request was sent
- `Confirmed`: whether the requested deletion outcome was verified

Validation:

- `Submitted` must be false for dry-run and no-target flows.
- Result items must correspond to the planned deletion scope.

## OrphanPurgeReport

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
- `SelectionFilters`
- `Discovery`
- `DeletionPlan`
- `Deletion`
- `DeleteRequested`
- `AutoConfirm`
- `Automation`
- `Errors`
- `Outcome`: `planned`, `deleted`, `partially_failed`, or `failed`

Validation:

- Must be serializable as deterministic JSON.
- Markdown and JSON reports must render from this model.
- Requested report files should be written even when cleanup fails after discovery.
