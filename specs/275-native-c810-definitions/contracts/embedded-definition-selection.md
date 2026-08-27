# Contract: Embedded Definition Selection

## Supported version mapping

| Configured Camunda version | Selected production family |
|---|---|
| 8.7 | `C87_` |
| 8.8 | `C88_` |
| 8.9 | `C89_` |
| 8.10 | `C810_` |

An unknown version has no production family and fails explicitly. It does not fall forward to C810 or backward to C89.

## Camunda 8.10 family

The V810 selection contains exactly:

- `processdefinitions/C810_DoubleUserTask.bpmn`
- `processdefinitions/C810_MultipleSubProcessesParent.bpmn`
- `processdefinitions/C810_NoOpCompletion.bpmn`
- `processdefinitions/C810_SimpleParent.bpmn`
- `processdefinitions/C810_SimpleParentWithIncidentSubprocess.bpmn`
- `processdefinitions/C810_SimpleServiceTask.bpmn`
- `processdefinitions/C810_SimpleUserTask.bpmn`
- `processdefinitions/C810_SimpleUserTaskWithIncident.bpmn`

## Command behavior

- `c8volt --camunda-version 8.10 embed list` returns only the C810 family.
- `c8volt --camunda-version 8.10 embed export --all` exports only the C810 family.
- `c8volt --camunda-version 8.10 embed deploy --all` deploys only the C810 family.
- Explicit `--file` selection remains unchanged and may select any existing embedded resource.
- Existing command names, flags, exit behavior, human layout, and machine-readable schemas remain unchanged.

## Smoke-test behavior

- Camunda 8.10 selects `embedded/processdefinitions/C810_MultipleSubProcessesParent.bpmn` with process ID `C810_MultipleSubProcessesParent`.
- Its deployment closure contains `C810_SimpleUserTask`, `C810_SimpleParent`, and `C810_MultipleSubProcessesParent`.
- Human output, JSON payloads, and audit reports expose the C810 fixture identity while continuing to report the configured compatibility line as `8.10`.
- Mutation, confirmation, traversal, cleanup, and failure semantics do not change.

## Compatibility guarantees

- C87, C88, and C89 assets and selection remain unchanged.
- C810 workflow behavior equals C89 after the permitted version-specific fields are normalized.
- No live 8.10 integration profile or target is part of this contract.
