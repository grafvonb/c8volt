---
title: "c8volt ops purge orphan-process-instances"
nav_exclude: true
---

## c8volt ops purge orphan-process-instances

Purge orphan child process instances

### Synopsis

Purge orphan child process instances.

The workflow discovers child process instances with missing parents, freezes the discovered key set, validates the delete plan, and then either reports the plan with --dry-run or submits deletion only after confirmation. Use --auto-confirm or --automation for unattended deletion, combine --automation with --json for deterministic machine output, and use --report-file to write an audit report.

```
c8volt ops purge orphan-process-instances [flags]
```

### Examples

```
  ./c8volt ops purge orphan-process-instances --dry-run
  ./c8volt ops purge orphan-process-instances --dry-run --bpmn-process-id order-process --limit 25
  ./c8volt ops purge orphan-process-instances --automation --json --dry-run
  ./c8volt ops purge orphan-process-instances --state completed --limit 25 --auto-confirm
  ./c8volt ops purge orphan-process-instances --dry-run --report-file orphan-purge.md
  ./c8volt ops purge orphan-process-instances --state completed --limit 25 --auto-confirm --report-file orphan-purge.json --report-format json
```

### Options

```
  -n, --batch-size int32            number of process instances to inspect per page (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string      BPMN process ID to filter process instances
      --dry-run                     discover and validate orphan process-instance cleanup without submitting deletion requests
      --end-date-after string       only include process instances with end date >= YYYY-MM-DD
      --end-date-before string      only include process instances with end date <= YYYY-MM-DD
      --end-date-newer-days int     only include process instances with end date N days old or newer (0 means today) (default -1)
      --end-date-older-days int     only include process instances with end date N days old or older (default -1)
      --fail-fast                   stop scheduling validation work after the first error
      --force                       force cancellation of the process instance(s), prior to deletion
  -h, --help                        help for orphan-process-instances
      --incidents-only              show only process instances that have incidents
  -l, --limit int32                 maximum number of matching child process instances to inspect across all pages
      --no-incidents-only           show only process instances that have no incidents
      --no-wait                     return after deletion requests are accepted without deletion confirmation
      --no-worker-limit             use all queued jobs as workers when --workers is unset
      --parent-key string           parent process instance key to narrow orphan-child discovery
      --pd-key string               process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)
      --pd-version int32            process definition version
      --pd-version-tag string       process definition version tag
      --report-file string          write an audit report to the given path
      --report-format string        audit report format: markdown, json (default inferred from report-file extension)
      --start-date-after string     only include process instances with start date >= YYYY-MM-DD
      --start-date-before string    only include process instances with start date <= YYYY-MM-DD
      --start-date-newer-days int   only include process instances N days old or newer (0 means today) (default -1)
      --start-date-older-days int   only include process instances N days old or older (default -1)
  -s, --state string                state to filter process instances: all, active, completed, canceled, terminated (default "all")
  -w, --workers int                 maximum concurrent workers when validating the delete plan (default: min(targets, 2*GOMAXPROCS, 32))
```

### Options inherited from parent commands

```
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
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt ops purge](c8volt_ops_purge)	 - Discover destructive operational cleanup workflows

