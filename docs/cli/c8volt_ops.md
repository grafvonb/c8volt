---
title: "c8volt ops"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt ops

Discover high-level operational workflows

### Synopsis

Discover high-level operational workflows.

The ops command family groups operational playbooks for execution, repair, and
future maintenance workflows. This root command is intentionally discovery-only;
target-specific subcommands will define concrete behavior as they are added.

```
c8volt ops [flags]
```

### Examples

```
  ./c8volt ops --help
  ./c8volt capabilities --json
```

### Options

```
  -h, --help   help for ops
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging
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

* [c8volt](c8volt)	 - Operate Camunda 8 workflows from the command line
* [c8volt ops execute](c8volt_ops_execute)	 - Discover predefined operational playbooks
* [c8volt ops repair](c8volt_ops_repair)	 - Discover repair and remediation workflows

