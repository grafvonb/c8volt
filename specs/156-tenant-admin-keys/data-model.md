# Data Model: Tenant Scope For Discovery And Explicit Admin Keys

## Selected Tenant

The effective tenant chosen through configuration, profile, environment, or `--tenant`.

**Fields**:
- `id`: tenant identifier selected for the command, including `<default>` where create/deploy/run flows use it.
- `source`: effective configuration source for the selected tenant when already available through existing config behavior.

**Validation rules**:
- Empty tenant remains an unscoped visible-tenants query for read/search commands unless an explicit tenant is supplied.
- Create, deploy, and run flows preserve existing `<default>` behavior when the effective tenant is empty.

## Discovery-Derived Candidate Set

Targets produced by c8volt from tenant-scoped discovery or search.

**Fields**:
- `selector`: search, list, or filter criteria used to discover candidates.
- `tenant`: selected tenant applied to discovery where supported.
- `keys`: candidate keys retained for preview, confirmation, dependency expansion, or mutation.
- `dependencyScope`: additional in-scope keys produced by existing safety behavior.

**Validation rules**:
- Candidate keys must come from the tenant-scoped result set where backend support exists.
- Dependency scope must remain tied to the discovered target meaning and must not include unrelated cross-tenant resources.

## Explicit Admin Input

Operator-supplied target identifiers that c8volt treats as intentional administrative input.

**Fields**:
- `kind`: process-instance key, process-definition key, resource ID, stdin key, or direct flag value.
- `values`: deduplicated operator-supplied identifiers.
- `source`: command flag, positional stdin marker, or direct command option.

**Validation rules**:
- Values must pass existing key or ID shape validation.
- Values are governed by Camunda backend authorization, not by c8volt-side selected-tenant equality checks.
- Existing command safety checks still apply for destructive operations.

## Returned Tenant Metadata

Tenant information returned by Camunda for an observed resource.

**Fields**:
- `tenantId`: tenant identifier from returned resource metadata, when present.
- `resourceKind`: process instance, process definition, resource, incident, or related runtime fact.

**Validation rules**:
- Returned tenant metadata may be displayed or logged for operator visibility.
- For explicit admin input, returned tenant metadata must not become a local tenant mismatch rejection by itself.

## Tenant-Sensitive Command Path

A command mode whose behavior depends on whether targets are discovered by c8volt or supplied directly by the operator.

**Fields**:
- `command`: user-facing command path.
- `mode`: discovery/search-derived, direct key, direct ID, stdin key, create/deploy/run, or mixed mode.
- `supportedVersions`: Camunda versions in scope for this feature.
- `expectedTenantSemantics`: tenant-scoped discovery or backend-authorized explicit input.

**Relationships**:
- Search-derived process-instance cancellation and deletion use a Discovery-Derived Candidate Set.
- Direct process-instance, process-definition, resource, and stdin modes use Explicit Admin Input.
- Create, deploy, and run flows use Selected Tenant with existing `<default>` behavior.
