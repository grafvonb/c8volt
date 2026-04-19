---
title: "c8volt embed export"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt embed export

Export embedded (virtual) resources to local files. Can be used to deploy updated versions of embedded resources using 'c8volt deploy'.

```
c8volt embed export [flags]
```

### Examples

```
  ./c8volt embed export --all --out ./fixtures
  ./c8volt embed export --file 'processdefinitions/*.bpmn' --out ./fixtures
  ./c8volt embed export --file processdefinitions/C88_SimpleUserTaskProcess.bpmn --out ./fixtures
```

### Options

```
      --all            export all embedded files
  -f, --file strings   embedded file(s) or a glob pattern to export (repeatable, quote patterns in the shell like zsh)
      --force          overwrite if destination file exists
  -h, --help           help for export
  -o, --out string     output base directory (default ".")
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

* [c8volt embed](c8volt_embed)	 - Manage embedded resources

