# CLI Command Contract: Cluster Topology Hierarchy

## Scope

This contract defines the expected public CLI behavior for cluster topology retrieval after the command-tree refactor.

## Preferred Command

- **Command**: `c8volt get cluster topology`
- **Purpose**: Retrieve the cluster topology of the connected Camunda 8 cluster.
- **Expected Behavior**:
  - Uses the same underlying topology retrieval logic as the legacy command path.
  - Prints the same successful topology payload format as the existing command.
  - Preserves existing failure output and process exit behavior.
  - Inherits the same root and `get` command flags available to other `get` subcommands.

## Legacy Compatibility Command

- **Command**: `c8volt get cluster-topology`
- **Purpose**: Preserve backward compatibility for existing users and scripts during the deprecation period.
- **Expected Behavior**:
  - Remains executable.
  - Produces the same successful topology payload format as the preferred command.
  - Preserves the same failure output and process exit behavior as the preferred command.
  - Does not print a runtime deprecation warning.
  - Is identified as deprecated in CLI help and generated documentation.

## Help and Documentation Rules

- `c8volt get` help must expose `cluster` as a subcommand.
- `c8volt get cluster` help must expose `topology` as a subcommand.
- CLI help and generated docs must identify `c8volt get cluster topology` as the preferred command path.
- CLI help and generated docs must identify `c8volt get cluster-topology` as deprecated but still supported.

## Testable Acceptance Signals

- Running either command path with the same valid configuration yields equivalent successful output.
- Running either command path with the same failing configuration yields equivalent failure semantics.
- Help output makes the nested hierarchy discoverable without requiring source inspection.
- Documentation reflects the preferred path and compatibility note in the same change set.
