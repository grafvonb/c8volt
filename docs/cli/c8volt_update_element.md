---
title: "c8volt update element"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update element

Update variables at explicit element scopes

### Synopsis

Update local variables on selected runtime element instances on Camunda 8.8 or newer.

Provide repeated or comma-separated --key values, or newline-separated keys from stdin with '-'. Supply exactly one non-null JSON object through --vars or --vars-file. Keys are deduplicated in first-input order. Explicit keys bypass the discovery tenant filter while retaining backend authorization.

The command reads every selected element and its complete local variables before approval. It freezes additions and changes at element:<key>, then writes only those values with local scope. Parent, sibling, descendant and called-process variables are not changed. An inherited same-name value does not prevent a local addition. Truncated values cannot prove equality. A null property assigns null; it does not delete a variable.

Use --dry-run to preview without prompts or writes. An empty object or an all-unchanged plan completes without confirmation, mutation or read-back. Normal execution confirms all requested values locally. --no-wait skips only read-back and reports accepted submissions, not confirmed values. Partial failures preserve acceptance and uncertainty with a nonzero exit; there is no rollback or resubmission after confirmation failure.

JSON requires --dry-run, --auto-confirm or --automation and cannot be combined with --verbose. Keys-only previews list changed destinations; executed results list accepted destinations even if confirmation failed. Quiet preserves explicit machine results and required errors.

```
c8volt update element [flags]
```

### Examples

```
  ./c8volt update element --key <element-key> --vars '{"approved":true}' --dry-run
  ./c8volt update ei --key <key-a>,<key-b> --key <key-a> --vars-file ./vars.json --auto-confirm
  printf '%s\n' "$ELEMENT_KEY_A" "$ELEMENT_KEY_B" | ./c8volt update element - --vars '{"status":null}' --auto-confirm
  ./c8volt --json update element --key <element-key> --vars '{"approved":true}' --auto-confirm --no-wait
```

### Options

```
      --dry-run            preview exact local changes without mutation
      --fail-fast          stop scheduling new updates after the first error
  -h, --help               help for element
  -k, --key strings        runtime element key(s); repeat, comma-separate or combine with stdin '-'
      --no-wait            return after accepted writes without local variable confirmation
      --no-worker-limit    use all queued jobs as workers when --workers is unset
      --vars string        JSON object with variables to set locally on each element
      --vars-file string   path to JSON object file with variables to set locally
  -w, --workers int        maximum concurrent workers (default: min(count, 2*GOMAXPROCS, 32))
```

### Options inherited from parent commands

```
      --all-tenants          clear configured tenant filtering and search all tenants visible to the authenticated user; mutually exclusive with --tenant
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
      --tenant string        tenant ID for discovery/search, selection, create, deploy, and run flows; explicit empty values can clear configured discovery filters, and explicit keys/IDs remain backend-authorized
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt update]({{ "/cli/c8volt_update" | relative_url }})	 - Update existing resources

