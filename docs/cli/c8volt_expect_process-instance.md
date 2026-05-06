---
title: "c8volt expect process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt expect process-instance

Wait for process instances to satisfy expectations

### Synopsis

Wait for process instances to satisfy requested state and incident expectations.

Use after `run`, `cancel`, or `delete` when a command returns before the final state or incident marker is visible.

```
c8volt expect process-instance [flags]
```

### Examples

```
  ./c8volt expect pi --key <process-instance-key> --state active
  ./c8volt expect pi --key <process-instance-key> --incident true
  ./c8volt expect pi --key <process-instance-key> --state active --incident false
  ./c8volt expect pi --key <process-instance-key> --state completed --state absent
  ./c8volt expect pi --key <process-instance-key> --state canceled
  ./c8volt get pi --key <process-instance-key> --keys-only | ./c8volt expect pi --incident true -
```

### Options

```
      --fail-fast         stop scheduling new instances after the first error
  -h, --help              help for process-instance
      --incident string   incident expectation; valid values are: [true, false]
  -k, --key strings       process instance key(s) to watch
      --no-worker-limit   disable limiting the number of workers to GOMAXPROCS when --workers > 1
  -s, --state strings     state expectation; valid values are: [active, completed, canceled, terminated, absent]
  -w, --workers int       maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))
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

* [c8volt expect](c8volt_expect)	 - Wait for process instances to satisfy expectations

