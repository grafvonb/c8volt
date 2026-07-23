# Quickstart: Runtime Listener Jobs Under Elements

## Prerequisites

- A c8volt configuration for a Camunda 8.8 or 8.9 environment with process instances that have runtime element instances and listener jobs.
- A known element instance key with at least one `EXECUTION_LISTENER` or `TASK_LISTENER` job.
- A known process-instance key with runtime elements and at least one element-owned listener job.
- For unsupported-version validation, a Camunda 8.7 test configuration.

## Targeted Validation

Run focused command tests while implementing:

```sh
go test ./cmd -run 'Test(GetElement|GetProcessInstance|WalkProcessInstance|OpsAnalyseSlowProcessInstances|CommandContract)' -count=1
```

Run focused facade and service tests while implementing:

```sh
go test ./c8volt/process ./c8volt/ops ./internal/services/processinstance ./internal/services/ops ./internal/services/job/... -count=1
```

Add or update targeted cases covering:

- Help and command capability metadata include `--with-listeners` on every in-scope command.
- `get element -k <element-instance-key> --with-listeners` renders listener jobs under the element.
- `get element --pi-key <process-instance-key> --with-listeners` attaches listeners only to matching elements.
- `get pi -k <process-instance-key> --with-elements --with-listeners` renders listeners under matching element rows.
- `walk pi -k <process-instance-key> --with-elements --with-listeners` preserves default, children, parent, and flat traversal structure.
- `ops analyse slow-process-instances -k <process-instance-key> --with-listeners` renders listener details under element timeline rows.
- JSON output includes `listeners` arrays only when listener enrichment is requested.
- Empty requested listeners produce empty arrays in JSON.
- Unmatched listener jobs are omitted from enriched output.
- `--with-listeners` without element context fails before remote listener lookup.
- `--keys-only --with-listeners` fails before remote listener lookup.
- Camunda 8.7 returns the existing unsupported job-search or listener-lookup error style.
- Existing output without `--with-listeners` remains unchanged.

## Validation Log

- 2026-07-23 11:41 Iteration 2: `go test ./internal/services/processinstance ./c8volt/process ./cmd -run 'Test.*Listener' -count=1` passed.
- 2026-07-23 11:41 Iteration 2: `go test ./c8volt/element ./c8volt/ops -count=1` passed.
- 2026-07-23 11:55 Iteration 3: `go test ./cmd ./c8volt/element -run 'Test(GetElement|CommandContract|GeneratedDocs).*Listener' -count=1` passed.
- 2026-07-23 11:55 Iteration 3: `go test ./internal/services/processinstance ./cmd -run 'Test.*Listener' -count=1` passed.
- 2026-07-23 11:55 Iteration 3: `go test ./cmd ./c8volt/element ./docsgen -count=1` passed.
- 2026-07-23 12:01 Iteration 4: `go test ./cmd -run 'TestGetProcessInstance.*Listener|TestCommandContract.*Listener|TestGenerated.*get pi' -count=1` passed.
- 2026-07-23 12:01 Iteration 4: `go test ./docsgen -run 'TestGeneratedGetProcessInstanceDocsDocumentVariableSearch' -count=1` passed.
- 2026-07-23 12:01 Iteration 4: `go test ./cmd ./docsgen -count=1` passed.
- 2026-07-23 12:09 Iteration 5: `go test ./cmd -run 'TestWalkProcessInstance.*Listener|TestCommandContract.*walk|TestGenerated.*walk' -count=1` passed.
- 2026-07-23 12:09 Iteration 5: `go test ./cmd ./docsgen -count=1` passed.
- 2026-07-23 12:18 Iteration 6: `go test ./cmd ./c8volt/ops ./internal/services/ops -run 'Test.*SlowProcess.*Listener|TestOps.*Listener|TestGenerated.*slow-process' -count=1` passed.
- 2026-07-23 12:18 Iteration 6: `go test ./cmd ./c8volt/ops ./internal/services/ops ./docsgen -count=1` passed.
- 2026-07-23 12:25 Iteration 7: `make docs-content` passed and regenerated listener command docs.
- 2026-07-23 12:25 Iteration 7: `gofmt -w cmd/*.go c8volt/*.go c8volt/*/*.go internal/domain/*.go internal/services/processinstance/*.go internal/services/element/*.go internal/services/ops/*.go` passed.
- 2026-07-23 12:25 Iteration 7: `go test ./cmd -run 'Test(GetElement|GetProcessInstance|WalkProcessInstance|OpsAnalyseSlowProcessInstances|CommandContract)' -count=1` passed.
- 2026-07-23 12:25 Iteration 7: `go test ./c8volt/process ./c8volt/ops ./internal/services/processinstance ./internal/services/ops ./internal/services/job/... -count=1` passed.
- 2026-07-23 12:25 Iteration 7: `go test ./c8volt/resource -count=1` passed after updating the test stub for the listener-aware process facade interface.
- 2026-07-23 12:25 Iteration 7: `make test` passed.

## Manual Smoke Scenarios

Inspect one runtime element with listeners:

```sh
./c8volt get element -k <element-instance-key> --with-listeners
```

Expected outcome:

- The element row is shown.
- Runtime listener jobs owned by the element appear beneath it.
- Elements with no listener jobs do not imply an error.

Inspect elements for a process instance:

```sh
./c8volt get element --pi-key <process-instance-key> --with-listeners
```

Expected outcome:

- Returned element rows match normal element search.
- Listener rows appear only under matching elements.

Inspect a process instance with elements and listeners:

```sh
./c8volt get pi -k <process-instance-key> --with-elements --with-listeners
```

Expected outcome:

- The process-instance row and element section match existing `--with-elements` behavior.
- Listener rows are nested inside the owning element row.

Inspect a process-instance family with elements and listeners:

```sh
./c8volt walk pi -k <process-instance-key> --with-elements --with-listeners
./c8volt walk pi -k <process-instance-key> --children --with-elements --with-listeners
./c8volt walk pi -k <process-instance-key> --parent --with-elements --with-listeners
./c8volt walk pi -k <process-instance-key> --flat --with-elements --with-listeners
```

Expected outcome:

- The same process-instance rows appear as the matching walk without listener enrichment.
- Listener rows stay inside the owning process instance's `elements:` block.
- Child process-instance rows remain process-tree rows.

Inspect slow-process analysis with listener context:

```sh
./c8volt ops analyse slow-process-instances -k <process-instance-key> --with-listeners
```

Expected outcome:

- Slow-analysis process-instance rows and element timeline rows remain readable.
- Listener rows appear under element rows, not under transition rows.

Inspect JSON output:

```sh
./c8volt --json get pi -k <process-instance-key> --with-elements --with-listeners
./c8volt --json walk pi -k <process-instance-key> --with-elements --with-listeners
./c8volt --json ops analyse slow-process-instances -k <process-instance-key> --with-listeners
```

Expected outcome:

- Element objects include `listeners` arrays when listener enrichment is requested.
- Existing structured fields remain stable.

Validate incompatible output modes:

```sh
./c8volt get pi -k <process-instance-key> --with-listeners
./c8volt --keys-only walk pi -k <process-instance-key> --with-elements --with-listeners
./c8volt --keys-only ops analyse slow-process-instances -k <process-instance-key> --with-listeners
```

Expected outcome:

- Each command fails with a clear validation error before remote listener lookup.

## Documentation Validation

After implementation updates command behavior or flags:

```sh
make docs-content
```

Confirm README and generated CLI docs describe `--with-listeners`, valid combinations, JSON shape, and unsupported-version behavior consistently.

## Full Validation

Before commit or merge:

```sh
make test
```
