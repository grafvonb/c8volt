# CLI Command Contract: Cluster License Command

## Scope

This contract defines the expected public CLI behavior for the new cluster license retrieval command.

## Command

- **Command**: `c8volt get cluster license`
- **Purpose**: Retrieve the connected Camunda 8 cluster's license information.
- **Expected Behavior**:
  - Calls the existing cluster license retrieval capability already exposed by the internal cluster service.
  - Prints the successful license payload in the CLI's standard structured JSON format.
  - Preserves current failure output and exit semantics used by other `get` commands.
  - Inherits the same root and `get` command flags available to other cluster read commands.

## Help and Discovery Rules

- `c8volt get` help must expose `cluster` as a subcommand.
- `c8volt get cluster` help must expose `license` as a subcommand.
- `c8volt get cluster license --help` must describe cluster license retrieval in terms consistent with existing cluster read commands.
- Help and docs must not advertise a legacy or alternate direct `c8volt get cluster-license` command.

## Output Rules

- Successful command output must include the existing cluster license fields returned by the service layer.
- Optional fields may be absent when the connected Camunda version does not provide them.
- The command must not invent placeholder values for omitted optional fields.

## Failure Rules

- Upstream transport failures must produce the established CLI error output and exit behavior for `get` commands.
- Non-success HTTP responses must surface as command failures rather than partial success.
- Malformed or empty successful responses must fail clearly through the existing error-handling path.

## Documentation Rules

- README and `docs/index.md` examples must be updated if they enumerate cluster read commands or discovery paths affected by this addition.
- Generated CLI docs under `docs/cli/` must be regenerated from Cobra metadata in the same change that introduces the command.

## Testable Acceptance Signals

- Running `c8volt get cluster license` with a valid configuration yields a structured license payload.
- Running `c8volt get cluster license` against a failing endpoint yields the expected failure semantics and exit code.
- Help output makes the command discoverable under `get cluster`.
- Documentation reflects the new command path in the same change set.
