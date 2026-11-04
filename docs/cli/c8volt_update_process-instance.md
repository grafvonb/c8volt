---
title: "c8volt update process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update process-instance

Update process-instance variables or assign a business ID

### Synopsis

Update root-scope variables on selected process instances.

Select by key or exact --bid (only c8.10+), then supply one JSON object through
--set-vars or --set-vars-file. The plan changes only root variables and waits
until requested values are visible. Unchanged plans need no writes.

Alternatively, --set-bid assigns the first native business ID to an active root
instance with uniqueness disabled (only c8.10+). Use explicit keys; assignment
cannot be combined with variable updates, and reassignment conflicts.
Use --dry-run to review the plan.

```
c8volt update process-instance [flags]
```

### Examples

```
  c8volt update process-instance --key <process-instance-key> --set-vars '{"customerTier":"gold"}' --dry-run
  c8volt update process-instance --key <process-instance-key> --set-vars-file vars.json
  c8volt update process-instance --bid ORDER-123 --set-vars '{"approved":true}' --dry-run
```

### Options

```
  -n, --batch-size int32       maximum process instances per discovery request (default 1000)
      --bid string             exact business ID to select in configured tenant scope (only c8.10+)
      --dry-run                preview variable updates or business-ID assignment without submitting mutation
      --fail-fast              stop scheduling new updates after the first error
  -h, --help                   help for process-instance
  -k, --key strings            process instance key(s) to update; repeat or combine with stdin '-'
  -l, --limit int32            maximum distinct matching instances; zero selects all; tree expansion is not limited
      --no-wait                return after the update request is accepted without confirmation
      --no-worker-limit        use all queued jobs as workers when --workers is unset
      --set-bid string         assign an exact native business ID to explicit process keys (only c8.10+)
      --set-vars string        JSON object with variables to set on each process instance
      --set-vars-file string   path to JSON object file with variables to set on each process instance
  -w, --workers int            maximum concurrent workers; omit for an automatic limit
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

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

