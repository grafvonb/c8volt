---
title: "c8volt embed deploy"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt embed deploy

Deploy embedded (virtual) resources

```
c8volt embed deploy [flags]
```

### Options

```
      --all            deploy all embedded files for the configured Camunda version
  -f, --file strings   embedded file(s) to deploy (repeatable)
  -h, --help           help for deploy
```

### Options inherited from parent commands

```
  -y, --auto-confirm        auto-confirm prompts for non-interactive use
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
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

* [c8volt embed](c8volt_embed.md)	 - Manage embedded resources

