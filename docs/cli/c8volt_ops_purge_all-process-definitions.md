---
title: "c8volt ops purge all-process-definitions"
nav_exclude: true
---

## c8volt ops purge all-process-definitions

Purge all selected process definitions

### Synopsis

Purge all selected process definitions.

The workflow discovers candidate process-definition versions using the same filters as `get pd`, freezes the candidate keys, validates the existing delete plan, and then either reports the plan with --dry-run or submits deletion only after confirmation. Preview with --dry-run before confirmed deletion. Use --auto-confirm or --automation for unattended deletion, combine --automation with --json for deterministic machine output, and use --report-file to write an audit report.

```
c8volt ops purge all-process-definitions [flags]
```

### Examples

```
  ./c8volt ops purge all-process-definitions --dry-run
  ./c8volt ops purge all-process-definitions --dry-run --report-file process-definition-purge.md
  ./c8volt ops purge all-pds --bpmn-process-id invoice --latest --dry-run
  ./c8volt ops purge all-process-definitions --bpmn-process-id invoice --pd-version 3 --dry-run --report-file invoice-purge.json --report-format json
  ./c8volt ops purge all-process-definitions --automation --json --dry-run
  ./c8volt ops purge all-process-definitions --key <process-definition-key> --auto-confirm --force
  ./c8volt ops purge all-process-definitions --key <process-definition-key> --auto-confirm --force --workers 4 --no-wait --report-file process-definition-purge.json --report-format json
```

### Options

```
  -b, --bpmn-process-id string   BPMN process ID to filter candidate process definitions
      --dry-run                  discover and validate process-definition cleanup without submitting deletion requests
      --fail-fast                stop scheduling validation or deletion work after the first error
      --force                    force cancellation of affected active process instances before deleting process definitions
  -h, --help                     help for all-process-definitions
  -k, --key string               process definition key to select for candidate discovery
      --latest                   only include the latest matching process-definition version(s)
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

* [c8volt ops purge](c8volt_ops_purge)	 - Discover destructive operational cleanup workflows

