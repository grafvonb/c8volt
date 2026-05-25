# Camunda 8.8 Mainstream Stability Validation Prompt

Use this prompt to run a real integration stability session for the c8volt
Camunda 8.8 variant against a caller-provided local Camunda 8.8 development
cluster. This is a runtime validation prompt, not a documentation/example
rewrite prompt.

Before using this prompt, follow `specs/prompts/AGENTS.md`. In particular, do
not treat files under `specs/prompts/` as release-change source material.

```text
Run a deep stability validation of c8volt's Camunda 8.8 behavior against the
real local Camunda 8.8 development cluster. Mirror the breadth of the previous
v89 real-cluster validation where possible, but correct expectations for C88
capabilities and unsupported C88 command paths.

Inputs:
- Target Camunda version: 8.8 only
- Target fixture prefix: C88
- Local cluster: caller's real local Camunda 8.8 development cluster
- Required config file: `config.yaml`
- Required profile: `kind-camunda-platform-local-c88`
- Optional build output path: /tmp/c8volt-c88-stability
- Scratch directory for substituted commands, logs, reports, discovered keys,
  and cleanup state: /tmp/c8volt-c88-stability-work

Goal:
Prove that c8volt is stable for the mainstream customer target, Camunda 8.8,
using real local cluster calls. Validate the same command families and workflows
that were exercised in the v89 session, but interpret C88 correctly:
unsupported C88 operations are expected unsupported outcomes when they fail
before mutation with clear unsupported-version diagnostics. Supported C88
operations must succeed against real fixture data.

Default mode:
This is a stability validation and repair task for runtime behavior and test
expectations. Do not update README examples, docs examples, generated CLI docs,
or command help examples. If a public example mentions C89, a v89-only fixture,
or a v89-only command path, create a private C88-substituted command in the
scratch directory and run that command shape against the C88 cluster. Keep the
public example unchanged unless the caller explicitly asks for documentation
work in a separate follow-up.

Hard boundaries:
1. Do not edit README.md, docs/**/*.md, generated CLI docs, command help text,
   or demos/vhs/**/*.tape as part of this prompt run.
2. Do not make examples C88-specific. The session may privately substitute
   fixture names, keys, config paths, or flags, but those substitutions must
   remain in scratch logs only.
3. Do not use the repository's checked-in examples as mutable test fixtures.
   Extract their command shapes if useful, then rewrite them privately in
   scratch space for C88.
4. Do not change the user's normal config file in place. Use `config.yaml` with
   `--profile kind-camunda-platform-local-c88` for every real-cluster command.
5. Do not treat unsupported C88 operations as product regressions when the
   command fails before mutation with a clear unsupported-version error.
6. Do not leave long-running commands alive. Stop or bound any command that
   floods output, scans too broadly, or waits too long for a stability session.
7. Prefer isolated C88 fixture data for destructive or mutating validation.
   Clean up run-owned process instances when the command surface supports it.
8. Mutating commands must be executed normally when the matrix command is not
   explicitly a `--dry-run` command. A dry-run can be run as an additional
   preview, but it does not replace the real mutation check.
9. Process-definition deletion is not a required success path on C88 for generic
   delete/purge process-definition commands. C88 smoke-test execution is the
   exception: run the smoke test normally and treat any unsupported cleanup
   failure as a product stability issue or C88 expectation bug to investigate,
   not as a reason to switch the smoke test to `--no-cleanup`.

Required temporary configuration workflow:
1. Create the scratch directory:
   `/tmp/c8volt-c88-stability-work`.
2. Build a temporary validation binary:
   `GOCACHE=/tmp/c8volt-gocache go build -o /tmp/c8volt-c88-stability .`
3. Use exactly this config/profile selector for every real-cluster command:
   `--config ./config.yaml --profile kind-camunda-platform-local-c88`
4. Do not create, copy, or edit a replacement config file unless the command
   under test is explicitly a config-writing command. Runtime test adaptation
   must happen in the scratch command matrix by changing fixture names,
   discovered keys, and expected outcomes, not by changing repository examples
   or the checked-in config.
5. Before running the command matrix, confirm the target really is C88:
   - `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 version`
   - `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 config validate`
   - `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 config test-connection`
   - `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 config test-connection --json`
   Stop if the effective config is not targeting the intended local Camunda 8.8
   cluster, if the version gate reports a different minor version, or if the
   cluster is unhealthy.

Required private C88 substitution workflow:
1. Extract the v89 validation command matrix or prior v89 session checklist if
   available from local notes, terminal history, prompt artifacts, or the
   caller's supplied material.
2. Build a private C88 matrix in scratch space. For every command shape:
   - Replace C89 fixture names with C88 fixture names only in the scratch
     command list.
   - Replace public placeholders with real keys discovered from the C88 cluster.
   - Replace process-definition deletion expectations with C88 unsupported
     expectations for generic delete/purge process-definition commands.
   - Keep normal smoke-test execution as a required real success path.
   - Add `--config ./config.yaml --profile kind-camunda-platform-local-c88` to
     every command.
   - Add bounds such as `--limit 5` where real cluster output could be large.
3. Discover fixture availability with:
   `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 embed list`
4. Use C88 fixtures for setup, for example:
   - `processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn`
   - C88 user-task, service-task, incident, and variable fixtures that exist in
     `embed list`
5. If a C89 fixture has no C88 equivalent, do not edit docs or examples.
   Mark the workflow as "not C88-covered" and explain whether that is expected
   or a stability gap.

Required command-family coverage:
Validate the C88-supported parts of these command families against real cluster
state, using private fixture data wherever practical:
- `version`
- `capabilities`
- `config validate`, `config show`, `config test-connection`
- `get cluster version`, `get cluster topology`, and license behavior when
  available
- `embed list`, `embed deploy`, and fixture run paths
- `run process-instance`
- `get process-definition` including latest/stat behavior supported by C88
- `get process-instance` by key, by state, with variables, with incidents, and
  with user-task lookup/fallback behavior where fixture data exists
- `walk process-instance`
- `cancel process-instance`
- `delete process-instance` where C88 supports the workflow
- `get incident` and `resolve incident` where supported
- `get job` and `update job` where supported
- variable update paths supported by C88
- user task lookup paths supported by C88
- tenant-aware paths when the local C88 config uses a tenant
- ops workflows, including normal `ops execute smoke-test`
- dry-run and automation/JSON modes for representative mutating workflows, plus
  the corresponding real mutation when the workflow is not explicitly dry-run

Required unsupported-operation expectations:
Create explicit expected-unsupported checks for command paths that are valid
c8volt commands but not supported by Camunda 8.8 or by c8volt's C88 backend.
These checks pass only when the command fails before mutation and the diagnostic
is clear. Include at least:
- process-definition deletion when it requires the C89 resource deletion path
- `ops purge all-process-definitions` real execution if it depends on
  process-definition deletion
- any other command discovered during the matrix that is intentionally C89-only

Required smoke-test coverage:
1. Run the C88 dry-run smoke test:
   `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 ops execute smoke-test --dry-run`
2. Run the C88 real smoke test normally:
   `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 ops execute smoke-test -y`
3. Verify it deploys/runs/walks the C88 fixture and completes with the correct
   normal C88 outcome.
4. Do not change public smoke-test examples or help output to C88-specific
   forms.
5. If normal C88 smoke-test execution fails because it reaches unsupported
   process-definition deletion, classify that as a smoke-test product stability
   issue to fix or as a C88-specific cleanup expectation bug. Do not paper over
   it by switching the required real smoke-test run to `--no-cleanup`.

Required execution discipline:
1. Keep a scratch run log with command, purpose, substituted values, exit code,
   and observed outcome.
2. For each mutating command, run a dry-run or preview first when available, but
   always run the real mutation too unless the matrix item is explicitly
   labeled dry-run-only.
3. For each destructive command, use isolated fixture-created resources and
   perform a post-check that proves only intended resources were affected.
4. For every unsupported C88 command, prove that the command did not perform
   the unsupported mutation.
5. If a command fails unexpectedly, classify it as:
   - test setup/config problem
   - fixture availability problem
   - C88 unsupported expected behavior
   - product stability bug
   - dirty-cluster interference
   - documentation/example mismatch that must not be edited in this run
6. Product stability fixes are allowed only when needed to make C88-supported
   behavior correct. Keep them scoped, add or update automated tests, and do not
   edit examples/docs/help.

Required automated validation after any code or test change:
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./c8volt/process ./c8volt/ops -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processinstance/v88 ./internal/services/processdefinition/v88 ./internal/services/resource/v88 ./internal/services/job/v88 ./internal/services/incident/v88 ./internal/services/usertask/v88 ./internal/services/variable/waiter -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/ops -count=1`
- `GOCACHE=/tmp/c8volt-gocache go test ./... -count=1` before final report when
  product behavior changed

Suggested investigation commands:
- `rg -n -- 'C89_|C88_|8\\.9|8\\.8|delete process-definition|purge all-process-definitions|smoke-test' README.md docs cmd internal -g '*.md' -g '*.go'`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 capabilities --json`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 config show --json`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 config test-connection --json`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 embed list`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 get cluster version`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 get cluster topology`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 get pd --latest --limit 5`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 run pi -b C88_MultipleSubProcessesParentProcess`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 get pi --state active --limit 5`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 get pi --with-vars --limit 5`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 get pi --with-incidents --limit 5`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 get incident --state active --limit 5`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 ops execute smoke-test --dry-run`
- `/tmp/c8volt-c88-stability --config ./config.yaml --profile kind-camunda-platform-local-c88 ops execute smoke-test -y`

Final output expectations:
- State that `config.yaml` profile `kind-camunda-platform-local-c88` was used
  and confirm no repository examples/docs/help were modified.
- State the cluster version, active profile, base URL class, tenant context, and
  fixture prefix validated.
- List the command families covered with real C88 calls.
- List supported C88 workflows that passed.
- List expected-unsupported C88 workflows and the exact diagnostic pattern that
  proved they failed before unsupported mutation.
- List any product stability bugs found, fixes made, and tests added.
- List any docs/example mismatches observed without editing them.
- Report cleanup performed and any intentionally retained run-owned resources.
- Report all automated tests run and their pass/fail results.
```
