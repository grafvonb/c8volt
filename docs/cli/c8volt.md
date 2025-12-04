---
title: "c8volt"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt

c8volt: Camunda 8 Operations CLI

### Synopsis

c8volt: Camunda 8 Operations CLI. The tool for Camunda 8 admins and developers to verify outcomes.

```
c8volt [flags]
```

### Options

```
  -y, --auto-confirm        auto-confirm prompts for non-interactive use
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
  -h, --help                help for c8volt
  -j, --json                output as JSON (where applicable)
      --keys-only           output as keys only (where applicable), can be used for piping to other commands, like cancel or delete
      --log-format string   log format (json, plain, text) (default "plain")
      --log-level string    log level (debug, info, warn, error) (default "info")
      --log-with-source     include source file and line number in logs
      --no-err-codes        suppress error codes in error outputs
      --profile string      config active profile name to use (e.g. dev, prod)
  -q, --quiet               suppress all output, except errors, overrides --log-level
      --tenant string       default tenant ID
```

### SEE ALSO

* [c8volt cancel](c8volt_cancel.md)	 - Cancel resources
* [c8volt config](c8volt_config.md)	 - Manage application configuration
* [c8volt delete](c8volt_delete.md)	 - Delete resources
* [c8volt deploy](c8volt_deploy.md)	 - Deploy resources
* [c8volt embed](c8volt_embed.md)	 - Manage embedded resources
* [c8volt expect](c8volt_expect.md)	 - Expect resources to be in a certain state
* [c8volt get](c8volt_get.md)	 - Get resources
* [c8volt run](c8volt_run.md)	 - Run resources
* [c8volt version](c8volt_version.md)	 - Print version information
* [c8volt walk](c8volt_walk.md)	 - Traverse (walk) the parent/child graph of resource type

