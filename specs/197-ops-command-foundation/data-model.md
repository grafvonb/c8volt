# Data Model: Ops Command Foundation

This feature does not introduce persistent storage entities. The model below describes command and report contract concepts that future ops workflows can reuse.

## Ops Command Family

Represents the top-level `c8volt ops` grouping command.

- **Command path**: `ops`
- **Purpose**: High-level operational workflows
- **Behavior**: Grouping/help only for this feature
- **Mutation classification**: Operational/state-changing command family
- **Runtime config requirement**: None for help and discovery

## Ops Execute Group

Represents `c8volt ops execute`.

- **Command path**: `ops execute`
- **Purpose**: Future predefined playbooks that discover a target set and execute existing resource actions
- **Behavior**: Grouping/help only for this feature
- **Concrete workflows in scope**: None

## Ops Repair Group

Represents `c8volt ops repair`.

- **Command path**: `ops repair`
- **Purpose**: Future repair/remediation workflows
- **Behavior**: Grouping/help only for this feature
- **Forbidden top-level input**: Ambiguous `--key`
- **Target semantics**: Defined later by target-specific repair commands

## Ops Workflow Contract

Shared conventions for future concrete ops workflows.

- **Mutation metadata**: Concrete state-changing workflows should be marked state-changing
- **Automation compatibility**: Concrete workflows must declare whether unattended operation is supported
- **Dry-run behavior**: Concrete workflows should preview target selection and planned actions without mutation
- **Report output**: Concrete workflows should build a structured report before rendering Markdown or JSON
- **Report format inference**: Future report file behavior should infer format from file extension where supported
- **Automation output**: `--automation --json` paths must keep deterministic JSON on stdout and progress/log output off stdout

## Workflow Step Status

Constrained status values for future workflow reports.

- `planned`: step has been selected for execution but not submitted
- `skipped`: step intentionally did not run
- `submitted`: mutation or operation was submitted
- `confirmed`: claimed outcome was verified
- `confirmation_failed`: submitted operation could not be confirmed
- `blocked`: step could not proceed due to a precondition or dependency
- `failed`: step execution failed

## Validation Rules

- Grouping commands must be independently testable through help output.
- Shared contracts must not call resource services or generated clients.
- Future resource-specific behavior must live below the ops facade boundary described in the spec.
