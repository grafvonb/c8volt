---
title: "c8volt update process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update process-instance

Update process-instance variables by key

### Synopsis

Update process-instance variables by key.

The command accepts repeated --key values or newline-separated keys from stdin with '-'. The --vars flag must be a JSON object and the same variable map is applied to every unique target key.

By default c8volt waits until requested process-instance-scope variables are visible through the same lookup path as `get pi --with-vars`; add --no-wait to return after the update request is accepted.

Variable updates are supported for Camunda 8.8 and 8.9. Camunda 8.7 returns an unsupported-version error before mutation.

```
c8volt update process-instance [flags]
```

### Examples

```
  ./c8volt update pi --key 2251799813711967 --vars '{"customerTier":"gold"}'
  ./c8volt update process-instance --key 2251799813711967 --vars '{"customerTier":"gold"}'
  ./c8volt update pi --key 2251799813711967 --key 2251799813711968 --vars '{"customerTier":"gold"}'
  printf '%s\n' 2251799813711967 2251799813711968 | ./c8volt update pi - --vars '{"customerTier":"gold"}'
  printf '%s\n' 2251799813711967 | ./c8volt update pi --key 2251799813711968 - --vars '{"customerTier":"gold"}'
  ./c8volt --json update pi --key 2251799813711967 --vars '{"customerTier":"gold"}' --no-wait
```

### Options

```
      --fail-fast         stop scheduling new updates after the first error
  -h, --help              help for process-instance
      --key strings       process instance key(s) to update; repeat or combine with stdin '-'
      --no-wait           return after the update request is accepted without variable confirmation
      --no-worker-limit   disable limiting the number of workers to GOMAXPROCS when --workers > 1
      --vars string       JSON object with variables to set on each process instance
  -w, --workers int       maximum concurrent workers when updating multiple process instances (default: min(count, GOMAXPROCS))
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

