# Camunda 8.10 Release Integration Suite Prompt

Use this prompt to run or improve the C810 real-cluster shell integration suite
in `integration/`. This is release validation harness work, not product
implementation work.

## Inputs

- Target Camunda minor: 8.10
- Initial release baseline: 8.10.3
- Fixture prefix: C810
- Default config: `./config.yaml`
- Default profile: `kind-camunda-platform-local-c810`
- Wrapper: `integration/scripts/run-c810-suite.sh`

## Goal

Run the broad operator-workflow shell pass against a configured disposable C810
cluster. Collect command-level evidence for daily read, mutation, and ops
workflows without presenting the result as aggregate release certification.

The target may already contain unrelated state. The suite uses C810 fixtures,
can perform broad destructive cleanup, and must never target a shared or
production cluster.

## Required Behavior

1. Build or explicitly select the current c8volt binary.
2. Prove configuration, connectivity, capabilities, gateway version, and
   topology with c8volt commands.
3. Require an exactly parsed Camunda 8.10 gateway minor and stop before fixture
   deployment or any other mutation on mismatch.
4. Generate run-owned disposable data from C810 embedded fixtures without
   requiring an empty cluster.
5. Exercise read-only, dry-run, mutating, automation, and key-pipeline paths.
6. Preview mutations where supported, execute them with explicit automation,
   and verify observable post-state.
7. Exercise supported process-definition deletion and purge workflows while
   retaining individual failures in the report.
8. Record command stdout, stderr, exit status, grouped progress, and UX
   findings without copying credentials into the report.

## Recommended Command

```sh
integration/scripts/run-c810-suite.sh
```

Optional larger data pass:

```sh
C8VOLT_IT_VOLUME_COUNT=50 integration/scripts/run-c810-suite.sh
```

## Result Boundary

This runner writes shell-suite `report.md`, `progress.tsv`, and command logs. A
successful result does not prove the required scenario catalog, five common
aggregate categories, batch product dependency, or tenant coverage. Use the
single-profile `make integration-test-all` workflow and inspect `release.json`
for aggregate release evaluation.

Do not edit product code while running this prompt unless the user explicitly
turns a finding into implementation work.
