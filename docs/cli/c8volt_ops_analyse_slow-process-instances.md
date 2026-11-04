---
title: "c8volt ops analyse slow-process-instances"
nav_exclude: true
---

## c8volt ops analyse slow-process-instances

Analyse slow process-instance timings

### Synopsis

Find slow process instances and the elements contributing to their duration.

Select a process with --bpmn-process-id or inspect one instance with --key.
Use --dur-longer for total duration, or --element-id and --dur-element-longer
for a specific element. Add --with-listeners to include listener jobs.

Durations use values such as 30s, 5m, or 24h. Use --limit to bound analysis.
Business-ID selection with --bid is available only c8.10+.

```
c8volt ops analyse slow-process-instances [-] [flags]
```

### Examples

```
  c8volt ops analyse slow-process-instances --bpmn-process-id order --state active --dur-longer 5m
  c8volt ops analyse slow-process-instances --key <process-instance-key> --with-listeners
  c8volt ops analyse slow-process-instances --bpmn-process-id order --element-id payment --dur-element-longer 30s
```

### Options

```
  -n, --batch-size int32            process instances per discovery page (server maximum 1000) (default 1000)
      --bid string                  exact process business ID to select (only c8.10+)
  -b, --bpmn-process-id string      BPMN process ID to discover process instances
      --dur-element-longer string   only include process instances with elements or transitions longer than this duration, for example 30s or 2m
      --dur-longer string           only include process instances whose whole duration is longer than this duration, for example 5m or 1h30m
      --element-id string           BPMN element ID to include in the analysis
      --element-state string        runtime element state to include in the analysis
      --end-date-after string       only include process instances with end date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --end-date-before string      only include process instances with end date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
  -h, --help                        help for slow-process-instances
  -k, --key strings                 process-instance key(s) to analyse; repeat or combine with stdin '-'
  -l, --limit int32                 maximum matching instances to analyse; omit for all matches
      --no-incidents-only           only include process instances without incidents during discovery
      --pd-key string               process definition key to discover process instances
      --start-date-after string     only include process instances with start date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
      --start-date-before string    only include process instances with start date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD
  -s, --state string                state to filter discovered process instances: all, active, suspended (only c8.10+), completed, canceled, terminated (default "all")
      --type string                 runtime element type to include in the analysis
      --with-full-timeline          show complete chronological element and transition detail
      --with-listeners              include runtime listener jobs
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

* [c8volt ops analyse]({{ "/cli/c8volt_ops_analyse" | relative_url }})	 - Discover read-only operational analyses

