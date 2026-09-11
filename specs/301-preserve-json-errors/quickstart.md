# Quickstart: Validate JSON Error Envelopes

## Prerequisites

Run from the repository root on `codex/301-preserve-json-errors`, after implementing the plan. Use the repository Go 1.26/toolchain go1.26.2 configuration, available dependencies, and an environment permitting subprocesses and loopback HTTP fixtures. A live cluster and credentials are not required.

The proposed new test prefix is `TestCommandErrorEnvelope`. It must exist after implementation; a “no tests to run” result is not validation evidence.

## Targeted command checks

```sh
go test ./cmd -list 'TestCommandErrorEnvelope'
go test ./cmd -run '^TestCommandErrorEnvelope' -count=1
go test ./cmd -run 'TestGetCluster|TestGetProcessDefinition|TestEmbedList|TestCommandCapability|TestOutputModes' -count=1
```

The first test group must include all corrected paths and all 12 stdin callers, with separate streams and exit-code checks. The regression group checks existing cluster, process-definition, embedded-list, and capability behavior. See the complete matrix in [plan.md](plan.md) and assertions in [contracts/cli-error-envelope.md](contracts/cli-error-envelope.md).

HTTP failure fixtures should use the existing test configuration helpers and local test servers. Cluster version/topology use the topology fixture and license uses the license fixture; retain existing adapter-specific paths from adjacent tests. Do not substitute a connection/bootstrap failure for the intended runtime error. Embedded failures use the command-local listing seam. No real mutations should be submitted; validation fixtures must assert zero post-validation requests.

## Manual command smoke check

Build a standalone executable so the observed exit status belongs to the CLI rather than `go run`:

```sh
check_dir=$(mktemp -d)
go build -o "$check_dir/c8volt" .
cat > "$check_dir/config.yaml" <<'CONFIG'
app:
  camunda_version: "8.8"
auth:
  mode: none
apis:
  camunda_api:
    base_url: "http://127.0.0.1:1"
CONFIG
```

These deliberate validation failures should finish before contacting that address:

```sh
printf '%s\n' 'filter: state=active' | "$check_dir/c8volt" --config "$check_dir/config.yaml" --json get process-instance - > "$check_dir/stdout.json" 2> "$check_dir/stderr.txt"
result_code=$?
cat "$check_dir/stdout.json"
cat "$check_dir/stderr.txt"
printf 'exit=%s\n' "$result_code"
```

Expected: one `invalid` / `invalid_input` envelope for `get process-instance`, the specific keys-only guidance, no repeated failure on stderr, and the established invalid-arguments exit code. The command is intentionally unsuccessful; if using a shell with automatic exit on error, run these checks in a shell that allows capturing the status.

```sh
"$check_dir/c8volt" --config "$check_dir/config.yaml" --json --no-err-codes get process-definition --xml --key 123 > "$check_dir/stdout.json" 2> "$check_dir/stderr.txt"
result_code=$?
cat "$check_dir/stdout.json"
printf 'exit=%s\n' "$result_code"
```

Expected: one invalid-input envelope explaining the existing XML/JSON incompatibility, no XML body or retrieval, and exit status zero. Repeat without `--no-err-codes` to verify the classified nonzero exit, and without `--json` using invalid stdin to verify human stderr reporting and zero stdout bytes. The subprocess suite, rather than these smoke checks, proves exact EOF, classifications, and all corrected runtime cases.

## Completion checks

Format the touched Go files, update source help and README, then run:

```sh
make docs-content
make test
git diff --check
```

`make test` runs the full race-enabled suite. Inspect generated CLI documentation and ensure contract-support declarations, output schemas, and unrelated behavior remain unchanged. Record any failed or unavailable validation explicitly; do not treat planning completion as implementation completion.

## Iteration 1 evidence

- Baseline before production changes: `go test ./cmd -run 'TestGetCluster|TestGetProcessDefinition|TestEmbedList|TestCommandCapability|TestOutputModes' -count=1` passed in 18.845 seconds.
- TDD failure confirmed: the new US1 JSON cases decoded EOF because the direct handlers emitted no envelope before T006/T007.
- US1 focused validation: `go test ./cmd -run '^TestCommandErrorEnvelope(DeleteValidation|Stdin)' -count=1` passed, executing deletion validation and all 12 stdin caller matrices.
- Broader command regression: `go test ./cmd -count=1` passed in 37.805 seconds.

## Iteration 2 evidence

- TDD failures confirmed: cluster, process-definition retrieval/selector/search/XML validation, and embedded-list JSON cases reached EOF before the planned dispatch corrections.
- US2 focused validation: `go test ./cmd -run '^TestCommandErrorEnvelope(Cluster|ProcessDefinition|EmbedList)|TestGetCluster|TestGetProcessDefinition|TestEmbedList' -count=1` passed in 88.023 seconds.
- Full race-enabled validation: `make test` passed, including `github.com/grafvonb/c8volt/cmd` in 341.643 seconds.

## Iteration 6 evidence

- Final production-diff audit passed: `cmd/command_contract.go`, `cmd/cmd_views_contract.go`, and `c8volt/ferrors/errors.go` are unchanged from the pre-feature baseline, so schema, capability eligibility, normalization, classification, and exit policy remain intact.
- The audit confirmed that changes are limited to the documented execution paths: the 12 stdin callers retain their prior key ordering and `.Unique()` placement, corrected failures invoke the shared renderer once and terminate immediately, and bootstrap/parsing, raw-XML retrieval/write, and defensive renderer-error exclusions remain unchanged.
- Integrated US3 validation: `go test ./cmd -run '^TestCommandErrorEnvelope|TestGetCluster|TestGetProcessDefinition|TestEmbedList|TestCommandCapability|TestOutputModes' -count=1` passed in 90.470 seconds, covering corrected human/JSON paths, ordinary/suppressed exits, mode precedence, non-full fallback, and successful workflows.

## Iteration 10 evidence

- Built a standalone executable with `go build -o "$check_dir/c8volt" .` and used the documented loopback-only Camunda 8.8 configuration; no live cluster or credentials were used.
- Invalid `get process-instance` stdin in JSON mode exited 2, wrote one 275-byte `invalid` / `invalid_input` envelope for `get process-instance`, preserved the specific `--keys-only` guidance, and wrote zero stderr bytes.
- `get process-definition --xml --key 123` with JSON mode wrote one 260-byte `invalid` / `invalid_input` incompatibility envelope and zero stderr bytes; it exited 0 with `--no-err-codes` and 2 without it.
- The same invalid stdin check in human mode exited 2, wrote zero stdout bytes, and wrote exactly one stderr diagnostic containing the `--keys-only` guidance.
- `jq -s` checks confirmed exactly one JSON document in each JSON capture. The documented smoke commands and implemented `TestCommandErrorEnvelope` prefix were current; no guide command or proposed test name required reconciliation.

## Iteration 11 evidence

- `go test ./cmd -list 'TestCommandErrorEnvelope'` passed and listed the implemented error-envelope test family, including cluster, delete validation, embedded list, fallback, modes, process definition, stdin, and success coverage.
- `go test ./cmd -run '^TestCommandErrorEnvelope' -count=1` passed in 72.627 seconds.
- `go test ./cmd -run 'TestGetCluster|TestGetProcessDefinition|TestEmbedList|TestCommandCapability|TestOutputModes' -count=1` passed in 19.749 seconds.
- `make test` passed the full `go test ./... -race -count=1` suite, including `github.com/grafvonb/c8volt/cmd` in 356.853 seconds.
- `git diff --check` passed with no whitespace errors.
