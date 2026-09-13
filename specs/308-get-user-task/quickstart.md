# Quickstart and Validation: Get User Tasks

This guide is for validation after implementation. The planning phase does not add the command or new tests. Run commands from the repository root.

## Prerequisites

- Go/toolchain matching `go.mod` (Go 1.26, toolchain go1.26.2) and the existing project dependencies.
- For deterministic automated checks, use fake HTTP backends through existing `testx` helpers; no live Camunda installation is required for those tests.
- For optional live reads, prepare existing c8volt configuration files for 8.8, 8.9, and 8.10 and a separate 8.7 configuration for unsupported-version checks. Select a stable set of already-existing native user tasks; this guide does not create or mutate tasks.
- Use Linux or macOS for the repository's real-terminal test runner. Do not substitute pipe-only stdin or a mocked terminal predicate for terminal acceptance.

See [CLI contract](contracts/cli.md), [facade/service contract](contracts/facade-service.md), and [data model](data-model.md) for exact inputs, output fields, and state rules.

## Automated validation

Run the closest suites as their implementation slices land:

```sh
go test ./internal/services/usertask/... -count=1
go test ./c8volt/task -count=1
go test ./cmd -run 'TestGetUserTask|Test.*UserTask|Test.*Stdin.*Envelope' -count=1
go test ./cmd -run 'Test.*UserTask.*Terminal|TestGetProcessInstanceKeysOnlyPagingTerminal' -count=1
go test ./cmd -run 'Test.*HasUserTasks|Test.*UserTask|Test.*CommandContract|Test.*Capabilities' -count=1
```

Task generation should use names matching these patterns for new command tests; confirm that each targeted run executes tests rather than reporting no matches. Preserve existing legacy resolver tests in each adapter and add facade/stub compatibility checks as needed.

Required fixture matrix:

| Scenario | Expected evidence |
| --- | --- |
| Single/repeated/comma keys; implicit and explicit stdin | Identical ordered unique task collection; flags precede stdin |
| Malformed flag/stdin key, explicit empty dash, invalid state, conflicts | Established invalid-input outcome; zero native task requests |
| Mixed existing/missing keys; denied access | Failure, no successful partial collection |
| Selected tenant and authorized explicit key outside it | Search request carries effective tenant; GET retains actual tenant and does not post-filter |
| Each filter and combined filters on 8.8/8.9/8.10 | Correct backend predicate and response mapping |
| Small batch, more pages, mid-page limit | Expected keys once, result count at most limit, no unnecessary next-page request |
| Empty/short intermediate page with continuation | Later tasks returned; no empty-result success or paging question at intermediate page |
| Exact total and capped total | Exact total returned; capped traversal reaches known population, no task accumulation needed for count |
| Repeated cursor, overflow, backend failure after earlier page | Failure with no complete-success JSON or numeric total |
| Empty human/JSON/keys/quiet and combinations | `found: 0` plus newline, one envelope with empty array, zero keys bytes, quiet human zero bytes |
| Automation/auto-confirm/JSON | No interactive prompt; valid output contract |
| Explicit limit with terminal stdin | Eligible prompts may occur before the boundary; reaching the limit stops without another prompt |
| 8.7 native reads and existing has-user-tasks workflows | New methods unsupported; previous resolver outcomes and fallback unchanged |

Capture stdout and stderr separately. For JSON, decode one envelope and require EOF after whitespace. For keys and empty output, compare exact bytes. Assert request counts, no mutation calls, and no added reads caused by rendering.

For paging tests use `testx.NewCmdTerminalRunner` with configured/inherited stderr destinations and redirected stdout. Exercise yes/yes, decline, EOF, empty search, sparse intermediate pages, keys-only output, auto-confirm, and automation. Prompt text must only reach stderr; each stdout key remains one complete line. Account for already emitted human/keys results when a later request fails.

## Optional live read smoke checks

Set paths and existing task identifiers appropriate to your environment; the assignments below are placeholders to replace:

```sh
export C8VOLT_TASK_CONFIG=/path/to/c8volt-v810.yaml
export C8VOLT_TASK_KEY=2251799815391233
export C8VOLT_TASK_KEY_2=2251799815391234
export C8VOLT_TASK_PI_KEY=2251799813711967
make build
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" get user-task --help
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" get ut --key "$C8VOLT_TASK_KEY"
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" get uts --key "$C8VOLT_TASK_KEY,$C8VOLT_TASK_KEY_2"
printf '%s\n' "$C8VOLT_TASK_KEY" "$C8VOLT_TASK_KEY_2" | ./bin/c8volt --config "$C8VOLT_TASK_CONFIG" get user-tasks
printf '%s\n' "$C8VOLT_TASK_KEY" | ./bin/c8volt --config "$C8VOLT_TASK_CONFIG" get ut -
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" --json get ut --pi-key "$C8VOLT_TASK_PI_KEY" --limit 20
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" get ut --assignee alice --state created --limit 20
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" get ut --candidate-group accounting --total
```

Expect compact task rows for human commands, a single collection envelope for JSON, and a numeric count for total. Run equivalent reads with the 8.8 and 8.9 configs. A valid new read with the 8.7 config must clearly fail as unsupported.

Choose a filter known to match nothing in your environment and compare modes:

```sh
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" get ut --element-id c8volt_nonexistent_validation_element
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" --json get ut --element-id c8volt_nonexistent_validation_element
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" --keys-only get ut --element-id c8volt_nonexistent_validation_element
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" --quiet --json get ut --element-id c8volt_nonexistent_validation_element
```

For manual terminal paging, use a selection with more than one page and no overall limit. From real terminal stdin, redirect stdout and stderr independently:

```sh
./bin/c8volt --config "$C8VOLT_TASK_CONFIG" --keys-only get ut --batch-size 1 > /tmp/c8volt-user-task-keys.txt 2> /tmp/c8volt-user-task-prompts.txt
```

The prompt is in the stderr file, so it may not be visible in the terminal. Automated terminal-runner tests are the authoritative repeatable check for continuation and EOF behavior. Never infer correctness from pipe-only tests.

## Final implementation gates

Run `gofmt` on touched Go files and inspect command declaration ownership. Update help, metadata, examples, and README, then:

```sh
make docs-content
git diff --check
make test
```

Verify the generated reference contains `get user-task` and its aliases/examples and that existing command docs remain accurate. All required checks must pass before accepting or committing implementation. Record any environment limitation explicitly; a missing test run is not a pass.
