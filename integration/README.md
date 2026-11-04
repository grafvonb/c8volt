# Release integration suites

These suites run the built CLI against a specifically selected Camunda target. They create, cancel, repair, resolve, and delete cluster data. Keep them separate from implementation validation: use unit tests, mocks, and test-owned HTTP fixtures for implementation work.

Read [AGENTS.md](AGENTS.md) before a separately requested live run. Examples below target Camunda 8.10. Compatibility suites for 8.9 and 8.8 remain available.

## Run a release suite

Select an existing local profile that points to the intended target:

```sh
C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 \
  make integration-test-all
```

The aggregate runs common checks, focused batch/tenant checks, baseline, volume, and real-state groups sequentially. It prints settings and asks once before mutation. It stops on the first failing group. Use `C8VOLT_IT_AUTOMATION=1` only for an explicitly authorized unattended run.

Profiles come from the normal local c8volt configuration under `$HOME/.config/c8volt`. Do not substitute a generated private config. Every family checks gateway and broker releases and partition health before mutating. Baseline and volume groups support 8.10, 8.9, and 8.8; real-state groups support 8.10 and 8.9.

## Select a group

| Scope | Make target |
| --- | --- |
| Common inventory, profile, and example checks | `integration-cli-common` |
| Baseline command families | `integration-test` |
| Volume scenarios | `integration-test-volume` |
| Runtime-state scenarios | `integration-test-real-state` |
| Batch operations and definition draining | `integration-cli-batch` |
| Logical and physical tenant boundaries | `integration-cli-tenants` |
| Suspension and resumption | `integration-cli-process-lifecycle` |

Individual family targets include `integration-cli-get`, `integration-cli-walk`, `integration-cli-update`, `integration-cli-cancel`, `integration-cli-delete`, `integration-cli-expect-resolve`, `integration-cli-deploy-embed-run`, and the `integration-cli-ops-*` families. Inspect `make help` for available volume and real-state slices.

```sh
C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 \
  make integration-cli-ops-repair C8VOLT_IT_GO_TEST_FLAGS=-v
```

Use a separate `C8VOLT_IT_WORKDIR` per invocation. The default timeouts are 60 minutes for baseline, 90 minutes for volume, and 90 minutes for real-state groups. `C8VOLT_IT_GO_TEST_FLAGS=-v` also enables subprocess scenario logs.

## Coverage and gates

- **Suspension/resumption:** the held-instance and lifecycle scenarios require an observed `8.10` server before fixture mutation. Later 8.10 patches remain eligible; older minors skip these scenarios. CLI suspension/resumption drives the main lifecycle proof. Independent reads confirm state and timestamp omission; cleanup resumes owned suspended instances before deletion.
- **Definition draining:** observe `DRAINING`, then cancel owned work and verify definition disappearance independently. A null batch operation can be valid. Definition disappearance does not prove that deferred history deletion has finished.
- **Logical tenants:** supply existing tenant-A, tenant-B, dual-authorized, and restricted-A profiles in the role file. Positive visibility precedes absence assertions. Explicit keys test authorization independently of discovery filters.
- **Physical tenants:** profiles must target distinct declared `/physical-tenants/<id>/v2` routes on one declared cluster. A default-route profile establishes equivalence. This proves routing and resource separation, not storage or compute isolation.

Tenant-role shape: [tenant-roles.example.json](assets/tenant-roles.example.json).

```sh
C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 \
C8VOLT_IT_TENANT_ROLES=/absolute/path/to/tenant-roles.json \
C8VOLT_IT_REQUIRE_TENANTS=physical \
  make integration-cli-tenants
```

Missing tenant infrastructure is an optional coverage gap unless required. Physical proof also requires the logical proof group. Cleanup failures remain separate failures.

## Results

Use Make/Go exit status and per-group evidence. A passing run covers executed scenarios; skipped or blocked examples do not count as acceptance.

| Evidence | Location |
| --- | --- |
| Command inventory and coverage | `inventory.json`, `coverage.json`, `coverage-<family>.json` |
| Profiles and read-only checks | `profiles.json`, `readonly-smoke.json` |
| Example eligibility and substitutions | `examples.json`, `data/` |
| Command stdout and stderr | `logs/` |
| Run summary | `run.json`, `summary.md` |

Expectation examples run only when their requested conditions match the fixture. Resource lookup requires an actual `resourceKey`; deployment and definition keys are not substitutes. The get slice deploys an inert generic resource for positive lookup and logical-ID keys-only checks. Variable-file examples receive a generated JSON fixture. The update slice also verifies bulk task assignment/unassignment/completion, task-variable ownership, post-creation business IDs, and an activated job’s timeout and completion business ID. Report-path regressions check preservation of existing files and execution with a fresh path. Retry-update validation does not establish bulk job-completion coverage.

See [command matrix](assets/command-matrix.md), [UX checklist](assets/ux-review-checklist.md), and the relevant `specs/` validation records for scenario evidence and remaining gaps. Historical records retain the versions and commands used at the time.

## Layout

- `cli/`: build-tagged Go suite, coverage manifest, and fixture helpers.
- `scripts/run-c810-suite.sh`: 8.10 shell workflow; older-version wrappers remain for compatibility.
- `scripts/run-c810-suspended.sh`: focused suspended cancellation/deletion checks.
- `scripts/tests/`: offline fake-CLI checks.
- `prompts/`: version-specific release workflow instructions.

Shell suites and Go family suites are separate; a passing shell workflow does not certify complete command coverage. The [all-command rules](assets/all-command-go-integration-rules.md) apply only when that suite is explicitly requested.
