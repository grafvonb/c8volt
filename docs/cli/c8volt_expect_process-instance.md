---
title: "c8volt expect process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt expect process-instance

Wait for process instances to reach states

### Synopsis

Wait for process instances to reach one of the requested states.

Use this command after `run`, `cancel`, or `delete` when the previous command returned before the final state was visible, or when you need an explicit post-action assertion.

For cancellation waits, `canceled` is the user-facing intent state. On Camunda `8.8` and `8.9`, the backend may report the same outcome as `terminated`; c8volt treats both as a match.

```
c8volt expect process-instance [flags]
```

### Examples

```
  ./c8volt expect pi --key <process-instance-key> --state active
  ./c8volt expect pi --key <process-instance-key> --state completed --state absent
  ./c8volt expect pi --key <process-instance-key> --state canceled
  ./c8volt get pi --key <process-instance-key> --keys-only | ./c8volt expect pi --state active -
```

### Options

```
      --fail-fast         stop scheduling new instances after the first error
  -h, --help              help for process-instance
  -k, --key strings       process instance key(s) to expect a state for
      --no-worker-limit   disable limiting the number of workers to GOMAXPROCS when --workers > 1
  -s, --state strings     state of a process instance; valid values are: [active, completed, canceled, terminated, absent]. On Camunda 8.8/8.9, canceled waits also match terminated
  -w, --workers int       maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))
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

* [c8volt expect](c8volt_expect)	 - Wait for process instances to reach a state

