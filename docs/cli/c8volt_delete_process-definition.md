---
title: "c8volt delete process-definition"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt delete process-definition

Delete process definition resources

### Synopsis

Delete process definition resources from Camunda.

By default c8volt first checks delete impact without changing anything: active process instances, required cancellation roots and process-instance tree scope when --force is used, and batch-operation read access before prompting. With --force, it cancels the root process instances, deletes the affected process-instance history, then asks Camunda to delete the process definition and remaining associated history.

Use --auto-confirm for unattended destructive runs.

```
c8volt delete process-definition [flags]
```

### Examples

```
  ./c8volt delete pd --key <process-definition-key> --auto-confirm
  ./c8volt delete pd --bpmn-process-id <bpmn-process-id> --latest --force
  ./c8volt delete pd --bpmn-process-id <bpmn-process-id> --latest --auto-confirm
  ./c8volt get pd --bpmn-process-id <bpmn-process-id> --latest --json
  ./c8volt get pd --bpmn-process-id <bpmn-process-id> --latest --keys-only | ./c8volt delete pd --auto-confirm -
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID of the process definition (all versions) to delete
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

* [c8volt delete](c8volt_delete)	 - Delete process instances or definitions

