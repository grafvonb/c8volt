---
title: "c8volt delete process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete process-instance

Delete process instances by key or filters

### Synopsis

Delete process instances by key or search filters.

c8volt validates the affected tree before any deletion, asks for confirmation, and waits until deletion is observed. Nonterminal instances block deletion unless --force is set.

--force allows cancellation when deletion encounters a nonterminal instance. Deletion traverses children first and retries after cancellation; the entire scope is not canceled before any deletion.

--tenant limits search-derived selection. An empty tenant or --all-tenants leaves discovery unfiltered across accessible tenants. Explicit --key and stdin keys use backend authorization without tenant filtering.

A --bpmn-process-id selector must match a visible process definition before instance discovery. An empty selection completes without confirmation or mutation.

Search mode plans all selected pages before one confirmation and deletion. --batch-size controls each discovery request; --limit caps the selected scope across all pages. --workers, --fail-fast, and --no-worker-limit control planning and deletion work.

Use --dry-run to preview the affected family without deleting or cancelling. Use --auto-confirm for unattended deletion.

```
c8volt delete process-instance [flags]
```

### Examples

```
  ./c8volt delete process-instance --key <process-instance-key> --force
  ./c8volt delete process-instance --key <process-instance-key> --dry-run
  ./c8volt --tenant tenant-a delete process-instance --key <process-instance-key> --dry-run
  ./c8volt --tenant tenant-a delete process-instance --state terminated --limit 5 --dry-run
  ./c8volt --tenant "" delete process-instance --state terminated --limit 5 --dry-run
  ./c8volt delete process-instance --state terminated --batch-size 250 --limit 5 --dry-run
  ./c8volt delete process-instance --state terminated --json --dry-run
  ./c8volt delete process-instance --state terminated --keys-only
  ./c8volt delete process-instance --state terminated --end-date-after 2026-05-01 --end-date-before 2026-05-31 --limit 5 --dry-run
  ./c8volt delete process-instance --bpmn-process-id <bpmn-process-id> --state terminated --batch-size 250 --limit 5 --dry-run
  ./c8volt --verbose delete process-instance --state terminated --limit 25 --auto-confirm
  ./c8volt expect process-instance --key <process-instance-key> --state absent
```

### Options

```
  -n, --batch-size int32            number of process instances to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server) (default 1000)
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
  -l, --limit int32                 maximum number of matching process instances to freeze for deletion across all pages; omit to continue through all matches
      --no-state-check              skip checking the current state of the process instance before deleting it
      --no-wait                     return after deletion is accepted
      --no-worker-limit             use all queued jobs as workers when --workers is unset
      --pd-version int32            process definition version
      --pd-version-tag string       process definition version tag
      --start-date-after string     only include process instances with start date >= YYYY-MM-DD
      --start-date-before string    only include process instances with start date <= YYYY-MM-DD
      --start-date-newer-days int   only include process instances N days old or newer (0 means today) (default -1)
      --start-date-older-days int   only include process instances N days old or older (default -1)
  -s, --state string                state to filter process instances: all, active, completed, canceled, terminated (default "all")
  -w, --workers int                 maximum concurrent workers for queued work; mutation work uses root trees (default: min(queued work, 2*GOMAXPROCS, 32)); independent of discovery page size
```

### Options inherited from parent commands

```
      --all-tenants        clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging
  -j, --json               output as JSON (where applicable)
      --keys-only          output keys only (where applicable)
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress output except errors
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt delete]({{ "/cli/c8volt_delete" | relative_url }})	 - Delete process instances or definitions

