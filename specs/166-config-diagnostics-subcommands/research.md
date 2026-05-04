# Research: Config Diagnostics Subcommands

## Decision: Keep `config show` as the compatibility owner and extract shared helpers

**Rationale**: `cmd/config_show.go` currently owns sanitized display, validation via `--validate`, and template rendering via `--template`. Extracting helper functions for validation and template rendering lets `config show --validate`, `config show --template`, `config validate`, and `config template` use one behavior path without changing existing command output by accident.

**Alternatives considered**: Duplicating validation/template logic in new command files was rejected because it creates drift between the legacy flags and dedicated subcommands. Replacing the flags with aliases was rejected because the issue explicitly requires existing flags to continue working.

## Decision: Implement dedicated Cobra subcommands under the existing `configCmd`

**Rationale**: The repository's command tree is Cobra-based and keeps leaf command behavior in `cmd/`. New `validate`, `template`, and `test-connection` commands under `configCmd` match existing command discovery, generated documentation, completion, and help-test patterns.

**Alternatives considered**: Adding top-level commands was rejected because the workflows are configuration diagnostics and belong under `config`. Adding hidden aliases only was rejected because the issue requires discoverable subcommands.

## Decision: Reuse the existing cluster topology facade and renderer for `test-connection`

**Rationale**: `c8volt get cluster topology` already proves the configured Camunda connection, routes through the versioned cluster service factory, and renders human-readable topology output via `renderClusterTopologyTree`. Reusing that path keeps service selection, authentication, error normalization, and output consistent with the existing command.

**Alternatives considered**: Shelling out to `get cluster topology` was rejected by the issue. Calling generated cluster clients directly from config command code was rejected because it would bypass the repository facade and duplicate service setup/error behavior.

## Decision: Log config source at `INFO` inside `test-connection`

**Rationale**: Root bootstrap currently logs the loaded config file only at debug level. The new diagnostic command has an explicit user-visible requirement to report the loaded path at `INFO`, and to state clearly when no config file was loaded. The command should obtain the source from the already initialized runtime context or a small repository-native helper and log it before validation/connection reporting.

**Alternatives considered**: Raising all root config source logging to `INFO` was rejected because it would change logging behavior for every command. Printing the path to stdout was rejected because the issue requests logging and topology output should remain stdout content.

## Decision: Compare configured and gateway versions by major/minor only after successful topology retrieval

**Rationale**: The issue requires patch differences to be ignored and major/minor mismatches to warn without failing. Running the comparison after topology retrieval ensures warnings are based on the actual gateway version and do not turn connection failures into version-comparison failures.

**Alternatives considered**: Full semantic-version comparison was rejected because patch-only differences must not warn. String prefix comparison was rejected because it is too brittle for configured values like `8.9` and gateway values like `8.9.2`.

## Decision: Update help, README, and generated CLI docs in the same feature

**Rationale**: The command surface changes from one multi-purpose `show` command to a clearer command family. The constitution requires user-facing docs to match behavior, and existing README/docs already list setup commands and command trees.

**Alternatives considered**: Updating command help only was rejected because README and generated CLI docs are the main durable references for setup and troubleshooting.
