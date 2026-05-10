---
title: "c8volt get incident"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get incident

List or fetch incidents

### Synopsis

Get Camunda incidents by key or by search criteria.

The command accepts repeated --key values or newline-separated keys from stdin with '-'. Each unique incident key is fetched once and rendered through the shared get output modes.

When no keys are supplied, incidents are searched by state, error type, error message, process context, flow-node context, and creation time. Search mode defaults to active incidents and follows the shared get paging and limit conventions.

Use --json for the stable incident payload, --keys-only for incident keys, --pi-keys-only for process instance keys, --error-message-limit to shorten long error messages, or --with-no-error-message to omit them.

```
c8volt get incident [flags]
```

### Examples

```
  ./c8volt get incident --key <incident-key>
  ./c8volt get inc --key <incident-key> --key <another-incident-key>
  printf '%s\n' "$INCIDENT_KEY_A" "$INCIDENT_KEY_B" | ./c8volt get incident -
  ./c8volt get pi --with-incidents --keys-only | ./c8volt get inc -
  ./c8volt get incident --state active --limit 5
  ./c8volt get incident --state resolved --error-type io_mapping_error --limit 5
  ./c8volt get incident --state active --error-type job_no_retries --pi-keys-only
  ./c8volt get incident --state active --error-type job_no_retries --pi-keys-only | ./c8volt cancel pi --dry-run -
  ./c8volt get incident --error-message "intentional" --limit 5
  ./c8volt get incident --creation-time-after 2026-05-01T00:00:00Z --creation-time-before 2026-05-31T00:00:00Z --limit 5
  ./c8volt get incident --pi-key <process-instance-key> --flow-node-id <flow-node-id>
  ./c8volt --json get incident --key <incident-key>
  ./c8volt --keys-only get incident --key <incident-key>
```

### Options

```
  -n, --batch-size int32              number of incidents to fetch per page (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string        BPMN process ID to filter incidents
      --creation-time-after string    only include incidents with creation time >= RFC3339 timestamp or YYYY-MM-DD
      --creation-time-before string   only include incidents with creation time <= RFC3339 timestamp or YYYY-MM-DD
      --error-message string          case-insensitive incident error message substring filter for search
      --error-message-limit int       maximum characters to show for incident messages; 0 keeps full messages
      --error-type string             case-insensitive incident error type filter for search
      --fail-fast                     stop scheduling new incident lookups after the first error
      --flow-node-id string           flow node ID to filter incidents
      --fni-key string                flow node instance key to filter incidents
  -h, --help                          help for incident
  -k, --key strings                   incident key(s) to fetch; repeat or combine with stdin '-'
  -l, --limit int32                   maximum number of matching incidents to return across all pages
      --no-worker-limit               disable limiting the number of workers to GOMAXPROCS when --workers > 1
      --pd-key string                 process definition key to filter incidents
      --pi-key string                 process instance key to filter incidents
      --pi-keys-only                  return only process instance keys for matching incidents
      --root-key string               root process instance key to filter incidents
  -s, --state string                  incident state scope for search: active, pending, resolved, migrated, unknown, all (default "active")
      --total                         return only the exact numeric total of matching incidents
      --with-no-error-message         omit error messages from incident output
  -w, --workers int                   maximum concurrent workers when fetching multiple incidents (default: min(count, GOMAXPROCS))
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

* [c8volt get](c8volt_get)	 - Inspect cluster, process, incident, tenant, and resource state

