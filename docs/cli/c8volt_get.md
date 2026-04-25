---
title: "c8volt get"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get

Inspect cluster, process, and resource state

### Synopsis

Inspect cluster, process, and resource state without changing it.

Use this command family when you want to look before acting: check cluster health,
list deployed process definitions, inspect process instances, or fetch a known resource.
The leaf commands document their filters and output modes.

```
c8volt get [flags]
```

### Examples

```
  ./c8volt get cluster topology
  ./c8volt get pd --latest
  ./c8volt get pi --state active
  ./c8volt get resource --id <resource-key>
```

### Options

```
  -h, --help   help for get
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json               output as JSON (where applicable)
      --keys-only          output as keys only (where applicable), can be used for piping to other commands
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress all output, except errors, overrides --log-level
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt](c8volt)	 - Operate Camunda 8 workflows from the command line
* [c8volt get cluster](c8volt_get_cluster)	 - Inspect cluster-wide topology and license information
* [c8volt get cluster-topology](c8volt_get_cluster-topology)	 - Show connected cluster topology
* [c8volt get process-definition](c8volt_get_process-definition)	 - List or fetch deployed process definitions
* [c8volt get process-instance](c8volt_get_process-instance)	 - List or fetch process instances
* [c8volt get resource](c8volt_get_resource)	 - Get a resource by id

