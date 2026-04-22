---
title: "c8volt get"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get

Read cluster, process, and resource state without changing it

### Synopsis

Read cluster, process, and resource state without changing it.

Use this command family when you need to inspect current Camunda state and choose a
resource-specific child command such as cluster topology, process definitions, process
instances, or resources. Prefer `get cluster` for cluster-wide inspection and
use the resource-specific leaf commands when you already know the object type you need.

Where a child command supports structured output, prefer `--json` for automation
and AI-assisted callers instead of scraping the default human-readable output.

```
c8volt get [flags]
```

### Examples

```
  ./c8volt get cluster --help
  ./c8volt get process-instance --json
  ./c8volt get process-definition --keys-only
```

### Options

```
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
  -h, --help                       help for get
```

### Options inherited from parent commands

```
  -y, --auto-confirm        auto-confirm prompts for non-interactive use
      --automation          enable the canonical non-interactive contract for commands that explicitly support it
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                output as JSON (where applicable)
      --keys-only           output as keys only (where applicable), can be used for piping to other commands
      --log-format string   log format (json, plain, text) (default "plain")
      --log-level string    log level (debug, info, warn, error) (default "info")
      --log-with-source     include source file and line number in logs
      --no-err-codes        suppress error codes in error outputs
      --no-indicator        disable transient terminal activity indicators
      --profile string      config active profile name to use (e.g. dev, prod)
  -q, --quiet               suppress all output, except errors, overrides --log-level
      --tenant string       tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose             adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt](c8volt)	 - Operate Camunda 8 with guided help and script-safe output modes
* [c8volt get cluster](c8volt_get_cluster)	 - Inspect cluster-wide topology and license information
* [c8volt get cluster-topology](c8volt_get_cluster-topology)	 - Get the cluster topology of the connected Camunda 8 cluster
* [c8volt get process-definition](c8volt_get_process-definition)	 - List or fetch deployed process definitions
* [c8volt get process-instance](c8volt_get_process-instance)	 - List or fetch process instances
* [c8volt get resource](c8volt_get_resource)	 - Get a resource by id

