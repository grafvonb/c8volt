---
title: "c8volt cancel process-instance"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt cancel process-instance

Cancel process instance(s) by key(s) and wait for the cancellation to complete

```
c8volt cancel process-instance [flags]
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID to filter process instances
      --dry-run                  perform a dry-run; show which process instances would be cancelled without actually cancelling them
      --fail-fast                stop scheduling new instances after the first error
      --force                    force cancellation of the root process instance if a process instance is a child, including all its child instances
  -h, --help                     help for process-instance
  -k, --key strings              process instance key(s) to cancel
      --no-state-check           skip checking the current state of the process instance before cancelling it
      --no-wait                  skip waiting for the cancellation to be fully processed (no status checks)
      --no-worker-limit          disable limiting the number of workers to GOMAXPROCS when --workers > 1
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
      --keys-only                  output as keys only (where applicable), can be used for piping to other commands
      --log-format string          log format (json, plain, text) (default "plain")
      --log-level string           log level (debug, info, warn, error) (default "info")
      --log-with-source            include source file and line number in logs
      --no-err-codes               suppress error codes in error outputs
      --profile string             config active profile name to use (e.g. dev, prod)
  -q, --quiet                      suppress all output, except errors, overrides --log-level
      --tenant string              default tenant ID
  -v, --verbose                    adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt cancel](c8volt_cancel.md)	 - Cancel resources

