---
title: "c8volt config validate"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt config validate

Validate effective configuration

### Synopsis

Validate effective configuration.

Loads the effective configuration through the normal config resolver and uses
the same validation behavior as `config show --validate`.

Tenant context describes configuration scope only: a named tenant is a discovery
filter, while an empty tenant means no configured tenant filter and is not
reported as <default>. Human diagnostics report explicit --tenant changes before
the resulting scope; --tenant "" warns when it clears a named configured filter.

```
c8volt config validate [flags]
```

### Examples

```
  ./c8volt --config ./config.yaml config validate
  ./c8volt --profile prod config validate
  ./c8volt --tenant tenant-a config validate
  ./c8volt --tenant "" config validate
```

### Options

```
  -h, --help   help for validate
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
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt config]({{ "/cli/c8volt_config" | relative_url }})	 - Inspect and validate c8volt configuration

