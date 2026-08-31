# Quickstart: All-Tenants Tenant Override

## Prerequisites

- Work on branch `282-all-tenants-override`.
- Read [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [the CLI contract](./contracts/all-tenants.md), and `specs/ralph-implementation-rules.md` before implementation.
- Preserve unrelated working-tree changes and do not edit generated Camunda clients.

## Implementation Order

1. Add the enum-like all-tenants support state and command annotation resolver in `cmd/command_contract.go`/`cmd/capabilities.go`.
2. Add the additive capability field and tests, then annotate all four concrete-destination leaves.
3. Add the root persistent boolean flag and early validation for explicit tenant conflict and destination rejection.
4. Apply the effective empty tenant after current configuration normalization and before config enters context or services.
5. Extend private provenance and the existing human tenant-context renderer with the exact warning.
6. Add representative discovery, output-mode, capability, and four-command no-side-effect tests.
7. Update source help/examples, README, operator guides, integration example flag recognition, and regenerate docs.

## Focused Test Sequence

Run the closest tests as each owner changes:

```bash
go test ./cmd -run 'Test.*(AllTenants|TenantOverride|TenantContext|CommandCapability|Capabilities)' -count=1
go test ./cmd -run 'Test.*(DeployProcessDefinition|EmbedDeploy|RunProcessInstance|SmokeTest).*AllTenants' -count=1
go test ./cmd -run 'Test.*(Search|List|Find|Cancel|Delete|Resolve|Ops).*AllTenants' -count=1
go test ./integration/cli -run 'Test.*(Example|Command)' -count=1
```

Use actual focused test names once introduced; keep failures local before broadening the run.

## Required Behavior Matrix

### Resolution

| Case | Expected result |
|------|-----------------|
| Named base config plus all-tenants | Effective tenant empty; exact warning in human output. |
| Named active profile plus all-tenants | Effective tenant empty; configured profile tenant shown as provenance. |
| Named environment override plus all-tenants | Effective tenant empty; configured environment tenant shown as provenance. |
| Camunda 8.7 omitted/default tenant plus all-tenants | Effective tenant empty after normalization. |
| Already-empty configured tenant plus all-tenants | Effective tenant empty; no override warning chatter. |
| `--all-tenants=false` | Existing configured/explicit tenant behavior. |
| Flag absent | All existing behavior and output unchanged. |

### Invalid input and side effects

Verify both `--tenant foo` and `--tenant ""` conflict with active all-tenants, return invalid input/exit 2, and make no request. For each command below, verify rejection occurs before the listed work:

| Command | Must not occur |
|---------|----------------|
| `deploy process-definition` | file inspection, prompt/activity, deployment request |
| `embed deploy` | embedded file access, deployment, optional run request |
| `run process-instance` | stdin/file input, prompt/activity, creation request |
| `ops execute smoke-test` | plan/report creation, prompt/activity, any request, including dry-run |

### Discovery and authorization

- Representative search/list requests must omit tenant-filter fields for every supported Camunda version.
- Mixed discovery/mutation and ops flows must use their existing unfiltered discovery behavior without a client-side tenant enumeration step.
- Direct resource-key calls must keep existing request shape and backend 403/404 handling; no ignore-tenant option is introduced.

### Output and discovery

- Assert the exact warning and line order for ordinary human output.
- Assert warning suppression and parseability for JSON/YAML.
- Assert keys-only remains one key per line, total-only remains only its total, and quiet remains silent.
- Assert capabilities expose `accepted` or `rejected_concrete_destination` from the same annotation used at runtime.
- Assert help and generated docs expose the inherited boolean flag without suggesting it is valid for concrete destinations.

### Usability evidence

Run a brief task-based review using root help, representative subcommand help, or generated documentation. Ask operators to find and invoke the all-visible-tenants behavior without mentioning `--tenant ""`. At least 90% must succeed within 30 seconds. Record participant count, elapsed outcomes, and any resulting help wording adjustment.

## Documentation Workflow

Update command source metadata first. Add representative examples for tenant-wide discovery, explicit safety wording to the four destination commands, README guidance, `docs/ops/index.md`, and affected non-generated ops playbooks. Then regenerate rather than hand-edit generated pages:

```bash
make docs-content
```

`docs/index.md` is generated from README. `docs/cli/*` is generated from command metadata.

## Final Validation

```bash
gofmt -w <touched-go-files>
make docs-content
make vet
make test
git diff --check
```

Review the final diff to ensure there are no facade/service/adapter/generated-client changes and no structured-schema provenance leak. If a real cross-tenant visibility check is available, run it as a gated/manual scenario with an identity limited to a subset of tenants; it is additional operational evidence, not a substitute for deterministic automated coverage.

## Completion Evidence

Implementation is ready for review only when:

- the complete resolution and rejection matrices pass;
- all four concrete-destination commands prove zero side effects;
- representative requests omit tenant filters;
- protected output modes retain their contracts;
- capability metadata, help, and generated docs agree;
- `make vet`, `make test`, and `git diff --check` pass.
