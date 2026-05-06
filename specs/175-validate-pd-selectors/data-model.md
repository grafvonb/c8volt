# Data Model: Validate Process Definition Selectors

## Process Definition Selector

Represents the process-definition portion of a process-instance command request.

**Fields**

- `bpmnProcessIds`: One or more BPMN process IDs provided by the command.
- `processVersion`: Optional exact process-definition version.
- `processVersionTag`: Optional process-definition version tag.
- `tenantContext`: Effective tenant context from root flags or persisted configuration.
- `selectionMode`: Whether the command needs any visible matching definition or the latest visible matching definition.

**Validation Rules**

- `get pi`, `cancel pi`, and `delete pi` use at most one BPMN process ID from shared process-instance search flags.
- `run pi` may use multiple BPMN process IDs and must validate every ID before creating any process instances.
- `--pd-version` remains incompatible with multi-ID `run pi` per existing command rules.
- `processVersion` and `processVersionTag` narrow visibility validation when provided.

## Visible Process Definition Match

Represents a deployed process definition visible to the current credentials and matching the selector.

**Fields**

- `key`: Process-definition key.
- `bpmnProcessId`: BPMN process ID.
- `processVersion`: Process-definition version.
- `processVersionTag`: Optional version tag.
- `tenantId`: Tenant ID displayed by existing process-definition output.

**Relationships**

- A selector is valid when each requested BPMN process ID has at least one visible match under the command's selector context.
- For latest-mode validation, a selector is valid when the latest visible process definition for each requested BPMN process ID can be resolved.

## Selector Validation Result

Represents the preflight outcome before process-instance work begins.

**Fields**

- `requestedSelectors`: The BPMN process IDs and narrowing fields that were checked.
- `missingSelectors`: BPMN process IDs with no visible process-definition match.
- `matches`: Visible process definitions found for valid selectors.
- `interactiveListAllowed`: Whether the command may ask the user to list visible definitions.

**State Transitions**

- `unchecked` -> `valid`: All requested BPMN process IDs have visible matches.
- `unchecked` -> `invalid`: One or more requested BPMN process IDs have no visible match.
- `invalid` -> `listed`: Interactive user accepts visible-definition listing after the command has already decided to fail.

## Process-Instance Operation

Represents the command work protected by selector validation.

**Fields**

- `command`: `get pi`, `cancel pi`, `delete pi`, or `run pi`.
- `operationKind`: Search, mutation, or start.
- `wouldPrompt`: Whether the command may ask human confirmation prompts.
- `machineMode`: Whether JSON, automation, keys-only, or non-TTY execution forbids prompts.

**Validation Rules**

- Process-instance search/mutation/start work must not begin when selector validation is invalid.
- Mutating and start operations must not submit partial work after any selector validation failure.
