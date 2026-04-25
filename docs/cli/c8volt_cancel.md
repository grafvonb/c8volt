---
title: "c8volt cancel"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt cancel

Cancel running process instances

### Synopsis

Cancel running process instances.

Use this command family when active workflow work should be stopped. The
process-instance command validates the affected tree, prompts before destructive
changes, and waits for the observed cancellation unless you opt out.

```
c8volt cancel [flags]
```

### Examples

```
  ./c8volt cancel pi --key <process-instance-key>
  ./c8volt cancel pi --key <process-instance-key> --force
  ./c8volt cancel pi --state active --count 200 --auto-confirm
```

### Options

```
  -h, --help   help for cancel
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
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

* [c8volt](c8volt)	 - Operate Camunda 8 workflows from the command line
* [c8volt cancel process-instance](c8volt_cancel_process-instance)	 - Cancel process instances by key or filters

