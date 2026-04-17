---
title: "c8volt deploy process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt deploy process-definition

Deploy BPMN process definition files

```
c8volt deploy process-definition [flags]
```

### Examples

```
  ./c8volt deploy pd --file ./order-process.bpmn
  ./c8volt deploy pd --file ./order-process.bpmn --run
  ./c8volt deploy pd --file - < ./order-process.bpmn
```

### Options

```
  -f, --file strings   paths to BPMN/YAML file(s) or '-' for stdin
  -h, --help           help for process-definition
      --no-wait        skip waiting for the deployment to be fully processed
      --run            run single process instance without vars after deploying process definition(s)
```

### Options inherited from parent commands

```
  -y, --auto-confirm               auto-confirm prompts for non-interactive use
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

* [c8volt deploy](c8volt_deploy)	 - Deploy resources

