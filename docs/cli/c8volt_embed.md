---
title: "c8volt embed"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt embed

Manage embedded resources

```
c8volt embed [flags]
```

### Options

```
  -h, --help   help for embed
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

* [c8volt](c8volt.md)	 - c8volt: Camunda 8 Operations CLI
* [c8volt embed deploy](c8volt_embed_deploy.md)	 - Deploy embedded (virtual) resources
* [c8volt embed export](c8volt_embed_export.md)	 - Export embedded (virtual) resources to local files. Can be used to deploy updated versions of embedded resources using 'c8volt deploy'.
* [c8volt embed list](c8volt_embed_list.md)	 - List embedded (virtual) files containing process definitions

