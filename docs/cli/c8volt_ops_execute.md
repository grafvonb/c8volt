---
title: "c8volt ops execute"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt ops execute

Discover predefined operational playbooks

### Synopsis

Discover predefined operational playbooks.

The execute command group is reserved for future playbooks that discover
target sets and execute existing c8volt resource actions. This grouping command
does not register or run concrete operational workflows by itself.

```
c8volt ops execute [flags]
```

### Examples

```
  ./c8volt ops execute --help
  ./c8volt capabilities --json
```

### Options

```
  -h, --help   help for execute
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

* [c8volt ops](c8volt_ops)	 - Discover high-level operational workflows

