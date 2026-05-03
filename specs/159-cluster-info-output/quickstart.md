# Quickstart: Improve Cluster Info Output And Version Command

## Targeted Validation

Run focused command tests while implementing:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetCluster(Topology|Version|License)|TestGetHelp|TestCapabilities' -count=1
```

Run the full repository validation before committing:

```bash
make test
```

## Manual Smoke Checks

Build or run the CLI against a configured environment and verify:

```bash
./c8volt get cluster topology
./c8volt get cluster topology --json
./c8volt get cluster version
./c8volt get cluster version --with-brokers
./c8volt get cluster license
./c8volt get cluster license --json
./c8volt get cluster licence
```

The removed direct command should no longer resolve:

```bash
./c8volt get cluster-topology
```

Expected result: Cobra reports an unknown command or equivalent invalid command error for `cluster-topology`.

## Documentation Refresh

After command metadata and README/docs content are updated, regenerate generated CLI docs:

```bash
make docs-content
```

Confirm generated docs contain `get cluster topology`, `get cluster version`, and `get cluster license`, and no longer contain a `c8volt_get_cluster-topology.md` page.
