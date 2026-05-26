# Camunda 8.9 Release Integration Suite Prompt

Use this prompt to run or improve the C89 real-cluster release integration
suite in `integration/`. This is release validation harness work, not product
implementation work.

## Inputs

- Target Camunda minor: 8.9
- Fixture prefix: C89
- Default config: `./config.yaml`
- Default profile: `kind-camunda-platform-local-c89`
- Wrapper: `integration/scripts/run-c89-suite.sh`

## Goal

Run a comprehensive release integration pass against the configured local C89
cluster. Validate daily operator workflows, especially mutating commands and
C89-only resource deletion behavior, while collecting enough evidence to fix
product issues after the run.

The target cluster is a dirty, disposable local release-lab environment. It may
already contain process definitions, process instances, incidents, jobs,
resources, and other state. The suite must run in that dirty environment and
does not need to preserve existing resources.

## Required Behavior

1. Build or select the current c8volt binary.
2. Use c8volt commands to prove configuration, connectivity, capabilities,
   gateway version, topology, and target C89 suitability.
3. Stop before mutation if the connected cluster is not Camunda 8.9.
4. Generate run-owned disposable data using C89 embedded fixtures without
   requiring the cluster to be clean.
5. Exercise read-only, dry-run, mutating, JSON, automation, and key-pipeline
   paths that matter for daily operations.
6. Run preview before real mutation where supported.
7. Verify post-conditions with c8volt commands.
8. Exercise C89 process-definition deletion and all-process-definition purge
   with full-force behavior. Broad dirty-cluster purge is allowed and expected
   in a disposable local cluster.
9. Use `--auto-confirm`, `--automation`, `--force`, and broad destructive
   scopes when needed; do not ask for confirmation during the suite.
10. Keep running after individual failures and write a complete report.
11. Capture UX findings about wording, grammar, aliases, confirmation behavior,
    output size, and automation friendliness.

## Recommended Command

```sh
integration/scripts/run-c89-suite.sh
```

Optional larger data pass:

```sh
C8VOLT_IT_VOLUME_COUNT=50 integration/scripts/run-c89-suite.sh
```

## Output Expectations

Report:

- c8volt version and binary path
- config/profile and cluster evidence
- fixture setup and generated data inventory
- grouped command results
- mutating command evidence and post-checks
- C89-only destructive workflow evidence
- dirty-environment setup and cleanup/purge outcome
- UX findings and recommended improvements

Do not edit product code while running this prompt unless the user explicitly
turns the findings into implementation work.
