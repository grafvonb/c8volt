---
title: "c8volt walk process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt walk process-instance

Inspect the parent/child tree of process instances

### Synopsis

Inspect the parent/child tree of process instances.

Use this read-only command after state-changing flows when you need to verify ancestor, child, or full-family relationships before cancelling, deleting, or confirming downstream effects. Choose --parent for ancestry, --children for descendants, and --family when you need the combined view or ASCII tree rendering.

Human-readable list and tree output remain the default. Use --json when another tool needs the shared result envelope around the returned traversal payload. `--automation` remains unsupported because traversal output semantics are still human-first.

```
c8volt walk process-instance [flags]
```

### Examples

```
  ./c8volt walk pi --key 2251799813711967 --family
  ./c8volt walk pi --key 2251799813711967 --family --tree
  ./c8volt walk pi --key 2251799813711977 --parent
  ./c8volt cancel pi --key 2251799813711967 --no-wait --auto-confirm
  ./c8volt walk pi --key 2251799813711967 --family --tree
  ./c8volt --json walk pi --key 2251799813711967 --children
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
      --automation          enable the canonical non-interactive contract for commands that explicitly support it
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
      --tenant string       tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose             adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt walk](c8volt_walk)	 - Inspect parent and child relationships for verification follow-up

