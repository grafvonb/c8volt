---
title: "c8volt expect process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt expect process-instance

Expect a process instance(s) to reach a certain state from list of states

```
c8volt expect process-instance [flags]
```

### Examples

```
  ./c8volt expect pi --key 2251799813685255 --state active
  ./c8volt expect pi --key 2251799813685255 --state completed --state absent
  ./c8volt get pi --bpmn-process-id order-process --keys-only | ./c8volt expect pi - --state terminated
```

### Options

```
      --fail-fast         stop scheduling new instances after the first error
  -h, --help              help for process-instance
  -k, --key strings       process instance key(s) to expect a state for
      --no-worker-limit   disable limiting the number of workers to GOMAXPROCS when --workers > 1
  -s, --state strings     state of a process instance; valid values aer: [active, completed, canceled, terminated or absent]
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
      --profile string             config active profile name to use (e.g. dev, prod)
  -q, --quiet                      suppress all output, except errors, overrides --log-level
      --tenant string              tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose                    adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt expect](c8volt_expect)	 - Expect resources to be in a certain state

