# Release Integration Command Matrix

This matrix describes the all-command Go integration suite coverage. The
executable source of truth for this suite is the coverage manifest and tests
under `integration/cli/`. The older shell suites under `integration/scripts/`
remain separate release checks.

## Preflight

| Command family | Coverage intent |
| --- | --- |
| `version` | Record c8volt build/version evidence. |
| `config validate` | Prove config shape before cluster calls. |
| `config test-connection --json` | Prove selected default-local profiles connect. |
| `get cluster version` | Gate selected Camunda minor before mutation. |
| `capabilities --json` | Capture command contract and automation support. |

## Inventory

| Command family | Coverage intent |
| --- | --- |
| all 55 command nodes | Validate live command inventory against the explicit manifest. |
| parent commands | Cover help/discovery and no-argument behavior. |
| leaf commands | Cover aliases, command-local flags, output modes, and destructive classification. |
| generated CLI docs and command help | Extract examples and record executable, blocked, skipped, or failed validation evidence. |

## Data Setup

| Command family | Coverage intent |
| --- | --- |
| dirty-cluster baseline | Accept existing local-cluster resources as normal test input. |
| `embed list` | Verify fixture availability. |
| `embed deploy` | Deploy version-matched embedded fixtures even when unrelated resources already exist. |
| `run process-instance` | Generate run-owned process instances with the suite marker for paging, stdin/key workflows, and mutation checks. |

## Read Workflows

| Command family | Coverage intent |
| --- | --- |
| `get` family | Cover cluster, process-definition, process-instance, resource, incident, job, element, and tenant surfaces. |
| `walk process-instance` | Cover parent, children, flat, variables, incidents, elements, and listener proposal fallback behavior. |
| `expect process-instance` | Cover key/stdin-oriented state expectation behavior. |

## Mutating Workflows

| Command family | Coverage intent |
| --- | --- |
| `deploy`, `embed`, `run` | Cover aliases, required selectors, variables, count, no-wait, and output checks. |
| `update process-instance` and `update job` | Cover dry-run, worker outcome, variables, and validation paths. |
| `cancel process-instance` | Cover key/filter selectors, dry-run, force, workers, no-wait, and validation paths. |
| `delete process-instance` and `delete process-definition` | Cover key/filter selectors, dry-run, force, latest/version flags, and validation paths. |
| `resolve incident` and `resolve process-instance` | Cover key/stdin selectors, dry-run, no-wait, and state checks. |

## Ops Workflows

| Command family | Coverage intent |
| --- | --- |
| `ops analyse slow-process-instances` | Cover key/filter/duration/timeline/listener/json/keys-only behavior. |
| `ops execute smoke-test` | Cover dry-run, report writing, count, workers, no-wait, and confirmed execution. |
| `ops execute retention-policy` | Cover dry-run planning, report writing, filters, workers, no-wait, and confirmed execution. |
| `ops purge` family | Cover all-process-definitions, orphan-process-instances, and process-instances-with-incidents with dry-run, reports, filters, workers, and confirmed execution. |
| `ops repair` family | Cover incident and process-instance repair with keys, search filters, vars, retries, timeouts, reports, dry-run, and confirmed execution. |

## Version-Specific Checks

| Target | Coverage intent |
| --- | --- |
| C87/C88/C89 selected profiles | Validate profile connectivity and expected minor before destructive scenarios. |
| command-family manifest | Records version-sensitive expectations and proposal gaps where embedded fixtures or commands cannot currently create required state. |

## Evidence

| Command family | Coverage intent |
| --- | --- |
| run summary | Write `run.json` and `summary.md` in the suite workdir. |
| inventory and coverage | Write `inventory.json`, `coverage.json`, and `coverage-<family>.json`. |
| readiness and smoke | Write `profiles.json` and `readonly-smoke.json`. |
| examples | Write `examples.json` plus seeded example substitution data under `data/`. |
| proposal reports | Write `proposals-command.json` and `proposals-embedded-bpmn.json`, using `[]` for no-gap reports. |
| command logs and data | Write subprocess stdout/stderr under `logs/` and selected keys/resources under `data/`. |
