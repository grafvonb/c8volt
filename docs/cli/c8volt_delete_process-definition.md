---
title: "c8volt delete process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete process-definition

Delete process definition resources

### Synopsis

Delete process definition resources from Camunda 8.9 or newer.

Before mutation, c8volt checks active-instance impact, required cancellation roots, affected instance families, and batch-operation read access. With --force, it cancels root instances, deletes affected instance history, then deletes the definition and remaining associated history.

--tenant limits BPMN selector discovery. An empty tenant or --all-tenants leaves discovery unfiltered across accessible tenants. Explicit --key and stdin keys use backend authorization without tenant filtering. A --bpmn-process-id selector must match visible definitions before impact planning.

Use --dry-run to preview impact without mutation, or --auto-confirm for unattended deletion. To delete only a definition's instances, use delete process-instance --bpmn-process-id <bpmn-process-id>.

```
c8volt delete process-definition [flags]
```

### Examples

```
  ./c8volt delete process-definition --key <process-definition-key> --auto-confirm
  ./c8volt delete process-definition --key <process-definition-key> --dry-run
  ./c8volt --tenant tenant-a delete process-definition --key <process-definition-key> --dry-run
  ./c8volt --tenant tenant-a delete process-definition --bpmn-process-id <bpmn-process-id> --latest --dry-run
  ./c8volt --tenant "" delete process-definition --bpmn-process-id <bpmn-process-id> --latest --dry-run
  ./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --force
  ./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --dry-run
  ./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --auto-confirm
  ./c8volt --verbose delete process-definition --bpmn-process-id <bpmn-process-id> --latest --auto-confirm
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest --json
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest --keys-only | ./c8volt delete process-definition --auto-confirm -
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID of the process definition (all versions) to delete
      --dry-run                  preview process-definition delete impact without submitting deletion or cancellation requests
      --fail-fast                stop scheduling new instances after the first error
      --force                    force cancellation of the process instance(s), prior to deletion
  -h, --help                     help for process-definition
  -k, --key strings              process definition key(s) to delete
      --latest                   fetch the latest version(s) of the given BPMN process(s)
      --no-state-check           skip checking process-instance state before deleting
      --no-wait                  return after deletion work is accepted
      --no-worker-limit          use all queued jobs as workers when --workers is unset
      --pd-version int32         process definition version
      --pd-version-tag string    process definition version tag
  -w, --workers int              maximum concurrent workers when --count > 1 (default: min(count, 2*GOMAXPROCS, 32))
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

* [c8volt delete]({{ "/cli/c8volt_delete" | relative_url }})	 - Delete process instances or definitions

