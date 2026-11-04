---
title: "c8volt delete process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete process-instance

Delete process instances by key or filters

### Synopsis

Delete process instances and their history by key or search filters.

The plan includes affected families and deletes children before parents.
Unfinished instances require --force, which permits cancellation before retrying
deletion. By default, wait until deletion is observed.

Use --dry-run to review the family. --limit bounds matching seeds, not relatives.
Business-ID selection with --bid is available only c8.10+.
Empty searches finish without confirmation or mutation.

```
c8volt delete process-instance [flags]
```

### Examples

```
  c8volt delete process-instance --key <process-instance-key> --dry-run
  c8volt delete process-instance --bpmn-process-id order --state completed --dry-run
  c8volt delete process-instance --key <process-instance-key> --force
```

### Options

```
  -n, --batch-size int32            process instances per discovery page (server maximum 1000) (default 1000)
      --bid string                  exact business ID to select (only c8.10+); one opaque value
  -b, --bpmn-process-id string      BPMN process ID to filter process instances
      --dry-run                     preview delete scope without submitting deletion or cancel-before-delete requests
      --end-date-after string       only include process instances with end date >= YYYY-MM-DD
      --end-date-before string      only include process instances with end date <= YYYY-MM-DD
      --end-date-newer-days int     only include process instances with end date N days old or newer (0 means today) (default -1)
      --end-date-older-days int     only include process instances with end date N days old or older (default -1)
      --fail-fast                   stop scheduling new instances after the first error
      --force                       allow cancellation when deletion encounters nonterminal process instances
  -h, --help                        help for process-instance
  -k, --key strings                 process instance key(s) to delete; repeat or combine with stdin '-'
  -l, --limit int32                 maximum matching instances to select; omit for all matches
      --no-state-check              skip checking the current state of the process instance before deleting it
      --no-wait                     return after deletion is accepted
      --no-worker-limit             use all queued jobs as workers when --workers is unset
      --pd-version int32            process definition version
      --pd-version-tag string       process definition version tag
      --start-date-after string     only include process instances with start date >= YYYY-MM-DD
      --start-date-before string    only include process instances with start date <= YYYY-MM-DD
      --start-date-newer-days int   only include process instances N days old or newer (0 means today) (default -1)
      --start-date-older-days int   only include process instances N days old or older (default -1)
  -s, --state string                state to filter process instances: all, active, suspended (only c8.10+), completed, canceled, terminated (default "all")
  -w, --workers int                 maximum concurrent workers; omit for an automatic limit
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

