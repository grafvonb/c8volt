---
title: "c8volt ops repair incident"
nav_exclude: true
---

## c8volt ops repair incident

Repair incidents by key or filter

### Synopsis

Repair incidents by key or search filters.

The workflow fixes the target set, applies requested variables, restores related
job retries, resolves incidents, and waits for clearance. Related-job timeout
updates are optional. --set-retries 0 skips retry restoration.

Use --dry-run to review the repair plan and --report-file to save an audit report.
Use --auto-confirm for unattended execution.

```
c8volt ops repair incident [flags]
```

### Examples

```
  c8volt ops repair incident --key <incident-key> --dry-run
  c8volt ops repair incident --key <incident-key> --set-retries 3 --set-vars-file vars.json --report-file repair.md
```

### Options

```
  -n, --batch-size int32               incidents per discovery page (server maximum 1000) (default 1000)
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
  -k, --key strings                    incident key(s) to repair; repeat or combine with stdin '-'
  -l, --limit int32                    maximum number of matching incidents to freeze for repair; omit to discover all matches
      --no-wait                        return after repair mutations are accepted without incident or retry confirmation
      --no-worker-limit                use all queued jobs as workers when --workers is unset
      --pd-key string                  process definition key to filter incidents
      --pi-key string                  process instance key to filter incidents
      --report-file string             plan an audit report at the given path
      --report-format string           audit report format: markdown, json (default inferred from report-file extension)
      --root-key string                root process instance key to filter incidents (only c8.9+)
      --set-job-timeout string         timeout duration to submit for related jobs, for example 60s, 5m, or 1h
      --set-retries int32              retry count to set on related jobs; 0 skips retry restoration (default 1)
      --set-vars string                JSON object with variables to set once per process-instance scope before resolving dependent incidents
      --set-vars-file string           path to JSON object file with variables to set once per process-instance scope
  -s, --state string                   incident state scope for search: active, pending, resolved, migrated, unknown, all (default "active")
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

* [c8volt ops repair]({{ "/cli/c8volt_ops_repair" | relative_url }})	 - Discover repair and remediation workflows

