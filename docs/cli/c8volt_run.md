---
title: "c8volt run"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt run

Start process instances

### Synopsis

Start process instances.

The process-instance command waits for active instances by default.

```
c8volt run [flags]
```

### Examples

```
  ./c8volt run pi -b C88_SimpleUserTask_Process
  ./c8volt run pi -b C88_SimpleUserTask_Process --vars '{"customerId":"1234"}'
  ./c8volt run pi -b C88_SimpleUserTask_Process --no-wait
```

### Options

```
  -h, --help   help for run
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
* [c8volt run process-instance](c8volt_run_process-instance)	 - Start process instances and confirm activation

