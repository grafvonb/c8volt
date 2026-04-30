---
title: "c8volt expect"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt expect

Wait for process instances to reach a state

### Synopsis

Wait for process instances to reach a state.

Use after run, cancel, or delete when success depends on an observed
process-instance state.

```
c8volt expect [flags]
```

### Examples

```
  ./c8volt expect pi --key <process-instance-key> --state active
  ./c8volt expect pi --key <process-instance-key> --state absent
  ./c8volt get pi --key <process-instance-key> --keys-only | ./c8volt expect pi --state active -
```

### Options

```
  -h, --help   help for expect
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
* [c8volt expect process-instance](c8volt_expect_process-instance)	 - Wait for process instances to reach states

