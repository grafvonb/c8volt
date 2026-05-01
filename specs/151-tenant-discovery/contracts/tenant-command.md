# Contract: Tenant Discovery CLI

## Command Surface

### `c8volt get tenant`

Returns all visible tenants for the configured environment.

Required behavior:

- Supports relevant global flags through the existing root command behavior.
- Uses read-only command classification.
- Produces compact human-readable output by default.
- Sorts by tenant name and then tenant ID.
- Returns an empty list successfully when no tenants are visible.

### `c8volt get tenant --key <tenant-id>`

Returns one tenant by tenant ID.

Required behavior:

- Uses tenant ID as the exact lookup key.
- Returns only the selected tenant.
- Uses the comparable not-found style when the tenant ID does not exist.
- Does not apply list filtering behavior.

### `c8volt get tenant --filter <text>`

Returns list results whose tenant names contain the supplied text.

Required behavior:

- Applies to list mode.
- Treats the filter as literal text.
- Does not interpret wildcard, glob, regex, or query-language syntax.
- Preserves final sorting after filtering.

### `--json`

Returns structured non-sensitive tenant output.

Required behavior:

- List mode returns structured data for all matching tenants.
- Keyed mode returns structured data for the selected tenant.
- Output includes tenant ID, name, and description when available.
- Output excludes sensitive data and generated-client relationship details.

## Error Contract

- Unsupported tenant-management versions fail with the repository's existing unsupported-capability style.
- Missing keyed tenants use the comparable not-found style.
- Invalid flag combinations, such as combining list-only filtering with keyed lookup, use existing invalid-input flag error style.
- Upstream authentication, authorization, and connectivity failures continue to flow through existing command error mapping.

## Documentation Contract

- Generated CLI documentation includes `get tenant`, `--key`, `--filter`, `--json`, aliases if any are added, and relevant inherited flags.
- README examples are updated only if the repository's user-facing command overview needs to list the new tenant discovery command.
