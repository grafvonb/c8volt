---
title: "c8volt update process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update process-instance

Update process-instance variables by key

### Synopsis

Update process-instance-scope variables on Camunda 8.8 or newer.

Provide repeated --key values or newline-separated keys from stdin with '-'. Supply exactly one payload source: --vars with a JSON object or --vars-file with its file path. The same variable map is applied to every unique key. Explicit keys use backend authorization without tenant filtering.

c8volt loads current variables, plans additions and changes, asks for confirmation, and waits until the requested variables are visible through the same lookup as get process-instance --with-vars.

Use --dry-run to inspect changes without mutation, or --auto-confirm for unattended updates.

```
c8volt update process-instance [flags]
```

### Examples

```
  ./c8volt update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update process-instance --key <process-instance-key> --vars-file ./vars.json --dry-run
  ./c8volt --tenant tenant-a update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update process-instance --key <process-instance-key-a> --key <process-instance-key-b> --vars '{"customerTier":"gold"}' --dry-run
  printf '%s\n' "$PROCESS_INSTANCE_KEY_A" "$PROCESS_INSTANCE_KEY_B" | ./c8volt update process-instance - --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt --json update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
```

### Options

```
      --dry-run            preview variable updates without submitting mutation
      --fail-fast          stop scheduling new updates after the first error
  -h, --help               help for process-instance
  -k, --key strings        process instance key(s) to update; repeat or combine with stdin '-'
      --no-wait            return after the update request is accepted without variable confirmation
      --no-worker-limit    use all queued jobs as workers when --workers is unset
      --vars string        JSON object with variables to set on each process instance
      --vars-file string   path to JSON object file with variables to set on each process instance
  -w, --workers int        maximum concurrent workers when updating multiple process instances (default: min(count, 2*GOMAXPROCS, 32))
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

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

