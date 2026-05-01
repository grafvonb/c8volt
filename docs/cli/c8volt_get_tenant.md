---
title: "c8volt get tenant"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get tenant

List tenants

### Synopsis

List tenants visible to the configured environment.

Human output includes tenant ID, name, and description when available.

```
c8volt get tenant [flags]
```

### Examples

```
  ./c8volt get tenant
  ./c8volt get tenant --key <tenant-id>
  ./c8volt get tenant --filter demo
  ./c8volt get tenant --json
  ./c8volt get tenant --key <tenant-id> --json
  ./c8volt get tenant --keys-only
```

### Options

```
  -f, --filter string   literal tenant name contains filter
  -h, --help            help for tenant
  -k, --key string      tenant ID to fetch
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

* [c8volt get](c8volt_get)	 - Inspect cluster, process, tenant, and resource state

