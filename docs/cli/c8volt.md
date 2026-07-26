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
state changes, walk process trees, cancel, and delete.

Supports Camunda 8.7, 8.8, and 8.9. Use capabilities for the machine-readable
command contract.

```
c8volt [flags]
```

### Examples

```
  ./c8volt config show --template
  ./c8volt --config ./config.yaml config show --validate
  ./c8volt get cluster topology
  ./c8volt embed deploy --all --run
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id>
  ./c8volt capabilities --json
  ./c8volt get --help
```

### Options

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging
  -h, --help               help for c8volt
  -j, --json               output as JSON (where applicable)
      --keys-only          output keys only (where applicable)
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress output except errors
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
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
* [c8volt ops]({{ "/cli/c8volt_ops" | relative_url }})	 - Discover high-level operational workflows
* [c8volt resolve]({{ "/cli/c8volt_resolve" | relative_url }})	 - Resolve operational incidents
* [c8volt run]({{ "/cli/c8volt_run" | relative_url }})	 - Start process instances
* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources
* [c8volt version]({{ "/cli/c8volt_version" | relative_url }})	 - Print version information
* [c8volt walk]({{ "/cli/c8volt_walk" | relative_url }})	 - Inspect process-instance relationships

