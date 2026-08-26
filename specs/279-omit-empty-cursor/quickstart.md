# Quickstart: Validate Empty-Cursor Omission

## Prerequisites

- Go 1.26 with the repository toolchain available.
- A disposable Camunda 8 Run 8.9.17 instance using its default H2/RDBMS secondary storage.
- A working profile named below as `<c89-run-profile>` in the operator's default local c8volt configuration.
- The profile declares Camunda compatibility line `8.9` and has valid endpoint/authentication settings.

Live commands deploy a process definition and create process instances. Do not target a production cluster.

## 1. Run Focused Automated Tests

From the repository root:

```bash
GOCACHE=/tmp/c8volt-gocache go test \
  ./internal/services/processdefinition/v88 \
  ./internal/services/processdefinition/v89 \
  ./internal/services/processdefinition/v810 \
  -count=1

GOCACHE=/tmp/c8volt-gocache go test \
  ./internal/services/processdefinition/... \
  -count=1
```

Expected outcome: all three adapters prove the initial limit-only page, exact non-empty cursor continuation, ordinary offset preservation, latest filters, and stable sort order described in [the request contract](contracts/latest-process-definition-search.md).

Run selector/facade regressions:

```bash
GOCACHE=/tmp/c8volt-gocache go test \
  ./c8volt/process ./cmd \
  -run 'ProcessDefinition|RunProcessInstance|Selector' \
  -count=1
```

Expected outcome: existing selector validation, no-partial-creation behavior, commands, and output contracts remain green.

## 2. Build and Check the External Profile

```bash
GOCACHE=/tmp/c8volt-gocache go build -o /tmp/c8volt-279 .

/tmp/c8volt-279 --profile <c89-run-profile> --json config test-connection
/tmp/c8volt-279 --profile <c89-run-profile> get cluster version
```

Expected outcome: connection succeeds and the reported Camunda release is 8.9.17. A 401 or invalid-client response is a profile/authentication failure that must be corrected before feature validation.

## 3. Reproduce the Corrected H2/RDBMS Workflow

Deploy one predictable embedded definition:

```bash
/tmp/c8volt-279 --profile <c89-run-profile> \
  embed deploy --file processdefinitions/C89_SimpleUserTask.bpmn
```

Start ten instances through latest-definition selector validation:

```bash
/tmp/c8volt-279 --profile <c89-run-profile> --keys-only \
  run process-instance \
  --bpmn-process-id C89_SimpleUserTask \
  --count 10 \
  --workers 4 | tee /tmp/c8volt-279-process-instance-keys.txt

wc -l /tmp/c8volt-279-process-instance-keys.txt
```

Expected outcome:

- The run command exits successfully.
- The key file contains exactly 10 process-instance keys.
- Selector validation does not return a 500 response from process-definition search.
- c8volt retains its established creation confirmation and keys-only behavior.

Keep the cluster disposable or clean up the created user-task instances using the repository's established integration cleanup workflow.

## 4. Optional Existing Integration Regressions

The existing integration targets mutate real state and require confirmation unless automation mode is enabled. They are broader than the defect proof but useful after the narrow H2 check:

```bash
C8VOLT_IT_PROFILES=<c89-run-profile> \
C8VOLT_IT_WORKDIR=/tmp/c8volt-279-it \
C8VOLT_IT_GO_TEST_FLAGS='-v -failfast' \
make integration-cli-deploy-embed-run

C8VOLT_IT_PROFILES=<c89-run-profile> \
C8VOLT_IT_VOLUME_COUNT=10 \
C8VOLT_IT_WORKDIR=/tmp/c8volt-279-volume \
C8VOLT_IT_GO_TEST_FLAGS='-v -failfast' \
make integration-cli-deploy-embed-run-volume
```

These targets cover related BPMN-ID workflows, but the explicit command in step 3 is the authoritative ten-instance acceptance check.

## 5. Run the Repository Gate

```bash
make test
git diff --check
git diff -- internal/clients/camunda README.md docs/cli
```

Expected outcome: the race-enabled test suite passes, no whitespace errors are reported, generated clients are unchanged, and no user-facing documentation diff appears.
