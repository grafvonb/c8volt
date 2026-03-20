# Quickstart: Add Process Definition XML Command

## Goal

Verify that `c8volt get process-definition` can return raw XML for one process definition and that the output is safe for normal shell redirection.

## Prerequisites

- A valid temp config file for tests or local execution
- A reachable Camunda environment with at least one deployed process definition
- A known process definition key

## Command Checks

1. Run help and confirm the XML flag is documented:

```bash
go test ./cmd -run 'TestGet.*ProcessDefinition.*Help' -count=1
```

2. Run the focused command tests for XML behavior:

```bash
go test ./cmd -run 'TestGetProcessDefinition.*XML' -count=1
```

3. Run any targeted process facade or service regression tests touched by the change:

```bash
go test ./c8volt/process ./internal/services/processdefinition/... -count=1
```

4. Regenerate CLI docs after help text changes:

```bash
make docs
```

5. Run the full repository validation required before commit:

```bash
make test
```

## Manual Smoke Check

With a valid config and process definition key:

```bash
./bin/c8volt --config /path/to/config.yaml get process-definition --key <key> --xml > /tmp/example.bpmn
```

Expected result:

- `/tmp/example.bpmn` contains the process definition XML
- the file does not include list summaries or JSON wrappers
- command failures still report an error and do not appear as a successful export
