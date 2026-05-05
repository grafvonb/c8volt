---
title: "c8volt"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
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
  ./c8volt run pi -b C88_SimpleUserTask_Process
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
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt cancel](c8volt_cancel)	 - Cancel running process instances
* [c8volt capabilities](c8volt_capabilities)	 - Describe commands for scripts and agents
* [c8volt config](c8volt_config)	 - Inspect and validate c8volt configuration
* [c8volt delete](c8volt_delete)	 - Delete process instances or definitions
* [c8volt deploy](c8volt_deploy)	 - Deploy BPMN resources to Camunda
* [c8volt embed](c8volt_embed)	 - Use bundled BPMN fixtures
* [c8volt expect](c8volt_expect)	 - Wait for process instances to satisfy expectations
* [c8volt get](c8volt_get)	 - Inspect cluster, process, tenant, and resource state
* [c8volt run](c8volt_run)	 - Start process instances
* [c8volt version](c8volt_version)	 - Print version information
* [c8volt walk](c8volt_walk)	 - Inspect process-instance relationships

