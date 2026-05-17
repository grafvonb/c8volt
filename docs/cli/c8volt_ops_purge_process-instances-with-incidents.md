---
title: "c8volt ops purge process-instances-with-incidents"
nav_exclude: true
---

## c8volt ops purge process-instances-with-incidents

Purge process instances selected by incidents

### Synopsis

Purge process instances selected by incidents.

The workflow discovers candidate incidents from incident filters, freezes the candidate process-instance keys, validates the delete plan, and then either reports the plan with --dry-run or submits deletion only after confirmation. Use --auto-confirm or --automation for unattended deletion, combine --automation with --json for deterministic machine output, and use --report-file to write an audit report.

```
c8volt ops purge process-instances-with-incidents [flags]
```

### Examples

```
  ./c8volt ops purge process-instances-with-incidents --dry-run
  ./c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --dry-run
  ./c8volt ops purge process-instances-with-incidents --state active --limit 5 --dry-run
  ./c8volt ops purge pi-with-incidents --state active --dry-run
  ./c8volt ops purge process-instances-with-incidents --automation --json --dry-run
  ./c8volt ops purge process-instances-with-incidents --dry-run --report-file incident-purge.md
  ./c8volt ops purge process-instances-with-incidents --state active --error-type job_no_retries --limit 5 --auto-confirm --force
  ./c8volt ops purge process-instances-with-incidents --state active --error-type job_no_retries --limit 5 --auto-confirm --force --workers 4 --report-file incident-purge.json --report-format json
```

### Options

```
  -n, --batch-size int32              number of incidents to inspect per page (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string        BPMN process ID to filter incidents
      --creation-time-after string    only include incidents with creation time >= RFC3339 timestamp or YYYY-MM-DD
      --creation-time-before string   only include incidents with creation time <= RFC3339 timestamp or YYYY-MM-DD
      --dry-run                       discover and validate incident-based process-instance cleanup without submitting deletion requests
      --error-message string          case-insensitive incident error message substring filter for discovery
      --error-type string             case-insensitive incident error type filter for discovery
      --fail-fast                     stop scheduling validation or deletion work after the first error
      --flow-node-id string           flow node ID to filter incidents
      --fni-key string                flow node instance key to filter incidents
      --force                         force cancellation of the process instance(s), prior to deletion
  -h, --help                          help for process-instances-with-incidents
  -k, --key strings                   incident key(s) to select for candidate discovery
  -l, --limit int32                   maximum number of matching incidents to inspect before candidate process-instance dedupe
      --no-wait                       return after deletion requests are accepted without deletion confirmation
      --no-worker-limit               use all queued jobs as workers when --workers is unset
      --pd-key string                 process definition key to filter incidents
      --pi-key string                 process instance key to filter incidents
      --report-file string            write an audit report to the given path
      --report-format string          audit report format: markdown, json (default inferred from report-file extension)
      --root-key string               root process instance key to filter incidents
  -s, --state string                  incident state scope for discovery: active, pending, resolved, migrated, unknown, all (default "active")
  -w, --workers int                   maximum concurrent workers when validating the delete plan and deleting roots (default: min(targets, 2*GOMAXPROCS, 32))
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

