---
title: "c8volt get process-instance"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-instance

Get process instances

```
c8volt get process-instance [flags]
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID to filter process instances
      --children-only            show only child process instances, meaning instances that have a parent key set
  -n, --count int32              number of process instances to fetch (max limit 1000 enforced by server) (default 1000)
  -h, --help                     help for process-instance
      --incidents-only           show only process instances that have incidents
  -k, --key string               process instance key to fetch
      --no-incidents-only        show only process instances that have no incidents
      --orphan-children-only     show only child instances where parent key is set but the parent process instance does not exist (anymore)
      --parent-key string        parent process instance key to filter process instances
      --pd-key string            process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
      --roots-only               show only root process instances, meaning instances with empty parent key
  -s, --state string             state to filter process instances: all, active, completed, canceled (default "all")
```

### Options inherited from parent commands

```
  -y, --auto-confirm               auto-confirm prompts for non-interactive use
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
      --config string              path to config file
      --debug                      enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                       output as JSON (where applicable)
      --keys-only                  output as keys only (where applicable), can be used for piping to other commands, like cancel or delete
      --log-format string          log format (json, plain, text) (default "plain")
      --log-level string           log level (debug, info, warn, error) (default "info")
      --log-with-source            include source file and line number in logs
      --no-err-codes               suppress error codes in error outputs
      --profile string             config active profile name to use (e.g. dev, prod)
  -q, --quiet                      suppress all output, except errors, overrides --log-level
      --tenant string              default tenant ID
```

### SEE ALSO

* [c8volt get](c8volt_get.md)	 - Get resources

