---
title: "c8volt ops purge all-process-definitions"
nav_exclude: true
---

## c8volt ops purge all-process-definitions

Purge all selected process definitions

### Synopsis

Delete selected process definitions and associated history.

Select versions by key or the same filters as get process-definition.
Unfinished instances block deletion. With --force, cancel affected roots, wait
for unfinished work to drain, then delete instance history and definitions.
Suspended instances count as unfinished only c8.10+.

Use --dry-run to review the affected families and --report-file to save an audit
report. Use --auto-confirm for unattended execution.

```
c8volt ops purge all-process-definitions [flags]
```

### Examples

```
  c8volt ops purge all-process-definitions --bpmn-process-id order --dry-run
  c8volt ops purge all-process-definitions --bpmn-process-id order --force --report-file purge.md
```

### Options

```
  -n, --batch-size int32         process definitions per discovery page (server maximum 1000) (default 1000)
  -b, --bpmn-process-id string   BPMN process ID to filter candidate process definitions
      --dry-run                  discover and validate process-definition cleanup without submitting deletion requests
      --fail-fast                stop scheduling validation or deletion work after the first error
      --force                    force cancellation of affected unfinished process instances before deleting process definitions
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
  -w, --workers int              maximum concurrent workers; omit for an automatic limit
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

* [c8volt ops purge]({{ "/cli/c8volt_ops_purge" | relative_url }})	 - Discover destructive operational cleanup workflows

