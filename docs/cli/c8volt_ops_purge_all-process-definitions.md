---
title: "c8volt ops purge all-process-definitions"
nav_exclude: true
---

## c8volt ops purge all-process-definitions

Purge all selected process definitions

### Synopsis

Delete selected process definitions and their associated history on Camunda 8.9 or newer.

Select definitions with the same filters as get process-definition, or provide explicit --key values. The workflow fixes the candidate set, validates delete impact, and requires confirmation before deletion.

Active-instance impact blocks deletion unless --force is set. Forced cleanup cancels root instances, waits for active instances to drain, deletes instance history, then deletes the definitions.

--tenant limits selector discovery; an empty tenant or --all-tenants searches across accessible tenants. Explicit keys use backend authorization without tenant filtering. --batch-size controls each discovery request; --limit caps the selected scope.

Use --dry-run to inspect impact without mutation, --auto-confirm or --automation for unattended deletion, and --report-file to save an audit report.

```
c8volt ops purge all-process-definitions [flags]
```

### Examples

```
  ./c8volt ops purge all-process-definitions --dry-run
  ./c8volt --tenant tenant-a ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --latest --dry-run
  ./c8volt --tenant "" ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --latest --dry-run
  ./c8volt ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --latest --dry-run
  ./c8volt ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --latest --force
  ./c8volt --verbose ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --latest --auto-confirm
  ./c8volt ops purge all-process-definitions --key <process-definition-key> --force --report-file process-definition-purge.md
```

### Options

```
  -n, --batch-size int32         number of process definitions to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string   BPMN process ID to filter candidate process definitions
      --dry-run                  discover and validate process-definition cleanup without submitting deletion requests
      --fail-fast                stop scheduling validation or deletion work after the first error
      --force                    force cancellation of affected active process instances before deleting process definitions
  -h, --help                     help for all-process-definitions
  -k, --key string               process definition key to select for candidate discovery
      --latest                   only include the latest matching process-definition version(s)
  -l, --limit int32              maximum number of matching process definitions to freeze for purge; omit to discover all matches
      --no-wait                  return after deletion requests are accepted without deletion confirmation
      --no-worker-limit          use all queued jobs as workers when --workers is unset
      --pd-version int32         process definition version to filter candidate discovery
      --pd-version-tag string    process definition version tag to filter candidate discovery
      --report-file string       write an audit report to the given path
      --report-format string     audit report format: markdown, json (default inferred from report-file extension)
  -w, --workers int              maximum concurrent workers when validating the delete plan and deleting process definitions (default: min(targets, 2*GOMAXPROCS, 32))
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
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt ops purge]({{ "/cli/c8volt_ops_purge" | relative_url }})	 - Discover destructive operational cleanup workflows

