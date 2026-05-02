# Quickstart: Walk Process-Instance Incident Details

## Human Output With Incidents

```bash
c8volt walk pi --key 2251799813711967 --with-incidents
```

Expected behavior:

- Walks the process-instance family tree using current defaults.
- Preserves existing process-instance row ordering.
- Prints returned incident messages as indented `incident <incident-key>:` lines below the matching process-instance row.
- Prints no incident lines for walked instances with no returned incidents.

## Family Tree With Incidents

```bash
c8volt walk pi --key 2251799813711967 --with-incidents
```

Expected behavior:

- Preserves existing ASCII tree structure and `(--key)` marker.
- Shows incident lines under the tree node that owns them.
- Preserves existing missing ancestor warnings after traversal output.

## JSON Output With Incidents

```bash
c8volt --json walk pi --key 2251799813711967 --children --with-incidents
```

Expected behavior:

- Preserves the shared JSON envelope.
- Preserves traversal metadata such as mode, outcome, root key, keys, edges, missing ancestors, and warning.
- Adds per-item `incidents` collections with incident details for each walked process instance.

## Validation Error

```bash
c8volt walk pi --with-incidents
```

Expected behavior:

- Fails before process-instance or incident lookup.
- Reports a clear validation error explaining that `--with-incidents` requires keyed walk input.

## Incident Lookup Failure

```bash
c8volt walk pi --key 2251799813711967 --with-incidents
```

Expected behavior when traversal succeeds but an incident lookup fails:

- Fails the command.
- Does not render a partially enriched human or JSON traversal.
- Does not turn the failed lookup into an empty incident collection.

## Version Boundary

```bash
c8volt --camunda-version 8.7 walk pi --key 2251799813711967 --with-incidents
```

Expected behavior:

- Returns the repository's existing unsupported-capability style when tenant-safe incident enrichment is unavailable.
- Does not silently fall back to tenant-unsafe direct incident lookup.

## Validation Commands

```bash
go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1
make test
```
