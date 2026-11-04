---
title: "c8volt resolve process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt resolve process-instance

Resolve process-instance incidents by key or business ID

### Synopsis

Resolve active incidents in selected process-instance families.

Select by key or exact --bid (only c8.10+). The workflow fixes the family and
incident scope before confirmation, then waits for clearance. It does not
change variables or job retries; use ops repair for those changes.

Use --dry-run to review the plan. Families without active incidents need no change.

```
c8volt resolve process-instance [flags]
```

### Examples

```
  c8volt resolve process-instance --key <process-instance-key> --dry-run
  c8volt resolve process-instance --key <process-instance-key>
```

### Options

```
  -n, --batch-size int32   maximum process instances per discovery request (default 1000)
      --bid string         exact business ID to select in configured tenant scope (only c8.10+)
      --dry-run            preview process-instance incident resolutions without submitting mutation
      --fail-fast          stop scheduling new process-instance resolutions after the first error
  -h, --help               help for process-instance
  -k, --key strings        process instance key(s) to resolve; repeat or combine with stdin '-'
  -l, --limit int32        maximum distinct matching instances; zero selects all; tree expansion is not limited
      --no-wait            return after resolution requests are accepted without incident confirmation
      --no-worker-limit    use all queued jobs as workers when --workers is unset
  -w, --workers int        maximum concurrent workers; omit for an automatic limit
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

* [c8volt resolve]({{ "/cli/c8volt_resolve" | relative_url }})	 - Resolve operational incidents

