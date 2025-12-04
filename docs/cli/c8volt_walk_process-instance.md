---
title: "c8volt walk process-instance"
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt walk process-instance

Traverse (walk) the parent/child graph of process instances

```
c8volt walk process-instance [flags]
```

### Options

```
      --children      shorthand for --mode=children
      --family        shorthand for --mode=family
  -h, --help          help for process-instance
  -k, --key string    start walking from this process instance key
      --mode string   walk mode: parent, children, family (default "children")
      --parent        shorthand for --mode=parent
      --tree          render family mode as an ASCII tree (only valid with --family)
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

* [c8volt walk](c8volt_walk.md)	 - Traverse (walk) the parent/child graph of resource type

