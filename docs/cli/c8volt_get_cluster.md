---
title: "c8volt get cluster"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get cluster

Inspect cluster-wide topology and license information

### Synopsis

Inspect cluster-wide topology and license information.

Use this parent command when you need cluster-level state rather than
process-specific resources. Choose `get cluster topology` to inspect
brokers, partitions, and gateway details, or `get cluster license` to
confirm the connected cluster's license payload.

These subcommands are read-only. Prefer `--json` on the leaf commands for
automation and AI-assisted callers.

```
c8volt get cluster [flags]
```

### Examples

```
  ./c8volt get cluster topology
  ./c8volt get cluster license --json
```

### Options

```
  -h, --help   help for cluster
```

### Options inherited from parent commands

```
  -y, --auto-confirm               auto-confirm prompts for non-interactive use
      --automation                 enable the canonical non-interactive contract for commands that explicitly support it
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
      --config string              path to config file
      --debug                      enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                       output as JSON (where applicable)
      --keys-only                  output as keys only (where applicable), can be used for piping to other commands
      --log-format string          log format (json, plain, text) (default "plain")
      --log-level string           log level (debug, info, warn, error) (default "info")
      --log-with-source            include source file and line number in logs
      --no-err-codes               suppress error codes in error outputs
      --profile string             config active profile name to use (e.g. dev, prod)
  -q, --quiet                      suppress all output, except errors, overrides --log-level
      --tenant string              tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose                    adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt get](c8volt_get)	 - Read cluster, process, and resource state without changing it
* [c8volt get cluster license](c8volt_get_cluster_license)	 - Get the cluster license of the connected Camunda 8 cluster
* [c8volt get cluster topology](c8volt_get_cluster_topology)	 - Get the cluster topology of the connected Camunda 8 cluster

