# Contract: Command Mode And Concern Reorganization

This contract defines the user-facing and maintainer-facing behavior that implementation tasks must preserve while reorganizing `cmd` package files.

## File Ownership Contract

Base command files must own:

- Command construction
- Aliases and examples
- Flag definition and validation
- Command metadata and capability annotations
- Top-level dispatch
- Ordinary non-mode execution

Focused mode files must own mode-specific lifecycle behavior when that behavior exists:

- Watch refresh loops, repaint, timing, status, and retry state
- Polling or follow loops
- Selector/search execution distinct from direct-key execution
- Dry-run planning coordination distinct from dry-run rendering
- Progress selection, pacing, and status tracking
- Report-file writing and report serialization

Renderer files must own only:

- View-model construction
- Layout and formatting
- Human output rendering
- Machine-readable output shaping when already part of the command presentation contract

Renderer files must not own:

- Facade calls
- Backend orchestration
- Page traversal
- Polling or retries
- Mutation planning
- Worker execution
- Version-specific service behavior

## Behavior Preservation Contract

Every reorganized slice must preserve the current behavior for:

- CLI flags and inherited flag handling
- Aliases and command paths
- Help text and examples
- Generated CLI documentation
- Capability metadata
- Human output text, summaries, ordering, and channels
- JSON output shape
- Keys-only output shape
- Quiet and verbose behavior
- Prompt wording and prompt timing
- Automation behavior
- Exit codes and failure classification
- Backend calls and request semantics

Any intentional behavior change must be removed from this feature or recorded as a separate follow-up before implementation proceeds.

## Command Mode Contract

Distinct command modes must be independently identifiable in source and tests when they have their own lifecycle, including:

- State
- Timing
- Retry or polling budget
- Status handling
- Request construction
- Refresh or repaint behavior
- Progress or milestone handling
- Direct-key versus selector/search execution

The process-definition watch reorganization must preserve the behavior baseline from the process-definition watch and repaint features.

## Resource View Contract

Resource-owned view areas must exist or remain clear for:

- Process-instance rendering
- Process-definition rendering
- Incident rendering
- Resource rendering
- Tenant rendering
- Generic flat-row layout

View tests must move or split along the same ownership lines as production rendering.

## Workflow Contract

Large workflow files must be reorganized by concern when the concern is distinct and reviewable:

- Job update command wiring, request parsing, worker-outcome handling, and planning
- Process-instance cancel selector/search execution and direct-key execution
- Process-instance delete selector/search execution and direct-key execution
- Root command wiring, configuration resolution, and service installation
- Slow-process analysis command behavior, validation, and progress
- Ops progress mode selection, milestone pacing, formatting, rendering, report-file helpers, Markdown serialization, JSON serialization, and workflow-specific report serialization

Destructive workflows must preserve:

- Confirmation behavior
- Auto-confirm behavior
- Automation behavior
- Force behavior
- Dry-run behavior
- Partial-completion reporting
- Worker controls
- Fail-fast behavior
- Deterministic exit behavior

## Test Ownership Contract

Tests must follow the production concern they verify:

- Watch tests beside watch mode behavior
- Renderer tests beside renderer ownership
- Process-instance support tests beside the relevant search, paging, dry-run, or mutation concern
- Job update tests beside the relevant command wiring, request parsing, outcome, or planning concern
- Ops progress/report tests beside the workflow concern they verify
- Subprocess helper tests beside the scenarios that require them

Large tests must be split incrementally with the production moves. A single broad test-only reorganization is outside this contract.

## Helper Removal Contract

A helper may be removed only after confirming:

- No production callers remain
- No test callers remain
- No subprocess helper scenario depends on it
- No examples or generated artifact checks depend on it
- Targeted tests and whitespace validation still pass

## Documentation And Metadata Contract

The default outcome is no intended user-facing documentation change.

If command behavior, flags, aliases, examples, output contracts, help text, or metadata change unintentionally, the implementation must either:

- Restore the previous behavior, or
- Split the change into a separate planned behavior change with source metadata, generated docs, README/docs updates, and targeted tests

Before completion, generated CLI documentation must be checked for unintended differences.
