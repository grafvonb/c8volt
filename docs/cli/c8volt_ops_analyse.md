---
title: "c8volt ops analyse"
nav_exclude: true
---

## c8volt ops analyse

Discover read-only operational analyses

### Synopsis

Discover read-only operational analyses.

The analyse command family groups inspection workflows that combine existing runtime resources without mutating cluster state.

```
c8volt ops analyse [flags]
```

### Examples

```
  ./c8volt ops analyse --help
  ./c8volt ops analyse slow-process-instances --help
```

### Options

```
  -h, --help   help for analyse
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
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt ops](c8volt_ops)	 - Discover high-level operational workflows
* [c8volt ops analyse slow-process-instances](c8volt_ops_analyse_slow-process-instances)	 - Analyse slow process-instance timings

