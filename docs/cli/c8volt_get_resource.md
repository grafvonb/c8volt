---
title: "c8volt get resource"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get resource

Get a resource by id

### Synopsis

Get a single resource by id.

Requires --id.

```
c8volt get resource [flags]
```

### Examples

```
  ./c8volt get resource --id <resource-key>
  ./c8volt --json get resource --id <resource-key>
  ./c8volt --keys-only get resource --id <resource-key>
```

### Options

```
  -h, --help        help for resource
  -i, --id string   resource id to fetch
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

* [c8volt get](c8volt_get)	 - Inspect cluster, process, and resource state

