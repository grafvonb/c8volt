---
title: "c8volt delete process-instance"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete process-instance

Delete a process instance by its key

```
c8volt delete process-instance [flags]
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID to filter process instances
      --fail-fast                stop scheduling new instances after the first error
      --force                    force cancellation of the process instance(s), prior to deletion
  -h, --help                     help for process-instance
  -k, --key strings              process instance key(s) to delete
      --no-state-check           skip checking the current state of the process instance before deleting it
      --no-wait                  skip waiting for the deletion to be fully processed (no status checks)
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
  -s, --state string             state to filter process instances: all, active, completed, canceled (default "all")
  -w, --workers int              maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))
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

* [c8volt delete](c8volt_delete.md)	 - Delete resources

