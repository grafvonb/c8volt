---
title: "c8volt embed"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt embed

Use bundled BPMN fixtures

### Synopsis

Use bundled BPMN fixtures.

Use `embed list` to see bundled files, `embed deploy` to deploy fixtures, or
`embed export` to inspect or edit files locally.

```
c8volt embed [flags]
```

### Examples

```
  ./c8volt embed list
  ./c8volt embed deploy --all --run
  ./c8volt embed export --all --out ./fixtures
```

### Options

```
  -h, --help   help for embed
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

* [c8volt](c8volt)	 - Operate Camunda 8 workflows from the command line
* [c8volt embed deploy](c8volt_embed_deploy)	 - Deploy bundled BPMN fixtures
* [c8volt embed export](c8volt_embed_export)	 - Export bundled BPMN fixtures to local files
* [c8volt embed list](c8volt_embed_list)	 - List bundled BPMN fixture files

