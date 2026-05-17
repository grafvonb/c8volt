---
title: "c8volt ops repair process-instance"
nav_exclude: true
---

## c8volt ops repair process-instance

Repair incidents selected by process instances

### Synopsis

Repair incidents selected by process instances.

The command accepts repeated --key values, newline-separated process-instance keys from stdin with '-', or process-instance search filters. Search mode automatically limits discovery to incident-bearing process instances; use --direct-incidents-only for stricter direct active incident matching. The workflow builds a fixed target set of repairable process instances and active incidents before mutation, applies process-instance-scope variable updates once per unique scope when requested, then reuses the incident repair steps for job updates, incident resolution, and confirmation. Use --report-file with markdown or json output for an audit record of discovery, targets, duplicate handling, skipped keys, step statuses, notices, errors, and final outcome.

```
c8volt ops repair process-instance [flags]
```

### Examples

```
  ./c8volt ops repair process-instance --key <process-instance-key>
  ./c8volt ops repair pi --key <process-instance-key> --key <another-process-instance-key>
  printf '%s\n' "$PI_KEY_A" "$PI_KEY_B" | ./c8volt ops repair process-instance -
  ./c8volt ops repair process-instance --state active --limit 5 --dry-run
  ./c8volt ops repair process-instance --direct-incidents-only --bpmn-process-id demo --limit 5 --dry-run
  ./c8volt ops repair process-instance --key <process-instance-key> --retries 0
  ./c8volt ops repair process-instance --key <process-instance-key> --job-timeout 5m
  ./c8volt ops repair process-instance --key <process-instance-key> --auto-confirm --report-file repair-process-instance.md
  ./c8volt --json ops repair process-instance --key <process-instance-key> --automation --dry-run
```

### Options

```
  -n, --batch-size int32                number of process instances to inspect per page (max limit 1000 enforced by server) (default 1000)
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
  -l, --limit int32                     maximum number of matching process instances to repair
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

* [c8volt ops repair](c8volt_ops_repair)	 - Discover repair and remediation workflows

