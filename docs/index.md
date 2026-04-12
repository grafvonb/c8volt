---
title: "c8volt"
permalink: /
nav_order: 1
has_toc: true
---

> Generated from build `c8volt v2.1.0-23-g6b6427c-dirty`, commit `6b6427c`, built `2026-04-12T07:32:59Z` | camunda: 8.7, 8.8

<img src="./logo/c8volt_orange_black_bkg_white_400x152.png" alt="c8volt logo" style="border-radius: 5px;" />

# c8volt Camunda 8 CLI

**c8volt is a Camunda 8 CLI for workflow operations. For workflows that must actually finish.**

> **done is done**
>
> If an action needs retries, waiting, tree traversal, state checks, or cleanup before it is truly finished, `c8volt` should do that work for you instead of making you script the last mile yourself.

`c8volt` is not just a CRUD shell around Camunda 8 APIs. It is a Camunda CLI shaped around operational intent:

- start a process and confirm it is really active
- cancel a child process by finding the root that must actually be cancelled
- delete a process instance family thoroughly, not partially
- deploy BPMN models and use them immediately

Standard read/list/get commands still matter, but they are not the headline. The headline is operational confidence.

`c8volt` helps when people are searching for a Camunda 8 CLI, a BPMN deployment CLI, or a reliable way to run, inspect, cancel, delete, and verify Camunda process instances from the terminal. It is designed for operators, developers, support engineers, CI pipelines, and local development environments that need outcome verification instead of "request accepted" ambiguity.

## What c8volt Is

`c8volt` is a command-line tool for Camunda 8 workflow operations. It focuses on the parts of workflow automation that teams often end up scripting around by hand:

- deploy BPMN process definitions from the CLI
- start Camunda 8 process instances and confirm they are active
- inspect parent and child process-instance trees
- cancel the correct root process instance when a child cannot be canceled directly
- delete a full process-instance family instead of one visible node
- wait for a target process state in scripts and CI jobs
- validate `c8volt` connection and authentication config before running production actions

If someone is looking for a Camunda 8 CLI, a Camunda CLI for operators, a BPMN deployment tool for Camunda 8, or a terminal workflow tool that behaves well in automation, those are the main problems `c8volt` is built to solve.

## At A Glance

- deploy BPMN and use it immediately
- run process instances and confirm they are active
- inspect process-instance trees before changing them
- cancel safely, including root escalation with `--force`
- delete process-instance families thoroughly
- wait for the state you actually need
- validate config and inspect cluster metadata

## c8volt vs c8ctl

[`c8ctl`](https://docs.camunda.io/build-with-camunda/) is Camunda's broader official CLI for the full Camunda 8 lifecycle. `c8volt` is more focused: it is built for operators and automation flows that care about what happened after the request, not just whether the request was accepted.

In practice, that means `c8volt` leans hardest into state confirmation, process-tree inspection, root-aware cancellation, thorough deletion, and shell-friendly operational workflows.

## Why It Feels Different

Many CLIs stop at "request accepted."

`c8volt` is designed for the moments after that:

- Did the process instance actually reach `ACTIVE`?
- Did the cancellation propagate through the whole family?
- Did deletion remove the tree, not just one visible node?
- Is the deployment already usable for the next command?

That is the gap `c8volt` closes.

## Signature Workflows

### 1. Bootstrap a local environment fast

Deploy bundled BPMN fixtures directly from the binary:

```bash
./c8volt embed list
./c8volt embed deploy --all
./c8volt embed deploy --all --run
```

This is the quickest path from "clean environment" to "real process instances to inspect and operate on."

### 2. Start and confirm, not just start

```bash
./c8volt get pd --latest
./c8volt run pi -b C88_SimpleUserTask_Process
```

By default, `c8volt` waits until the process instance is actually active. If you explicitly want asynchronous behavior:

```bash
./c8volt run pi -b C88_SimpleUserTask_Process --no-wait
```

For batch execution:

```bash
./c8volt run pi -b C88_SimpleUserTask_Process -n 100 --workers 8
```

### 3. Understand the tree before you change it

```bash
./c8volt walk pi --key 2251799813711967 --family
./c8volt walk pi --key 2251799813711967 --family --tree
```

This is where `c8volt` becomes an operations tool instead of just a resource browser: it helps you see the structure that explains why a cancellation or deletion may behave the way it does.

### 4. Cancel the thing that actually needs cancelling

Camunda may reject a direct cancellation of a child instance when the real action must happen at the root.

```bash
./c8volt cancel pi --key 2251799813711977
./c8volt cancel pi --key 2251799813711977 --force
./c8volt cancel pi --state active --start-date-before 2026-03-31
./c8volt cancel pi --state active --start-date-newer-days 30
```

With `--force`, `c8volt` escalates from the selected child to the root process instance and waits for the family-level outcome.
The same search-driven flow also supports inclusive absolute `--start-date-*` / `--end-date-*` filters and relative `--start-*-days` / `--end-*-days` shortcuts when you want to target matching instances without collecting keys first.

### 5. Delete thoroughly

```bash
./c8volt delete pi --key 2251799813711967 --force
./c8volt delete pi --state completed --end-date-after 2026-01-01 --end-date-before 2026-01-31 --auto-confirm
./c8volt delete pi --state completed --end-date-older-days 7 --end-date-newer-days 60 --auto-confirm
./c8volt get pi --state completed --keys-only | ./c8volt delete pi - --auto-confirm
```

Deletion in real environments often means cancel-first, then remove, then verify. `c8volt` is built for that operational sequence.

### 6. Page through large process-instance result sets safely

```bash
./c8volt get pi --state active
./c8volt get pi --state active --count 250
./c8volt cancel pi --state active --count 250
./c8volt delete pi --state completed --count 250 --auto-confirm
```

Search-based `get pi`, `cancel pi`, and `delete pi` now work page by page instead of silently stopping at the first `1000` matches. They report the page size used, the current-page count, the cumulative processed count, and whether another page remains. When more matches exist, the commands prompt before continuing unless `--auto-confirm` is set. Direct `--key` workflows still bypass paging and keep their existing behavior.

## Precision Tools

The project also includes a strong set of supporting commands that are fully implemented and useful in day-to-day operator work.

### Wait for a known-good state

Use `expect` when a script or operator workflow needs a concrete state transition before moving on:

```bash
./c8volt expect pi --key 2251799813685255 --state active
./c8volt expect pi --key 2251799813685255 --state completed --state absent
./c8volt get pi --bpmn-process-id order-process --keys-only | ./c8volt expect pi - --state terminated
```

`expect` is where `c8volt` becomes a strong automation partner instead of just a command runner: it can wait for `active`, `completed`, `canceled`, `terminated`, or `absent` and it works naturally with piped keys for bulk verification flows.

### Inspect the environment before acting

```bash
./c8volt get cluster topology
./c8volt get cluster license
./c8volt config show
./c8volt config show --validate
./c8volt config show --template
```

This makes `c8volt` useful not only for action commands, but also for environment checks, support flows, and CI diagnostics. The `config` command family is about `c8volt` itself: how the CLI connects, authenticates, renders logs, and selects profiles.

### Pull exact artifacts and metadata

```bash
./c8volt get pd --key 2251799813686017 --xml
./c8volt get pd --bpmn-process-id order-process --latest --stat
./c8volt get resource --id resource-id-123
```

When you need more than "list everything," `c8volt` can pull the sharp edges too: the latest deployed definition, raw BPMN XML for one exact definition, definition statistics, and single resources by id.

### Find the exact process instances you want

```bash
./c8volt get pi --state active --incidents-only
./c8volt get pi --roots-only --with-age
./c8volt get pi --children-only
./c8volt get pi --orphan-children-only
```

This is one of the quiet strengths of the tool. `c8volt` can narrow process-instance views to roots, children, orphaned children, instances with incidents, instances without incidents, or age-annotated views when you need a faster operational read on what is actually happening.

### Narrow process instances by start or end day

```bash
./c8volt get pi --start-date-after 2026-01-01 --start-date-before 2026-01-31
./c8volt get pi --start-date-older-days 7 --start-date-newer-days 30
./c8volt get pi --end-date-after 2026-02-01
./c8volt get pi --end-date-before 2026-03-31 --state completed
./c8volt cancel pi --state active --start-date-after 2026-03-01 --start-date-before 2026-03-31
./c8volt cancel pi --state active --start-date-newer-days 30
./c8volt delete pi --state completed --end-date-after 2026-01-01 --end-date-before 2026-01-31 --auto-confirm
./c8volt delete pi --state completed --end-date-older-days 7 --end-date-newer-days 60 --auto-confirm
```

The `--start-date-*` and `--end-date-*` flags are inclusive `YYYY-MM-DD` bounds for search/list usage. For relative day filters, `--*-date-older-days N` means `N` days old or older, and `--*-date-newer-days N` means `N` days old or newer (`0` means today). Both forms narrow `get pi` results and also support search-driven `cancel pi` / `delete pi` selection, they exclude missing `endDate` values when end-date filters are used, they reject mixing absolute and relative filters for the same field, and they are intentionally not supported with `--key` direct lookup.

### Export bundled fixtures for editing or custom deployment

```bash
./c8volt embed list
./c8volt embed list --details
./c8volt embed export --all --out ./fixtures
./c8volt embed export --file 'processdefinitions/*.bpmn' --out ./fixtures
```

The embedded-fixture workflow is more than a demo convenience: you can list what ships in the binary, switch between short names and full embedded paths, export selected assets, and turn the built-in fixtures into a fast local lab for repeatable testing.

## Command Map

```text
c8volt
├── embed                     Work with bundled BPMN fixtures
│   ├── list                  List bundled BPMN assets available in the binary
│   ├── deploy                Deploy bundled fixtures directly to Camunda
│   └── export                Export bundled fixtures for editing or custom deployment
├── deploy                    Deploy resources from local files or stdin
│   └── pd                    Deploy BPMN process definitions
├── run                       Start runnable resources
│   └── pi                    Start process instances and confirm activation by default
├── walk                      Inspect parent/child relationships
│   └── pi                    Walk ancestors, descendants, or full family trees
├── cancel                    Cancel resources and wait for a confirmed state change
│   └── pi                    Cancel process instances, including root escalation with --force
├── delete                    Delete resources, optionally forcing cleanup first
│   ├── pi                    Delete process instance trees
│   └── pd                    Delete process definitions with safety warnings
├── expect                    Wait until resources reach a target state
│   └── pi                    Wait for active, completed, canceled, terminated, or absent
├── get                       Read state, metadata, and resources
│   ├── cluster topology      Show connected Camunda cluster topology
│   ├── cluster license       Show cluster license details
│   ├── process-definition    List definitions, fetch latest versions, or retrieve XML
│   ├── process-instance      List or fetch process instances
│   └── resource              Fetch a single resource by id
└── config                    Inspect and validate c8volt configuration
    └── show                  Render, validate, or template c8volt config

plus:
  version                     Print build and compatibility information
```

## Quick Start

### Run Camunda 8 locally

Download [Camunda 8 Run](https://downloads.camunda.cloud/release/camunda/c8run), unpack it, and start it:

```bash
./start.sh
```

For local `c8run` on Camunda `8.8`, a minimal `config.yaml` for `c8volt` looks like this:

```yaml
apis:
  version: "88"
  camunda_api:
    base_url: "http://localhost:8080/v2"
auth:
  mode: none
log:
  level: info
```

Common config locations:

- `./config.yaml`
- `$HOME/.c8volt/config.yaml`
- `$HOME/.config/c8volt/config.yaml`

## Configure c8volt

The `c8volt config` command family is for the CLI itself, not for configuring Camunda.

Use it to:

- inspect the effective `c8volt` configuration
- validate connection and auth settings before running operational commands
- generate a template file to start from
- switch between connection profiles such as `local`, `staging`, and `prod`
- set the default tenant for tenant-aware operations

The most useful starting commands are:

```bash
./c8volt config show
./c8volt config show --validate
./c8volt config show --template
./c8volt --profile prod config show
```

### Process-instance paging defaults

Search-based `get process-instance`, `cancel process-instance`, and `delete process-instance` share one default page-size setting:

```yaml
app:
  process_instance_page_size: 250
```

- Default behavior stays at `1000` when this key is unset.
- `--count` overrides the configured value for one command run.
- `C8VOLT_APP_PROCESS_INSTANCE_PAGE_SIZE` provides the same setting through the environment.
- `--auto-confirm` continues to the next page automatically; without it, the CLI prompts between pages.
- Direct `--key` usage for `cancel pi` and `delete pi` remains non-paged.

### Tenant support

`c8volt` supports tenant-aware operations through:

- `app.tenant` in the config file
- the global `--tenant` flag for per-command override

This tenant value is used by commands such as deploy and run, and tenant IDs are also visible in process-definition and process-instance output where the API returns them.

Set a default tenant in config:

```yaml
app:
  tenant: "tenant-a"
```

Or override it for one command:

```bash
./c8volt --tenant tenant-a run pi -b order-process
./c8volt --tenant tenant-a deploy pd --file ./order-process.bpmn
./c8volt --tenant tenant-a get pd --latest
```

This makes it practical to keep one base config while switching tenant context explicitly in scripts, CI jobs, or operator sessions.

### Common OAuth connection scenarios

#### 1. Local or simple cluster, no auth

This is the smallest setup and works well for local `c8run` or unsecured development environments:

```yaml
apis:
  version: "88"
  camunda_api:
    base_url: "http://localhost:8080/v2"
auth:
  mode: none
log:
  level: info
```

#### 2. One OAuth-protected Camunda API endpoint

Use this when the cluster API is the main entry point and one token is enough for the operations you run:

```yaml
apis:
  camunda_api:
    base_url: "https://camunda.example.com/v2"
auth:
  mode: oauth2
  oauth2:
    token_url: "https://login.example.com/oauth/token"
    client_id: "c8volt"
    client_secret: "${set via env}"
log:
  level: info
```

This is the best default mental model for `c8volt`: connect the CLI to one cluster API, authenticate with client credentials, then operate through the CLI.

If your cluster is tenant-aware and you usually operate in one tenant, add:

```yaml
app:
  tenant: "tenant-a"
```

#### 3. OAuth with explicit scopes per API

Use this when your identity provider or platform setup requires scopes for specific APIs:

```yaml
apis:
  camunda_api:
    base_url: "https://camunda.example.com"
    require_scope: true
  operate_api:
    base_url: "https://operate.example.com"
    require_scope: true
  tasklist_api:
    base_url: "https://tasklist.example.com"
    require_scope: true
auth:
  mode: oauth2
  oauth2:
    token_url: "https://login.example.com/oauth/token"
    client_id: "c8volt"
    client_secret: "${set via env}"
    scopes:
      camunda_api: "camunda-api.read"
      operate_api: "operate-api.read"
      tasklist_api: "tasklist-api.read"
```

This is the most common "real cluster" scenario when different endpoints or policies exist behind one identity provider.

#### 4. One file, multiple profiles

Use profiles when the same operator needs to switch between environments without copying files:

```yaml
active_profile: local

auth:
  oauth2:
    client_secret: "${set via env}"

profiles:
  local:
    app:
      tenant: ""
    apis:
      camunda_api:
        base_url: "http://localhost:8080/v2"
    auth:
      mode: none

  prod:
    app:
      tenant: "tenant-a"
    apis:
      camunda_api:
        base_url: "https://camunda.example.com"
        require_scope: true
    auth:
      mode: oauth2
      oauth2:
        token_url: "https://login.example.com/oauth/token"
        client_id: "c8volt"
        scopes:
          camunda_api: "camunda-api.read"
```

Switch profiles with:

```bash
./c8volt --profile local get cluster topology
./c8volt --profile prod get cluster topology
```

### Recommended workflow for OAuth setups

1. Generate a starting point with `./c8volt config show --template`.
2. Fill in `apis.camunda_api.base_url`, `auth.mode`, and the OAuth credentials.
3. If you usually operate in a specific tenant, set `app.tenant`.
4. Keep `client_secret` out of the YAML file when possible and inject it through environment variables.
5. Run `./c8volt config show --validate`.
6. Confirm connectivity with `./c8volt get cluster topology`.
7. Confirm tenant-aware behavior with a command such as `./c8volt get pd --latest` or an explicit `--tenant` override.
8. Only then move on to deploy/run/cancel/delete workflows.

### Environment variables for secrets

Sensitive values are safer in environment variables than in committed config files. The config system supports `C8VOLT_...` environment variables, so a common pattern is:

```bash
export C8VOLT_AUTH_OAUTH2_CLIENT_SECRET='super-secret'
export C8VOLT_AUTH_OAUTH2_CLIENT_ID='c8volt'
./c8volt --profile prod config show --validate
```

### Install c8volt

Download the appropriate archive from [c8volt Releases](https://github.com/grafvonb/c8volt/releases), unpack it, then verify the binary:

```bash
./c8volt version
```

Release archives are the main installation path for local operator machines, CI runners, and ephemeral environments. The project is especially useful when you want a portable Camunda 8 CLI binary for Linux or macOS that can deploy BPMN, inspect workflow state, and automate process-instance operations from shell scripts.

### Verify connectivity

```bash
./c8volt get cluster topology
./c8volt get cluster license
```

Or point to an explicit config file:

```bash
./c8volt get cluster topology --config ./config.yaml
```

## Everyday Commands

The supporting read and deployment commands are still part of the core toolbox:

```bash
./c8volt deploy pd --file ./order-process.bpmn
./c8volt deploy pd --file ./order-process.bpmn --run
./c8volt --tenant tenant-a deploy pd --file ./order-process.bpmn
./c8volt get pd --bpmn-process-id order-process --latest
./c8volt get pd --bpmn-process-id order-process --latest --stat
./c8volt run pi -b order-process --vars '{"customerId":"1234"}'
./c8volt get pi --state active
./c8volt get pi --state active --incidents-only --with-age
./c8volt get cluster topology
./c8volt get resource --id resource-id-123
./c8volt config show
./c8volt version
```

## Good in Pipelines

`c8volt` is built to behave well in scripts, CI jobs, and operator toolchains:

- `--json` for structured output
- `--keys-only` for command chaining
- `--auto-confirm` for non-interactive runs
- `--workers` for controlled concurrency
- `--fail-fast` when one error should stop the next wave of work
- `--backoff-*` retry controls for API-facing flows
- `--quiet` and `--verbose` for different execution contexts
- `--profile` and `--config` for environment switching without shell gymnastics
- stable operational error handling and exit behavior

Example:

```bash
./c8volt get pi --bpmn-process-id C88_SimpleUserTask_Process --state active --keys-only
```

And when you want to move from "query" to "bulk action" without leaving the shell:

```bash
./c8volt get pi --state active --keys-only | ./c8volt cancel pi -
./c8volt get pd --bpmn-process-id order-process --latest --keys-only | ./c8volt delete pd - --auto-confirm
```

## Documentation

- Project site: [c8volt.info](https://c8volt.info)
- Search-oriented use cases and FAQ: [c8volt.info/use-cases](https://c8volt.info/use-cases.html)
- Generated CLI reference: [c8volt.info/cli](https://c8volt.info/cli/)

## Copyright

(c) 2026 Adam Bogdan Boczek | <a href="https://boczek.info" target="_blank" rel="noopener noreferrer">boczek.info</a>
