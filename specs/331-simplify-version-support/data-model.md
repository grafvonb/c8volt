# Data Model: Supported-Version Cleanup

No new persistent entities, schemas, or migrations are introduced.

## Version selection

- **Identity**: Existing `toolx.CamundaVersion` value.
- **Allowed canonical values**: `8.8`, `8.9`, `8.10`.
- **Default**: `8.9` when configuration omits selection.
- **Aliases**: Existing aliases for those three identities remain accepted.
- **Validation**: Any other identity follows generic unsupported-version errors without default substitution or remote work.
- **Relationship**: Each valid identity selects the existing matching adapter for every service family; capability differences remain explicit.

## Tenant selection

- **Discovery tenant**: Empty remains all visible tenants under existing supported-version rules.
- **Creation tenant**: Empty uses the existing concrete default tenant through `TargetTenant()`.
- **Validation**: Explicit tenant values, all-tenants overrides and configuration-source precedence retain their current contracts.

## Traversal and mutation results

Existing domain/facade response shapes remain stable. Retain complete/partial traversal outcomes, missing-ancestor facts, exact/lower-bound/indeterminate totals, tenant evidence and accepted-versus-confirmed mutation states. Removing the exclusive key-only path does not remove unknown tenant evidence when supported responses omit metadata.

## Support artifact inventory

A review inventory classifies each tracked path/reference as retired support to remove, retained behavior to preserve, or incidental numeric/provenance data. This is a validation record, not a new product schema. Git history, external issues and ignored local configuration are outside its boundary.
