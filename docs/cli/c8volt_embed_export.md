---
title: "c8volt embed export"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt embed export

Export bundled BPMN fixtures to local files

### Synopsis

Export bundled BPMN fixtures to local files.

Use --all for the full set, or repeat --file with exact names or quoted globs.

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
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging
  -j, --json               output as JSON (where applicable)
      --keys-only          output keys only (where applicable)
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress output except errors
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt embed](c8volt_embed)	 - Use bundled BPMN fixtures

