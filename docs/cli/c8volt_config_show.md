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
Tenant context in the sanitized document describes configuration scope only:
a named tenant is a discovery filter, while an empty tenant means no configured
tenant filter and is not reported as <default>.
Human diagnostics report explicit --tenant changes before the resulting scope;
--tenant "" warns when it clears a named configured filter.
The --validate and --template flags remain supported as compatibility shortcuts
for validation and template rendering.

```
c8volt config show [flags]
```

### Examples

```
  ./c8volt config show
  ./c8volt --config ./config.yaml --profile prod config show
  ./c8volt --tenant tenant-a config show
  ./c8volt --tenant "" config show
  ./c8volt --config ./config.yaml config show --validate
  ./c8volt config show --template
```

### Options

```
  -h, --help       help for show
      --template   compatibility shortcut: print a blank configuration template
      --validate   compatibility shortcut: validate the effective configuration and exit with an error code if invalid
```

### Options inherited from parent commands

```
      --all-tenants        clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
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

