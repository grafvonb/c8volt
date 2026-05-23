# Data Model: BPMN Selector Validation for Operational Commands

## BPMN Process Definition Selector

Represents the command-visible selector used to identify one or more visible process definitions.

**Fields**

- `bpmnProcessId`: BPMN process ID provided by `--bpmn-process-id`.
- `processVersion`: Optional exact process-definition version.
- `processVersionTag`: Optional process-definition version tag.
- `tenantContext`: Effective tenant context from flags or configuration.
- `selectionMode`: Whether the command needs any visible matching definition or the latest visible matching definition.

**Validation Rules**

- The selector is checked only when a command directly receives `--bpmn-process-id`.
- Version, version tag, and tenant context narrow the visible match where the command supports those selectors.
- Missing or invisible selectors are command failures before protected runtime search or mutation work begins.

## Operational Command Selector Decision

Represents the audit outcome for each command that registers `--bpmn-process-id`.

**Fields**

- `command`: User-facing command path.
- `operationKind`: Read-only search, runtime mutation, direct process-definition selection, or process start.
- `selectorSource`: Direct BPMN selector, stdin key stream, or explicit key flag.
- `validationRequired`: Whether visible process-definition validation must run before the command proceeds.
- `documentedOutcome`: The help/docs/test outcome for missing selectors.

**Validation Rules**

- `get pi` and `run pi` remain aligned with existing validation behavior.
- `cancel pi`, `delete pi`, and `get incident` require validation when `--bpmn-process-id` is set.
- `get pd` and `delete pd` require an explicit audit outcome and tests or documentation for missing BPMN selectors.
- Commands without a direct BPMN selector remain unchanged.

## Selector Validation Outcome

Represents the preflight result before command-specific resource work.

**Fields**

- `requestedSelectors`: Selectors checked in the effective command context.
- `matches`: Visible process definitions matching each selector.
- `missingSelectors`: Selectors with no visible match.
- `nearMatches`: Process definitions that match BPMN ID but not version, version tag, or tenant context.
- `promptAllowed`: Whether human recovery listing may be offered.

**State Transitions**

- `unchecked` -> `valid`: Every requested selector has a visible match.
- `unchecked` -> `invalid`: One or more requested selectors has no visible match.
- `invalid` -> `listed`: A prompt-eligible human user accepts recovery listing after the command has already decided to fail.

## Runtime Resource Result

Represents process-instance or incident results after selector validation succeeds.

**Fields**

- `resourceKind`: Process instance or incident.
- `items`: Matching runtime resources.
- `total`: Displayed or structured result count when available.
- `emptyReason`: Empty because no runtime resource matched, not because selector visibility was unknown.

**Validation Rules**

- Empty runtime results are valid only after required selector validation succeeds.
- Dry-run, confirmation, paging, and mutation planning must not start from an invalid selector.
