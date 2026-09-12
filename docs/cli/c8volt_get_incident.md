---
title: "c8volt get incident"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get incident

List or fetch incidents

### Synopsis

Get Camunda incidents by key or search criteria.

Provide repeated --key values or newline-separated keys from stdin with '-'. Each unique incident key is fetched once.

Without keys, search by state, error type, error message, process context, element context, and creation time. Search defaults to active incidents. --batch-size controls each discovery request; --limit caps incidents across all pages; --total counts matching incidents.

A --bpmn-process-id selector must match a visible process definition before discovery.

```
c8volt get incident [flags]
```

### Examples

```
  ./c8volt get incident --key <incident-key>
  ./c8volt get incident --key <incident-key> --key <another-incident-key>
  printf '%s\n' "$INCIDENT_KEY_A" "$INCIDENT_KEY_B" | ./c8volt get incident -
  ./c8volt get incident --state active --keys-only | ./c8volt get incident -
  ./c8volt get incident --state active --limit 5
  ./c8volt get incident --state resolved --error-type io_mapping_error --limit 5
  ./c8volt get incident --state active --error-type io_mapping_error --pi-keys-only
  ./c8volt get incident --state active --error-type io_mapping_error --pi-keys-only | ./c8volt cancel process-instance --dry-run -
  ./c8volt get incident --error-message "intentional" --limit 5
  ./c8volt get incident --creation-time-after 2026-05-01T00:00:00Z --creation-time-before 2026-05-31T00:00:00Z --limit 5
  ./c8volt get incident --pi-key <process-instance-key> --element-id <element-id>
  ./c8volt --json get incident --key <incident-key>
  ./c8volt --keys-only get incident --key <incident-key>
```

### Options

```
  -n, --batch-size int32               number of incidents to request per page; does not cap total results (max limit 1000 enforced by server) (default 1000)
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
      --no-worker-limit                use all queued jobs as workers when --workers is unset
      --pd-key string                  process definition key to filter incidents
      --pi-key string                  process instance key to filter incidents
      --pi-keys-only                   return only process instance keys for matching incidents
      --root-key string                root process instance key to filter incidents
  -s, --state string                   incident state scope for search: active, pending, resolved, migrated, unknown, all (default "active")
      --total                          return only the exact numeric total of matching incidents
      --with-no-error-message          omit error messages from incident output
  -w, --workers int                    maximum concurrent workers when fetching multiple incidents (default: min(count, 2*GOMAXPROCS, 32))
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

* [c8volt get]({{ "/cli/c8volt_get" | relative_url }})	 - Inspect cluster, process, job, element, incident, tenant, and resource state

