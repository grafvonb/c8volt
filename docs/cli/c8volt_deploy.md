---
title: "c8volt deploy"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt deploy

Deploy resources

### Synopsis

Deploy resources such as BPMN process definitions.
It is a root command and requires a subcommand to specify the resource type to deploy.

```
c8volt deploy [flags]
```

### Options

```
      --backoff-max-retries int    max retry attempts (0 = unlimited)
      --backoff-timeout duration   overall timeout for the retry loop (default 2m0s)
  -h, --help                       help for deploy
```

### Options inherited from parent commands

```
  -y, --auto-confirm        auto-confirm prompts for non-interactive use
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
  -j, --json                output as JSON (where applicable)
      --keys-only           output as keys only (where applicable), can be used for piping to other commands
      --log-format string   log format (json, plain, text) (default "plain")
      --log-level string    log level (debug, info, warn, error) (default "info")
      --log-with-source     include source file and line number in logs
      --no-err-codes        suppress error codes in error outputs
      --profile string      config active profile name to use (e.g. dev, prod)
  -q, --quiet               suppress all output, except errors, overrides --log-level
      --tenant string       default tenant ID
  -v, --verbose             adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt](c8volt.md)	 - c8volt: Camunda 8 Operations CLI
* [c8volt deploy process-definition](c8volt_deploy_process-definition.md)	 - Deploy a process definition

