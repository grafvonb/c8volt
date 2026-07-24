---
title: "c8volt ops analyse slow-process-instances"
nav_exclude: true
---

## c8volt ops analyse slow-process-instances

Analyze slow process-instance timings

### Synopsis

Analyze slow process-instance timings.

The command is read-only. Select process instances by explicit --key values or by exactly one process-definition selector, then inspect process and runtime element timing without changing cluster state.

Search mode pages through discovered process instances by default. --batch-size controls each discovery page request, --limit caps the frozen analysis scope, and explicit keys bypass discovery paging. JSON and keys-only output stay free of progress text.

Use --dur-longer to keep only process-instance roots whose whole duration is above a threshold. Detail filters such as --element-id, --type, --element-state, and --dur-element-longer keep only process instances with matching element or transition detail rows, then show those matching rows under the root.

Default output shows compact slowest element contributors. Use --with-full-timeline to inspect complete chronological element and transition detail.

Use --with-listeners to include runtime listener jobs under matching element timeline rows.

Duration thresholds use Go duration syntax such as 500ms, 30s, 5m, 1h, 1h30m, or 24h. Calendar units such as 1d are not accepted.

JSON output exposes stable duration, comparison, and timeline fields. Keys-only output prints selected process-instance keys in result order, one per line.

```
c8volt ops analyse slow-process-instances [-] [flags]
```

### Examples

```
  ./c8volt ops analyse slow-process-instances --key 2251799813685249
  ./c8volt ops analyze slow-process-instances --bpmn-process-id OrderProcess --state all --limit 20
  ./c8volt ops analyse spi --bpmn-process-id OrderProcess --dur-longer 5m
  ./c8volt ops analyse slow-process-instances --pd-key 2251799813687001 --dur-element-longer 30s
  ./c8volt ops analyse slow-process-instances --key 2251799813685249 --with-full-timeline
  ./c8volt ops analyse slow-process-instances --key 2251799813685249 --with-listeners
  ./c8volt ops analyse spi --bpmn-process-id OrderProcess --dur-longer 1h30m --dur-element-longer 30s
  ./c8volt get pi --state active --keys-only | ./c8volt ops analyse slow-process-instances -
```

### Options

```
  -n, --batch-size int32            number of process instances to inspect per discovery page; does not cap explicit keys or timeline details (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string      BPMN process ID to discover process instances
      --dur-element-longer string   only include process instances with element or transition detail rows longer than this duration, for example 30s or 2m
      --dur-longer string           only include process instances whose whole duration is longer than this duration, for example 5m or 1h30m
      --element-id string           BPMN element ID to keep in detail rows
      --element-state string        runtime element state to keep in detail rows
      --end-date-after string       only include process instances with end date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --end-date-before string      only include process instances with end date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
  -h, --help                        help for slow-process-instances
  -k, --key strings                 process-instance key(s) to analyze; repeat or combine with stdin '-'
  -l, --limit int32                 maximum number of matching process instances to freeze during discovery; omit to discover all matches
      --no-incidents-only           only include process instances without incidents during discovery
      --pd-key string               process definition key to discover process instances
      --start-date-after string     only include process instances with start date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --start-date-before string    only include process instances with start date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
  -s, --state string                state to filter discovered process instances: all, active, completed, canceled, terminated (default "all")
      --type string                 runtime element type to keep in detail rows
      --with-full-timeline          show complete chronological element and transition detail
      --with-listeners              include runtime listener jobs under matching element timeline rows
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
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt ops analyse](c8volt_ops_analyse)	 - Discover read-only operational analyses

