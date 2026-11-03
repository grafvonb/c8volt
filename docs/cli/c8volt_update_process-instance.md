---
title: "c8volt update process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update process-instance

Update process-instance variables or assign a business ID

### Synopsis

Update process-instance-scope variables on Camunda 8.8 or newer, or assign a native business ID on Camunda 8.10.

Use --set-bid with explicit --key values or newline stdin to assign one exact value. It cannot be combined with --bid, --vars, --vars-file, --limit or --batch-size. All keys and the value are validated before planning; distinct targets and the value are frozen before one approval. Deliberate empty stdin succeeds without approval or mutation. Keys retain explicit backend authorization, without tenant filtering or parent/family expansion.

Assignment requires an active root instance with no business ID and uniqueness disabled. The backend decides eligibility; even identical reassignment conflicts. Empty, whitespace-only, invalid UTF-8 and values exceeding 256 UTF-16 units are rejected. Spaces, commas and case in accepted values are preserved exactly. A variable named businessId remains an ordinary variable, independent of native identity. Job, task and listener snapshots are not refreshed by assignment.

Assignment dry-run previews requested intent and prior observations without reserving eligibility; a later execution can race. Normal execution confirms accepted assignments by reading the exact instance. --no-wait skips confirmation and reports acceptance only. Ambiguous responses remain unknown and are never automatically replayed. Partial results preserve rejected, unknown and not-submitted outcomes with a nonzero exit status. Keys-only prints planned keys in dry-run and known accepted keys in execution; empty scope prints nothing.

Select by exact business ID with --bid on Camunda 8.10, as an alternative to --key or stdin. BID discovery respects configured tenant scope and completes before approval; --limit bounds distinct matching seeds and --batch-size controls request size. Duplicate matches count once. Empty completed discovery succeeds without confirmation or mutation, including --no-wait. Errors during discovery prevent mutation. JSON retains the original criterion and reviewed scope; execution never searches the BID again after approval. A separate dry-run and later execution perform separate discoveries.

BID updates freeze each selected instance's existing root-variable plan and payload without family expansion. Per-instance failures retain partial machine evidence and one compact configured ERROR; use --verbose for outcome details.

Variable plans retain each selected instance's observed business ID when available. Detailed plan rows quote it safely, and JSON preserves it exactly. Key/status-only outcomes do not add a lookup.

Provide repeated --key values or newline-separated keys from stdin with '-'. For variable updates, supply exactly one payload source: --vars with a JSON object or --vars-file with its file path. The same variable map is applied to every unique key. Explicit keys use backend authorization without tenant filtering.

c8volt reads every selected process instance's complete root-variable scope before planning additions and changes. Nested-scope variables are excluded, and truncated values remain planned changes because they cannot prove equality. After confirmation, the command writes only the prepared additions and changes locally to their recorded roots, then waits until all requested variables are visible through the same lookup as get process-instance --with-vars. Detailed previews label every planned write with its selected process-instance root as pi:<key>.

Use --dry-run to inspect changes without mutation, or --auto-confirm for unattended updates. A completed plan with no additions or changes finishes without confirmation, mutation, or verification. Use --no-wait to return after accepted root writes without confirmation. Partial failures retain accepted, failed, and skipped outcomes while returning a nonzero exit status.

```
c8volt update process-instance [flags]
```

### Examples

```
  ./c8volt update pi --key <process-instance-key> --set-bid 'ORDER-123' --dry-run
  ./c8volt --auto-confirm update pi --key <process-instance-key> --set-bid 'ORDER-123' --no-wait
  ./c8volt --tenant tenant-a update pi --bid 'ORDER-123' --vars '{"customerTier":"gold"}' --dry-run --json
  ./c8volt update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update process-instance --key <process-instance-key> --vars-file ./vars.json --dry-run
  ./c8volt --tenant tenant-a update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update process-instance --key <process-instance-key-a> --key <process-instance-key-b> --vars '{"customerTier":"gold"}' --dry-run
  printf '%s\n' "$PROCESS_INSTANCE_KEY_A" "$PROCESS_INSTANCE_KEY_B" | ./c8volt update process-instance - --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt --json update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
```

### Options

```
  -n, --batch-size int32   maximum process instances per discovery request (default 1000)
      --bid string         exact business ID to select in configured tenant scope (Camunda 8.10)
      --dry-run            preview variable updates or business-ID assignment without submitting mutation
      --fail-fast          stop scheduling new updates after the first error
  -h, --help               help for process-instance
  -k, --key strings        process instance key(s) to update; repeat or combine with stdin '-'
  -l, --limit int32        maximum distinct matching instances; zero selects all; tree expansion is not limited
      --no-wait            return after the update request is accepted without confirmation
      --no-worker-limit    use all queued jobs as workers when --workers is unset
      --set-bid string     assign an exact native business ID to explicit process keys (Camunda 8.10)
      --vars string        JSON object with variables to set on each process instance
      --vars-file string   path to JSON object file with variables to set on each process instance
  -w, --workers int        maximum concurrent workers when updating multiple process instances (default: min(count, 2*GOMAXPROCS, 32))
```

### Options inherited from parent commands

```
      --all-tenants          clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
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
      --tenant string        tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

