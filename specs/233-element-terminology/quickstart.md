# Quickstart: Element Terminology Standardization

## Prerequisites

- Work on branch `233-element-terminology`.
- Apply `--implementation-context specs/ralph-implementation-rules.md` to every Ralph implementation iteration.
- Do not edit generated Camunda clients by hand.

## Verification Scenarios

### Incident Filters

```sh
./c8volt get incident --element-id task-a --limit 5
./c8volt get incident --element-instance-key 2251799813685300 --limit 5
./c8volt get incident --flow-node-id task-a
./c8volt get incident --fni-key 2251799813685300
```

Expected results:
- The first two commands parse and filter incidents.
- The last two commands fail locally as unknown flags.

### Incident Output

```sh
./c8volt get incident --element-id task-a --json
./c8volt get incident --element-id task-a
```

Expected results:
- JSON contains `elementId` and `elementInstanceKey`.
- JSON does not contain `flowNodeId` or `flowNodeInstanceKey`.
- Human rows use `e:` and `ei:`.
- Human rows do not use `fn:` or `fni:`.

### Process Context

```sh
./c8volt get pi --with-incidents --json <process-instance-key>
./c8volt walk pi --with-incidents <process-instance-key>
```

Expected results:
- Process-instance JSON uses `parentElementInstanceKey`.
- Process-instance JSON does not use `parentFlowNodeInstanceKey`.
- Incident-enriched output uses canonical element terminology.

### Ops Workflows

```sh
./c8volt ops repair incident --element-id task-a --dry-run
./c8volt ops purge process-instances-with-incidents --element-instance-key 2251799813685300 --dry-run
```

Expected results:
- Ops workflows accept canonical incident filters.
- Help, dry-run output, confirmation text, JSON, and report content do not expose legacy public names.

## Validation Commands

Run targeted checks as each story completes, then broader checks before commit readiness:

```sh
go test ./cmd -run 'Incident|ProcessInstance|Walk|OpsRepair|OpsPurge|CommandContract' -count=1
go test ./c8volt/incident ./c8volt/process ./c8volt/ops ./c8volt/resource -count=1
go test ./internal/services/incident/... ./internal/services/processinstance/... -count=1
make docs-content
make test
```
