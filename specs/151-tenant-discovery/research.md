# Research: Tenant Discovery Command

## Generated Tenant Client Support

Decision: Support tenant discovery through generated Camunda `v8.8` and `v8.9` clients.

Rationale: Both `internal/clients/camunda/v88/camunda/client.gen.go` and `internal/clients/camunda/v89/camunda/client.gen.go` expose `SearchTenants`, `GetTenant`, `TenantSearchQueryRequest`, `TenantFilter`, `TenantSearchQuerySortRequest`, and `TenantResult`. `TenantResult` exposes the non-sensitive display fields needed by the issue: `tenantId`, `name`, and `description`.

Shape review 2026-05-01: No field mismatch found. `v8.8` and `v8.9` tenant shapes are aligned for the planned conversion path: `TenantResult.TenantId`, `TenantResult.Name`, nullable `TenantResult.Description`, request-level `TenantFilter` with optional `TenantId` and `Name`, and search sort requests are present in both versions. `v8.7` still has no matching `SearchTenants` or `GetTenant` generated methods.

Alternatives considered: Calling generated clients directly from command code was rejected because the issue explicitly requires service-layer support and the repository consistently hides generated clients behind versioned internal services.

## Unsupported Version Handling

Decision: Treat `v8.7` tenant discovery as unsupported unless implementation discovers an equivalent generated tenant-management surface during coding.

Rationale: The current `v8.7` generated client search did not show `SearchTenants` or `GetTenant` equivalents, while the repository already has `domain.ErrUnsupported` and facade error mapping for unsupported capability behavior.

Alternatives considered: Returning an empty list for `v8.7` was rejected because it would be misleading. Local fallback discovery through unrelated APIs was rejected because it would not provide the tenant-management contract requested in the issue.

## Filtering Semantics

Decision: Apply `--filter` as local literal `contains` matching against tenant names.

Rationale: The issue explicitly asks for simple contains behavior and explicitly rejects wildcard, glob, regex, and query-language support. Local filtering also keeps behavior deterministic and easy to test regardless of upstream filter semantics.

Alternatives considered: Passing name filters to the upstream API was rejected as the only filtering mechanism because generated filter semantics may not be simple contains. Case-insensitive matching was deferred because the issue does not request it and the simplest literal contains behavior is case-sensitive in Go.

## Keyed Lookup and Filtering

Decision: Reject `--key` and `--filter` together as an invalid flag combination.

Rationale: `--key` selects one tenant by tenant ID, while `--filter` is a list-mode convenience against tenant names. Rejecting the combination keeps command behavior explicit and matches existing CLI validation patterns for incompatible filters.

Alternatives considered: Ignoring `--filter` in keyed mode was rejected because it silently discards user input. Applying the filter after keyed lookup was rejected because it makes a direct ID lookup unexpectedly depend on tenant name text.

## Sorting Boundary

Decision: Sort final list results by tenant name and then tenant ID in the facade or command-facing layer.

Rationale: The issue asks for predictable sorting, preferably by tenant name and then tenant ID. Applying final local sorting makes tests independent of upstream order and still allows internal services to request upstream sorting when useful.

Alternatives considered: Relying only on upstream sort parameters was rejected because responses may vary between versions or mocked tests. Sorting by tenant ID only was rejected because the issue prefers name first.

## Output Shape

Decision: Human and JSON output should expose tenant ID, name, and optional description only.

Rationale: These are the relevant non-sensitive fields present in the generated tenant results. Keeping the public model narrow reduces accidental disclosure and keeps list output compact.

Alternatives considered: Emitting raw generated tenant structs was rejected because it would couple public JSON output to generated clients and could include future fields without review.
