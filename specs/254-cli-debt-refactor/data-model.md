# Data Model: CLI Debt Refactor

This feature does not introduce persistent storage. The model defines repository artifacts, command behavior records, and validation/reporting concepts used to plan and verify the refactor.

## Command Node

Represents one CLI command or grouping node in the live command tree.

**Fields**

- `path`: Canonical command path, such as `get process-instance` or `ops repair incident`
- `aliases`: Supported aliases
- `family`: One of grouping, basic read, basic mutation, high-level ops, compatibility, config, or embed
- `mutation`: Read-only or state-changing
- `contractSupport`: Full, limited, or unsupported
- `automationSupport`: Full or unsupported
- `outputModes`: Supported human and machine output modes
- `flags`: Relevant flags, including `--batch-size`, `--limit`, `--workers`, `--fail-fast`, `--no-worker-limit`, `--automation`, `--quiet`, `--json`, and `--keys-only` where present
- `pages`: Whether the command pages backend data
- `mutates`: Whether the command can change remote state
- `activityIndicator`: Whether the command uses transient activity output
- `durableProgress`: Whether the command emits persistent progress or discovery summaries
- `ownership`: Current owner of paging, discovery, query strategy, rendering, confirmation, and mutation planning behavior
- `executionStyle`: Mostly serial, bounded concurrent, or mixed
- `performanceRisk`: Low, medium, or high for thousands of resources

**Validation Rules**

- Every one of the 55 command nodes must have exactly one record in the assessment.
- Grouping/discovery commands may have limited contract support but must still be classified.
- Unsupported compatibility/config/embed nodes must be listed rather than silently excluded.

## Command Behavior Assessment

Represents the checked-in baseline produced before refactoring decisions are marked complete.

**Fields**

- `createdDate`: Date the assessment was generated or last materially updated
- `scope`: Command tree version and feature branch
- `commands`: Complete set of command node records
- `highRiskPaths`: Commands or workflows with the greatest high-volume risk
- `duplicationFindings`: Repeated mechanics that are candidates for extraction
- `intentionalDifferences`: Similar patterns retained because workflow semantics differ
- `performanceCharacterizationPlan`: How high-volume paths will be measured

**Relationships**

- Contains many Command Node records.
- Informs Refactor Slice ordering and validation scope.

**Validation Rules**

- Must cover all 55 nodes before refactoring work is considered ready.
- Must distinguish command-owned behavior from facade-owned and service-owned behavior.
- Must document real duplicated helpers or justify why similar helpers remain separate.

## Refactor Slice

Represents an independently testable implementation unit.

**Fields**

- `name`: Slice name
- `priority`: P1, P2, P3, or P4
- `commandsAffected`: Command nodes touched
- `serviceAreasAffected`: Internal service/facade areas touched
- `behaviorBaseline`: User-visible behavior to preserve
- `expectedOwnershipChange`: Mechanics moved or deliberately retained
- `performanceValidation`: Targeted test, fake-latency check, benchmark-style scenario, or smoke scenario
- `documentationImpact`: None, help update, generated docs update, README/docs update, or capability metadata update

**Validation Rules**

- Each slice must be independently testable.
- A slice that changes user-visible command behavior must include documentation and capability metadata validation.
- A slice that touches destructive workflows must include confirmation, partial-completion, auto-confirm, automation, and exit-behavior validation.

## Paged Discovery Workflow

Represents a multi-page retrieval or discovery process used by read, mutation, or ops commands.

**Fields**

- `scope`: What is being discovered, such as jobs, elements, incidents, process instances, process definitions, or ops candidates
- `pageSize`: Per-page request size
- `limit`: Total user cap or frozen-scope cap
- `advanceMode`: Cursor, offset, or version-specific fallback
- `completionStatus`: Complete, user-limited, warning-stop, partial-complete, or failed
- `itemsSeen`: Count fetched or inspected
- `itemsSelected`: Count rendered, frozen, or planned
- `reportedTotal`: Exact, lower-bound, unavailable, or ignored because local filtering changes meaning
- `progressBehavior`: Spinner, verbose durable progress, discovery summary, prompt, or silence

**Validation Rules**

- `--batch-size` controls page size only.
- `--limit` controls total user cap or frozen-scope cap.
- Machine output modes must not include incidental progress or prompt text.
- Warning-stop and partial-complete statuses must not be reported as complete.

## Progress Policy

Represents the CLI-wide output behavior contract for long-running work.

**Fields**

- `activityIndicatorRule`: Transient indicator policy for opaque long-running work
- `verboseProgressRule`: Durable progress policy for paging and long-running mutations
- `opsDiscoverySummaryRule`: Stable summary policy for complete vs user-limited discovery
- `promptRule`: Interactive continuation and confirmation text requirements
- `machineOutputRule`: Output silence requirements for JSON, keys-only, quiet, and automation modes

**Validation Rules**

- Activity indicators must use the shared activity writer and respect no-indicator, quiet, automation, and JSON log constraints.
- Verbose progress must go to stderr/log paths rather than stdout.
- JSON output must remain one valid document.
- Keys-only output must print one key per line and nothing else.

## High-Volume Workflow

Represents commands or service flows expected to handle thousands of process instances or related resources.

**Fields**

- `workflow`: Name of the command or service flow
- `resourceTypes`: Process instances, jobs, elements, incidents, process definitions, variables, runtime elements, or related resources
- `currentExecutionStyle`: Serial, bounded concurrent, or mixed
- `independentWork`: Lookup, enrichment, planning, confirmation checks, or mutation work that can safely run concurrently
- `operatorControls`: Worker, batch, limit, fail-fast, no-worker-limit, auto-confirm, automation, and dry-run controls that bound behavior
- `performanceRisk`: Low, medium, or high
- `measurement`: Fake-latency test, benchmark-style test, targeted smoke scenario, or documented manual verification

**Validation Rules**

- Independent work may use bounded concurrency when it preserves output order and safety semantics.
- Page traversal must avoid uncontrolled fan-out.
- No changed workflow may be slower in targeted validation without a documented tradeoff.

## Machine Output Contract

Represents parseable output guarantees for automation consumers.

**Fields**

- `mode`: JSON, keys-only, quiet, or automation
- `stdoutContract`: Expected stdout shape
- `stderrContract`: Allowed progress or diagnostic behavior
- `promptContract`: Whether prompting is allowed
- `failureContract`: Exit and error reporting expectation

**Validation Rules**

- JSON emits one valid document or envelope.
- Keys-only emits one key per line and no extra text.
- Automation mode must not prompt unexpectedly.
- Quiet mode suppresses nonessential human output.

## State Transitions

### Paged Discovery Workflow

```text
NotStarted
  -> FetchingPage
  -> RenderingOrPlanningPage
  -> WaitingForPrompt
  -> FetchingPage
  -> Complete | UserLimited | WarningStop | PartialComplete | Failed
```

### Refactor Slice

```text
Proposed
  -> Assessed
  -> Implemented
  -> TargetValidated
  -> DocsValidated
  -> BroadValidated
```
