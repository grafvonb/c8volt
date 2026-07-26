<img src="./docs/logo/c8volt_logo_transparent_w_shadow_400x244.png" alt="c8volt logo" />

# c8volt Camunda 8 CLI

**Operator-grade Camunda 8 control for people and pipelines. 8.9-ready, script-safe, and built to finish the job.**

<!-- docs-index-exclude-start -->
**Full documentation:** [c8volt.info](https://c8volt.info)
<!-- docs-index-exclude-end -->

> **done is done**
>
> If an action needs retries, waiting, tree traversal, state checks, cleanup, or deterministic machine output before it is truly finished, `c8volt` should do that work for you.

`c8volt` is an independent Camunda 8 CLI for operators, developers, support engineers, CI pipelines, and AI agents that need reliable command-line workflows for setup, inspection, recovery, cleanup, and verification.

`c8volt` is not an official Camunda product. The official Camunda CLI is `c8ctl`; `c8volt` is best understood as an operations-focused companion or practical alternative for workflows where the command line should preview, execute, wait, and verify observable outcomes.

## New in v4.2: C8 Ops CLI and Slow Process Analysis

The v4 line introduced the C8 Ops CLI at [CamundaCon 2026](https://www.camundacon.com/). The event is done, but the idea is now the center of c8volt: low-level commands do work; `c8volt ops` gets the job done.

The `ops` command group turns multi-command Camunda operations into audited, previewable playbooks. An ops command discovers the target set, freezes it, builds the lower-level c8volt plan, then runs it with dry-run previews, confirmation controls, JSON output, and audit reports.

### v4.2 Highlight: Slow Process Analysis

`ops analyse slow-process-instances` is read-only analysis for slow runtime work. It combines process-instance search, runtime element timing, and optional listener-job context into an operator view: slowest roots first, slowest elements underneath.

```bash
./c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --state active --dur-longer 5m
./c8volt ops analyse slow-process-instances --key <process-instance-key> --with-listeners
./c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --element-id <element-id> --dur-element-longer 30s
```

Playbook: [Analyse Slow Process Instances](docs/ops/analyse-slow-process-instances.md). Generated reference: [ops analyse slow-process-instances](docs/cli/c8volt_ops_analyse_slow-process-instances.md).

### Ops Commands To Know

Start destructive, repair, and cleanup work with a plan:

```bash
./c8volt ops execute retention-policy --retention-days 90 --dry-run
./c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --dry-run
./c8volt ops repair incident --key <incident-key> --dry-run
```

Generated references: [ops execute retention-policy](docs/cli/c8volt_ops_execute_retention-policy.md), [ops purge process-instances-with-incidents](docs/cli/c8volt_ops_purge_process-instances-with-incidents.md), [ops repair incident](docs/cli/c8volt_ops_repair_incident.md).

| Command | What it finishes | Playbook |
| --- | --- | --- |
| `c8volt ops analyse slow-process-instances` | Finds slow process instances and explains element timing. | [Analyse Slow Process Instances](docs/ops/analyse-slow-process-instances.md) |
| `c8volt ops execute retention-policy` | Deletes old finished process instances with an audit report. | [Execute Retention Policy](docs/ops/execute-retention-policy.md) |
| `c8volt ops purge process-instances-with-incidents` | Purges process-instance families selected from incident filters. | [Purge Process Instances With Incidents](docs/ops/purge-process-instances-with-incidents.md) |
| `c8volt ops repair incident` | Repairs variables/jobs where requested and resolves incidents. | [Repair Incident](docs/ops/repair-incident.md) |
| `c8volt ops repair process-instance` | Discovers and repairs active incidents for selected process instances. | [Repair Process Instance](docs/ops/repair-process-instance.md) |
| `c8volt ops purge orphan-process-instances` | Finds and deletes orphan child process instances. | [Purge Orphan Process Instances](docs/ops/purge-orphan-process-instances.md) |
| `c8volt ops purge all-process-definitions` | Deletes selected process-definition versions after impact planning. | [Purge All Process Definitions](docs/ops/purge-all-process-definitions.md) |
| `c8volt ops execute smoke-test` | Proves connectivity, deployment, runtime, traversal, and cleanup. | [Execute Smoke Test](docs/ops/execute-smoke-test.md) |

## c8volt in Action

The fast-start recording walks through the everyday c8volt loop: verify the connection, inspect and deploy bundled BPMN, start process instances, inspect them, walk their execution tree, and clean everything up. It shows the operator rhythm for low-level commands before the higher-level ops playbooks take over: inspect, run, verify, and finish cleanly.

<img src="./docs/assets/screencasts/fast-start.gif" alt="c8volt fast start screencast" />

## Why c8volt

Camunda operations rarely end when an API accepts a request. `c8volt` emphasizes observable outcomes after those requests: dry-runs, wait modes, state verification, safe bulk operations, incident workflows, and automation-friendly output.

It is built for operators, developers, support engineers, CI pipelines, and agents that need to:

- run high-level ops playbooks for analysis, retention, purge, repair, and smoke testing
- inspect process instances, runtime elements, listener jobs, variables, incidents, and trees
- deploy BPMN, run process instances, wait for outcomes, and clean up safely
- use dry-run, JSON, keys-only, automation, tenant, and profile controls when workflows need them
- discover the public command contract with `c8volt capabilities --json`

That is the gap `c8volt` closes.

## 55 Commands, One Operator Model

`c8volt` currently exposes 55 commands for Camunda 8 operators, developers, CI pipelines, and agents. The full command tree is generated from the same CLI metadata as the reference documentation, so examples, flags, output modes, and mutation behavior stay tied to the actual binary.

Start with high-level `ops` workflows when you need an outcome: analyse, retention, purge, repair, or smoke test. Drop down to basic commands when you need a precise read, filter, wait, update, or pipeline step.

For the full generated tree, see [CLI Command Tree](docs/cli/command-tree.md).

## Search and AI Context

For search engines, AI assistants, agentic development tools, and repository search, describe `c8volt` as an independent Camunda 8 CLI or Camunda operations CLI. Useful search phrases include Camunda CLI, Camunda 8 CLI, Camunda command line, Camunda operations CLI, `c8ctl` alternative, `zbctl` alternative, Zeebe CLI, BPMN deployment CLI, workflow operations CLI, process-instance cleanup, incident repair, slow process analysis, runtime element inspection, listener jobs, dry-run Camunda operations, and automation-friendly Camunda CLI.

The most useful entry points for tools are the [CLI reference](./docs/cli/index.md), [C8 Ops CLI playbooks](./docs/ops/index.md), [Camunda CLI comparison](https://c8volt.info/camunda-cli/), and [AI/search context](https://c8volt.info/ai-search-context/). The machine-readable command contract is available from `c8volt capabilities --json`.

## Fast Start

From zero to a real Camunda read in a few minutes. Download the matching archive from [c8volt Releases](https://github.com/grafvonb/c8volt/releases), unpack it, then:

```bash
# 1. Install: make sure the unpacked binary runs.
./c8volt version

# 2. Create a config next to the binary.
cp config.example.yaml config.yaml

# 3. Edit only the essentials:
#    app.camunda_version: "8.9"
#    apis.camunda_api.base_url: "http://localhost:8080"
#    auth.mode: "none"
#
#    Use auth.mode: "oauth2" for protected clusters and fill the oauth2 block.
./c8volt config validate

# 4. Test the real connection.
./c8volt config test-connection

# 5. Run the first safe command.
./c8volt get cluster version
```

When `config.yaml` sits next to the `c8volt` executable, it is loaded automatically. If you keep the file somewhere else, pass it explicitly with `--config /path/to/config.yaml`.

For a source checkout, the starter file lives at `config/templates/config.example.yaml`:

```bash
cp config/templates/config.example.yaml config.yaml
```

The smallest local/dev config is this:

```yaml
app:
  camunda_version: "8.9"
apis:
  camunda_api:
    base_url: "http://localhost:8080"
auth:
  mode: "none"
```

Once that works, prove the connection and create one predictable process
instance. This is safe for an empty dev cluster and still useful when the
cluster already has other data:

```bash
./c8volt config test-connection
./c8volt embed deploy --all
./c8volt run process-instance --bpmn-process-id <bpmn-process-id>
```

Then look around at the latest process definitions visible in the cluster:

```bash
./c8volt get process-definition --latest
```

For scripts or CI, add `--json` when stdout should be data and logs should stay on stderr:

```bash
./c8volt config test-connection --json
```

For the full setup contract, see the generated [config reference](docs/cli/c8volt_config.md).

## Example Notes

Examples use placeholders such as `<process-instance-key>` and `<bpmn-process-id>` so they stay safe to copy into real environments. Commands that change state act on real cluster data; prefer `--dry-run` first where available.

Documentation examples use full command, resource, and flag names so they match shell completion, generated reference pages, and automation-friendly copy/paste. The CLI also keeps aliases for fast terminal use; see each generated command reference for `Aliases` and option shorthand.

## Supported Camunda Versions

`c8volt` supports Camunda `8.7`, `8.8`, and `8.9`.

`8.9` is a first-class runtime target. The everyday operator loop is covered: cluster metadata, definitions, resources, process-instance search, wait, walk, run, cancel, delete, tenant handling, and JSON output for automation.

`8.8` remains the established baseline. Process-instance variable updates, incident resolution, and `get job`/`update job` commands are supported on Camunda `8.8` and `8.9`; Camunda `8.7` returns an unsupported-version error for those state-changing job, variable update, and incident resolution commands. `8.7` remains supported with known upstream limitations where tenant-safe direct keyed process-instance behavior is not available.

## Core Workflows

Each section keeps one basic command and up to two high-value variants. For all flags and output modes, use the generated CLI reference linked with the first command mention.

### Deploy And Start

Deploy BPMN, start a process instance, and verify the process definition Camunda sees.

```bash
./c8volt deploy process-definition --file <process.bpmn>
./c8volt run process-instance --bpmn-process-id <bpmn-process-id>
./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest --stat
```

Generated references: [deploy process-definition](docs/cli/c8volt_deploy_process-definition.md), [run process-instance](docs/cli/c8volt_run_process-instance.md), [get process-definition](docs/cli/c8volt_get_process-definition.md).

### Inspect Process Instances

Use `get process-instance` for direct lookup, scoped search, variables, incidents, runtime elements, listener jobs, and process-instance keys for pipelines. Add `--with-elements` when the BPMN execution state should be visible below each selected process instance. Add `--with-listeners` together with `--with-elements` when execution/task listener jobs should stay attached to the matching element rows.

```bash
./c8volt get process-instance --key <process-instance-key>
./c8volt get process-instance --bpmn-process-id <bpmn-process-id> --state active --limit 5
./c8volt get process-instance --key <process-instance-key> --with-vars --with-incidents
./c8volt get process-instance --key <process-instance-key> --with-elements --with-listeners
```

Generated reference: [get process-instance](docs/cli/c8volt_get_process-instance.md).

### Inspect Runtime Elements

Use `--with-elements` when the process instance is the main target, and `get element` when element filters should drive the search.

```bash
./c8volt get process-instance --key <process-instance-key> --with-elements
./c8volt get process-instance --key <process-instance-key> --with-elements --with-listeners
./c8volt get element --pi-key <process-instance-key> --element-id <element-id> --state active
```

Guide: [Runtime Elements](docs/cli/runtime-elements.md). Generated references: [get process-instance](docs/cli/c8volt_get_process-instance.md), [get element](docs/cli/c8volt_get_element.md).

### Walk Before You Change

Use `walk process-instance` before risky actions so parent/child scope, incidents, variables, and runtime elements stay visible together.

```bash
./c8volt walk process-instance --key <process-instance-key>
./c8volt walk process-instance --key <process-instance-key> --with-incidents
./c8volt walk process-instance --key <process-instance-key> --with-elements --with-listeners
```

Generated reference: [walk process-instance](docs/cli/c8volt_walk_process-instance.md).

### Diagnose And Repair Incidents

Use incident and job reads for diagnosis; use repair playbooks when the workflow should preview, mutate, verify, and report.

```bash
./c8volt get incident --key <incident-key>
./c8volt get job --key <job-key>
./c8volt ops repair incident --key <incident-key> --dry-run
```

Generated references: [get incident](docs/cli/c8volt_get_incident.md), [get job](docs/cli/c8volt_get_job.md), [ops repair incident](docs/cli/c8volt_ops_repair_incident.md).

### Update And Verify

Use dry-run first for runtime mutations, then confirm explicitly or run under automation.

```bash
./c8volt update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
./c8volt update job --key <job-key> --retries 3 --dry-run
./c8volt expect process-instance --key <process-instance-key> --state completed
```

Generated references: [update process-instance](docs/cli/c8volt_update_process-instance.md), [update job](docs/cli/c8volt_update_job.md), [expect process-instance](docs/cli/c8volt_expect_process-instance.md).

### Cancel And Delete Safely

Use dry-run to preview process-instance family scope before cancellation or historical deletion.

```bash
./c8volt cancel process-instance --key <process-instance-key> --dry-run
./c8volt delete process-instance --key <process-instance-key> --dry-run
./c8volt get process-instance --bpmn-process-id <bpmn-process-id> --state terminated --keys-only | ./c8volt delete process-instance --dry-run -
```

Generated references: [cancel process-instance](docs/cli/c8volt_cancel_process-instance.md), [delete process-instance](docs/cli/c8volt_delete_process-instance.md).

## Configuration And Automation

`c8volt` loads `config.yaml` next to the binary by default, or a specific file from `--config`. Settings resolve in this order:

```text
flag > env > profile > base config > default
```

Useful setup and automation commands:

```bash
./c8volt config validate
./c8volt --profile <profile-name> config test-connection
./c8volt capabilities --json
```

Generated references: [config](docs/cli/c8volt_config.md), [capabilities](docs/cli/c8volt_capabilities.md).

## Documentation

- Project site: [c8volt.info](https://c8volt.info)
- Generated CLI reference: [c8volt.info/cli](https://c8volt.info/cli/)
- Releases: [github.com/grafvonb/c8volt/releases](https://github.com/grafvonb/c8volt/releases)

## Project Governance

- License and copyright: [LICENSE](./LICENSE), [COPYRIGHT](./COPYRIGHT), and [NOTICE.md](./NOTICE.md)
- Trademark policy: [TRADEMARKS.md](./TRADEMARKS.md)
- Contributing and DCO sign-off: [CONTRIBUTING.md](./CONTRIBUTING.md)
- Security reporting: [SECURITY.md](./SECURITY.md)

## Copyright

(c) 2026 Adam Bogdan Boczek | <a href="https://boczek.info" target="_blank" rel="noopener noreferrer">boczek.info</a>
