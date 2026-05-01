# Data Model: Tenant Discovery Command

## Tenant

Represents a Camunda tenant visible to the configured environment.

Fields:

- `tenantId` string, required. Stable tenant identifier and the value accepted by `--key`.
- `name` string, required. Human-readable tenant name used for display, sorting, and list filtering.
- `description` string, optional. Non-sensitive descriptive text returned when available.

Validation rules:

- `tenantId` must be preserved exactly as returned by the service.
- `name` must be preserved exactly as returned by the service.
- `description` may be empty or absent and must not break human or JSON rendering.
- Public tenant models must not include credentials, tokens, assignments, authorizations, client secrets, user membership, group membership, roles, mapping rules, or other sensitive relationship data.

## Tenant List Result

Represents the command-facing collection returned by list mode.

Fields:

- `items` list of `Tenant`.
- `total` optional count when the repository pattern calls for list totals.

Validation rules:

- Items are sorted by tenant name ascending and tenant ID ascending before presentation.
- Empty lists are successful results.
- Filtering must not mutate tenant data; it only includes or excludes items.

## Tenant Lookup Result

Represents a single tenant returned by `--key`.

Fields:

- `item` or direct `Tenant`, depending on the repository's existing single-item JSON rendering pattern.

Validation rules:

- A matching tenant ID returns exactly one tenant.
- A missing tenant ID maps to the existing comparable not-found outcome.
- `--filter` must not broaden or modify keyed lookup behavior; combining `--key` and `--filter` is invalid.

## Tenant Name Filter

Represents literal text supplied through `--filter`.

Fields:

- `text` string, optional in list mode.

Validation rules:

- Matching uses literal contains behavior against `Tenant.name`.
- Wildcard, glob, regex, and query-language syntax are treated as literal text.
- Empty filter text should behave like no filter unless existing flag validation rejects empty string values.
- Tenant name filters apply only to list mode and are rejected when `--key` is also supplied.

## Unsupported Tenant Capability

Represents a version-specific inability to list or lookup tenants.

Fields:

- `version` configured Camunda version.
- `operation` tenant discovery operation that is unsupported.

Validation rules:

- Unsupported versions return the repository's existing unsupported-capability style.
- Unsupported behavior occurs before partial tenant output is rendered.
