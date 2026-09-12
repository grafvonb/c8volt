---
title: "c8volt ops purge"
nav_exclude: true
---

## c8volt ops purge

Discover destructive operational cleanup workflows

### Synopsis

Remove operational resources through validated cleanup workflows.

Choose a subcommand to purge process definitions, orphan instances, or instances selected by incidents.

```
c8volt ops purge [flags]
```

### Examples

```
  ./c8volt ops purge --help
  ./c8volt ops purge orphan-process-instances --dry-run
  ./c8volt ops purge orphan-process-instances --state completed --limit 25 --auto-confirm --report-file orphan-purge.md
```

### Options

```
  -h, --help   help for purge
```

### Options inherited from parent commands

```
      --all-tenants        clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
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
      --tenant string      tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output and API request diagnostics on stderr
```

### SEE ALSO

* [c8volt ops]({{ "/cli/c8volt_ops" | relative_url }})	 - Run operational playbooks
* [c8volt ops purge all-process-definitions]({{ "/cli/c8volt_ops_purge_all-process-definitions" | relative_url }})	 - Purge all selected process definitions
* [c8volt ops purge orphan-process-instances]({{ "/cli/c8volt_ops_purge_orphan-process-instances" | relative_url }})	 - Purge orphan child process instances
* [c8volt ops purge process-instances-with-incidents]({{ "/cli/c8volt_ops_purge_process-instances-with-incidents" | relative_url }})	 - Purge process instances selected by incidents

