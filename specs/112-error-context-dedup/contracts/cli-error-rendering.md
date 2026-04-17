# Contract: CLI Error Rendering Deduplication

## Shared Prefix Contract

All affected CLI failures must preserve the existing normalized error-class prefix chosen by `c8volt/ferrors`.

| Situation | Required behavior |
|--------|-------------------|
| Normalized not-found failure | Keep `resource not found:` as the prefix |
| Normalized unsupported failure | Keep `unsupported capability:` as the prefix |
| Other normalized shared class | Keep that class’s existing normalized prefix |

This feature must not change the class-selection or exit-code contract.

## Helper Boundary Contract

The shared helper boundary for this feature is intentionally narrow.

| Helper seam | Required behavior |
|-------------|-------------------|
| `c8volt/ferrors.Normalize` and `WrapClass` | Keep owning only shared classification, prefix rendering, and exit-code inputs; they must not rewrite or deduplicate wrapped detail text |
| `cmd.normalizeCommandError` and `cmd.normalizeBootstrapError` | Map known command/bootstrap sentinels to shared classes, then defer to `ferrors` for all remaining normalization |
| `cmd.handleNewCliError` and `cmd.handleBootstrapError` | Resolve logger and `noErrCodes` context, then render through `ferrors.HandleAndExit` without adding another message-composition layer |

These helpers are not allowed to introduce a second dedup system. Story-specific cleanup must happen in the upstream command, helper, and service wrappers that compose the breadcrumb chain.

## Breadcrumb Contract

| Rule | Required behavior |
|------|-------------------|
| Ordered context | Breadcrumbs remain in outer-to-inner stage order |
| Stage meaning | Breadcrumbs must still identify the failing stage |
| Shortening | Breadcrumbs may be shortened only when their meaning remains equivalent |
| No repeated failure meaning | Breadcrumbs must not restate the same root failure meaning once a lower layer already owns it |

## Root Detail Contract

| Rule | Required behavior |
|------|-------------------|
| Single root detail | The final rendered error includes the underlying root failure detail once |
| Identifier deduplication | Repeated identifiers should be removed when they do not add new context |
| Cross-class rule | The same single-root-detail contract applies to matching non-not-found error classes |
| Helper ownership | Shared helpers preserve the wrapped detail as-is; upstream wrappers own whether the detail is already concise enough |

## Scope Contract

The feature applies to every duplicated CLI error path found during the repository audit that shares the same duplication pattern and feeds user-facing output.

Covered pattern families include:

- process-instance lookup and traversal
- process-instance mutation and wait follow-up
- single-resource command fetch wrappers
- resource/client orchestration wrappers that bubble into CLI output

The audited owner layers for those families are:

| Pattern family | Owner layer | Representative files |
|--------|-------------|----------------------|
| Process-instance lookup/traversal | walker plus versioned process-instance services | `internal/services/processinstance/walker/walker.go`, `internal/services/processinstance/v87/service.go`, `internal/services/processinstance/v88/service.go`, `internal/services/processinstance/v89/service.go` |
| Process-instance mutation/wait | versioned process-instance services plus waiter-backed follow-up wrappers | `internal/services/processinstance/v87/service.go`, `internal/services/processinstance/v88/service.go`, `internal/services/processinstance/v89/service.go`, `internal/services/processinstance/waiter/waiter.go` |
| Single-resource fetch wrappers | command handlers | `cmd/get_processdefinition.go`, `cmd/get_resource.go`, `cmd/get_cluster_license.go`, `cmd/get_cluster_topology.go`, `cmd/get_processinstance.go` |
| Resource/client orchestration wrappers | CLI-facing resource client wrappers | `c8volt/resource/client.go` |

## Regression Contract

Representative automated coverage is required for each affected duplication-pattern family.

| Family | Minimum regression expectation |
|--------|-------------------------------|
| Process-instance lookup/traversal | Covered by walk/get/helper tests that assert preserved prefix, preserved breadcrumbs, and deduplicated root detail |
| Process-instance mutation/wait | Covered by cancel/delete/helper tests that assert the same rendering contract |
| Single-resource fetch wrappers | Covered by representative get-style command tests |
| Non-not-found class pattern | Covered by at least one representative failure path if the audit changes such a family |

Confirmed regression anchors for the setup audit are:

| Family | Anchor files |
|--------|--------------|
| Process-instance lookup/traversal | `cmd/walk_test.go`, `internal/services/processinstance/walker/walker_test.go` |
| Process-instance mutation/wait | `cmd/cancel_test.go`, `cmd/delete_test.go`, `internal/services/processinstance/v87/service_test.go`, `internal/services/processinstance/v88/service_test.go` |
| Single-resource fetch wrappers | `cmd/get_test.go` and focused `get_*` command tests |
| Shared class/exit behavior | `c8volt/ferrors/errors_test.go`, `cmd/bootstrap_errors_test.go` |
| Cluster license/topology unavailable and malformed-response paths | `cmd/get_test.go` plus shared unavailable-prefix checks in `c8volt/ferrors/errors_test.go` |

The tests should assert semantic rendering outcomes, not every exact original wording choice, except where the preserved shared class prefix is part of the public contract.
