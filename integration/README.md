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
- Embedded C88/C89/C810 fixtures can be deployed and used to generate disposable
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
  deletion and purge paths supported from C89 onward.
- UX findings are collected: wording, grammar, aliases, confirmation behavior,
  verbosity, rhythm, bounded output, and automation friendliness.

## Structure

- `AGENTS.md`: guardrails for future agent runs.
- `README.md`: this usage guide.
- `prompts/camunda-88-release-integration-suite.md`: C88 release validation
  prompt.
- `prompts/camunda-89-release-integration-suite.md`: C89 release validation
  prompt.
- `prompts/camunda-810-release-integration-suite.md`: C810 release validation
  prompt.
- `scripts/run-c88-suite.sh`: wrapper for the C88 suite.
- `scripts/run-c89-suite.sh`: wrapper for the C89 suite.
- `scripts/run-c810-suite.sh`: wrapper for the C810 suite.
- `scripts/run-c810-suspended.sh`: focused Bash suspended cancellation/deletion checks.
- `scripts/run-suite.sh`: shared executable suite driver.
- `scripts/lib/suite-lib.sh`: reporting, progress, and command helpers.
- `scripts/tests/run-suite_test.sh`: offline fake-CLI version-gate coverage.
- `cli/`: build-tagged Go all-command integration suite.
- `assets/all-command-go-integration-rules.md`: context-only rules for a
  destructive Go suite that covers the complete command tree; not normal
  Speckit or Ralph implementation context.
- `assets/command-matrix.md`: intended command-family coverage map.
- `assets/ux-review-checklist.md`: reusable UX review checklist applied while
  reading suite output.

## Quick Start

The existing `integration-cli-get` slice includes `TestGetFamilySuspendedState`
for suspended process-instance reads. It uses an isolated held definition and
checks the observed server patch is **8.10.4 before any fixture mutation**;
the configured release remains `8.10`. Older-minor profiles skip this scenario,
which does not count as suspension acceptance.

For a focused run on the designated disposable profile:

```sh
go test -tags=integration ./integration/cli -list '^TestGetFamilySuspendedState$'
C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 \
  go test -tags=integration ./integration/cli -run '^TestGetFamilySuspendedState$' -count=1 -v -timeout=20m
```

Require the named test to be listed before running it. The scenario compares
suspended, active and terminal owned instances, exercises state and combined
expectations, and verifies timestamp omission after resumption. Generated
suspend/resume calls are fixture setup only. Cleanup resumes owned instances
before ordinary deletion. Evidence includes `suspended-revision.json` (Git
revision, dirty status and unchanged 8.10.3 client provenance), profile and
command records, raw observations under `data/`, separate streams under `logs/`,
and `suspended-cleanup-<profile>.json`. Cleanup failures are reported separately.
Use a separate `C8VOLT_IT_WORKDIR` for each invocation.

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
make integration-test C8VOLT_IT_GO_TEST_FLAGS=-v
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
make integration-test-volume C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_VOLUME_TIMEOUT=90m
make integration-cli-get-volume C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-walk-volume C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-update-volume C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-cancel-volume C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-delete-volume C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-expect-resolve-volume C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-deploy-embed-run-volume C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-ops-analyse-volume C8VOLT_IT_GO_TEST_FLAGS=-v
```

Planned volume target names remain reserved for the other families. Until their
matching `TestVolume*Family` scenarios are implemented, those Make targets exit
with a clear not-implemented message instead of reporting a false pass.

Real-state targets support Camunda 8.9 and 8.10 without duplicating the family
suite. They are more
specific than volume targets: they must prove actual jobs, incidents with
related jobs, listener state, BPMN error jobs, retention candidates, destructive
post-state, mixed failures, and gap-boundary tracking against real cluster state.
The target names are reserved first so scripts can depend on the stable entry
points while each slice is implemented:

```sh
make integration-test-real-state C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_REAL_STATE_TIMEOUT=90m
make integration-cli-real-state-gaps C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-jobs C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-incidents C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-listeners C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-bpmn-error C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-real-state-retention C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_REAL_STATE_TIMEOUT=90m
make integration-cli-real-state-destructive C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_REAL_STATE_TIMEOUT=90m
make integration-cli-real-state-cascade C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_REAL_STATE_TIMEOUT=90m
```

The common target keeps nonmutating inventory, manifest, profile, and smoke
checks ahead of the separately confirmed example slice, whose eligible examples
may mutate a disposable target:

```sh
make integration-cli-common C8VOLT_IT_GO_TEST_FLAGS=-v
```

Camunda 8.10 batch compatibility has a focused destructive slice. It records
submission-only cancellation separately from later terminal observation,
checks cancellation and deletion accounting against independent process-instance
post-state, and holds an owned instance while observing process-definition
`DRAINING` before bounded cleanup:

```sh
C8VOLT_IT_PROFILES=dev810 make integration-cli-batch \
  C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_REAL_STATE_TIMEOUT=90m
```

The process-definition scenario accepts a valid null `batchOperation` when
Camunda defers history deletion. It requires observing `DRAINING` while an owned
instance is active, then cancels that instance and verifies the definition
disappears. Rejecting the valid null batch remains a regression failure.
Definition disappearance does not independently prove that all associated
history deletion has completed.

Logical tenant validation uses the credential-free role shape in
`assets/tenant-roles.example.json`. Each role names an existing default-local
profile; credentials and authorization configuration remain in the operator's
config. The tenant-A, tenant-B, dual-authorized, and restricted-A profiles must
all target Camunda 8.10 on the declared physical endpoint. The dual-authorized
profile deliberately keeps tenant A as its configured filter so explicit and
stdin tenant-B keys prove that filtering is not confused with authorization.

```sh
C8VOLT_IT_TENANT_ROLES=/absolute/path/to/tenant-roles.json \
C8VOLT_IT_REQUIRE_TENANTS=logical \
C8VOLT_IT_PROFILES=dev810 make integration-cli-tenants \
  C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_REAL_STATE_TIMEOUT=90m
```

The slice establishes positive owned visibility before absence assertions,
then verifies scoped cancellation leaves tenant B unchanged, restricted-A
credentials cannot plan a proven tenant-B key, dual authorization accepts
explicit/stdin tenant-B keys, and `--all-tenants` discovery and mutation remain
inside the selected physical endpoint. It also covers conflicting selectors
and single-destination restrictions. Missing roles are a named optional gap;
setting `C8VOLT_IT_REQUIRE_TENANTS=logical` makes missing or invalid role
infrastructure unsuccessful when the tenant group is checked.

Physical proof uses the optional `physical` group in the same role file. Its
two profiles must use normalized Camunda base URLs ending in
`/physical-tenants/<declared-id>/v2`, target the same logical tenant ID, and
name distinct physical IDs in the same operator-declared `clusterId`. A third
profile must use the default `/v2` route and explicitly declare which physical
tenant it is expected to equal. URL similarity does not establish shared
cluster identity; the declaration and topology readiness are separate gates.

```sh
C8VOLT_IT_TENANT_ROLES=/absolute/path/to/tenant-roles.json \
C8VOLT_IT_REQUIRE_TENANTS=physical \
C8VOLT_IT_PROFILES=dev810 make integration-cli-tenants \
  C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_REAL_STATE_TIMEOUT=90m
```

The physical scenario proves positive local visibility, absence of the other
endpoint's owned fixtures, unchanged control resources after scoped mutation,
`--all-tenants` confinement to the selected endpoint, and default-route
equivalence. It uses topology for readiness and does not treat a nondefault
status endpoint response as a gate. This is observable API routing,
authorization, and resource-separation evidence only; it does not certify
storage, backup, provisioning, or compute isolation. Missing physical roles
remain a named optional gap unless `physical` coverage is explicitly required;
physical mode also requires the complete logical proof group.

`make integration-test-all` uses the existing Make targets directly, in order:
common preflight/examples, focused batch and tenant checks, baseline, volume,
and real-state groups. Output goes straight to the terminal, including each
Go test command and group result. Make stops on the first failing group.
The nested Make invocation uses `-j1` so destructive groups do not overlap,
even when the outer invocation uses parallel Make.

Defaults are profile `c810local` and requested Camunda minor `8.10`:

```sh
make integration-test-all

# Override the selected environment:
make integration-test-all C8VOLT_IT_PROFILES=dev89 C8VOLT_IT_RELEASE_VERSION=8.9
```

Before confirmation, Make prints profiles, requested Camunda version, evidence
location, tenant requirements, build setting, and slice timeouts. Common
preflight verifies selected profiles before mutation; each family's existing
readiness checks also verify the configured and requested minor against the
observed gateway version. Baseline/volume support 8.8, 8.9 and 8.10; real-state
coverage supports 8.9 and 8.10 with version-matched fixtures.

The aggregate confirms once. For unattended execution:

```sh
make integration-test-all C8VOLT_IT_AUTOMATION=1 C8VOLT_IT_GO_TEST_FLAGS=-v
```

## Results and coverage limits

There is no separate runner, frozen coverage catalog, or aggregate release
verdict. Use normal Make/Go exit status and the existing per-group reports.
A successful invocation means its executed tests passed; skipped optional
scenarios and missing tenant infrastructure do not constitute coverage.
The job retry-update validation scenario covers only its supplied flags; it
does not establish successful bulk job completion.
Tenant roles remain optional unless explicitly required with
`C8VOLT_IT_REQUIRE_TENANTS`. Existing real-state prerequisite-gap documentation
and focused scenario reports remain available.

The gaps target is non-destructive and validates that `gaps.md` plus the
coverage matrix keep missing setup and fixture prerequisites in spec-owned
artifacts. Other real-state targets create or discover suite-owned data and now
cover jobs, incidents, listener state, BPMN error dry-runs, retention,
destructive post-state, parent/child cascade delete and process-definition
purge, incident-selected purge, repair, and mixed-failure branches. Remaining
orphan purge, durable standalone resolve, and repair-specific mixed-failure
branches stay visible in
`specs/257-c89-real-state-integration/`.
Except for `integration-cli-real-state-gaps`, real-state targets are
destructive and must only be run against disposable Camunda 8.9 or 8.10 clusters
selected from the default local c8volt configuration.

Pass extra `go test` flags with `C8VOLT_IT_GO_TEST_FLAGS`. For example, use `-v`
when you want scenario-level command logs in addition to the evidence files.
The Make targets automatically set `C8VOLT_IT_VERBOSE=1` when `-v` is present:

```sh
make integration-cli-get C8VOLT_IT_GO_TEST_FLAGS=-v
make integration-cli-ops-repair C8VOLT_IT_GO_TEST_FLAGS='-v -failfast'
make integration-cli-get-volume C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_VOLUME_TIMEOUT=90m
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

Run C810 against the C810 profile in `config.yaml`:

```sh
integration/scripts/run-c810-suite.sh
```

These shell suites are broad operator-workflow checks with grouped Markdown
evidence. Their exact gateway-minor gate runs before fixture deployment or any
other mutation. Their success covers the scenarios executed by that shell suite;
it does not certify omitted scenarios or unavailable infrastructure.

### Suspended cancellation and deletion (Bash)

`scripts/run-c810-suspended.sh` is a focused shell suite using the existing
`suite-lib.sh` report/progress helpers and the embedded `C810_SimpleUserTask`,
`C810_SimpleParent`, and `C810_MultipleSubProcessesParent` fixtures. It builds the
CLI with `go build` by default and executes CLI commands directly; it never
invokes `go test`. It is a separate entry point from `make integration-test-all`,
which still runs the existing Go suite.

It covers cancellation and forced deletion of a standalone suspended root and
both mixed active/suspended tree combinations, each with normal waiting and
`--no-wait` (12 scenarios). Tree cases select a child and check the reported
root. Deletion first checks refusal without force and unchanged family states.
Waiting commands require completed postconditions at return; no-wait commands
must report `accepted`, followed by separate bounded completion checks for every
family member. No instance is resumed during these scenarios.

For a separately requested live run against a disposable target:

```sh
C8VOLT_IT_CONFIG=./config.yaml \
C8VOLT_IT_PROFILE='<profile-name>' \
C8VOLT_IT_CAMUNDA_URL='https://<camunda-host>/v2' \
integration/scripts/run-c810-suspended.sh
```

The script requires Bash, `jq`, `curl`, and either Go or an existing binary
(`C8VOLT_IT_BUILD=0` with `C8VOLT_IT_BIN`). The API URL is also applied to the CLI
through `C8VOLT_APIS_CAMUNDA_API_BASE_URL`, so fixture setup and CLI operations
address the same server. The observed server must be **8.10.4 before mutations**.
The config/profile defaults match the C810 shell wrapper.

Only suspension setup uses the REST API, since there is no public suspend
command. For authenticated targets, set `C8VOLT_IT_CURL_CONFIG` to a private curl
configuration containing the required authentication (for example, a bearer
header or session cookie). Use the same principal as the selected CLI profile;
the CLI continues to use its configured authentication. Do not put secrets in
the script or report directory. Without this variable, curl uses no credentials
and ignores the user's default `.curlrc`.

Evidence includes separate stdout/stderr, exit codes, JSON snapshots, owned
keys, `report.md`, and `progress.tsv` under `C8VOLT_IT_WORKDIR` (a fresh temporary
directory by default). Exit cleanup deletes only created roots/families and
retains the shared embedded definitions. Cleanup failures are recorded
separately. These checks establish CLI results and final engine state; isolated
tests retain responsibility for exact internal request ordering/counts.

The wrappers use these defaults:

| Suite | Target version | Fixture prefix | Default profile |
| --- | --- | --- | --- |
| C88 | `8.8` | `C88` | `kind-camunda-platform-local-c88` |
| C89 | `8.9` | `C89` | `kind-camunda-platform-local-c89` |
| C810 | `8.10` | `C810` | `kind-camunda-platform-local-c810` |

Override configuration with environment variables:

```sh
C8VOLT_IT_CONFIG=./config.yaml \
C8VOLT_IT_PROFILE=kind-camunda-platform-local-c810 \
C8VOLT_IT_VOLUME_COUNT=25 \
integration/scripts/run-c810-suite.sh
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
| `C8VOLT_IT_PROFILES` | `c810local` for `make integration-test-all`; active profile for direct slices | Comma-separated profile names selected from the default local c8volt config. |
| `C8VOLT_IT_RELEASE_VERSION` | `8.10` for `make integration-test-all`; otherwise empty | Require the observed gateway minor to match this value before mutation. |
| `C8VOLT_IT_TENANT_ROLES` | empty | Absolute path to credential-free logical role metadata naming default-local profiles. |
| `C8VOLT_IT_REQUIRE_TENANTS` | `none` | Required tenant proof: `none`, `logical`, or `physical`; physical proof also requires logical proof. |
| `C8VOLT_IT_BIN` | built into the workdir | Existing c8volt binary to test when `C8VOLT_IT_BUILD=0`. |
| `C8VOLT_IT_BUILD` | build enabled | Set `0` with `C8VOLT_IT_BIN` to skip building the binary. |
| `C8VOLT_IT_WORKDIR` | temp directory | Evidence directory for reruns and artifact review. |
| `C8VOLT_IT_VOLUME_COUNT` | `12` | Number of suite-owned process instances created by Go volume targets for paging, limit, and filtering coverage. |
| `C8VOLT_IT_GO_TEST_FLAGS` | empty | Extra flags passed to `go test`, such as `-v` or `-failfast`. |
| `C8VOLT_IT_GO_TEST` | local integration `go test` command | Make-level command template used by integration targets. |
| `C8VOLT_IT_VERBOSE` | derived from `C8VOLT_IT_GO_TEST_FLAGS` | Set to `1` when verbose harness logging is required without Make flag derivation. |
| `C8VOLT_IT_TIMEOUT` | `60m` | Make-level timeout used by baseline `integration-cli-*` targets. |
| `C8VOLT_IT_VOLUME_TIMEOUT` | `90m` | Make-level timeout used by `integration-cli-*-volume` targets. |
| `C8VOLT_IT_REAL_STATE_TIMEOUT` | `90m` | Make-level timeout used by `integration-cli-real-state-*` and `integration-cli-batch` targets. |
| `C8VOLT_IT_AUTOMATION` | `0` | Set `1` to skip the Make-level mutation confirmation prompt. |

## Outputs

The shell suites write:

- `report.md`: grouped release validation report with evidence links.
- `progress.tsv`: append-only progress log for resume/rerun comparison.
- `summary.env`: shell-readable summary values such as run id and binary path.
- `logs/*.stdout` and `logs/*.stderr`: command evidence.
- `data/*.keys`: generated process-instance keys and other dynamic data.
- `reports/*`: command-generated reports, for example smoke-test reports.

Validate the shell tooling without a cluster before a live run:

```sh
bash -n integration/scripts/run-suite.sh integration/scripts/run-c*-suite.sh \
  integration/scripts/lib/suite-lib.sh integration/scripts/tests/run-suite_test.sh
bash integration/scripts/tests/run-suite_test.sh
```

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
  `real-state-ops-reports-<family>.json`: opt-in Camunda 8.9/8.10 real-state
  evidence.
- `batch-evidence.json` and `process-definition-draining-evidence.json`:
  focused 8.10 submission, terminal accounting, DRAINING, and post-state proof.
- `logs/`: per-command stdout and stderr evidence.
- `data/`: seeded data, selected keys, resource IDs, and substitution evidence.

Each direct Go test group uses the existing harness work directory. By default
it creates and retains a separate temporary directory. `C8VOLT_IT_WORKDIR` still
selects an exact directory, not an aggregate parent; use it for individual-group
reruns because reusing it across groups can overwrite shared report filenames.
No `release.json` or runner manifest is generated. The CLI binary is built by
each group's existing harness unless `C8VOLT_IT_BUILD=0` and `C8VOLT_IT_BIN` name
an existing binary.

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

## User-task variable coverage

The existing get targets include version-matched user-task variable scenarios:

- `integration-cli-get`: standalone local filtering, effective display and a called process.
- `integration-cli-get-volume`: filtered paging, keys, totals, limits and empty output.
- `integration-test-all`: includes both through the existing target groups.

For a focused run against an existing disposable profile:

```sh
C8VOLT_IT_PROFILES=c89local go test -tags=integration ./integration/cli -run '^TestGetFamilyUserTaskVariables$' -count=1 -v -timeout=10m
C8VOLT_IT_PROFILES=c89local C8VOLT_IT_VOLUME_COUNT=3 go test -tags=integration ./integration/cli -run '^TestVolumeGetFamilyUserTaskVariables$' -count=1 -v -timeout=10m
```

Replace the profile name with your default-local configured profile. Direct Go runs do not ask Make's confirmation question. Supported profiles use their existing C88/C89/C810 models. Wrong-version profiles fail readiness before setup.

These scenarios use the incident model with `hasIncident=false` to create local `incident=1`, contrasted with process `incident=99`. Operator failures are retained and do not prevent subsequent operator/display/called-process checks. Volume comparisons include a recorded preexisting definition-scoped baseline; no global count or exclusive model ownership is assumed. Evidence uses the existing record format in `user-task-variables-<profile>.json` and `volume-get-user-task-variables-<profile>.json`. Use separate evidence directories for separate invocations. Failed or unavailable version coverage is never reported as passing.

Live `$notIn` cases are excluded across versions after an observed C89 HTTP 500, at user request. Similar behavior on other versions is an assumption, not live verification. Unit request-encoding coverage remains unchanged.
