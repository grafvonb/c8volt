# All-Command Go Integration Suite Rules

Status: context-only integration harness guidance.

This file is reusable context for designing, implementing, reviewing, or rerunning a destructive Go integration suite that covers the complete `c8volt` command tree against real local Camunda clusters.

It is not a Spec Kit feature artifact, Ralph implementation context, product requirement source, public documentation page, generated documentation input, or normal implementation-run dependency. Do not load or apply this file during ordinary feature implementation unless the user explicitly asks to work on the all-command integration suite or its scripts.

## Purpose

The all-command Go integration suite must prove that the real `c8volt` CLI works end to end against real Camunda clusters. It complements the existing unit tests and release integration scripts by exercising command execution, flags, examples, output modes, destructive workflows, and version behavior through the built binary.

The suite must cover every command reported by:

```sh
c8volt capabilities --json
```

At the time this rule set was written, that inventory contains 55 command nodes, including grouping commands and leaf commands.

## Configuration Contract

- Use the operator's default local c8volt configuration from `$HOME/.config/c8volt`.
- Do not pass `--config` from the suite.
- Do not generate a private test config.
- Do not override auth mode in test helpers.
- It is acceptable to select a profile that already exists in the default local config, for example through a suite variable such as `C8VOLT_IT_PROFILES=dev87,dev88,dev89`.
- Profile selection must only name profiles from the default local config. It must not point to an alternate config file.
- The suite should fail early with a clear message when the required version profiles are not available or do not connect.

## Cluster State Contract

The suite must be prepared for both states:

- a completely clean cluster
- an already dirty cluster with unrelated process definitions, instances, incidents, jobs, variables, tenants, and resources

The suite must not assume exclusive ownership of the cluster and must not fail merely because unrelated data exists.

The selected cluster is considered disposable for this suite. It is explicitly allowed to mutate existing data when required to cover command behavior. Existing process definitions, process instances, incidents, jobs, resources, variables, tenants, and other cluster state may be updated, cancelled, deleted, resolved, repaired, or purged by the suite.

Do not run this suite against shared, production, customer, or non-disposable clusters.

## Data Ownership

Prefer run-owned data when it can satisfy the scenario:

- Generate one unique run marker at suite startup, for example `c8voltITRunId`.
- Pass the marker through command-created process variables whenever possible.
- Use c8volt commands to create test data before falling back to direct Camunda APIs.
- Record whether evidence came from seeded data or pre-existing cluster data.

Allowed evidence classes:

- `seeded`: created by this suite run
- `preexisting`: already present before this suite run
- `mutated`: changed by this suite run
- `retained`: intentionally left behind
- `cleanup_failed`: cleanup was attempted but did not complete

Cleanup is best effort unless the command under test is itself the cleanup behavior. A cleanup failure should be reported with evidence, but it must not hide the actual command coverage result.

## Command Coverage Contract

Use the live Cobra command contract as the inventory source. A coverage manifest should be generated or validated from `capabilities --json`.

Every command node must have an explicit integration coverage entry. For each leaf command, cover:

- command path
- aliases
- required flags
- every command-local flag
- relevant persistent flags
- supported output modes
- success behavior
- local validation failures
- remote failure or not-found behavior where practical
- destructive confirmation or automation behavior where applicable
- version support for Camunda 8.7, 8.8, and 8.9 where the command claims version-sensitive behavior

Parent and grouping commands must at least cover help/discovery behavior and no-argument behavior.

The suite should be organized by command family:

- `get`
- `run`
- `deploy`
- `embed`
- `update`
- `delete`
- `cancel`
- `walk`
- `expect`
- `resolve`
- `ops analyse`
- `ops execute`
- `ops purge`
- `ops repair`
- `config`
- `capabilities`
- `version`

High-level ops commands must each have explicit scenarios, especially:

- `ops analyse slow-process-instances`
- `ops execute retention-policy`
- `ops execute smoke-test`
- `ops purge all-process-definitions`
- `ops purge orphan-process-instances`
- `ops purge process-instances-with-incidents`
- `ops repair incident`
- `ops repair process-instance`

## Data Creation Preference

Create test data through c8volt commands whenever possible:

1. `embed list`
2. `embed deploy`
3. `deploy process-definition`
4. `run process-instance`
5. `update process-instance`
6. `update job`
7. `cancel`, `delete`, `resolve`, and `ops` workflows

Direct Camunda API setup is allowed only when no c8volt command can create the required state. Each direct API fallback must be recorded in a proposal report as a candidate c8volt command addition or command extension.

The proposal report should include:

- required test state
- API endpoint or operation used
- reason no existing c8volt command could create it
- proposed command or flag extension
- affected Camunda versions
- whether the need is test-only or operator-useful

## Embedded Process Definition Preference

Use embedded c8volt BPMN models before introducing external BPMN files.

If existing embedded models do not cover a required behavior, do not modify them in place. Record a proposal for a new embedded process definition instead.

Common fixture gaps to report when encountered:

- listener jobs for `--with-listeners`
- BPMN error boundary flows for job error handling
- richer variable shapes for variable search and filtering
- long-running and completed instances for duration and retention workflows
- controlled incident/job states that cannot be produced by existing models

The proposal should name the required BPMN behavior, the command coverage it enables, and the Camunda versions that need version-prefixed fixtures.

## Dirty-Cluster Assertions

Assertions must tolerate unrelated cluster data.

Use these patterns:

- Assert valid output shape instead of exact global counts.
- Assert run-owned resources are present when the scenario created them.
- Use impossible selectors for no-match scenarios.
- Use specific keys for targeted destructive tests.
- Use dry-run before broad destructive workflows when the command supports it.
- Allow broad destructive workflows to touch pre-existing data when that is the behavior being covered.

Avoid these patterns:

- assuming search results are empty before setup
- asserting exact total counts across the whole cluster
- deleting all cluster data as a precondition
- relying on process-definition version numbers being stable
- relying on latest deployment belonging to the suite unless it was just created and selected by key

## Examples And Documentation Validation

The suite must validate examples from command help and generated CLI documentation.

Example validation should:

- extract examples from Cobra command metadata and generated CLI pages
- substitute placeholders with run-created keys, resource IDs, or embedded files
- execute read-only examples directly
- execute mutating examples only against disposable targets
- verify JSON examples produce valid JSON where promised
- verify keys-only examples print one key per line and nothing else

Mutating examples are allowed, but they must be visibly marked with a warning about potentially dangerous actions. The example validation should fail when a destructive example is not clearly marked.

## Go Test Structure

The Go integration suite should live outside unit-test packages and use a build tag:

```go
//go:build integration
```

Recommended package structure:

```text
integration/cli/
  all_commands_test.go
  cancel_test.go
  config_test.go
  delete_test.go
  deploy_embed_run_test.go
  examples_test.go
  expect_resolve_test.go
  get_test.go
  ops_analyse_test.go
  ops_execute_test.go
  ops_purge_test.go
  ops_repair_test.go
  update_test.go
  walk_test.go
  harness_test.go
```

`TestMain` should build one binary and run all cases as subprocesses. Tests should exercise the same CLI path an operator uses, including config resolution, stdout, stderr, prompts, exit codes, and output modes.

Recommended execution:

```sh
go test -tags=integration ./integration/cli -count=1 -timeout=60m
```

## Reporting Requirements

Each run should write a reusable evidence directory outside generated docs:

- command inventory snapshot
- profile and version summary
- run marker
- per-command stdout/stderr
- per-command exit code
- created keys and resource IDs
- mutated pre-existing keys when known
- cleanup results
- API fallback proposal report
- embedded BPMN proposal report
- example validation report

Reports must make it clear whether a failure is caused by product behavior, harness setup, missing fixture support, missing command support, or cluster/environment availability.

## Non-Goals

- Do not turn this rule file into public product documentation.
- Do not place generated evidence under `docs/`.
- Do not require normal unit tests, Speckit workflows, Ralph iterations, or docs generation to read this file.
- Do not treat this file as permission to mutate clusters outside the explicitly selected disposable integration target.
- Do not use this file as a reason to change command behavior without a separate accepted product requirement.
