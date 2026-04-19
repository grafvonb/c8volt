---
title: "c8volt delete process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete process-definition

Delete a process definition(s)

```
c8volt delete process-definition [flags]
```

### Examples

```
  ./c8volt delete pd --key 2251799813686017 --auto-confirm
  ./c8volt delete pd --bpmn-process-id order-process --latest --force
  ./c8volt get pd --bpmn-process-id order-process --latest --keys-only | ./c8volt delete pd - --auto-confirm
```

### Options

```
      --allow-inconsistent       allow deletion of process definitions even if their state will become inconsistent (not deleted from Operate's data)
  -b, --bpmn-process-id string   BPMN process ID of the process definition (all versions) to delete
      --fail-fast                stop scheduling new instances after the first error
      --force                    force cancellation of the process instance(s), prior to deletion
  -h, --help                     help for process-definition
  -k, --key strings              process definition key(s) to delete
      --latest                   fetch the latest version(s) of the given BPMN process(s)
      --no-state-check           skip checking the current state of the process instance(s) of the process definition before deleting it
      --no-wait                  skip waiting for the deletion to be fully processed
      --no-worker-limit          disable limiting the number of workers to GOMAXPROCS when --workers > 1
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
  -w, --workers int              maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))
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

* [c8volt delete](c8volt_delete)	 - Delete resources

