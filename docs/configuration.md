---
title: "Configuration"
permalink: /configuration/
nav_order: 2
has_toc: true
---

# Configuration

Set the gateway URL, authentication, and Camunda version before running operational commands. These examples target Camunda 8.10. Match `app.camunda_version` to your cluster; 8.9 and 8.8 remain supported.

## Local cluster

Save `config.yaml` in your working directory, or pass `--config /path/to/config.yaml`:

```yaml
app:
  camunda_version: "8.10"
apis:
  camunda_api:
    base_url: "http://localhost:8080"
auth:
  mode: "none"
```

Without `--config`, c8volt searches the working directory, then the XDG and user configuration directories, then `/etc/c8volt`. The built-in version default remains 8.9; set 8.10 explicitly as shown. Use `c8volt config template` for the full template, or the release archive's `config.example.yaml`.

## Protected cluster

For OAuth, replace the authentication block and set the gateway URL under `apis.camunda_api.base_url`:

```yaml
auth:
  mode: "oauth2"
  oauth2:
    token_url: "https://login.example.com/oauth"
    client_id: "c8volt"
    client_secret: ""
```

Set `token_url` to the provider path before `/token`; c8volt appends `/token`. Supply the secret through `C8VOLT_AUTH_OAUTH2_CLIENT_SECRET`. Cookie authentication is also supported; see the [config reference](/cli/c8volt_config).

## Profiles and precedence

Settings resolve as **flag > environment > profile > base config > default**. Profiles override the base settings:

```yaml
profiles:
  production:
    app:
      camunda_version: "8.10"
      tenant: "tenant-a"
    apis:
      camunda_api:
        base_url: "https://camunda.example.com"
```

```bash
c8volt --profile production config validate
c8volt --profile production config test-connection
c8volt --profile production config show
```

`config show` redacts secrets. Validation is local; `test-connection` contacts Camunda. Before operational requests, an automatic gateway-version advisory warns on a release-line mismatch and keeps the configured adapter.

## Tenant scope

`app.tenant` sets the default tenant. `--tenant tenant-a` overrides it; `--all-tenants` clears the discovery filter within the authenticated identity's access. The two flags cannot be combined. Explicit keys use backend authorization without discovery filtering. Creation, deployment, and smoke tests require one destination tenant.

## Timeouts and load

`--timeout` bounds an HTTP request; it does not set a job timeout. Use `update job --set-timeout` for that. Workflow waiting uses the backoff settings in the full config template.

Use `--request-rate 10` to pace shared request starts per second. The default, zero, disables pacing. Pacing does not change worker concurrency or extend timeouts.

## Output and diagnostics

Query results, JSON, and keys use stdout. Human mutation summaries and diagnostics use the configured logger; interactive prompts use stderr. `--quiet` suppresses informational output while retaining errors and requested machine results.

Use `--verbose` for per-resource outcomes. `--debug` or `log.level: debug` enables redacted HTTP diagnostics independently of verbose. Configure logging with `log.level` and `log.format`; formats are `plain-time`, `plain`, `text`, and `json`.
