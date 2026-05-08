---
title: "c8volt update"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update

Update existing resources

### Synopsis

Update existing resources.

The process-instance command updates process-instance-scope variables on
existing Camunda 8.8 and 8.9 process instances. The job command updates
job retries and timeout by key, with dry-run planning, confirmation prompts,
and optional no-wait submitted output. Camunda 8.7 configurations return an
unsupported-version error before these mutations.

```
c8volt update [flags]
```

### Examples

```
  ./c8volt update pi --key 2251799813711967 --vars '{"customerTier":"gold"}'
  ./c8volt update pi --key 2251799813711967 --vars-file ./vars.json
  ./c8volt update pi --key 2251799813711967 --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update job --key 2251799813711967 --retries 3 --dry-run
  ./c8volt update job --key 2251799813711967 --timeout 5m --auto-confirm
  ./c8volt update job --key 2251799813711967 --retries 3 --no-wait --auto-confirm
  ./c8volt update process-instance --key 2251799813711967 --vars '{"customerTier":"gold"}'
  printf '%s\n' 2251799813711967 2251799813711968 | ./c8volt update pi - --vars '{"customerTier":"gold"}'
  ./c8volt --automation --json update pi --key 2251799813711967 --vars '{"customerTier":"gold"}' --no-wait
```

### Options

```
  -h, --help   help for update
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

* [c8volt](c8volt)	 - Operate Camunda 8 workflows from the command line
* [c8volt update job](c8volt_update_job)	 - Update a job by key
* [c8volt update process-instance](c8volt_update_process-instance)	 - Update process-instance variables by key

