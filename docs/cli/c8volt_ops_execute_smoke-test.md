---
title: "c8volt ops execute smoke-test"
nav_exclude: true
---

## c8volt ops execute smoke-test

Execute a cluster smoke test workflow

### Synopsis

Execute a cluster smoke test workflow.

Tenant contract: smoke-test setup is a creation operation. A named tenant is reported as "creation target: <tenant>" before deployment and start; empty tenant configuration targets and reports "creation target: default tenant". This command does not accept --all-tenants because it creates resources in one concrete tenant. The audit report carries the same context for created resources and cleanup evidence.

The workflow validates the configured profile, selects the embedded multiple-subprocess fixture for the configured Camunda version, deploys it, creates process instances, walks their families, and cleans up resources it can safely attribute to the run unless --no-cleanup is set. Default human output keeps deploy, start, walk, and cleanup progress on one workflow activity and writes compact stderr milestones at most once per 10-second interval, plus immediate failure warnings. Verbose and debug output replace aggregate milestones with one per-stage or per-item completion line. JSON and automation output remain free of human progress text; quiet mode suppresses successful progress and retains failure warnings. Cleanup always removes created process instances. Process-definition cleanup runs only when no unrelated instances still use the deployed fixture definition; dirty clusters skip that final definition cleanup and report retained resources instead of failing the smoke proof. Use --dry-run to validate the requested plan without submitting mutation requests.

```
c8volt ops execute smoke-test [flags]
```

### Examples

```
  ./c8volt ops execute smoke-test --dry-run
  ./c8volt --tenant tenant-a ops execute smoke-test --dry-run
  ./c8volt ops execute smoke-test --report-file smoke-test.md
  ./c8volt --verbose ops execute smoke-test --count 5 --auto-confirm
  ./c8volt ops execute smoke-test --count 5 --report-file smoke-test.md
```

### Options

```
  -n, --count int              number of process instances to create from the deployed smoke-test definition (default 1)
      --dry-run                validate the smoke-test plan without submitting mutation requests
      --fail-fast              stop scheduling smoke-test work after the first error
  -h, --help                   help for smoke-test
      --no-cleanup             retain created process instances and the deployed process definition
      --no-wait                return after cleanup requests are accepted without deletion confirmation
      --no-worker-limit        use all queued smoke-test jobs as workers when --workers is unset
      --report-file string     write an audit report to the given path
      --report-format string   audit report format: markdown, json (default inferred from report-file extension)
  -w, --workers int            maximum concurrent workers when creating, walking, or cleaning smoke-test resources (default: min(count, 2*GOMAXPROCS, 32))
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

* [c8volt ops execute]({{ "/cli/c8volt_ops_execute" | relative_url }})	 - Discover predefined operational playbooks

