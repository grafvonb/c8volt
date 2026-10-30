---
title: "c8volt"
permalink: /cli/c8volt/
parent: "CLI Reference"
nav_order: 1
nav_exclude: false
---

## c8volt

Operate Camunda 8 workflows from the command line

### Synopsis

c8volt: Camunda 8 Operations CLI.

Deploy BPMN models, start process instances, inspect workflow state, wait for
state changes, walk process trees, suspend, cancel, and delete.

Supports Camunda 8.8, 8.9, and 8.10.
Camunda 8.10 baseline: 8.10.3 (final).
Camunda 8.10 aliases: 8.10, 810, v810, v8.10. Default: 8.9.
Before the first operational API request, check the gateway major/minor version once.
Mismatch or lookup failure warns and retains the configured adapter. The check
waits at most two seconds (or a shorter HTTP/workflow timeout), uses shared
request pacing, and can add a pacing slot. Disable it with --no-version-check.
Local commands perform no check. Explicit version and connection tests still run.

Use capabilities to discover supported commands and automation options.

Use --request-rate to pace shared request starts per second for this invocation
(default 0, disabled; positive fractions allowed). It does not change worker
concurrency or extend request/workflow deadlines. Explicit zero overrides config.
Local wait failures identify an unsent request; earlier work may already be accepted.

DEBUG request-rate summaries work with or without pacing. Configure
http.request_rate_report_interval (default 5s; e.g. 10s; 0s disables summaries).
They report actual starts, elapsed time and requests/second through the configured
logger. Summaries pause during manual input and exclude that time from the reported
duration and rate. Reporting stops with the invocation; short commands may emit no summary.

On API-backed commands, --debug emits redacted request diagnostics at DEBUG
alongside other configured logs. --quiet and log-level filtering still apply;
log format, timestamps, and source settings use the existing logger.

```
c8volt [flags]
```

### Examples

```
  ./c8volt config show --template
  ./c8volt --config ./config.yaml config show --validate
  ./c8volt get cluster topology
  ./c8volt --request-rate 10 cancel process-instance --state active
  ./c8volt --config ./config.yaml --debug get process-definition --latest
  ./c8volt --all-tenants get process-instance --state active
  ./c8volt embed deploy --all --run
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id>
  ./c8volt capabilities --json
  ./c8volt get --help
```

### Options

```
      --all-tenants          clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
  -y, --auto-confirm         auto-confirm prompts for non-interactive use
      --automation           enable non-interactive mode for commands that explicitly support it
      --config string        path to config file
      --debug                enable debug logging
  -h, --help                 help for c8volt
  -j, --json                 output as JSON (where applicable)
      --keys-only            output keys only (where applicable)
      --log-level string     log level (debug, info, warn, error) (default "info")
      --no-indicator         disable transient terminal activity indicators
      --no-version-check     skip the automatic advisory gateway version check
      --profile string       config active profile name to use (e.g. dev, prod)
  -q, --quiet                suppress output except errors
      --request-rate float   maximum shared request starts per second (fractions allowed; 0 disables pacing)
      --tenant string        tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt cancel]({{ "/cli/c8volt_cancel" | relative_url }})	 - Cancel running process instances
* [c8volt capabilities]({{ "/cli/c8volt_capabilities" | relative_url }})	 - Describe commands for scripts and agents
* [c8volt config]({{ "/cli/c8volt_config" | relative_url }})	 - Inspect and validate c8volt configuration
* [c8volt delete]({{ "/cli/c8volt_delete" | relative_url }})	 - Delete process instances or definitions
* [c8volt deploy]({{ "/cli/c8volt_deploy" | relative_url }})	 - Deploy BPMN resources to Camunda
* [c8volt embed]({{ "/cli/c8volt_embed" | relative_url }})	 - Use bundled BPMN fixtures
* [c8volt expect]({{ "/cli/c8volt_expect" | relative_url }})	 - Wait for process instances to satisfy expectations
* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state
* [c8volt ops]({{ "/cli/c8volt_ops" | relative_url }})	 - Run operational playbooks
* [c8volt resolve]({{ "/cli/c8volt_resolve" | relative_url }})	 - Resolve operational incidents
* [c8volt resume]({{ "/cli/c8volt_resume" | relative_url }})	 - Resume suspended process instances
* [c8volt run]({{ "/cli/c8volt_run" | relative_url }})	 - Start process instances
* [c8volt suspend]({{ "/cli/c8volt_suspend" | relative_url }})	 - Suspend active process instances
* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources
* [c8volt version]({{ "/cli/c8volt_version" | relative_url }})	 - Print version information
* [c8volt walk]({{ "/cli/c8volt_walk" | relative_url }})	 - Inspect process-instance relationships

