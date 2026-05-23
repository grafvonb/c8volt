# Quickstart: BPMN Selector Validation for Operational Commands

## Prerequisites

- A configured c8volt environment.
- One visible BPMN process definition.
- One missing or intentionally misspelled BPMN process ID.
- Optional tenant, version, and version-tag contexts for selector narrowing tests.

## Scenario 1: Mutating PI Commands Reject Missing BPMN Selectors

```sh
./c8volt cancel pi --bpmn-process-id MissingProcess --state active --dry-run
./c8volt delete pi --bpmn-process-id MissingProcess --state terminated --dry-run
```

Expected result:

- Each command fails before process-instance search paging or mutation planning.
- Output includes the shared missing visible process-definition diagnostic.
- No cancellation or deletion confirmation is shown.

## Scenario 2: Valid Selectors With No Runtime Resources Stay Empty

```sh
./c8volt cancel pi --bpmn-process-id ExistingProcessWithNoActiveInstances --state active --dry-run
./c8volt delete pi --bpmn-process-id ExistingProcessWithNoTerminatedInstances --state terminated --dry-run
```

Expected result:

- Process-definition validation succeeds.
- Existing valid empty-result behavior is preserved.
- No mutation request is submitted because no process instances match.

## Scenario 3: Incident Search Rejects Missing BPMN Selectors

```sh
./c8volt get incident --bpmn-process-id MissingProcess --state active
./c8volt get incident --bpmn-process-id MissingProcess --pi-keys-only
```

Expected result:

- Each command fails before incident search paging.
- Machine-friendly modes do not prompt for recovery output.
- A visible definition with no matching incidents still produces the existing empty incident result.

## Scenario 4: Direct Process-Definition Commands Have Explicit Missing Behavior

```sh
./c8volt get pd --bpmn-process-id MissingProcess
./c8volt delete pd --bpmn-process-id MissingProcess --latest --auto-confirm
```

Expected result:

- `get pd` has tested and documented missing-selector behavior.
- `delete pd` fails before impact planning, confirmation, or deletion when no process definition matches.

## Scenario 5: Version, Tag, and Tenant Context Narrow Validation

```sh
./c8volt cancel pi --bpmn-process-id ExistingProcess --pd-version 999 --dry-run
./c8volt delete pi --bpmn-process-id ExistingProcess --pd-version-tag missing-tag --dry-run
./c8volt --tenant missing-tenant get incident --bpmn-process-id ExistingProcess
```

Expected result:

- Each command validates the full selector context.
- A version, tag, or tenant mismatch fails as a selector validation error.

## Scenario 6: Pipeline Boundary Remains Upstream

```sh
./c8volt get pi --bpmn-process-id MissingProcess --keys-only | ./c8volt cancel pi --dry-run -
```

Expected result:

- `get pi` reports the missing BPMN selector.
- `cancel pi -` and `delete pi -` continue to operate only on key input and do not invent BPMN selector validation.

## Suggested Validation Commands

```sh
GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*ProcessDefinitionSelector|Test.*Bpmn.*Selector|Test(GetIncident|Cancel|Delete).*Bpmn|Test(Get|Delete)ProcessDefinition.*Bpmn|Test(Get|Cancel|Delete)ProcessInstance.*Pipeline|Test.*SelectorValidationHelpContract|TestCommandCapabilityForCommand_BpmnSelectorAlignedCommandContracts' -count=1
GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./c8volt/incident ./internal/services/processdefinition ./internal/services/incident -count=1
make docs-content
make test
```
