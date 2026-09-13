---
title: "c8volt ops analyse slow-process-instances"
nav_exclude: true
---

## c8volt ops analyse slow-process-instances

Analyse slow process-instance timings

### Synopsis

Analyse process-instance and runtime-element durations without changing cluster state.

Select explicit --key values or exactly one process-definition selector. --batch-size controls each discovery request; --limit caps selected instances across all pages. Explicit keys bypass discovery paging.

--dur-longer selects roots whose total duration exceeds a threshold. --element-id, --type, --element-state, and --dur-element-longer restrict analysis to matching element or transition details. Use --with-full-timeline to inspect the complete chronology, or --with-listeners to include runtime listener jobs.

Durations use Go syntax such as 500ms, 30s, 5m, 1h30m, or 24h. Calendar units such as 1d are not supported.

```
c8volt ops analyse slow-process-instances [-] [flags]
```

### Examples

```
  ./c8volt ops analyse slow-process-instances --key <process-instance-key>
  ./c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --state active --dur-longer 5m
  ./c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --batch-size 500 --limit 2000
  ./c8volt ops analyse slow-process-instances --pd-key <process-definition-key> --dur-element-longer 30s
  ./c8volt ops analyse slow-process-instances --key <process-instance-key> --with-full-timeline
  ./c8volt ops analyse slow-process-instances --key <process-instance-key> --with-listeners
  ./c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --element-id <element-id> --dur-element-longer 30s
  ./c8volt get process-instance --state active --keys-only | ./c8volt ops analyse slow-process-instances -
```

### Options

```
  -n, --batch-size int32            number of process instances to inspect per discovery page; does not cap frozen analysis scope, explicit keys, or timeline details (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string      BPMN process ID to discover process instances
      --dur-element-longer string   only include process instances with elements or transitions longer than this duration, for example 30s or 2m
      --dur-longer string           only include process instances whose whole duration is longer than this duration, for example 5m or 1h30m
      --element-id string           BPMN element ID to include in the analysis
      --element-state string        runtime element state to include in the analysis
      --end-date-after string       only include process instances with end date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --end-date-before string      only include process instances with end date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
  -h, --help                        help for slow-process-instances
  -k, --key strings                 process-instance key(s) to analyse; repeat or combine with stdin '-'
  -l, --limit int32                 maximum number of matching process instances to freeze for analysis across all discovery pages; omit to discover all matches
      --no-incidents-only           only include process instances without incidents during discovery
      --pd-key string               process definition key to discover process instances
      --start-date-after string     only include process instances with start date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --start-date-before string    only include process instances with start date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
  -s, --state string                state to filter discovered process instances: all, active, completed, canceled, terminated (default "all")
      --type string                 runtime element type to include in the analysis
      --with-full-timeline          show complete chronological element and transition detail
      --with-listeners              include runtime listener jobs
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

* [c8volt ops analyse]({{ "/cli/c8volt_ops_analyse" | relative_url }})	 - Discover read-only operational analyses

