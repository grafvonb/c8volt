# CLI Contract: Config Diagnostics Subcommands

## Supported Commands

```bash
./c8volt config show
./c8volt config show --validate
./c8volt config show --template
./c8volt config validate
./c8volt config template
./c8volt config test-connection
```

## Compatibility Behavior

- `config show` continues to print the sanitized effective configuration.
- `config show --validate` remains supported as a compatibility shortcut and validates the effective configuration.
- `config show --template` remains supported as a compatibility shortcut and prints the blank configuration template.
- `--validate` and `--template` on `config show` remain mutually exclusive.
- Existing global flags such as `--config`, `--profile`, `--tenant`, `--log-level`, `--quiet`, `--debug`, and `--no-err-codes` keep their established behavior.

## `config validate`

```bash
./c8volt --config ./config.yaml config validate
```

- Loads the effective configuration through the existing config bootstrap flow.
- Uses the same validation behavior as `config show --validate`.
- Exits `0` when the effective configuration is valid.
- Returns non-zero through the standard error path when the effective configuration is invalid.

## `config template`

```bash
./c8volt config template
```

- Prints the same blank configuration template as `config show --template`.
- Exits `0` when template rendering succeeds.
- Returns non-zero through the standard error path when template rendering fails.

## `config test-connection`

```bash
./c8volt --config ./config.yaml config test-connection
```

- Loads the effective configuration through the existing config bootstrap flow.
- Logs the loaded config file path at `INFO` level without requiring `--debug`.
- If no config file was loaded, logs at `INFO` that configuration came from defaults, environment, or other non-file sources.
- Validates the effective configuration before attempting a remote connection.
- Stops before topology retrieval when validation fails.
- Tests the configured Camunda connection through the same topology capability used by `get cluster topology`.
- On success, logs an `INFO` connection success message, prints the same human-readable topology output as `get cluster topology`, and exits `0`.
- On remote failure, logs through the standard `ERROR` path and returns non-zero.

## Version Warning Behavior

- Gateway version and configured Camunda version are compared by major/minor only.
- `8.9` and `8.9.2` are treated as matching.
- `8.9.2` and `8.8` produce a `WARNING`.
- Version mismatch warnings do not change an otherwise successful exit code.

## Documentation Contract

- `config --help` lists `show`, `validate`, `template`, and `test-connection`.
- `config show --help` documents `--validate` and `--template` as supported compatibility shortcuts.
- README setup/troubleshooting examples include the dedicated validation, template, and connection-test commands.
- Generated CLI docs include pages for all new config subcommands.
