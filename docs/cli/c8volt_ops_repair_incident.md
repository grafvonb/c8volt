---
title: "c8volt ops repair incident"
nav_exclude: true
---

## c8volt ops repair incident

Repair incidents by key or filter

### Synopsis

Repair incidents by key or filter.

Tenant contract: incident-filter mode uses discovery semantics, where a named tenant scopes candidate discovery and empty tenant configuration leaves discovery unfiltered. Explicit --tenant changes are reported before scope, and --tenant "" warns when it clears a named configured filter. Direct --key and stdin input use explicit-key semantics and report that the tenant filter is not applied. Selection context appears before discovery or explicit-key resolution; validated affected tenants appear before the confirmation question and the first mutation. --auto-confirm skips only the question and does not suppress tenant context permitted by the selected output mode. Frozen plans and audit reports show one known resource tenant informationally, emit one warning-level "affected tenants" summary when the scope spans multiple tenants, and warn separately for targets with unknown tenant metadata.

The command accepts repeated --key values, newline-separated keys from stdin with '-', or incident search filters. Keyed mode and search mode are mutually exclusive. Search mode pages through all matching incidents by default. --batch-size tunes per-page discovery requests only, and --limit intentionally caps the frozen scope. Human, JSON, and audit report output identify whether discovery completed or was user-limited. It builds a fixed incident target set before mutation, applies process-instance-scope variable updates once per unique scope when requested, applies job retry and timeout updates only when an incident has a related job, resolves each incident, and confirms clearance unless --no-wait is set. Default human output keeps repair progress on one workflow activity and writes compact stderr milestones at most once per 10-second interval, plus immediate failure warnings. Verbose and debug output replace aggregate milestones with one per-incident completion line. JSON and automation output remain free of human progress text; quiet mode suppresses successful progress and retains failure warnings. Incidents without related jobs are reported and still proceed to incident resolution. Use --report-file with Markdown or JSON output for an audit record of discovery, targets, step statuses, notices, errors, and final outcome.

```
c8volt ops repair incident [flags]
```

### Examples

```
  ./c8volt ops repair incident --key <incident-key> --dry-run
  ./c8volt --tenant tenant-a ops repair incident --key <incident-key> --dry-run
  ./c8volt --tenant "" ops repair incident --state active --limit 5 --dry-run
  ./c8volt ops repair incident --state active --error-type io_mapping_error --limit 5 --dry-run
  ./c8volt ops repair incident --key <incident-key> --vars '{"hasIncident":false}' --dry-run
  ./c8volt --verbose ops repair incident --state active --error-type io_mapping_error --limit 5 --auto-confirm
  ./c8volt ops repair incident --key <incident-key> --vars '{"hasIncident":false}' --report-file repair-incident.md
```

### Options

```
  -n, --batch-size int32               number of incidents to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string         BPMN process ID to filter incidents
      --creation-time-after string     only include incidents with creation time >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --creation-time-before string    only include incidents with creation time <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --creation-time-newer-days int   only include incidents with creation time N days old or newer (0 means today) (default -1)
      --creation-time-older-days int   only include incidents with creation time N days old or older (default -1)
      --dry-run                        freeze repair targets and preview repair steps without submitting mutations
      --element-id string              BPMN element ID to filter incidents
      --element-instance-key string    element instance key to filter incidents
      --error-message string           case-insensitive incident error message substring filter for search
      --error-type string              case-insensitive incident error type filter for search
      --fail-fast                      stop scheduling incident repairs after the first error
  -h, --help                           help for incident
      --job-timeout string             timeout duration to submit for related jobs, for example 60s, 5m, or 1h
  -k, --key strings                    incident key(s) to repair; repeat or combine with stdin '-'
  -l, --limit int32                    maximum number of matching incidents to freeze for repair; omit to discover all matches
      --no-wait                        return after repair mutations are accepted without incident or retry confirmation
      --no-worker-limit                use all queued jobs as workers when --workers is unset
      --pd-key string                  process definition key to filter incidents
      --pi-key string                  process instance key to filter incidents
      --report-file string             plan an audit report at the given path
      --report-format string           audit report format: markdown, json (default inferred from report-file extension)
      --retries int32                  retry count to set on related jobs; 0 skips retry restoration (default 1)
      --root-key string                root process instance key to filter incidents
  -s, --state string                   incident state scope for search: active, pending, resolved, migrated, unknown, all (default "active")
      --vars string                    JSON object with variables to set once per process-instance scope before resolving dependent incidents
      --vars-file string               path to JSON object file with variables to set once per process-instance scope
  -w, --workers int                    maximum concurrent workers when repairing multiple incidents (default: min(count, 2*GOMAXPROCS, 32))
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

* [c8volt ops repair]({{ "/cli/c8volt_ops_repair" | relative_url }})	 - Discover repair and remediation workflows

