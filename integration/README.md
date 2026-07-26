# c8volt Release Integration Suites

This folder contains release-oriented, real-cluster integration validation for
c8volt. The suites complement unit tests by exercising the built CLI against
local disposable Camunda clusters through the same commands an operator uses.

These assets are intentionally separate from product implementation, Speckit
feature artifacts, docs examples, and Ralph PRDs. See `integration/AGENTS.md`
before using this folder in an agent run.

## What The Suites Check

- The configured cluster is reachable and matches the expected Camunda minor.
- c8volt can discover its own command contract, help tree, config, cluster
  version, topology, and capabilities.
- Embedded C88/C89 fixtures can be deployed and used to generate disposable
  run-owned data in an already dirty cluster.
- Daily ops workflows work end to end:
  - process-definition discovery
  - process-instance creation, lookup, paging, variable search, variable update
  - walk, expect, cancel, delete
  - incident/job discovery where fixture data exists
  - `ops execute`, `ops purge`, and `ops repair` command families
- Mutating workflows run preview first where supported, then real execution with
  full force/auto-confirm behavior.
- Unsupported-version behavior is explicit, especially C88 process-definition
  deletion and C89-only purge paths.
- UX findings are collected: wording, grammar, aliases, confirmation behavior,
  verbosity, rhythm, bounded output, and automation friendliness.

## Structure

- `AGENTS.md`: guardrails for future agent runs.
- `README.md`: this usage guide.
- `prompts/camunda-88-release-integration-suite.md`: C88 release validation
  prompt.
- `prompts/camunda-89-release-integration-suite.md`: C89 release validation
  prompt.
- `scripts/run-c88-suite.sh`: wrapper for the C88 suite.
- `scripts/run-c89-suite.sh`: wrapper for the C89 suite.
- `scripts/run-suite.sh`: shared executable suite driver.
- `scripts/lib/suite-lib.sh`: reporting, progress, and command helpers.
- `cli/`: build-tagged Go all-command integration suite.
- `assets/all-command-go-integration-rules.md`: context-only rules for a
  destructive Go suite that covers the complete command tree; not normal
  Speckit or Ralph implementation context.
- `assets/command-matrix.md`: intended command-family coverage map.
- `assets/ux-review-checklist.md`: reusable UX review checklist applied while
  reading suite output.

## Quick Start

Run the Go all-command integration suite inventory slice:

```sh
go test -tags=integration ./integration/cli -run TestCommandInventory -count=1 -timeout=10m
```

The Go suite uses the default local c8volt configuration under
`$HOME/.config/c8volt`. It must not be pointed at a generated private config.
Command-family slices are destructive and should only target disposable
clusters. To select specific default-local profiles, use a comma-separated
profile list:

```sh
C8VOLT_IT_PROFILES=kind-camunda-platform-local-c88,kind-camunda-platform-local-c89 \
go test -tags=integration ./integration/cli -count=1 -timeout=60m
```

Common Go suite commands:

```sh
go test -tags=integration ./integration/cli -run '^$' -count=1
go test -tags=integration ./integration/cli -run 'TestProfiles|TestReadOnlySmoke' -count=1 -timeout=10m
go test -tags=integration ./integration/cli -run TestSeededData -count=1 -timeout=20m
go test -tags=integration ./integration/cli -run 'TestGetFamily|TestWalkFamily|TestOps' -count=1 -timeout=60m
go test -tags=integration ./integration/cli -run TestExamples -count=1 -timeout=20m
go test -tags=integration ./integration/cli -count=1 -timeout=60m
go test ./integration/cli -count=1
```

Family targets are available through `make`:

```sh
make integration-test IT_GO_TEST_FLAGS=-v
make integration-cli-get
make integration-cli-walk
make integration-cli-update
make integration-cli-cancel
make integration-cli-delete
make integration-cli-expect-resolve
make integration-cli-deploy-embed-run
make integration-cli-ops-analyse
make integration-cli-ops-execute
make integration-cli-ops-purge
make integration-cli-ops-repair
```

Volume targets are slower and intentionally destructive. They seed or discover
their own data in clean or dirty disposable clusters, write separate
`volume-*.json` evidence files, and keep the baseline family targets quick.
The currently implemented volume targets are:

```sh
make integration-test-volume IT_GO_TEST_FLAGS=-v IT_VOLUME_TIMEOUT=90m
make integration-cli-get-volume IT_GO_TEST_FLAGS=-v
make integration-cli-walk-volume IT_GO_TEST_FLAGS=-v
make integration-cli-update-volume IT_GO_TEST_FLAGS=-v
make integration-cli-cancel-volume IT_GO_TEST_FLAGS=-v
make integration-cli-delete-volume IT_GO_TEST_FLAGS=-v
make integration-cli-expect-resolve-volume IT_GO_TEST_FLAGS=-v
make integration-cli-deploy-embed-run-volume IT_GO_TEST_FLAGS=-v
make integration-cli-ops-analyse-volume IT_GO_TEST_FLAGS=-v
```

Planned volume target names remain reserved for the other families. Until their
matching `TestVolume*Family` scenarios are implemented, those Make targets exit
with a clear not-implemented message instead of reporting a false pass.

Real-state targets are a separate Camunda 8.9 foundation layer. They are more
specific than volume targets: they must prove actual jobs, incidents with
related jobs, listener state, BPMN error jobs, retention candidates, destructive
post-state, mixed failures, and gap-boundary tracking against real cluster state.
The target names are reserved first so scripts can depend on the stable entry
points while each slice is implemented:

```sh
make integration-test-real-state IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
make integration-cli-real-state-gaps IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-jobs IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-incidents IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-listeners IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-bpmn-error IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-retention IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
make integration-cli-real-state-destructive IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
```

To run every baseline, volume, and real-state integration slice in sequence:

```sh
make integration-test-all IT_GO_TEST_FLAGS=-v IT_VOLUME_TIMEOUT=90m IT_REAL_STATE_TIMEOUT=90m
```

Make asks for confirmation before running integration slices because they may
mutate real Camunda cluster state. A single `make` invocation asks once, even
when an aggregate target expands to many slices. Automation can skip the prompt
with `IT_CONFIRM=0`:

```sh
make integration-test-real-state IT_CONFIRM=0 IT_GO_TEST_FLAGS=-v IT_REAL_STATE_TIMEOUT=90m
```

The gaps target is non-destructive and validates that `gaps.md` plus the
coverage matrix keep missing setup and fixture prerequisites in spec-owned
artifacts. Other real-state targets create or discover suite-owned data and now
cover jobs, incidents, listener state, BPMN error dry-runs, retention,
destructive post-state, incident-selected purge, repair, and mixed-failure
branches. Remaining process-definition purge, orphan purge, durable standalone
resolve, and repair-specific mixed-failure branches stay visible in
`specs/257-c89-real-state-integration/`.
Except for `integration-cli-real-state-gaps`, real-state targets are
destructive and must only be run against disposable Camunda 8.9 clusters
selected from the default local c8volt configuration.

Pass extra `go test` flags with `IT_GO_TEST_FLAGS`. For example, use `-v`
when you want scenario-level command logs in addition to the evidence files.
The Make targets automatically set `C8VOLT_IT_VERBOSE=1` when `-v` is present:

```sh
make integration-cli-get IT_GO_TEST_FLAGS=-v
make integration-cli-ops-repair IT_GO_TEST_FLAGS='-v -failfast'
make integration-cli-get-volume IT_GO_TEST_FLAGS=-v IT_VOLUME_TIMEOUT=90m
```

Verbose integration output includes each c8volt subprocess scenario, arguments,
exit code, duration, stdout/stderr evidence paths, and compact output snippets.
Set `C8VOLT_IT_VERBOSE=1` directly when running raw `go test -v` commands.

When a slice validates help or generated-documentation examples, destructive
examples must stay executable against disposable test clusters and must be
marked with a warning in the source help text or generated docs. Integration
tests may prove those examples, but missing setup commands or embedded BPMN
assets belong in spec-owned gap artifacts rather than runtime proposal files.

Run C88 against the C88 profile in `config.yaml`:

```sh
integration/scripts/run-c88-suite.sh
```

Run C89 against the C89 profile in `config.yaml`:

```sh
integration/scripts/run-c89-suite.sh
```

The wrappers use these defaults:

| Suite | Target version | Fixture prefix | Default profile |
| --- | --- | --- | --- |
| C88 | `8.8` | `C88` | `kind-camunda-platform-local-c88` |
| C89 | `8.9` | `C89` | `kind-camunda-platform-local-c89` |

Override configuration with environment variables:

```sh
C8VOLT_IT_CONFIG=./config.yaml \
C8VOLT_IT_PROFILE=kind-camunda-platform-local-c89 \
C8VOLT_IT_VOLUME_COUNT=25 \
integration/scripts/run-c89-suite.sh
```

Useful variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `C8VOLT_IT_CONFIG` | `./config.yaml` | Config file passed to c8volt. |
| `C8VOLT_IT_PROFILE` | wrapper-specific | Profile passed to c8volt. Empty means use config default. |
| `C8VOLT_IT_BIN` | built into `/tmp` | Existing c8volt binary to test. |
| `C8VOLT_IT_BUILD` | `1` | Set `0` to skip `go build` when `C8VOLT_IT_BIN` is set. |
| `C8VOLT_IT_WORKDIR` | `/tmp/c8volt-it-<suite>-<timestamp>` | Report/progress/log directory. |
| `C8VOLT_IT_VOLUME_COUNT` | `12` | Number of user-task process instances created for paging/batch coverage. |
| `C8VOLT_IT_FULL_FORCE` | `1` | Run broad destructive checks that may modify/delete existing cluster resources. |
| `C8VOLT_IT_KEEP_DATA` | `0` | Set `1` to skip final cleanup of run-created data. Broad full-force checks may still mutate existing data when `C8VOLT_IT_FULL_FORCE=1`. |

Go suite variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `C8VOLT_IT_PROFILES` | active profile from default local config | Comma-separated profile names selected from the default local c8volt config. |
| `C8VOLT_IT_BIN` | built into the workdir | Existing c8volt binary to test when `C8VOLT_IT_BUILD=0`. |
| `C8VOLT_IT_BUILD` | build enabled | Set `0` with `C8VOLT_IT_BIN` to skip building the binary. |
| `C8VOLT_IT_WORKDIR` | temp directory | Evidence directory for reruns and artifact review. |
| `C8VOLT_IT_VOLUME_COUNT` | `12` | Number of suite-owned process instances created by Go volume targets for paging, limit, and filtering coverage. |
| `IT_VOLUME_TIMEOUT` | `90m` | Make-level timeout used by `integration-cli-*-volume` targets. |
| `IT_REAL_STATE_TIMEOUT` | `90m` | Make-level timeout used by `integration-cli-real-state-*` targets. |
| `IT_CONFIRM` | `1` | Set `0` to skip the Make-level mutation confirmation prompt. |

## Outputs

The shell suites write:

- `report.md`: grouped release validation report with evidence links.
- `progress.tsv`: append-only progress log for resume/rerun comparison.
- `summary.env`: shell-readable summary values such as run id and binary path.
- `logs/*.stdout` and `logs/*.stderr`: command evidence.
- `data/*.keys`: generated process-instance keys and other dynamic data.
- `reports/*`: command-generated reports, for example smoke-test reports.

The Go all-command suite writes evidence under `C8VOLT_IT_WORKDIR` or a temp
directory:

- `summary.md`: high-level run summary and expected evidence files.
- `run.json`: run marker, binary path, workdir, and selected profiles.
- `inventory.json` and `coverage.json`: live command contract and manifest drift report.
- `coverage-<family>.json`: per-family manifest entries and subprocess records.
- `profiles.json` and `readonly-smoke.json`: readiness and smoke evidence.
- `examples.json`: help/generated-doc example validation results.
- `proposals-command.json` and `proposals-embedded-bpmn.json`: setup gap proposals.
- `volume-<family>.json`, `volume-data-<family>.json`,
  `volume-progress-<family>.json`, `volume-pipelines-<family>.json`, and
  `volume-ops-reports-<family>.json`: opt-in volume-suite evidence.
- `real-state-<family>.json`, `real-state-data-<family>.json`,
  `real-state-progress-<family>.json`, and
  `real-state-ops-reports-<family>.json`: opt-in Camunda 8.9 real-state
  evidence.
- `logs/`: per-command stdout and stderr evidence.
- `data/`: seeded data, selected keys, resource IDs, and substitution evidence.

The suite exits non-zero when any required case fails or any expected failure
does not fail in the expected way. It keeps running after individual failures so
the final report is useful for fixing a batch of issues.

## Rerun Workflow

1. Run the suite and inspect `report.md`.
2. Fix product or docs issues separately from the harness.
3. Rerun the same suite. To compare in the same directory, reuse
   `C8VOLT_IT_WORKDIR`.
4. Review which command groups changed from `FAIL` to `PASS` in `progress.tsv`
   and the new report.

Example:

```sh
export C8VOLT_IT_WORKDIR=/tmp/c8volt-it-c89-release
integration/scripts/run-c89-suite.sh
# apply fixes
integration/scripts/run-c89-suite.sh
```

## Safety Model

The suite is designed for a dirty, disposable local Camunda cluster. Existing
process definitions, process instances, incidents, jobs, resources, and other
cluster state may already be present before the run. The suite must not require
a clean environment.

When the suite runs, it has full permission to mutate, cancel, delete, resolve,
repair, or purge existing resources without asking for additional confirmation.
It previews risky mutations first where the command supports dry-run, then uses
`--auto-confirm`, `--automation`, `--force`, and broad selection scopes where
that is the realistic release check. The report records evidence for both
run-owned data and any broad dirty-cluster cleanup/purge commands.

Do not run these suites against shared, production, or non-disposable clusters.
