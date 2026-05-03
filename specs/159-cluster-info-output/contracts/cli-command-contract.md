# Contract: Cluster Info Output Commands

## Supported Commands

### `c8volt get cluster topology`

- Fetches cluster topology through the existing cluster topology retrieval path.
- Default output is human-readable tree text.
- The first line summarizes cluster gateway version, broker count, partition count, replication factor, and last change when available.
- Broker rows are sorted by `NodeId`.
- Partition rows are sorted by `PartitionId` under each broker.
- Partition rows include partition id, role, and health state when available.

### `c8volt get cluster topology --json`

- Fetches the same topology data.
- Prints the structured topology response using the established result envelope.
- Does not include human tree text.

### `c8volt get cluster version`

- Fetches cluster topology through the same retrieval path as `get cluster topology`.
- Prints only the gateway version and a trailing newline by default.
- Does not list brokers unless `--with-brokers` is present.

### `c8volt get cluster version --with-brokers`

- Fetches cluster topology through the same retrieval path as `get cluster topology`.
- Prints `GatewayVersion: <version>`, a blank separator line, `Brokers:`, and one broker line per broker.
- Broker lines follow `Broker <NodeId>: <Version> (<Host>)`.
- Broker lines are sorted by `NodeId`.

### `c8volt get cluster license`

- Fetches cluster license data through the existing cluster license retrieval path.
- Default output is flat human-readable text.
- Uses license domain field names.
- Prints one field per line.
- Omits absent optional fields without placeholder values.

### `c8volt get cluster license --json`

- Fetches the same license data.
- Prints the structured license response using the established result envelope.
- Does not include flat human text.

### `c8volt get cluster licence`

- Alias for `c8volt get cluster license`.
- Supports the same `--json` behavior as the canonical spelling.

## Removed Commands

### `c8volt get cluster-topology`

- Must no longer be registered, listed in help, advertised in capabilities, or documented in generated CLI references.
- Existing aliases for the direct command (`ct`, `cluster-info`, `ci`) must no longer resolve to topology retrieval.
- Users should use `c8volt get cluster topology`.

## Error Behavior

- Cluster topology, version, and license commands preserve existing error wrapping and exit behavior for unavailable clusters, malformed responses, authentication failures, and configuration failures.
- `--json` changes successful output shape only; it does not create a different failure semantics contract.

## Output Contract Notes

- Human-readable topology, version, and license output is written only on successful command execution.
- Topology and version human output must be deterministic for command tests by sorting brokers by `NodeId` and partitions by `PartitionId`.
- Optional topology and license fields should be omitted or rendered as a neutral missing marker only where the command-specific contract says so; commands must not invent replacement domain values.
- JSON output for topology and license remains the established result-envelope path and must not be mixed with human-readable tree, version, or flat license lines.
- Generated CLI reference pages are derived from Cobra command metadata; the removed direct `get cluster-topology` page should disappear by command removal plus docs regeneration, not by maintaining a hand-edited replacement page.
