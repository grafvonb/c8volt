---
title: "c8volt version"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt version

Print version information

### Synopsis

Print version information.

Use --json for version metadata.

```
c8volt version [flags]
```

### Examples

```
  ./c8volt version
  ./c8volt version --json
```

### Options

```
  -h, --help   help for version
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

