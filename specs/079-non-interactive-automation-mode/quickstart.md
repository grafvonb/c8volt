# Quickstart: Define Non-Interactive Automation Mode

## Intended Outcome

- `c8volt` exposes one dedicated root automation flag for AI agents, scripts, and CI.
- Supported commands can run unattended without hanging on prompts or continuation requests.
- Unsupported commands reject automation mode explicitly instead of falling back to interactive behavior.
- JSON output remains the machine-readable execution surface, and stdout stays clean when automation callers request it.
- `--no-wait` continues to mean accepted-but-not-yet-confirmed work rather than becoming the default automation behavior.

## Implementation Starting Points

1. Start at the root flag and config seams:
   - [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go)
   - [`cmd/cmd_cli.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_cli.go)
2. Extend command metadata and discovery:
   - [`cmd/command_contract.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/command_contract.go)
   - [`cmd/capabilities.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/capabilities.go)
3. Wire representative command behavior through the existing prompt and result seams:
   - `get`: [`cmd/get_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processinstance.go)
   - `run`: [`cmd/run_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/run_processinstance.go)
   - `deploy`: [`cmd/deploy_processdefinition.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/deploy_processdefinition.go)
   - `delete`: [`cmd/delete_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_processinstance.go)
   - `cancel`: [`cmd/cancel_processinstance.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_processinstance.go)
4. Keep shared JSON behavior inside the current render helpers:
   - [`cmd/cmd_views_rendermode.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_rendermode.go)
   - [`cmd/cmd_views_contract.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cmd_views_contract.go)
5. Update user-facing guidance once command behavior is stable:
   - [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md)
   - [`docs/use-cases.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/use-cases.md)
   - generated docs under [`docs/cli/`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/docs/cli)

## Verification Focus

1. The root automation flag is discoverable and documented as the canonical non-interactive entry point.
2. Supported prompting commands no longer block when automation mode is active.
3. Unsupported commands reject automation mode explicitly and actionably.
4. JSON automation runs keep stdout machine-safe.
5. Automation mode plus `--no-wait` returns `accepted` instead of implying confirmed completion.
6. Human-mode behavior remains unchanged when the automation flag is absent.

## Delivered Command Boundary

- Supported automation paths: `capabilities`, `get process-instance`, `run process-instance`, `deploy process-definition`, `delete process-instance`, and `cancel process-instance`.
- Explicit unsupported automation paths: `expect process-instance` and `walk process-instance`.
- Canonical machine-readable invocation pattern: `--automation --json`, optionally combined with `--no-wait` when accepted-but-not-yet-complete work is desired.

## Suggested Test Order

```bash
go test ./cmd -count=1
make docs
make docs-content
make test
```

Run the focused `cmd` suite first so prompt behavior, result envelopes, and unsupported-command rejection can be debugged before the full repository test gate.

## Manual Smoke Ideas

### Discovery

```bash
./c8volt capabilities --json
```

Check that the output shows the dedicated automation flag and reports which commands support automation mode.

### Supported read flow

```bash
./c8volt --config /tmp/c8volt.yaml --automation --json get process-instance --state active --count 250
```

Check that the command:

- does not block on paging continuation
- returns one machine-readable result on stdout
- does not mix progress chatter into stdout

### Supported write flow

```bash
./c8volt --config /tmp/c8volt.yaml --automation --json delete process-instance --state completed --count 50
```

Check that the command:

- does not require a separate `--auto-confirm`
- either proceeds under the documented automation contract or fails explicitly if the path is not yet supported
- keeps machine-readable output deterministic

### Accepted work

```bash
./c8volt --config /tmp/c8volt.yaml --automation --json run process-instance --bpmn-process-id order-process --no-wait
```

Check that the result:

- uses the shared result envelope
- reports `accepted`
- does not imply confirmed completion

### Unsupported automation invocation

```bash
./c8volt --config /tmp/c8volt.yaml --automation some-unsupported-command
```

Check that the command:

- fails immediately and explicitly
- does not drop into interactive behavior
- returns a machine-readable failure if JSON mode is also requested
