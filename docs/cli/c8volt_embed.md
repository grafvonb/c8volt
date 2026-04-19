---
title: "c8volt embed"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt embed

Inspect, export, or deploy embedded BPMN resources

### Synopsis

Inspect, export, or deploy embedded BPMN resources.

Use this command family when the workflow starts from BPMN assets already embedded in
the c8volt binary. Choose `embed list` to discover packaged resources,
`embed export` to write them to disk, and `embed deploy` when you want
to deploy an embedded process definition to Camunda.

```
c8volt embed [flags]
```

### Examples

```
  ./c8volt embed list
  ./c8volt embed export --name invoice.bpmn --output-dir ./tmp
  ./c8volt embed deploy --name invoice.bpmn
```

### Options

```
  -h, --help   help for embed
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

* [c8volt](c8volt)	 - Operate Camunda 8 with guided help and script-safe output modes
* [c8volt embed deploy](c8volt_embed_deploy)	 - Deploy bundled BPMN fixtures for quick testing
* [c8volt embed export](c8volt_embed_export)	 - Export embedded (virtual) resources to local files. Can be used to deploy updated versions of embedded resources using 'c8volt deploy'.
* [c8volt embed list](c8volt_embed_list)	 - List embedded (virtual) files containing process definitions

