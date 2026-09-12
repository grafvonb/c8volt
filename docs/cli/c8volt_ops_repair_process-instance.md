---
title: "c8volt ops repair process-instance"
nav_exclude: true
---

## c8volt ops repair process-instance

Repair incidents selected by process instances

### Synopsis

Repair active incidents associated with selected process instances.

Provide repeated --key values, newline-separated keys from stdin with '-', or process-instance search filters. Search selects incident-bearing instances; --direct-incidents-only restricts matching to direct active incidents.

The workflow fixes the repairable instance and incident sets, applies requested variables once per process-instance scope, then updates related jobs, resolves incidents, and confirms clearance unless --no-wait is set.

--tenant limits search; an empty tenant or --all-tenants searches across accessible tenants. Explicit keys use backend authorization without tenant filtering. --batch-size controls each discovery request; --limit caps the selected scope.

Use --dry-run to inspect planned repairs without mutation, --auto-confirm or --automation for unattended repair, and --report-file to save an audit report.

```
c8volt ops repair process-instance [flags]
```

### Examples

```
  ./c8volt ops repair process-instance --key <process-instance-key> --dry-run
  ./c8volt --tenant tenant-a ops repair process-instance --key <process-instance-key> --dry-run
  ./c8volt --tenant "" ops repair process-instance --state active --limit 5 --dry-run
  ./c8volt ops repair process-instance --state active --limit 5 --dry-run
  ./c8volt ops repair process-instance --direct-incidents-only --bpmn-process-id <bpmn-process-id> --limit 5 --dry-run
  ./c8volt --verbose ops repair process-instance --state active --limit 5 --auto-confirm
  ./c8volt ops repair process-instance --key <process-instance-key> --vars '{"hasIncident":false}' --report-file repair-process-instance.md
```

### Options

```
  -n, --batch-size int32                number of process instances to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string          BPMN process ID to filter process instances
      --children-only                   select only child process instances
      --direct-incidents-only           select only process instances with direct active incidents
      --dry-run                         freeze repair targets and preview repair steps without submitting mutations
      --end-date-after string           only include process instances with end date >= YYYY-MM-DD
      --end-date-before string          only include process instances with end date <= YYYY-MM-DD
      --end-date-newer-days int         only include process instances with end date N days old or newer (0 means today) (default -1)
      --end-date-older-days int         only include process instances with end date N days old or older (default -1)
      --fail-fast                       stop scheduling incident repairs after the first error
  -h, --help                            help for process-instance
      --incident-error-message string   case-insensitive incident error message substring filter for --direct-incidents-only
      --incident-error-type string      case-insensitive incident error type filter for --direct-incidents-only
      --incident-state string           incident state scope for --direct-incidents-only: active, pending, resolved, migrated, unknown, all (default "active")
      --job-timeout string              timeout duration to submit for related jobs, for example 60s, 5m, or 1h
  -k, --key strings                     process-instance key(s) whose active incidents should be repaired; repeat or combine with stdin '-'
  -l, --limit int32                     maximum number of matching process instances to freeze for repair; omit to discover all matches
      --no-wait                         return after repair mutations are accepted without incident or retry confirmation
      --no-worker-limit                 use all queued jobs as workers when --workers is unset
      --parent-key string               parent process instance key to filter process instances
      --pd-key string                   process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)
      --pd-version int32                process definition version
      --pd-version-tag string           process definition version tag
      --report-file string              plan an audit report at the given path
      --report-format string            audit report format: markdown, json (default inferred from report-file extension)
      --retries int32                   retry count to set on related jobs; 0 skips retry restoration (default 1)
      --roots-only                      select only root process instances
      --start-date-after string         only include process instances with start date >= YYYY-MM-DD
      --start-date-before string        only include process instances with start date <= YYYY-MM-DD
      --start-date-newer-days int       only include process instances N days old or newer (0 means today) (default -1)
      --start-date-older-days int       only include process instances N days old or older (default -1)
  -s, --state string                    state to filter process instances: all, active, completed, canceled, terminated (default "all")
      --vars string                     JSON object with variables to set once per process-instance scope before resolving dependent incidents
      --vars-file string                path to JSON object file with variables to set once per process-instance scope
  -w, --workers int                     maximum concurrent workers when repairing multiple incidents (default: min(count, 2*GOMAXPROCS, 32))
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

