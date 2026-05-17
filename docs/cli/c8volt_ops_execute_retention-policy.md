---
title: "c8volt ops execute retention-policy"
nav_exclude: true
---

## c8volt ops execute retention-policy

Execute process-instance retention cleanup

### Synopsis

Execute process-instance retention cleanup.

The workflow discovers process instances older than the required retention age, freezes that candidate set, validates the delete plan, and then either reports the plan with --dry-run or submits deletion after confirmation. Use compatible process-instance filters to narrow discovery, --auto-confirm or --automation for unattended deletion, and --report-file to write an audit report.

```
c8volt ops execute retention-policy [flags]
```

### Examples

```
  ./c8volt ops execute retention-policy --retention-days 90 --dry-run
  ./c8volt ops execute retention-policy --retention-days 90 --state completed --bpmn-process-id order-process --dry-run
  ./c8volt ops execute retention-policy --retention-days 90 --automation --json --dry-run
  ./c8volt ops execute retention-policy --retention-days 90 --state completed --bpmn-process-id order-process --limit 25 --auto-confirm
  ./c8volt ops execute retention-policy --retention-days 90 --state completed --bpmn-process-id order-process --limit 25 --auto-confirm --force --workers 4
  ./c8volt ops execute retention-policy --retention-days 90 --dry-run --report-file retention-report.md
  ./c8volt ops execute retention-policy --retention-days 90 --state completed --bpmn-process-id order-process --limit 25 --auto-confirm --report-file retention-report.json --report-format json
```

### Options

```
  -n, --batch-size int32         number of process instances to inspect per page (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string   BPMN process ID to filter process instances
      --children-only            discover only child process instances
      --dry-run                  discover and validate retention cleanup without submitting deletion requests
      --fail-fast                stop scheduling validation or deletion work after the first error
      --force                    force cancellation of the process instance(s), prior to deletion
  -h, --help                     help for retention-policy
      --incidents-only           discover only process instances that have incidents
  -k, --key strings              unsupported explicit process-instance key selector
  -l, --limit int32              maximum number of matching process instances to inspect across all pages
      --no-incidents-only        discover only process instances that have no incidents
      --no-state-check           skip checking process-instance state before deleting
      --no-wait                  return after deletion requests are accepted without deletion confirmation
      --no-worker-limit          use all queued jobs as workers when --workers is unset
      --parent-key string        parent process instance key to narrow retention discovery
      --pd-key string            process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
      --report-file string       write an audit report to the given path
      --report-format string     audit report format: markdown, json (default inferred from report-file extension)
      --retention-days int       required non-negative age in days for process-instance retention eligibility
      --roots-only               discover only root process instances
  -s, --state string             state to filter process instances: all, active, completed, canceled, terminated (default "all")
  -w, --workers int              maximum concurrent workers when validating the delete plan and deleting roots (default: min(targets, 2*GOMAXPROCS, 32))
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

* [c8volt ops execute](c8volt_ops_execute)	 - Discover predefined operational playbooks

