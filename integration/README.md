# c8volt Release Integration Suites

This folder contains release-oriented, real-cluster integration validation for
c8volt. The suites complement unit tests by exercising the built CLI against a
local Camunda 8.8 or 8.9 cluster through the same commands an operator uses.

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
Later command-family slices are destructive and should only target disposable
clusters.

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

## Outputs

Each run writes:

- `report.md`: grouped release validation report with evidence links.
- `progress.tsv`: append-only progress log for resume/rerun comparison.
- `summary.env`: shell-readable summary values such as run id and binary path.
- `logs/*.stdout` and `logs/*.stderr`: command evidence.
- `data/*.keys`: generated process-instance keys and other dynamic data.
- `reports/*`: command-generated reports, for example smoke-test reports.

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
