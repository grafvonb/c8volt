---
title: "c8volt get cluster topology"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get cluster topology

Show connected cluster topology as a tree

### Synopsis

Show connected cluster topology as a sorted human-readable tree.

This command reports brokers, partitions, and gateway metadata for the configured Camunda cluster. Use --json for the structured topology payload.

```
c8volt get cluster topology [flags]
```

### Examples

```
  ./c8volt get cluster topology
  ./c8volt get cluster topology --json
```

### Options

```
  -h, --help   help for topology
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

* [c8volt get cluster](c8volt_get_cluster)	 - Inspect cluster-wide topology, version, and license information

