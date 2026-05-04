---
title: "c8volt config template"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt config template

Print a blank configuration template

### Synopsis

Print a blank configuration template.

Renders the same blank configuration template as `config show --template`.

```
c8volt config template [flags]
```

### Examples

```
  ./c8volt config template
```

### Options

```
  -h, --help   help for template
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

