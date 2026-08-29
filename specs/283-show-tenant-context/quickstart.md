# Quickstart: Implement and Validate Tenant Context

## Prerequisites

- Work on branch `283-show-tenant-context`.
- Read [spec.md](./spec.md), [plan.md](./plan.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/tenant-context.md](./contracts/tenant-context.md), and `specs/ralph-implementation-rules.md` before implementation.
- Ralph runs must include `--implementation-context specs/ralph-implementation-rules.md`.
- Do not create/switch branches or edit generated Camunda clients for this feature.

## Suggested implementation order

1. Add domain/public context values, conversions, constructors, and the service accumulator.
2. Preserve and aggregate tenant evidence in PI, process-definition, job, incident, and ops plans using already-resolved objects.
3. Attach base operation semantics in cmd and extend shared/raw structured views.
4. Add human preflight/confirmation rendering and creation-target lines before deploy/run mutation calls.
5. Replace misleading ops report enrichment and update Markdown/JSON report views.
6. Extend focused tests, then help/docs and full validation.

## Focused acceptance checks

### Shared model and services

```bash
go test ./internal/services/common ./internal/services/processinstance/... ./internal/services/processdefinition/... -run 'Test.*Tenant|Test.*DryRun|Test.*Plan' -count=1
go test ./c8volt/tenant ./c8volt/process ./c8volt/resource ./c8volt/job ./c8volt/incident ./c8volt/ops -run 'Test.*Tenant|Test.*Convert|Test.*Plan' -count=1
```

Verify stable deduplication/sorting, unknown-only, known-plus-unknown, known cross-tenant, and known cross-tenant-plus-unknown plans. Assert no additional mock backend calls.

### Configuration and shared machine contract

```bash
go test ./cmd -run 'Test.*Config|Test.*TenantContext|Test.*CommandContract|Test.*JSON' -count=1
```

Expected cases:

- named configuration renders `Tenant filter: tenant-a`;
- empty configuration renders no configured tenant and never calls it `<default>`;
- `config show` remains parseable YAML with `tenantContext`;
- `config test-connection --json` remains one parseable document;
- existing envelope payloads are unchanged and the optional object is present only where applicable.

### Process-instance mutations

```bash
go test ./cmd -run 'Test.*(Cancel|Delete|Resolve|Update).*ProcessInstance|Test.*DryRun.*Tenant|Test.*MutationProgress' -count=1
```

Expected cases:

- named and empty selector paths show the correct discovery line before confirmation;
- explicit keys state that the filter is not applied;
- a configured `tenant-a` and resolved `tenant-b` succeeds subject to backend behavior and reports `tenant-b`;
- multiple known tenants produce one sorted prominent warning;
- unknown metadata produces a non-blocking warning, including alongside a cross-tenant warning;
- JSON, quiet, and keys-only contracts remain exact.

### Process-definition, job, deploy, and run

```bash
go test ./cmd -run 'Test.*Delete.*ProcessDefinition|Test.*UpdateJob|Test.*Deploy|Test.*Run.*ProcessInstance' -count=1
```

Expected cases:

- definition deletion distinguishes selector and explicit-key semantics and includes nested cancellation evidence;
- job update reports the current job tenant without filtering a mismatched explicit key;
- deploy/run print `Create in tenant: <default>` or the named target before the backend call;
- non-interactive flows gain no prompt;
- JSON and key streams remain valid.

### Operations and audit reports

```bash
go test ./cmd -run 'Test.*Ops.*(Retention|Purge|Repair|Smoke|Report)|Test.*Report.*(JSON|Markdown|Tenant)' -count=1
```

Expected cases:

- retention, all-definition purge, orphan purge, incident purge, repair, and smoke-test preflight use the correct mode;
- unfiltered JSON/Markdown audit evidence never reports `<default>` as its filter;
- all affected JSON reports contain the common object and remain parseable;
- legacy `tenantId` remains only for truthful compatibility cases;
- report schema identifiers and Markdown/JSON format support remain unchanged.

## Output contract spot checks

For at least one representative command in each family, capture stdout and stderr separately:

1. Human mode: tenant context precedes the mutation/confirmation.
2. JSON mode: stdout contains exactly one JSON document with `tenantContext` and the existing payload shape.
3. Quiet mode: no new human context appears.
4. Keys-only mode: stdout is exactly `key\n` for each result with no labels or warnings.
5. Config YAML: output parses as one YAML document.

## Documentation and full validation

```bash
gofmt -w <touched-go-files>
make docs-content
make vet
make test
git diff --check
```

Update source command `Long` text/examples first, then regenerate `docs/cli/*`. Update README safety/output guidance and the affected `docs/ops/*.md` pages for retention, smoke test, process-definition purge, orphan/incident purge, and repair. Documentation must explain named discovery, unfiltered discovery, default creation, explicit keys, cross-tenant warnings, unknown metadata, and quiet/keys-only behavior.

The implementation is ready for review only when targeted tests and `make test` pass, generated docs match command metadata, and no output mode contains contradictory tenant semantics.
