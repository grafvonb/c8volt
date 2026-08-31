# Quickstart: Validate Stable Tenant-Aware Process-Definition Ordering

## Prerequisites

- Work from branch `286-tenant-pd-ordering`.
- Read [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/process-definition-ordering.md](contracts/process-definition-ordering.md), and `specs/ralph-implementation-rules.md` before implementation.
- Use Go 1.26 with the repository toolchain and run `gofmt` on touched Go files.

## Focused Automated Validation

Run the closest tests first. Test names may be refined when tasks introduce the exact fixtures, but coverage must retain these boundaries.

```bash
go test ./internal/domain -run 'Test.*ProcessDefinition.*(Sort|Order|Latest)' -count=1
go test ./internal/services/processdefinition -run 'Test.*(SearchProcessDefinitionsPages|Latest|WatchSnapshot|Order)' -count=1
go test ./internal/services/processdefinition/v87 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort)' -count=1
go test ./internal/services/processdefinition/v88 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort|Stat)' -count=1
go test ./internal/services/processdefinition/v89 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort|Stat)' -count=1
go test ./internal/services/processdefinition/v810 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort|Stat)' -count=1
go test ./c8volt/process -run 'TestClient_(SearchProcessDefinitions|SearchProcessDefinitionsLatest|SearchProcessDefinitionsPages|CollectProcessDefinitionWatchSnapshot)' -count=1
go test ./cmd -run 'Test.*ProcessDefinition.*(Order|Latest|Paging|Stat|JSON|Keys|Watch|XML)|TestCommandContract' -count=1
```

The fixture set must include:

- `<default>` and named tenants whose exact text order is observable;
- tenant and BPMN IDs differing only by case;
- several BPMN processes and versions 9 and 10;
- equal tenant/process/version rows with text keys such as `10` and `2`;
- shuffled results split across pages of sizes 1, 2, and 1000;
- statistics changes that do not change row order;
- native latest behavior on 8.8-8.10 and local latest reduction on 8.7.

## Documentation Validation

Regenerate source-derived CLI documentation and inspect the contract wording:

```bash
make docs-content
rg -n "tenant|BPMN|version|order|latest" README.md cmd/get_processdefinition.go docs/cli
```

Confirm README, command help, and generated CLI pages state tenant ascending, BPMN process ID ascending, version descending, and key ascending without promising special `<default>` placement.

## Repository Validation

```bash
make vet
make test
git diff --check
```

`make test` is the required repository gate and runs the full suite with the race detector.

## Optional Cluster-Backed Smoke Test

Build outside the repository tree:

```bash
go build -o /tmp/c8volt-pd-ordering .
```

Against a controlled profile with multiple tenants, processes, and versions, capture and compare key sequences:

```bash
/tmp/c8volt-pd-ordering --profile PROFILE get process-definition --all-tenants --batch-size 1 --keys-only > /tmp/pd-order-1.txt
/tmp/c8volt-pd-ordering --profile PROFILE get process-definition --all-tenants --batch-size 2 --keys-only > /tmp/pd-order-2.txt
/tmp/c8volt-pd-ordering --profile PROFILE get process-definition --all-tenants --batch-size 1000 --keys-only > /tmp/pd-order-1000.txt
diff -u /tmp/pd-order-1.txt /tmp/pd-order-2.txt
diff -u /tmp/pd-order-1.txt /tmp/pd-order-1000.txt
```

Then verify:

1. `--latest` returns one newest definition for each exact tenant/BPMN pair.
2. Adding `--stat` on supported versions leaves the ordered keys unchanged.
3. Human and JSON arrays follow the same key sequence as keys-only output.
4. Watch rows do not move when only counts change.
5. Direct retrieval by key and XML retrieval retain their prior content and behavior.
6. Equivalent visible populations on 8.7, 8.8, 8.9, and 8.10 normalize to the same sequence, acknowledging the existing 8.7 1000-definition ceiling.
