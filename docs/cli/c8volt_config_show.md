---
title: "c8volt config show"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt config show

Show effective configuration

### Synopsis

Show effective configuration with sensitive values sanitized.

Precedence: flag > env > profile > base config > default.
Use --validate to check the effective config, or --template for a blank template.

```
c8volt config show [flags]
```

### Examples

```
  ./c8volt config show
  ./c8volt --config ./config.yaml --profile prod config show
  ./c8volt --config ./config.yaml config show --validate
  ./c8volt config show --template
```

### Options

```
  -h, --help       help for show
      --template   print a blank configuration template
      --validate   validate the effective configuration and exit with an error code if invalid
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

