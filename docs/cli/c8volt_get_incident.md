---
title: "c8volt get incident"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get incident

List or fetch incidents

### Synopsis

Inspect incidents by key or search filters. Search defaults to active incidents.

Use --error-type or --pi-key to narrow the scope. Add --pi-keys-only to pass
affected process-instance keys to another command. Use --limit to bound a search.

```
c8volt get incident [flags]
```

### Examples

```
  c8volt get incident --key <incident-key>
  c8volt get incident --state active --error-type io_mapping_error --limit 10
  c8volt get incident --pi-key <process-instance-key> --json
```

### Options

```
  -n, --batch-size int32               incidents per discovery page (server maximum 1000) (default 1000)
  -b, --bpmn-process-id string         BPMN process ID to validate and filter incidents
      --creation-time-after string     only include incidents with creation time >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --creation-time-before string    only include incidents with creation time <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --creation-time-newer-days int   only include incidents with creation time N days old or newer (0 means today) (default -1)
      --creation-time-older-days int   only include incidents with creation time N days old or older (default -1)
      --element-id string              BPMN element ID to filter incidents
      --element-instance-key string    element instance key to filter incidents
      --error-message string           case-insensitive incident error message substring filter for search
      --error-message-limit int        maximum characters to show for incident messages; 0 keeps full messages
      --error-type string              case-insensitive incident error type filter for search
      --fail-fast                      stop scheduling new incident lookups after the first error
  -h, --help                           help for incident
  -k, --key strings                    incident key(s) to fetch; repeat or combine with stdin '-'
  -l, --limit int32                    maximum number of matching incidents to return across all pages; omit to continue through all matches
      --no-error-message               omit error messages from incident output
      --no-worker-limit                use all queued jobs as workers when --workers is unset
      --pd-key string                  process definition key to filter incidents
      --pi-key string                  process instance key to filter incidents
      --pi-keys-only                   return only process instance keys for matching incidents
      --root-key string                root process instance key to filter incidents
  -s, --state string                   incident state scope for search: active, pending, resolved, migrated, unknown, all (default "active")
      --total                          return only the exact numeric total of matching incidents
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

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

