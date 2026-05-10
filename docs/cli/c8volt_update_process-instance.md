---
title: "c8volt update process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update process-instance

Update process-instance variables by key

### Synopsis

Update process-instance variables by key.

The command accepts repeated --key values or newline-separated keys from stdin with '-'. Provide exactly one variable payload source: --vars with a JSON object or --vars-file with a path to a JSON object file. The same variable map is applied to every unique target key.

By default c8volt loads current process-instance-scope variables, previews planned additions and changes, asks for confirmation, then waits until requested variables are visible through the same lookup path as `get pi --with-vars`. Use --dry-run to preview without mutating, or --auto-confirm for unattended mutation.

Variable updates are supported for Camunda 8.8 and 8.9. Camunda 8.7 returns an unsupported-version error before mutation.

```
c8volt update process-instance [flags]
```

### Examples

```
  ./c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}'
  ./c8volt update pi --key <process-instance-key> --vars-file ./vars.json
  ./c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}' --auto-confirm
  ./c8volt update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}'
  ./c8volt update pi --key <process-instance-key-a> --key <process-instance-key-b> --vars '{"customerTier":"gold"}'
  printf '%s\n' "$PROCESS_INSTANCE_KEY_A" "$PROCESS_INSTANCE_KEY_B" | ./c8volt update pi - --vars '{"customerTier":"gold"}'
  printf '%s\n' "$PROCESS_INSTANCE_KEY_A" | ./c8volt update pi --key "$PROCESS_INSTANCE_KEY_B" - --vars '{"customerTier":"gold"}'
  ./c8volt --json update pi --key <process-instance-key> --vars '{"customerTier":"gold"}' --auto-confirm
```

### Options

```
      --dry-run            preview variable updates without submitting mutation
      --fail-fast          stop scheduling new updates after the first error
  -h, --help               help for process-instance
      --key strings        process instance key(s) to update; repeat or combine with stdin '-'
      --no-wait            return after the update request is accepted without variable confirmation
      --no-worker-limit    disable limiting the number of workers to GOMAXPROCS when --workers > 1
      --vars string        JSON object with variables to set on each process instance
      --vars-file string   path to JSON object file with variables to set on each process instance
  -w, --workers int        maximum concurrent workers when updating multiple process instances (default: min(count, GOMAXPROCS))
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

* [c8volt update](c8volt_update)	 - Update existing resources

