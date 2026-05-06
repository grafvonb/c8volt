# Quickstart: Validate Process Definition Selectors

## Prerequisites

- A configured c8volt environment with at least one visible process definition.
- A known missing or misspelled BPMN process ID.
- Optional tenant/version/version-tag contexts for selector narrowing tests.

## Scenario 1: Missing BPMN Process ID Does Not Look Empty

```sh
./c8volt get pi --bpmn-process-id MissingProcess
```

Expected result:

- Command fails before treating process-instance search as a valid empty result.
- Human output explains that no visible process definition matched the selector.
- Output is not only `found: 0`.

## Scenario 2: Existing Process Definition With No Instances Still Looks Empty

```sh
./c8volt get pi --bpmn-process-id ExistingProcessWithNoInstances
```

Expected result:

- Process-definition validation succeeds.
- Existing process-instance search behavior is preserved.
- `found: 0` remains valid when no process instances match.

## Scenario 3: Selector Context Narrows Visibility

```sh
./c8volt get pi --bpmn-process-id ExistingProcess --pd-version 999
./c8volt get pi --bpmn-process-id ExistingProcess --pd-version-tag missing-tag
./c8volt --tenant missing-tenant get pi --bpmn-process-id ExistingProcess
```

Expected result:

- Each command validates the full selector context.
- A version, tag, or tenant mismatch fails as a selector validation error.

## Scenario 4: Automation Modes Do Not Prompt

```sh
./c8volt --json get pi --bpmn-process-id MissingProcess
./c8volt --automation get pi --bpmn-process-id MissingProcess
./c8volt get pi --bpmn-process-id MissingProcess --keys-only
```

Expected result:

- Each command fails clearly without asking whether to list visible process definitions.
- Structured output modes keep existing error conventions.

## Scenario 5: Mutating Commands Fail Before Mutation

```sh
./c8volt cancel pi --bpmn-process-id MissingProcess --state active --auto-confirm
./c8volt delete pi --bpmn-process-id MissingProcess --state completed --auto-confirm
```

Expected result:

- Each command fails before cancellation or deletion planning submits mutation work.

## Scenario 6: Multi-ID Run Is All-Or-Nothing

```sh
./c8volt run pi --bpmn-process-id ExistingProcess --bpmn-process-id MissingProcess
```

Expected result:

- Validation reports the missing BPMN process ID.
- No process instances are created for any ID in the request.

## Suggested Validation Commands

```sh
GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'Test.*ProcessDefinitionSelector|Test.*Bpmn.*Selector|TestRunProcessInstance.*Bpmn' -count=1
GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processdefinition/v87 ./internal/services/processdefinition/v88 ./internal/services/processdefinition/v89 -count=1
make docs-content
make test
```
