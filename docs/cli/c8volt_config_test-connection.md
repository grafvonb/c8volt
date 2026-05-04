---
title: "c8volt config test-connection"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt config test-connection

Test configured Camunda connection

### Synopsis

Test configured Camunda connection.

Loads the effective configuration and logs the config source. The command
validates local configuration before retrieving cluster topology, then warns
when the configured Camunda version differs from the gateway version by
major/minor version.

Use --json for a structured diagnostic payload on stdout; logs remain on stderr.

```
c8volt config test-connection [flags]
```

### Examples

```
  ./c8volt --config ./config.yaml config test-connection
  ./c8volt --config ./config.yaml config test-connection --json
  ./c8volt --profile prod config test-connection
```

### Options

```
  -h, --help   help for test-connection
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

* [c8volt config](c8volt_config)	 - Inspect and validate c8volt configuration

