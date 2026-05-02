# Quickstart: Keyed Process-Instance Incident Details

## Manual Smoke Scenarios

### Keyed human output with incidents

```bash
c8volt get pi --key 2251799813711967 --with-incidents
```

Expected result:

- The process-instance row is shown.
- Incident message text is visible as an indented `incident:` line below the matching process-instance row when the process instance has returned incidents.
- The command exits successfully for process instances without incidents.

### Keyed JSON output with incidents

```bash
c8volt get pi --key 2251799813711967 --with-incidents --json
```

Expected result:

- JSON payload contains an enriched collection.
- Each item contains the process instance and its own `incidents` collection.
- Each returned incident includes `errorMessage` when available.

### Multiple keys

```bash
c8volt get pi --key 2251799813711967 --key 2251799813711977 --with-incidents --json
```

Expected result:

- Each process instance is returned once.
- Incident details are attached to the matching process-instance key.

### Invalid search-mode use

```bash
c8volt get pi --with-incidents
c8volt get pi --state active --with-incidents
c8volt get pi --incidents-only --with-incidents
```

Expected result:

- Each command fails with a clear invalid flag-combination error before incident lookup.

### Unsupported v8.7 behavior

```bash
c8volt --camunda-version 8.7 get pi --key 2251799813711967 --with-incidents
```

Expected result:

- The command fails using the repository's existing unsupported-capability style when tenant-safe incident enrichment is unavailable.

## Automated Validation

Targeted validation:

```bash
go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1
```

Repository validation:

```bash
make test
```
