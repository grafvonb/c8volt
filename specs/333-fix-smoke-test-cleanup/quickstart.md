# Validation Guide: Complete Smoke-Test Definition Cleanup

## Prerequisites

Use the repository Go toolchain (Go 1.26, toolchain 1.26.2). Unit and command tests use local fixtures and do not require a cluster. Live checks require a configured supported Camunda environment, deployment/deletion permissions, and a disposable tenant. Live scenarios below mutate that tenant and are validation instructions, not actions performed during planning.

See [contract](contracts/smoke-test.md) for output expectations and [data model](data-model.md) for report fields.

## Targeted automated checks after implementation

From the repository root:

```sh
go test ./internal/services/ops -run 'Test.*SmokeTest' -count=1
go test ./c8volt/ops -run 'Test.*SmokeTest' -count=1
go test ./cmd -run 'Test.*SmokeTest' -count=1
go test ./internal/services/processdefinition -run 'TestFindUnrelatedProcessInstancesForDefinition' -count=1
make docs-content
git diff --check
```

Format touched Go files with gofmt. Include relevant command-contract tests when metadata assertions change. Broaden to the affected package suites when targeted coverage misses conversion/output consumers; use `make test` only for justified shared-runtime impact or unresolved regressions. Do not run runtime tests merely for planning artifacts.

## Regression fixtures

- Return all three definitions from deployment; assert three distinct exact-key deletes and confirmed outcomes across every supported fixture family.
- Block a dependency with unrelated instances while allowing siblings; assert no blocked/unrelated deletion and truthful retained counts.
- Simulate safety-discovery errors, deletion errors, and accepted responses followed by wait errors. Assert per-key attribution, partial failure, and preserved earlier results.
- Repeat with fail-fast and cancellation; unscheduled entries remain unattempted. Default behavior attempts independent eligible siblings.
- Cover duplicate keys, conflicting metadata, missing keys/main identity, unrelated versions/tenants, and safety discovery through sparse pages.
- Cover no-cleanup, no-wait, dry-run, automation/auto-confirm, human/verbose/JSON/quiet and supported mode combinations. Check request counts, output streams, reports, progress totals, and EOF after one JSON envelope.

## Live default cleanup

```sh
go build -o /tmp/c8volt-smoke333 .
/tmp/c8volt-smoke333 --tenant smoke333 get pd --stat
/tmp/c8volt-smoke333 --tenant smoke333 ops execute smoke-test --auto-confirm --report-file /tmp/smoke333.json --report-format json
/tmp/c8volt-smoke333 --tenant smoke333 get pd --stat
```

Start with an empty disposable tenant. The final list contains no fixture definitions. The report lists all three deployed identities, three confirmed definition results, and confirmed instance cleanup. Repeat using configured supported compatibility versions. A live result is not implied by passing fake-server tests.

## Retention and submitted-state checks

```sh
/tmp/c8volt-smoke333 --tenant smoke333 ops execute smoke-test --dry-run
/tmp/c8volt-smoke333 --tenant smoke333 ops execute smoke-test --no-cleanup --auto-confirm --report-file /tmp/smoke333-retained.json
/tmp/c8volt-smoke333 --tenant smoke333 ops execute smoke-test --no-wait --auto-confirm --report-file /tmp/smoke333-submitted.json
```

Use a fresh disposable tenant/reset between scenarios. Dry-run changes nothing. No-cleanup retains all created resources and reports their identities. No-wait reports submitted requests, not confirmed removal. Resources deliberately retained by no-cleanup require explicit teardown using the recorded exact keys and normal deletion safety controls; do not delete by shared fixture names. Safety-blocked and failure scenarios are deterministic automated fixtures rather than timing-dependent live mutations.
