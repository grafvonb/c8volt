# Validation Guide: Tenant Context Before Ops Mutations

## Prerequisites

- Repository checkout on `295-tenant-context-before-mutations` with the implementation applied.
- Go toolchain from `go.mod` (Go 1.26, toolchain 1.26.2), Make, and repository dependencies available.
- Read [contract](contracts/tenant-reporting.md) and [data model](data-model.md) for expected behavior.
- Commands below run from the repository root. Automated scenarios use existing fake backends and require no live destructive environment.

## Targeted validation

Run the narrowest new regression tests first. Then exercise existing relevant suites:

```sh
go test ./internal/services/ops -run 'Test.*(Purge|Retention|Repair|Tenant|Progress)' -count=1
go test ./c8volt/ops -run 'Test.*(Progress|Tenant|Purge|Retention|Repair)' -count=1
go test ./cmd -run 'Test.*(TenantContext|OpsPurgeAllProcessDefinitions|OpsPurgeOrphanProcessInstances|OpsPurgeProcessInstancesWithIncidents|OpsExecuteRetentionPolicy|OpsRepairIncident|OpsRepairProcessInstance|OpsAuditReport|MarkdownTenantContext)' -count=1
```

Expected: all relevant tests pass, with new regression cases discoverable under these patterns. Verify the test output or `go test -list` includes the new cases; an empty test selection is not validation.

## End-to-end command scenarios

Use the command subprocess and fake-server fixtures in the existing six `cmd/ops_*_test.go` files. New tests must execute real command parsing/configuration and command handlers rather than only invoking render helpers.

1. For each affected command, execute auto-confirm against known targets. At the first discovery request/activity observation, assert stderr already contains selection context. At the first mutation request, assert affected tenants and warnings are already present. For repair with variable changes, observe the variable update as the first mutation.
2. Repeat interactive execution with accepted and declined input. Verify all applicable context precedes the question; decline causes zero mutations. Existing planned candidate reuse and request counts must match baseline.
3. Exercise the selection and evidence combinations in the contract matrix. Count each applicable summary/warning across the full output; each occurs once. Empty scopes emit no fabricated tenant or unknown warning.
4. Run supported JSON, quiet, automation, and other protected modes. Parse the JSON envelope and check both output streams for unintended tenant chatter. Retain existing quiet failure diagnostics.
5. Request JSON and Markdown audit files in temporary directories. Verify full tenant evidence survives human deduplication, including unknown and cross-tenant cases. Reuse existing report overwrite/preservation expectations.
6. Cover discovery/impact errors and force-blocked paths: no mutation occurs and no partial scope is claimed as validated. Cover dry-run and repeated command execution to catch stale rendering state.

Use existing `testx.SafeSlice`/`testx.AtomicCounter` or synchronized fixture state when request handlers run concurrently. Avoid sleep-based ordering assertions and unsynchronized buffer reads. Document each new or materially changed test's regression purpose.

## Documentation and complete validation

After implementation, format touched Go files with `gofmt`, update README and command source guidance, and run:

```sh
make docs-content
make test
git diff --check
```

Expected: generated CLI references match the corrected order, the full `go test ./... -race -count=1` suite passes, and the diff has no whitespace errors. Inspect regenerated docs for unrelated changes before committing. Do not hand-edit generated command references.

This guide describes validation to perform during implementation. No implementation test results are claimed by the planning artifacts.
