---
title: "c8volt get cluster license"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get cluster license

Get the cluster license of the connected Camunda 8 cluster

### Synopsis

Get the cluster license of the connected Camunda 8 cluster.

This read-only command requires a configured Camunda 8 connection. Prefer `--json` when automation needs the raw license payload instead of the default rendered output.

```
c8volt get cluster license [flags]
```

### Examples

```
  ./c8volt get cluster license
  ./c8volt get cluster license --json
```

### Options

```
  -h, --help   help for license
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

* [c8volt get cluster](c8volt_get_cluster)	 - Inspect cluster-wide topology and license information

