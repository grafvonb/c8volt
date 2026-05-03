# Data Model: Improve Cluster Info Output And Version Command

## Cluster Topology Result

- **Purpose**: Source data for topology tree output and cluster version output.
- **Fields**:
  - `GatewayVersion`: version string shown in the topology summary and default version command output.
  - `ClusterSize`: broker count or cluster size value shown in the topology summary when available.
  - `PartitionsCount`: total partition count shown in the topology summary when available.
  - `ReplicationFactor`: replication factor shown in the topology summary when available.
  - `LastCompletedChangeId`: last-change value shown in the topology summary, rendered as `-` when empty.
  - `Brokers`: broker rows sorted by node id for human output.
- **Validation rules**:
  - Human output must not reorder or mutate the returned domain data.
  - Missing optional values should render cleanly without invented domain facts.

## Broker Topology Row

- **Purpose**: Human-readable broker node inside the topology tree and optional version output.
- **Fields**:
  - `NodeId`: primary sort key and row identifier.
  - `Host`: host value shown in broker and version output when available.
  - `Port`: port value shown in topology output when available.
  - `Version`: broker version shown in broker and version output when available.
  - `Partitions`: partition rows sorted by partition id for topology tree output.
- **Validation rules**:
  - Broker rows are sorted by `NodeId`.
  - Version output follows `Broker <NodeId>: <Version> (<Host>)`.

## Partition Topology Row

- **Purpose**: Human-readable partition child row under each broker.
- **Fields**:
  - `PartitionId`: primary sort key and row identifier.
  - `Role`: leader/follower role shown when available.
  - `Health`: health state shown when available.
- **Validation rules**:
  - Partition rows are sorted by `PartitionId` under each broker.
  - Rows remain readable when role or health values are empty.

## Cluster License Result

- **Purpose**: Source data for flat license output and JSON license output.
- **Fields**:
  - `ValidLicense`: required validity status.
  - `LicenseType`: license type.
  - `ExpiresAt`: optional expiration timestamp.
  - `IsCommercial`: optional commercial-use marker.
- **Validation rules**:
  - Flat output uses domain field names, one field per line.
  - Optional fields appear only when present.
  - `--json` output preserves the existing structured envelope.

## Command Surface

- **Supported command paths**:
  - `c8volt get cluster topology`
  - `c8volt get cluster topology --json`
  - `c8volt get cluster version`
  - `c8volt get cluster version --with-brokers`
  - `c8volt get cluster license`
  - `c8volt get cluster license --json`
  - `c8volt get cluster licence`
- **Removed command paths**:
  - `c8volt get cluster-topology`
  - `c8volt get ct`
  - `c8volt get cluster-info`
  - `c8volt get ci`
