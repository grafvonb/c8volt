---
title: "c8volt get cluster-topology"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get cluster-topology

Show connected cluster topology

### Synopsis

Show connected cluster topology.

This legacy command reports brokers, partitions, and gateway metadata. Prefer `c8volt get cluster topology` for new usage.

```
c8volt get cluster-topology [flags]
```

### Examples

```
  ./c8volt get cluster-topology
  ./c8volt get cluster topology --json
```

### Options

```
  -h, --help   help for cluster-topology
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

