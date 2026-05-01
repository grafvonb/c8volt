# Quickstart: Tenant Discovery Command

## Prerequisites

- A configured c8volt profile or config file with authentication for a supported Camunda version.
- Tenant-management support is available for the configured version.

## Scenario 1: List Tenants

```bash
./c8volt get tenant
```

Expected result:

- Human-readable output lists tenant ID, name, and description when available.
- Results are sorted by tenant name and then tenant ID.
- No sensitive tenant relationship or authorization data is shown.

## Scenario 2: Show One Tenant

```bash
./c8volt get tenant --key tenant-a
```

Expected result:

- Only tenant `tenant-a` is returned.
- Missing tenant IDs use the comparable not-found command style.

## Scenario 3: Filter List by Name

```bash
./c8volt get tenant --filter demo
```

Expected result:

- Only tenants whose names contain `demo` are returned.
- Characters such as `*`, `[abc]`, `.*`, and `name:demo` are treated as literal filter text.

Invalid combination:

```bash
./c8volt get tenant --key tenant-a --filter demo
```

Expected result:

- The command fails with the existing invalid flag-combination style.
- No tenant data is rendered.

## Scenario 4: Use JSON Output

```bash
./c8volt get tenant --json
./c8volt get tenant --key tenant-a --json
```

Expected result:

- List mode returns structured tenant data for all matching tenants.
- Keyed mode returns structured tenant data for the selected tenant.
- JSON output contains non-sensitive tenant information only.

## Scenario 5: Unsupported Version

```bash
./c8volt --camunda-version 8.7 get tenant
```

Expected result:

- The command fails using the existing unsupported-capability style.
- No partial tenant data is rendered.

## Validation Commands

```bash
go test ./internal/services/tenant/... -count=1
go test ./c8volt/tenant ./cmd -run 'Test.*Tenant' -count=1
make docs-content
make test
```
