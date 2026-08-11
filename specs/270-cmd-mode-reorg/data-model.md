# Data Model: Command Mode And Concern Reorganization

This feature does not introduce persistent storage. The model defines repository ownership concepts, behavior contracts, and validation states used to plan and verify the reorganization.

## Command Mode

Represents a distinct user-facing execution path with behavior beyond ordinary command dispatch.

**Fields**

- `commandPath`: Canonical command path, such as `get process-definition`, `cancel process-instance`, or `ops analyse slow-process-instances`
- `modeName`: Watch, polling, selector search, direct-key execution, dry-run, progress, report, or ordinary execution
- `lifecycleResponsibilities`: State, timing, retry, polling, repaint, status, request construction, or worker handling owned by the mode
- `baselineBehavior`: Output, prompts, exit behavior, backend calls, and metadata expected before reorganization
- `focusedFile`: Target ownership file or files for the mode
- `validationScope`: Targeted tests and representative output checks required for the mode

**Validation Rules**

- A mode with three or more mode-specific declarations must have focused ownership.
- A mode move must preserve baseline behavior before follow-up ownership corrections begin.
- Machine-output compatibility must be checked whenever output routing or render-mode decisions are touched.

## Base Command Area

Represents ordinary command ownership after mode-specific behavior is separated.

**Fields**

- `commandPath`: Canonical command path
- `constructionResponsibilities`: Command creation, aliases, examples, metadata, and child wiring
- `flagResponsibilities`: Local flags, inherited flag interpretation, and validation
- `dispatchResponsibilities`: Selection of ordinary execution or focused mode execution
- `retainedBehavior`: Behavior that intentionally remains in the base command file

**Validation Rules**

- Base command files must stay focused on construction, flags, validation, top-level dispatch, and ordinary execution.
- Base command files must not take ownership of backend traversal, polling, mutation planning, or worker execution.

## Renderer Area

Represents output ownership for a resource or shared layout concern.

**Fields**

- `resource`: Process instance, process definition, incident, resource, tenant, operation, or shared layout
- `viewModels`: User-facing data shapes rendered by the area
- `renderModes`: Human, JSON, keys-only, quiet, verbose, report, or other supported presentation modes
- `outputBaseline`: Text, field ordering, summaries, and machine-readable shape before reorganization
- `nonRenderingWorkMoved`: Facade calls, orchestration, or planning that must leave the renderer if present

**Validation Rules**

- Renderer areas may build view models, choose layout, and format output.
- Renderer areas must not perform facade calls, backend orchestration, traversal, polling, mutation planning, or workflow execution.
- Generic layout helpers must live in shared rendering ownership rather than resource-specific mode-selection files.

## Workflow Concern

Represents a separable responsibility within a larger command workflow.

**Fields**

- `workflow`: Job update, process-instance cancel, process-instance delete, root command support, slow-process analysis, ops progress, or ops report workflow
- `concern`: Request parsing, direct-key execution, selector execution, planning, progress selection, milestone pacing, terminal formatting, report writing, JSON serialization, Markdown serialization, or worker-outcome handling
- `currentOwner`: Existing source file or area
- `targetOwner`: Focused source file or follow-up issue
- `behaviorRisk`: Low, medium, or high
- `validationScope`: Tests and output checks required after movement

**Validation Rules**

- Distinct workflow concerns should be separately identifiable and testable.
- Destructive workflow concerns must preserve confirmation, auto-confirm, automation, force, partial-completion, and deterministic exit behavior.
- Report serialization must remain distinct from terminal rendering when those outputs have different contracts.

## Behavior Contract

Represents observable CLI behavior that must remain stable.

**Fields**

- `flagsAndAliases`: Existing command flags, aliases, and inherited options
- `helpAndMetadata`: Help text, examples, generated docs, and capability metadata
- `stdoutContract`: Human, JSON, keys-only, quiet, or report stdout shape
- `stderrContract`: Progress, verbose details, prompts, warnings, and activity output
- `exitContract`: Success, user cancellation, validation failure, partial failure, and backend failure behavior
- `backendCallContract`: Remote calls and facade/service interactions expected before reorganization

**Validation Rules**

- Behavior contracts must not change unintentionally.
- JSON output must remain one valid document.
- Keys-only output must print one key per line and nothing else.
- Automation mode must not prompt unexpectedly.
- Help, docs, and metadata diffs must be intentional and documented.

## Reorganization Slice

Represents a small, reviewable unit of implementation.

**Fields**

- `name`: Slice name
- `priority`: P1, P2, P3, or P4
- `sourceFilesMoved`: Production files or declarations moved
- `testFilesMoved`: Tests moved or split with the production concern
- `behaviorBaseline`: User-visible behavior checked before and after
- `followUpsIdentified`: Ownership issues deferred because they exceed mechanical movement
- `validationEvidence`: Commands run and outcomes observed

**Validation Rules**

- Each slice must be independently testable.
- File moves should precede behavior or ownership corrections in the same area.
- A slice is not complete until targeted validation and whitespace checks pass.

## Follow-Up Ownership Correction

Represents a non-mechanical issue discovered during reorganization.

**Fields**

- `area`: Command, renderer, facade, service, or report area affected
- `finding`: Ownership concern found during review
- `impact`: Potential behavior, architecture, or validation effect
- `recommendedNextStep`: Separate issue, later task, or documented no-op
- `reasonDeferred`: Why it should not be blended into the current behavior-preserving slice

**Validation Rules**

- Follow-ups must be explicit when an ownership correction could affect behavior or architecture.
- Follow-ups must not block mechanical reorganization unless the current state makes safe movement impossible.

## State Transitions

### Reorganization Slice

```text
Proposed
  -> BaselineReviewed
  -> FilesMoved
  -> TestsAligned
  -> TargetValidated
  -> DocsChecked
  -> Complete
```

### Follow-Up Ownership Correction

```text
Discovered
  -> ImpactAssessed
  -> DeferredWithReason | IncludedWithValidation
```
