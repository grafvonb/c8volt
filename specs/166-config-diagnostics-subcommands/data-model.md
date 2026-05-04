# Data Model: Config Diagnostics Subcommands

## Effective Configuration

- **Purpose**: Represents the resolved c8volt configuration used by config diagnostics and cluster connection checks.
- **Fields**:
  - `activeProfile`: optional selected profile.
  - `app.camundaVersion`: configured Camunda version used for version compatibility comparison.
  - `apis.camundaApi.baseUrl`: configured Camunda endpoint.
  - `auth`: configured authentication mode and credentials.
  - `warnings`: non-fatal configuration warnings emitted after normalization.
- **Validation**:
  - Must normalize through the existing configuration loading flow.
  - Must validate through the existing `config.Config` validation behavior.
  - Sensitive values must remain sanitized when displayed by `config show`.
- **Lifecycle**: Loaded during command bootstrap, attached to command context, then consumed by `config show`, `config validate`, `config template`, or `config test-connection`.

## Config Source Description

- **Purpose**: Describes where the effective configuration came from for `config test-connection` logging.
- **Fields**:
  - `loadedPath`: config file path when Viper loaded a file.
  - `sourceKind`: file-loaded or no-file-loaded.
  - `fallbackDescription`: text explaining defaults/environment/non-file sources when no config file was loaded.
- **Validation**:
  - If `loadedPath` is non-empty, log that path at `INFO`.
  - If `loadedPath` is empty, log a clear no-file-loaded message at `INFO`.
- **Lifecycle**: Captured during bootstrap or reconstructed from the active command context before `test-connection` runs its remote proof.

## Configuration Validation Result

- **Purpose**: Represents the observable outcome of validating the effective configuration.
- **States**:
  - `valid`: exits with code `0`.
  - `invalid`: exits non-zero through the standard local precondition/error path.
- **Validation**:
  - `config validate` and `config show --validate` must produce equivalent outcomes for equivalent inputs.
  - `config test-connection` must stop on `invalid` before remote connection.
- **Lifecycle**: Produced by shared validation helper used by legacy and dedicated command paths.

## Configuration Template

- **Purpose**: Represents the blank configuration document printed for users who need a starter config.
- **Fields**:
  - `yaml`: rendered blank template content.
- **Validation**:
  - `config template` and `config show --template` must produce equivalent output.
  - Rendering failures must return non-zero through the standard error path.
- **Lifecycle**: Produced by shared template helper without using the active effective configuration.

## Connection Test Result

- **Purpose**: Represents the `config test-connection` diagnostic result after local validation and remote topology retrieval.
- **States**:
  - `validation-failed`: local configuration invalid; remote topology not attempted.
  - `connection-failed`: topology retrieval failed; standard error path returns non-zero.
  - `connected-version-match`: topology retrieved and configured/gateway versions match by major/minor.
  - `connected-patch-difference`: topology retrieved and only patch versions differ; no warning.
  - `connected-version-warning`: topology retrieved and major/minor versions differ; warning logged while exit remains `0`.
- **Validation**:
  - Only connected states print topology output.
  - Version warnings must never turn a successful connection into a failing command.
- **Lifecycle**: Produced by `config test-connection` after calling the existing cluster topology facade and renderer.

## Gateway Version

- **Purpose**: The Camunda version reported by cluster topology.
- **Fields**:
  - `raw`: upstream gateway version string.
  - `majorMinor`: normalized major/minor tuple when parseable.
- **Validation**:
  - Patch values are ignored for mismatch decisions.
  - Missing or unparseable values should avoid false mismatch warnings and should be covered by tests if the existing topology model allows that case.
- **Lifecycle**: Read from topology response after successful connection.

## Configured Camunda Version

- **Purpose**: The configured Camunda version selected by the effective configuration.
- **Fields**:
  - `raw`: configured version string.
  - `majorMinor`: normalized major/minor tuple when parseable.
- **Validation**:
  - Compared to gateway version by major/minor only.
  - Major/minor mismatch produces a warning and preserves success.
- **Lifecycle**: Read from effective config during `config test-connection`.
