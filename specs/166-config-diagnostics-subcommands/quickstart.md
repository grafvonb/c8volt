# Quickstart: Config Diagnostics Subcommands

## Prerequisites

- A built `c8volt` binary or `go run .` from the repository root.
- A valid `config.yaml` for a reachable Camunda cluster for connection-success verification.
- At least one invalid config fixture for validation failure verification.

## Verify Compatibility Commands

```bash
./c8volt --config ./config.yaml config show
./c8volt --config ./config.yaml config show --validate
./c8volt config show --template
```

Expected result: sanitized config output remains unchanged, validation behavior remains unchanged, and template output matches the previous compatibility command output.

## Verify Dedicated Validation

```bash
./c8volt --config ./config.yaml config validate
./c8volt --config ./invalid-config.yaml config validate
```

Expected result: the valid config exits `0`; the invalid config exits non-zero through the standard local precondition/error path.

## Verify Dedicated Template Rendering

```bash
./c8volt config template
```

Expected result: output matches `./c8volt config show --template`.

## Verify Connection Test

```bash
./c8volt --config ./config.yaml config test-connection
```

Expected result: the command logs the loaded config path at `INFO`, validates the config, retrieves cluster topology, logs connection success at `INFO`, prints human-readable topology output, and exits `0`.

## Verify No-File Config Source Logging

```bash
C8VOLT_APIS_CAMUNDA_API_BASE_URL=http://localhost:8080 \
C8VOLT_AUTH_MODE=none \
./c8volt config test-connection
```

Expected result: the command logs that no config file was loaded and configuration came from defaults/environment or other non-file sources.

## Verify Version Comparison

- Gateway `8.9.2` with configured `8.9`: no warning.
- Gateway `8.9.2` with configured `8.9.1`: no warning.
- Gateway `8.9.2` with configured `8.8`: warning logged; exit remains `0` if connection succeeded.

## Validation Commands

```bash
go test ./cmd -run 'TestConfig|TestGetClusterTopology' -count=1
make test
```
