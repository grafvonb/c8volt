---
title: "c8volt expect"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt expect

Wait for verification targets to reach the expected state

### Synopsis

Wait for verification targets to reach the expected state.

Use this read-only command family after a state-changing operation when success depends
on a later observed state. Child commands document the wait contract, the acceptable
target states, and which output modes are safe for follow-up verification.

```
c8volt expect [flags]
```

### Examples

```
  ./c8volt expect process-instance --help
  ./c8volt expect process-instance --key 2251799813711967 --state active
  ./c8volt expect process-instance --key 2251799813711967 --state absent
```

### Options

```
  -h, --help   help for expect
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable the canonical non-interactive contract for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json               output as JSON (where applicable)
      --keys-only          output as keys only (where applicable), can be used for piping to other commands
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress all output, except errors, overrides --log-level
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt](c8volt)	 - Operate Camunda 8 with guided help and script-safe output modes
* [c8volt expect process-instance](c8volt_expect_process-instance)	 - Expect a process instance(s) to reach a certain state from list of states

