---
title: "c8volt get cluster"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get cluster

Inspect cluster-wide topology and license information

### Synopsis

Inspect cluster-wide topology and license information.

Use `get cluster topology` to check brokers, partitions, and gateway details.
Use `get cluster license` to inspect the connected cluster's license payload.

```
c8volt get cluster [flags]
```

### Examples

```
  ./c8volt get cluster topology
  ./c8volt get cluster license
```

### Options

```
  -h, --help   help for cluster
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
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

* [c8volt get](c8volt_get)	 - Inspect cluster, process, and resource state
* [c8volt get cluster license](c8volt_get_cluster_license)	 - Show connected cluster license
* [c8volt get cluster topology](c8volt_get_cluster_topology)	 - Show connected cluster topology

