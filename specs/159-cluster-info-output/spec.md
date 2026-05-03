# Feature Specification: Improve Cluster Info Output And Version Command

**Feature Branch**: `159-cluster-info-output`  
**Created**: 2026-05-03  
**Status**: Draft  
**Input**: User description: "https://github.com/grafvonb/c8volt/issues/159"

## GitHub Issue Traceability

- **Issue Number**: 159
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/159
- **Issue Title**: feat(cluster): improve cluster info output and add version command

## Clarifications

### Session 2026-05-03

- Q: What should happen to the legacy `c8volt get cluster-topology` command while `get cluster topology` gets new human-readable output? -> A: Remove the legacy `c8volt get cluster-topology` command path.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read Cluster Topology As A Tree (Priority: P1)

As a Camunda operator inspecting a cluster interactively, I want `c8volt get cluster topology` to show brokers and partitions as a compact tree by default so that I can understand leader and replica placement without parsing JSON.

**Why this priority**: This is the largest operator-facing readability improvement and changes the default output of an existing command.

**Independent Test**: Run `c8volt get cluster topology` against fixture data with multiple brokers and partitions, then verify the output starts with one cluster summary line and renders sorted broker and partition rows in the same visual tree style as `walk pi`.

**Acceptance Scenarios**:

1. **Given** the connected cluster returns gateway, broker, and partition topology data, **When** the operator runs `c8volt get cluster topology`, **Then** the command completes successfully and prints a concise tree with a cluster summary line, broker rows, and partition rows.
2. **Given** topology data contains brokers and partitions in any upstream order, **When** human-readable topology output is rendered, **Then** broker rows are ordered by `NodeId` and partition rows under each broker are ordered by `PartitionId`.
3. **Given** topology data contains leader and health information for partitions, **When** the tree is rendered, **Then** each partition row includes its partition id, role, and health state in a scannable single line.

---

### User Story 2 - Preserve Machine-Readable Cluster Responses (Priority: P2)

As an automation author, I want `--json` on cluster topology and license commands to preserve the established structured result envelope so that scripts do not break when human defaults become more readable.

**Why this priority**: The feature intentionally changes default output, so preserving the machine-readable contract is necessary for compatibility.

**Independent Test**: Run `c8volt get cluster topology --json` and `c8volt get cluster license --json` against fixture data, then verify each command returns the existing structured JSON envelope instead of the new human-readable text.

**Acceptance Scenarios**:

1. **Given** topology data is available, **When** the operator runs `c8volt get cluster topology --json`, **Then** the command returns the structured topology response using the established result envelope format.
2. **Given** license data is available, **When** the operator runs `c8volt get cluster license --json`, **Then** the command returns the structured license response using the established result envelope format.
3. **Given** a command fails while JSON output is requested, **When** the failure is reported, **Then** the command preserves the CLI's established error reporting and exit semantics.

---

### User Story 3 - Check Gateway And Broker Versions (Priority: P3)

As a Camunda operator, I want `c8volt get cluster version` to show the gateway version by default and optionally list broker versions so that I can quickly confirm whether the cluster is running the expected version.

**Why this priority**: Version lookup is a new command, but it can be delivered by reusing topology data and should not require operators to inspect the full topology tree.

**Independent Test**: Run `c8volt get cluster version` and `c8volt get cluster version --with-brokers` against topology fixture data, then verify the default output is only the gateway version and the broker-enabled output contains the gateway version plus one sorted line per broker.

**Acceptance Scenarios**:

1. **Given** topology data includes a gateway version, **When** the operator runs `c8volt get cluster version`, **Then** the command prints only that gateway version on a single line.
2. **Given** topology data includes broker versions and hosts, **When** the operator runs `c8volt get cluster version --with-brokers`, **Then** the command prints `GatewayVersion: <version>`, a blank separator line, `Brokers:`, and one broker version line per broker.
3. **Given** broker data is returned in any upstream order, **When** broker versions are rendered, **Then** broker lines are ordered by `NodeId` and follow `Broker <NodeId>: <Version> (<Host>)`.

---

### User Story 4 - Read Cluster License As Flat Information (Priority: P4)

As a Camunda operator checking license status interactively, I want `c8volt get cluster license` to show one field per line by default so that validity, type, and expiration information are readable without JSON formatting.

**Why this priority**: License output is already available, but the issue asks for a flatter human default to match operator workflows.

**Independent Test**: Run `c8volt get cluster license` against license fixture data and verify the output uses domain field names with one field per line, omits unavailable optional values cleanly, and keeps `--json` as the structured path.

**Acceptance Scenarios**:

1. **Given** license data includes validity and license type, **When** the operator runs `c8volt get cluster license`, **Then** the command prints concise human-readable fields such as validity and license type, one field per line.
2. **Given** license data includes optional attributes such as expiration or commercial-use status, **When** flat output is rendered, **Then** available optional attributes appear with field names matching the existing license domain model.
3. **Given** optional license attributes are absent, **When** flat output is rendered, **Then** the command does not invent placeholder values or fail solely because optional fields are missing.

### Edge Cases

- Topology output must remain stable when brokers or partitions are returned unsorted by the upstream API.
- Topology tree rendering must handle clusters with zero brokers or brokers with no listed partitions without producing malformed tree connectors.
- Missing optional topology fields must not create misleading values in human-readable output; available fields should still be shown.
- `get cluster version` must fail with the established cluster-command error behavior if topology retrieval fails.
- `get cluster version --with-brokers` must handle brokers with missing optional host or version information without corrupting surrounding output.
- The command spelling `license` remains canonical, and `licence` should work as an alias if it is not already present.
- JSON output must not be mixed with human tree or flat text when `--json` is present.
- Documentation and help must remove the legacy `get cluster-topology` path and direct users to `get cluster topology`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST render `c8volt get cluster topology` as human-readable tree output by default.
- **FR-002**: Topology tree output MUST include a concise cluster summary line with gateway version, broker count, partition count, replication factor, and last-change information when those values are available.
- **FR-003**: Topology tree output MUST render broker rows with node id, host, port when available, and broker version when available.
- **FR-004**: Topology tree output MUST render partition rows under each broker with partition id, role, and health state when available.
- **FR-005**: Human-readable topology output MUST sort brokers by `NodeId` and partitions by `PartitionId`.
- **FR-006**: Topology tree output MUST follow the existing `walk pi` visual tree style for branch connectors, nesting, and row readability.
- **FR-007**: `c8volt get cluster topology --json` MUST return the structured topology response using the established result envelope format.
- **FR-008**: The system MUST expose `c8volt get cluster version` under the existing `get cluster` hierarchy.
- **FR-009**: `c8volt get cluster version` MUST reuse the same cluster topology retrieval capability as `c8volt get cluster topology`.
- **FR-010**: `c8volt get cluster version` MUST print only the gateway version by default.
- **FR-011**: `c8volt get cluster version --with-brokers` MUST print the gateway version plus one broker version line per broker.
- **FR-012**: Broker version lines MUST follow `Broker <NodeId>: <Version> (<Host>)` and MUST be sorted by `NodeId`.
- **FR-013**: The system MUST render `c8volt get cluster license` as concise flat human-readable information by default.
- **FR-014**: Flat license output MUST use field names that match the existing license domain model and print one field per line.
- **FR-015**: `c8volt get cluster license --json` MUST return the structured license response using the established result envelope format.
- **FR-016**: The system MUST add `licence` as an alias for `c8volt get cluster license` if that alias is not already present.
- **FR-017**: Human-readable output changes MUST preserve established success, failure, logging, and exit behavior for cluster read commands.
- **FR-018**: Command help, capability metadata, README examples, and generated CLI documentation MUST be updated wherever the new version command, new human-readable defaults, JSON behavior, or `licence` alias are user-visible.
- **FR-019**: Automated tests MUST cover topology tree output, topology JSON output, version-only output, version output with brokers, license flat output, license JSON output, and the `licence` alias.
- **FR-020**: The system MUST remove the legacy `c8volt get cluster-topology` command path, including its help/discovery surface and generated documentation.
- **FR-021**: The implementation MUST remain bounded to cluster command output, version command wiring, legacy topology command removal, documentation, and tests without redesigning the underlying cluster service or versioned API clients.

### Key Entities *(include if feature involves data)*

- **Cluster Topology Result**: The connected cluster metadata used for topology and version output, including gateway version, brokers, partitions, replication factor, and last-change information when available.
- **Broker Topology Row**: A human-readable broker representation including node id, host, port when available, and broker version when available.
- **Partition Topology Row**: A human-readable partition representation including partition id, role, and health state.
- **Cluster Version Command**: The user-visible command that derives gateway and optional broker version output from topology data.
- **Cluster License Result**: The connected cluster license metadata, including validity, license type, expiration, commercial-use status, and any optional fields available for the configured Camunda version.
- **Machine-Readable Result Envelope**: The established JSON output shape used by cluster read commands when `--json` is requested.

## Assumptions

- The existing cluster service already returns enough topology data to support gateway-only and broker-inclusive version output without a new upstream API call.
- The existing topology domain model contains the fields needed to render the requested tree or can expose already-returned data without changing the upstream request contract.
- The existing license domain model is the source of truth for flat license field names.
- Human-readable output is the default for interactive command usage; `--json` remains the explicit machine-readable path.
- The legacy `c8volt get cluster-topology` command path can be removed in this feature; users should move to `c8volt get cluster topology`.
- Generated CLI documentation under `docs/cli/` will be refreshed from command metadata rather than hand-edited when command help changes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated command tests show `c8volt get cluster topology` renders a sorted tree with cluster, broker, and partition rows in the same visual family as `walk pi`.
- **SC-002**: Automated command tests show `c8volt get cluster topology --json` preserves the structured topology envelope.
- **SC-003**: Automated command tests show `c8volt get cluster version` prints exactly one gateway version line by default.
- **SC-004**: Automated command tests show `c8volt get cluster version --with-brokers` prints gateway information plus one sorted broker line per broker.
- **SC-005**: Automated command tests show `c8volt get cluster license` renders flat one-field-per-line human-readable output.
- **SC-006**: Automated command tests show `c8volt get cluster license --json` preserves the structured license envelope.
- **SC-007**: Automated command tests or help/discovery assertions show `licence` reaches the same behavior as `license`.
- **SC-008**: Automated help/discovery tests show `c8volt get cluster-topology` is no longer listed or accepted.
- **SC-009**: Users reviewing README examples or generated CLI docs can discover the version command, the new human-readable defaults, and the JSON option without reading source code.
