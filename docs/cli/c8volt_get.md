---
title: "c8volt get"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get

Get resources

### Synopsis

Get resources such as process definitions or process instances.
It is a root command and requires a subcommand to specify the resource type to get.

```
c8volt get [flags]
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
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                output as JSON (where applicable)
      --keys-only           output as keys only (where applicable), can be used for piping to other commands
      --log-format string   log format (json, plain, text) (default "plain")
      --log-level string    log level (debug, info, warn, error) (default "info")
      --log-with-source     include source file and line number in logs
      --no-err-codes        suppress error codes in error outputs
      --profile string      config active profile name to use (e.g. dev, prod)
  -q, --quiet               suppress all output, except errors, overrides --log-level
      --tenant string       default tenant ID
  -v, --verbose             adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt](c8volt.md)	 - c8volt: Camunda 8 Operations CLI
* [c8volt get cluster-topology](c8volt_get_cluster-topology.md)	 - Get the cluster topology of the connected Camunda 8 cluster
* [c8volt get process-definition](c8volt_get_process-definition.md)	 - Get deployed process definitions
* [c8volt get process-instance](c8volt_get_process-instance.md)	 - Get process instances

