---
title: "c8volt resolve process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt resolve process-instance

Resolve process-instance incidents by key or business ID

### Synopsis

Resolve active incidents in process-instance families.

Provide repeated --key values or newline-separated keys from stdin with '-'. c8volt expands each target to its family, validates the affected scope, and asks for confirmation. Explicit keys use backend authorization without tenant filtering.

Only incidents discovered at command start are resolved. Instances with no active incidents require no mutation. By default c8volt waits until those incidents are no longer active.

Use --dry-run to inspect the family and incident plan without mutation.

On Camunda 8.10, --bid selects one exact opaque business ID in configured tenant scope. It cannot be combined with --key or stdin. All distinct matches are selected before family expansion; --limit bounds matching seeds, never relatives. --batch-size controls discovery requests.

BID resolution freezes the complete family and active incident targets before confirmation. New matches or incidents after approval are not submitted. Dry-run performs discovery reads only; empty selections and families with no active incidents never prompt or submit mutation, including with --no-wait. JSON and verbose output retain the original criterion and complete reviewed work. No job or variable updates are performed.

```
c8volt resolve process-instance [flags]
```

### Examples

```
  ./c8volt resolve process-instance --bid ORDER-123 --dry-run
  ./c8volt resolve process-instance --key <process-instance-key> --dry-run
  ./c8volt resolve process-instance --key <process-instance-key>
  ./c8volt --tenant tenant-a resolve process-instance --key <process-instance-key> --dry-run
  ./c8volt resolve process-instance --key <process-instance-key> --key <another-process-instance-key>
  printf '%s\n' "$PROCESS_INSTANCE_KEY_A" "$PROCESS_INSTANCE_KEY_B" | ./c8volt resolve process-instance -
```

### Options

```
  -n, --batch-size int32   maximum process instances per discovery request (default 1000)
      --bid string         exact business ID to select in configured tenant scope (Camunda 8.10)
      --dry-run            preview process-instance incident resolutions without submitting mutation
      --fail-fast          stop scheduling new process-instance resolutions after the first error
  -h, --help               help for process-instance
  -k, --key strings        process instance key(s) to resolve; repeat or combine with stdin '-'
  -l, --limit int32        maximum distinct matching instances; zero selects all; tree expansion is not limited
      --no-wait            return after resolution requests are accepted without incident confirmation
      --no-worker-limit    use all queued jobs as workers when --workers is unset
  -w, --workers int        maximum concurrent workers when resolving multiple process instances (default: min(count, 2*GOMAXPROCS, 32))
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

* [c8volt resolve]({{ "/cli/c8volt_resolve" | relative_url }})	 - Resolve operational incidents

