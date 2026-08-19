# Quickstart: Validate Camunda 8.10 Guidance Alignment

## Prerequisites

- Work from branch `277-align-v810-guidance`.
- Use the repository Go toolchain declared in `go.mod`.
- Have `rg`, Git, and Make available.
- Use commit `d7f87d5d` as the feature baseline for protected-path comparisons.

## 1. Verify Maintainer Version Inventories

Inspect the active maintainer and architecture sources:

```bash
rg -n 'v87|v88|v89|v810|V87|V88|V89|V810|newest supported|CurrentCamundaVersion' \
  AGENTS.md \
  specs/ralph-implementation-rules.md \
  .specify/memory/architecture.md \
  .specify/memory/architecture-repo-facts.md
```

Expected:

- Every complete supported-runtime adapter/client inventory includes V810.
- V810 is identified as newest where a newest line is named.
- V88 remains the default.
- Version-neutral layering rules remain unchanged.

## 2. Verify the Shipped Configuration Template

```bash
rg -n 'Supported values|default|camunda_version' config/templates/config.example.yaml
go test ./cmd -run 'TestConfigTemplate.*SupportedCamundaVersions' -count=1
```

Expected:

- The supported values include 8.7, 8.8, 8.9, and 8.10.
- The comment identifies 8.8 as the default.
- The rendered template exposes the same guidance.

## 3. Verify Gateway Requirement and Help Semantics

```bash
rg -n 'FR-006|SC-002|match|mismatch|unrecognizable|diagnostic' \
  specs/273-camunda-v810-support/spec.md \
  specs/273-camunda-v810-support/contracts/version-selection.md \
  cmd/config_test_connection.go

go test ./cmd -run 'TestConfigTestConnection.*(Help|V810GatewayReleaseLineWarnings)' -count=1
```

Expected:

- Plain, patch, and prerelease 8.10 gateway values are matches.
- Another major/minor is a diagnostic non-match.
- Empty and unparseable values are unverifiable diagnostics.
- No issue #277 wording requires a new command failure.

## 4. Verify Active C810 Guidance and Historical Preservation

```bash
rg -n 'V810.*C89|8\.10.*C89|C89.*8\.10' \
  specs/273-camunda-v810-support/spec.md \
  specs/273-camunda-v810-support/plan.md \
  specs/273-camunda-v810-support/research.md \
  specs/273-camunda-v810-support/data-model.md \
  specs/273-camunda-v810-support/quickstart.md \
  specs/273-camunda-v810-support/contracts/service-compatibility.md

rg -n 'Supersession Note|V810-to-C89' \
  specs/273-camunda-v810-support/tasks.md \
  specs/273-camunda-v810-support/progress.md \
  specs/273-camunda-v810-support/ralph-memory.md
```

Expected:

- Active guidance contains no C89 runtime fallback. A C89 reference is acceptable only when it explains the behavioral source or a rejected alternative.
- Historical records retain the former C89 implementation language.
- The #275 supersession note remains present and makes the active C810 decision unambiguous.

## 5. Verify the Operator Contract

```bash
rg -n '8\.10|810|v810|v8\.10|8\.10\.0-alpha4|default' \
  README.md \
  api/README.md \
  docs/index.md \
  docs/cli/c8volt.md \
  docs/cli/c8volt_version.md \
  docs/cli/c8volt_config_test-connection.md \
  config/templates/config.example.yaml

go test ./cmd ./docsgen -count=1
```

Expected:

- Operator sources agree on the canonical identity, four aliases, V88 default, active prerelease baseline, and in-place replacement model where each topic is relevant.
- Gateway help reflects the full match/diagnostic matrix.
- Focused command and documentation tests pass.

Regenerate derived documentation through the repository workflow:

```bash
make docs-content
```

Review generated diffs and retain only source-driven changes. Do not hand-edit `docs/index.md` or `docs/cli/*`.

## 6. Verify Intentional Integration Scope

```bash
rg -n '8\.7|8\.8|8\.9|8\.10|V810|C810' integration
```

Expected: integration guidance and assets remain limited to the established stable V87–V89 profiles. Their omission of V810 is intentional and must not be treated as stale product-support guidance.

## 7. Verify Protected Scope

```bash
git diff --exit-code d7f87d5d -- \
  internal/clients/camunda \
  internal/services \
  c8volt \
  embedded/processdefinitions \
  integration \
  api
```

Expected: no diff. Focused documentation assertions under `cmd/` are allowed; production runtime changes are not.

## 8. Run the Delivery Gate

```bash
git diff --check
make test
git status --short
```

Expected:

- No whitespace errors.
- The full race-enabled repository test suite passes.
- Final changes are limited to #277 artifacts, current guidance, the configuration template, the #273 wording correction, authored gateway help, source-driven generated documentation, and focused documentation assertions.
