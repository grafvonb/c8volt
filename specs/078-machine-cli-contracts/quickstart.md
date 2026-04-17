# Quickstart: Define Machine-Readable CLI Contracts

## Intended Outcome

- `c8volt` exposes one dedicated top-level discovery command for machine consumers.
- Supported commands return one shared machine-readable result envelope with command-specific payloads.
- The envelope uses only `succeeded`, `accepted`, `invalid`, and `failed`.
- Existing exit codes remain the primary process-level signal.
- Discovery shows whether each command has `full`, `limited`, or `unsupported` contract support.

## Implementation Starting Points

1. Start with the command tree and render helpers:
   - [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go)
   - [`cmd/cmd_views_rendermode.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_rendermode.go)
   - [`c8volt/ferrors/errors.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors.go)
2. Add the dedicated discovery command and define the capability-record shape from the real Cobra tree.
3. Wrap one representative command from each required family in the shared result envelope:
   - `get`: [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go)
   - `run`: [`cmd/run_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_processinstance.go)
   - `expect`: [`cmd/expect_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/expect_processinstance.go)
   - `walk`: [`cmd/walk_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_processinstance.go)
   - `deploy`: [`cmd/deploy_processdefinition.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_processdefinition.go)
   - `delete`: [`cmd/delete_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go)
   - `cancel`: [`cmd/cancel_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go)
4. Use existing public payload models under [`c8volt/process/model.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/process/model.go) and [`c8volt/resource/model.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/model.go) as the envelope payload.
5. Update README and regenerate docs after help text and command surfaces are final.

## Verification Focus

1. Discovery output lists nested command paths, flags, output modes, mutation type, and contract support status.
2. Representative `get` and `expect` commands produce `succeeded` envelopes on confirmed success.
3. Representative `run`, `deploy`, `delete`, or `cancel` flows produce `accepted` when `--no-wait` intentionally returns before confirmation.
4. Invalid invocations return `invalid` while preserving current process exit behavior.
5. Remote or infrastructure failures return `failed` while preserving current process exit behavior.
6. Unsupported or limited commands remain visible in discovery and are not misreported as full support.

## Final Contract Snapshot

- Discovery is exposed through `c8volt capabilities --json` and reports top-level plus nested command metadata from the live Cobra tree.
- Shared machine-readable execution results are now rolled out for the representative `get`, `run`, `expect`, `walk`, `deploy`, `delete`, and `cancel` command families.
- Human-oriented output modes such as plain text and `--keys-only` remain intact; the shared envelope is layered onto structured JSON mode instead of replacing operator-facing flows.

## Suggested Test Order

```bash
go test ./c8volt/ferrors -count=1
go test ./cmd -count=1
make docs
make docs-content
make test
```

Run the focused suites first so failures in outcome mapping or discovery metadata are isolated before the repository-wide test gate.

## Verification Baseline

- Verified on 2026-04-17 with `go test ./c8volt/ferrors -count=1`
- Verified on 2026-04-17 with `go test ./cmd -count=1`
- Verified on 2026-04-17 with `make docs`
- Verified on 2026-04-17 with `make docs-content`
- Verified on 2026-04-17 with `make test`

## Manual Smoke Ideas

### Discovery

```bash
./c8volt capabilities --json
```

Check that the output includes:

- nested command paths such as `get process-instance`
- output modes such as `json` and `keys-only`
- a mutation classification
- a contract support status

### Confirmed success

```bash
./c8volt --config /tmp/c8volt.yaml --json get resource --id resource-id-123
```

Check that the result:

- uses the shared envelope
- reports `succeeded`
- preserves the resource payload under `payload`

### Accepted work

```bash
./c8volt --config /tmp/c8volt.yaml --json run pi --bpmn-process-id order-process --no-wait
```

Check that the result:

- uses the shared envelope
- reports `accepted`
- still exits with the same process-level semantics the command already uses

### Invalid input

```bash
./c8volt --config /tmp/c8volt.yaml --json run pi
```

Check that the result:

- reports `invalid`
- explains what the caller must correct
- preserves the existing invalid-args exit behavior
