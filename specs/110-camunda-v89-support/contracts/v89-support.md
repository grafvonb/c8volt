# Contract: Camunda v8.9 Runtime Support

## Version Support Contract

| Surface | Required behavior |
|--------|-------------------|
| Version normalization | `8.9`, `89`, `v89`, and `v8.9` normalize to `toolx.V89` |
| Supported-version lists | User-facing support lists include `8.9`; until native factories land, runtime error text may still list only implemented versions |
| Factory selection | `cluster`, `processdefinition`, `processinstance`, and `resource` factories return native `v89` implementations when `app.camunda_version` is `v8.9` |
| Unsupported version handling | Missing or unsupported versions continue to fail through the existing unknown-version error path |
| Implemented-version error text | Runtime unsupported-version errors now list `8.7`, `8.8`, and `8.9` because all four versioned service families have native `v89` implementations |

## Foundational Phase Gate

- Phase 2 is complete when the repository has compile-safe `internal/services/*/v89` package scaffolds, version-local client contracts, and shared `api.go` assertions that reserve the `v89` service shape without routing runtime traffic to it yet.
- Until User Story 1 lands, `toolx` may advertise `v8.9` while factory error messages continue to use the narrower implemented-version list so runtime failures stay truthful.
- Release readiness is still blocked until native `v89` factory selection, execution coverage, and documentation parity all exist together.

## Final Native v8.9 Client Boundary Contract

| Final native `v8.9` surface | Required client boundary |
|----------------------------|--------------------------|
| Cluster service | `internal/clients/camunda/v89/camunda/client.gen.go` only |
| Process-definition service | `internal/clients/camunda/v89/camunda/client.gen.go` only |
| Process-instance service | `internal/clients/camunda/v89/camunda/client.gen.go` only |
| Resource service | `internal/clients/camunda/v89/camunda/client.gen.go` only |

Mixed-client internals are allowed only inside documented temporary fallback paths and must not remain in the final accepted native `v8.9` runtime path.

## Temporary Fallback Contract

| Situation | Required behavior |
|----------|-------------------|
| Native `v8.9` path is still incomplete during implementation | Temporary fallback may be used if the user-facing command contract is preserved |
| Temporary fallback exists | It must be documented for maintainers and tracked as non-final |
| Feature reaches final acceptance for a service family that already uses versioned services | Temporary fallback must be removed and replaced by the native `v8.9` path |

Current feature status: no remaining repository command family relies on a temporary fallback path for `v8.9`; the transition-only rule remains documented here only as a bounded implementation guard.

## Repository Command-Family Parity Contract

Every repository command family already supported on `v8.8` must remain supported on `v8.9`:

| Command family | Minimum parity expectation |
|---------------|----------------------------|
| `get cluster topology`, `get cluster license` | Same successful metadata retrieval contract on `v8.9` |
| `get process-definition` | Same list/latest/XML/statistics behavior on `v8.9` |
| `deploy process-definition`, `delete process-definition`, `get resource` | Same resource lifecycle contract on `v8.9` |
| `run`, `get process-instance`, `walk`, `expect`, `cancel`, `delete` process-instance | Same process-instance lifecycle and traversal contract on `v8.9` |

At least one explicit `v8.9` execution test is required for each repository command family.

## Preserved Older-Version Contract

| Version | Required behavior |
|--------|-------------------|
| `v8.7` | Existing runtime behavior remains unchanged |
| `v8.8` | Existing runtime behavior remains unchanged |
| `v8.9` | Adds the same command-family support scope currently available on `v8.8` |

Adding `v8.9` support must not silently regress or reclassify `v8.7` or `v8.8` behavior.

## Documentation Release Gate Contract

| Surface | Required behavior before release readiness |
|--------|--------------------------------------------|
| `README.md` | Version-support guidance reflects `v8.9` support truthfully |
| `docs/index.md` | Synced homepage content reflects the same support statement |
| `docs/cli/*` | Generated CLI reference reflects the same runtime truth and help text |
| Root command help | No longer claims runtime support stops at `v8.8` |

Documentation updates are part of release readiness and are not optional follow-up work.

## Repository Implementation Seams

The main implementation seams for this contract are:

- `toolx/version.go`
- `cmd/root.go`
- `c8volt/client.go`
- `internal/services/cluster/{factory.go,v89/...}`
- `internal/services/processdefinition/{factory.go,v89/...}`
- `internal/services/processinstance/{factory.go,v89/...}`
- `internal/services/resource/{factory.go,v89/...}`
- `README.md`
- `docs/index.md`
- generated `docs/cli/`
