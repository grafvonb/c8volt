---
title: "c8volt ops purge process-instances-with-incidents"
nav_exclude: true
---

## c8volt ops purge process-instances-with-incidents

Purge process instances selected by incidents

### Synopsis

Delete process-instance families selected by incidents.

Select incidents by --inc-key or filters such as --error-type. The workflow
plans family deletion before confirmation; unfinished instances require --force.
Use --dry-run to review the scope and --report-file to save an audit report.
Use --auto-confirm for unattended execution.

```
c8volt ops purge process-instances-with-incidents [flags]
```

### Examples

```
  c8volt ops purge process-instances-with-incidents --error-type io_mapping_error --dry-run
  c8volt ops purge process-instances-with-incidents --inc-key <incident-key> --force --report-file purge.md
```

### Options

```
  -n, --batch-size int32               incidents per discovery page (server maximum 1000) (default 1000)
  -b, --bpmn-process-id string         BPMN process ID to filter incidents
      --creation-time-after string     only include incidents with creation time >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --creation-time-before string    only include incidents with creation time <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --creation-time-newer-days int   only include incidents with creation time N days old or newer (0 means today) (default -1)
      --creation-time-older-days int   only include incidents with creation time N days old or older (default -1)
      --dry-run                        discover and validate incident-based process-instance cleanup without submitting deletion requests
      --element-id string              BPMN element ID to filter incidents
      --element-instance-key string    element instance key to filter incidents
      --error-message string           case-insensitive incident error message substring filter for discovery
      --error-type string              case-insensitive incident error type filter for discovery
      --fail-fast                      stop scheduling validation or deletion work after the first error
      --force                          allow cancellation when deletion encounters nonterminal process instances
  -h, --help                           help for process-instances-with-incidents
      --inc-key strings                incident key(s) to select for candidate discovery
  -l, --limit int32                    maximum matching incidents to select; omit for all matches
      --no-wait                        return after deletion requests are accepted without deletion confirmation
      --no-worker-limit                use all queued jobs as workers when --workers is unset
      --pd-key string                  process definition key to filter incidents
      --pi-key string                  process instance key to filter incidents
      --report-file string             write an audit report to the given path
      --report-format string           audit report format: markdown, json (default inferred from report-file extension)
      --root-key string                root process instance key to filter incidents
  -s, --state string                   incident state scope for discovery: active, pending, resolved, migrated, unknown, all (default "active")
  -w, --workers int                    maximum concurrent workers; omit for an automatic limit
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

* [c8volt ops purge]({{ "/cli/c8volt_ops_purge" | relative_url }})	 - Discover destructive operational cleanup workflows

