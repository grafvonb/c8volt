---
title: "c8volt expect process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt expect process-instance

Expect a process instance(s) to reach a certain state from list of states

### Synopsis

Wait for process instance(s) to reach one of the requested states.

Use this read-only command after `run`, `cancel`, or `delete` when the operation returned before the final state was visible, or when you need an explicit post-action assertion. The command waits until each keyed process instance reaches one of the requested states or fails with a shared error model. For cancellation waits, `canceled` is the user-facing intent state; on Camunda `8.8` and `8.9`, that same outcome may be surfaced by the backend as `terminated`, and `c8volt` treats them as equivalent.

Default output stays human-oriented. Use --json when another tool needs the final wait report. `--automation` remains unsupported because the broader waiting contract is not yet defined there.

```
c8volt expect process-instance [flags]
```

### Examples

```
  ./c8volt expect pi --key 2251799813685255 --state active
  ./c8volt expect pi --key 2251799813685255 --state completed --state absent
  ./c8volt expect pi --key 2251799813711967 --state canceled
  ./c8volt run pi --bpmn-process-id order-process --no-wait --json
  ./c8volt expect pi --key 2251799813711967 --state active
  ./c8volt get pi --bpmn-process-id order-process --keys-only | ./c8volt expect pi - --state terminated
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
  -y, --auto-confirm               auto-confirm prompts for non-interactive use
      --automation                 enable the canonical non-interactive contract for commands that explicitly support it
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
      --config string              path to config file
      --debug                      enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                       output as JSON (where applicable)
      --keys-only                  output as keys only (where applicable), can be used for piping to other commands
      --log-format string          log format (json, plain, text) (default "plain")
      --log-level string           log level (debug, info, warn, error) (default "info")
      --log-with-source            include source file and line number in logs
      --no-err-codes               suppress error codes in error outputs
      --no-indicator               disable transient terminal activity indicators
      --profile string             config active profile name to use (e.g. dev, prod)
  -q, --quiet                      suppress all output, except errors, overrides --log-level
      --tenant string              tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose                    adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt expect](c8volt_expect)	 - Wait for verification targets to reach the expected state

