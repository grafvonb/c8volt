---
title: "c8volt capabilities"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt capabilities

Describe commands for scripts and agents

### Synopsis

Describe the public c8volt command contract for scripts, CI jobs, and agents.

Use --json for command paths, flags, output modes, mutation behavior, contract support, and automation support.

```
c8volt capabilities [flags]
```

### Examples

```
  ./c8volt capabilities
  ./c8volt capabilities --json
```

### Options

```
  -h, --help   help for capabilities
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

