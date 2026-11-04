---
title: "c8volt ops"
nav_exclude: true
---

## c8volt ops

Run operational playbooks

### Synopsis

Run playbooks for incident repair, retention, cleanup, runtime analysis,
and connection testing. Start with --dry-run on workflows that change data.

```
c8volt ops [flags]
```

### Examples

```
  c8volt ops repair incident --key <incident-key> --dry-run
  c8volt ops execute retention-policy --retention-days 90 --dry-run
  c8volt ops analyse slow-process-instances --bpmn-process-id order --dur-longer 5m
```

### Options

```
  -h, --help   help for ops
```

### Options inherited from parent commands

```
      --all-tenants          search all accessible tenants; cannot combine with --tenant
  -y, --auto-confirm         auto-confirm prompts for non-interactive use
      --automation           enable non-interactive mode for commands that explicitly support it
      --config string        path to config file
      --debug                enable debug logging
  -j, --json                 output as JSON (where applicable)
      --keys-only            output keys only (where applicable)
      --log-level string     log level (debug, info, warn, error) (default "info")
      --no-indicator         disable transient terminal activity indicators
      --no-version-check     skip the automatic advisory gateway version check
      --profile string       config active profile name to use (e.g. dev, prod)
  -q, --quiet                suppress output except errors
      --request-rate float   maximum shared request starts per second (fractions allowed; 0 disables pacing)
      --tenant string        tenant for discovery and creation; explicit keys use backend authorization
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt]({{ "/cli/c8volt/" | relative_url }})	 - Operate Camunda 8 workflows from the command line
* [c8volt ops analyse]({{ "/cli/c8volt_ops_analyse" | relative_url }})	 - Discover read-only operational analyses
* [c8volt ops execute]({{ "/cli/c8volt_ops_execute" | relative_url }})	 - Discover predefined operational playbooks
* [c8volt ops purge]({{ "/cli/c8volt_ops_purge" | relative_url }})	 - Discover destructive operational cleanup workflows
* [c8volt ops repair]({{ "/cli/c8volt_ops_repair" | relative_url }})	 - Discover repair and remediation workflows

