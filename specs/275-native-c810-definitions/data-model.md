# Data Model: Native Camunda 8.10 Embedded Process Definitions

This feature adds static embedded resources and one version-selection mapping. It introduces no persistent runtime data model.

## C810 Embedded Definition

| Field | Value or rule |
|---|---|
| compatibility line | `8.10` |
| filename prefix | `C810_` |
| process ID prefix | `C810_` |
| process name prefix | `C810 ` |
| execution platform | `Camunda Cloud` |
| execution platform version | `8.10.0` |
| process version tag | `v2.0.0` |
| behavioral source | same-suffix `C89_` definition |
| allowed source differences | version identity and platform version only |

### Required inventory

| Role | C810 filename and process ID | C89 source |
|---|---|---|
| Double User Task | `C810_DoubleUserTask` | `C89_DoubleUserTask` |
| Multiple Sub-Processes Parent | `C810_MultipleSubProcessesParent` | `C89_MultipleSubProcessesParent` |
| No-Op Completion | `C810_NoOpCompletion` | `C89_NoOpCompletion` |
| Simple Parent | `C810_SimpleParent` | `C89_SimpleParent` |
| Parent With Incident Subprocess | `C810_SimpleParentWithIncidentSubprocess` | `C89_SimpleParentWithIncidentSubprocess` |
| Simple Service Task | `C810_SimpleServiceTask` | `C89_SimpleServiceTask` |
| Simple User Task | `C810_SimpleUserTask` | `C89_SimpleUserTask` |
| User Task With Incident | `C810_SimpleUserTaskWithIncident` | `C89_SimpleUserTaskWithIncident` |

Each filename is the listed identity plus `.bpmn`.

### Validation rules

- The inventory contains exactly eight C810 files.
- Filename, process ID, process name, and owning BPMN plane use the C810 identity.
- Every called process uses a C810 identity and resolves within the inventory.
- No C89 reference remains in a C810 file.
- All non-version workflow and diagram content equals the corresponding C89 source.

## Production Fixture Mapping

| Camunda version | Production prefix |
|---|---|
| 8.7 | `C87_` |
| 8.8 | `C88_` |
| 8.9 | `C89_` |
| 8.10 | `C810_` |
| unknown | no mapping; selection fails |

### Consumers

- Embedded list filtering
- Embedded `--all` export and deployment
- Smoke-test parent selection
- Smoke-test dependency-closure selection
- Fixture identity in human, JSON, and report output

## Relationships and lifecycle

- Every C810 definition has exactly one same-suffix C89 behavioral source.
- `C810_MultipleSubProcessesParent` calls `C810_SimpleParent` and `C810_SimpleUserTask`.
- `C810_SimpleParent` calls `C810_SimpleUserTask`.
- `C810_SimpleParentWithIncidentSubprocess` calls `C810_SimpleUserTaskWithIncident`.
- Definitions are immutable embedded assets after build; they have no runtime state transition in c8volt.
