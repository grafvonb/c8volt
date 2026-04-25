---
title: "c8volt config"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt config

Manage application configuration

### Synopsis

Manage application configuration.

Use this command family to inspect the effective configuration, validate the values a
command would run with, or generate a copy-pasteable template config file. Choose
`config show` when you need to understand how flags, environment variables,
profiles, and base config resolve into one effective command context.

```
c8volt config [flags]
```

### Examples

```
  ./c8volt config show
  ./c8volt --config ./config.yaml --profile prod config show --validate
  ./c8volt config show --template
```

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable the canonical non-interactive contract for commands that explicitly support it
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

* [c8volt](c8volt)	 - Operate Camunda 8 with guided help and script-safe output modes
* [c8volt config show](c8volt_config_show)	 - Show effective configuration

