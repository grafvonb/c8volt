---
title: "c8volt delete process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete process-definition

Delete process definition resources

### Synopsis

Delete deployed process definitions and associated history.

Unfinished instances block deletion. With --force, cancel affected roots, verify
that unfinished work has drained, then delete instance history and definitions.
Suspended instances count as unfinished only c8.10+.

Use --dry-run to review impact. By default, wait for definition removal and any
returned deletion batch. Deferred history deletion without a batch cannot be
independently confirmed. Use delete process-instance to remove instances alone.

```
c8volt delete process-definition [flags]
```

### Examples

```
  c8volt delete process-definition --key <process-definition-key> --dry-run
  c8volt delete process-definition --bpmn-process-id order --force
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID of the process definition (all versions) to delete
      --dry-run                  preview process-definition delete impact without submitting deletion or cancellation requests
      --fail-fast                stop scheduling new instances after the first error
      --force                    cancel unfinished process-instance roots before deleting history and definitions
  -h, --help                     help for process-definition
  -k, --key strings              process definition key(s) to delete
      --latest                   fetch the latest version(s) of the given BPMN process(s)
      --no-state-check           skip checking process-instance state before deleting
      --no-wait                  skip final deletion confirmation; required unfinished-instance drain checks still run
      --no-worker-limit          use all queued jobs as workers when --workers is unset
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
  -w, --workers int              maximum concurrent workers; omit for an automatic limit
```

### Options inherited from parent commands

```
      --all-tenants          search all accessible tenants; cannot combine with --tenant
  -y, --auto-confirm         auto-confirm prompts for non-interactive use
      --automation           enable non-interactive mode for commands that explicitly support it
      --config string        path to config file
      --debug                enable debug logging
  -j, --json                 output as JSON (where applicable)
      --keys-only            output keys only (where applicable)
      --log-level string     log level (debug, info, warn, error) (default "info")
      --no-indicator         disable transient terminal activity indicators
      --no-version-check     skip the automatic advisory gateway version check
      --profile string       config active profile name to use (e.g. dev, prod)
  -q, --quiet                suppress output except errors
      --request-rate float   maximum shared request starts per second (fractions allowed; 0 disables pacing)
      --tenant string        tenant for discovery and creation; explicit keys use backend authorization
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt delete]({{ "/cli/c8volt_delete" | relative_url }})	 - Delete process instances or definitions

