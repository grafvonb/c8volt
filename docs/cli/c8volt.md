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

Operate Camunda 8 with c8volt v5: inspect, repair, deploy, and clean up.

Documentation and examples target Camunda 8.10; 8.9 and 8.8 remain supported.
Set app.camunda_version to match your cluster. Start with config test-connection,
then use ops playbooks for repair, retention, purge, and runtime analysis.

Preview changes with --dry-run where supported. Use --auto-confirm for unattended
execution and --json for scripts. See capabilities --json for command support.

```
c8volt [flags]
```

### Examples

```
  c8volt config test-connection
  c8volt get process-instance --state active --limit 10
  c8volt ops repair incident --key <incident-key> --dry-run
```

### Options

```
      --all-tenants          search all accessible tenants; cannot combine with --tenant
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
      --tenant string        tenant for discovery and creation; explicit keys use backend authorization
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

