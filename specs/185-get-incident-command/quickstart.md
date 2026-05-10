# Quickstart: Get Incident Command

## Targeted Validation

Run command, facade, and incident-service tests first:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/incident/... -count=1
```

Run broader validation before committing:

```bash
GOCACHE=/tmp/c8volt-gocache go test ./... -count=1
make test
```

## Operator Examples

Fetch a known incident:

```bash
c8volt get incident --key 2251799813817616
```

Fetch multiple incident keys:

```bash
c8volt get incident --key 2251799813817616 --key 2251799813817617
```

Pipe incident keys from process-instance incident output:

```bash
c8volt get pi --with-incidents --keys-only | c8volt get incident -
```

List active incidents:

```bash
c8volt get incident
c8volt get incident --state active
```

Search resolved incidents by error type:

```bash
c8volt get incident --state resolved --error-type io_mapping_error
```

Search incident messages with case-insensitive substring semantics:

```bash
c8volt get incident --error-message "intentional incident"
```

Search incidents in a time window:

```bash
c8volt get incident --creation-time-after 2026-05-08T00:00:00Z --creation-time-before 2026-05-09T00:00:00Z
```

Count exact matching incidents:

```bash
c8volt get incident --total --error-type io_mapping_error
```

Render JSON:

```bash
c8volt get incident --state active --json
```

Render only keys:

```bash
c8volt get incident --state active --keys-only
```

Limit error message length:

```bash
c8volt get incident --state active --error-message-limit 120
```
