# Documentation recordings

VHS tapes show Camunda 8.10 workflows with current v5 syntax. They are source files; checked-in GIFs are recordings of earlier builds and must be regenerated before reuse in v5 documentation.

Recording contacts a real cluster and changes data. Run it only as a separately requested recording workflow against a chosen target; it is not implementation validation.

## Setup

Install `vhs`, `ttyd`, `ffmpeg`, and Go. Provide a Camunda 8.10 target and OAuth credentials:

```bash
export C8VOLT_VHS_C810_BASE_URL="https://camunda.example.com"
export C8VOLT_VHS_OAUTH_TOKEN_URL="https://login.example.com/oauth/token"
export C8VOLT_VHS_OAUTH_CLIENT_ID="c8volt"
export C8VOLT_VHS_OAUTH_CLIENT_SECRET="..."
```

Optional API scopes are listed in [recording.env.example](config/recording.env.example). Keep credentials out of the repository.

## Record

| Recording | Target |
| --- | --- |
| Quick start | `make demo-vhs-fast-start` |
| Smoke test | `make demo-vhs-st` |
| Retention | `make demo-vhs-rp` |
| Orphan purge | `make demo-vhs-opi` |
| Incident purge | `make demo-vhs-piwi` |
| Definition purge | `make demo-vhs-apd` |
| Incident repair | `make demo-vhs-inc` |
| Instance repair | `make demo-vhs-pi` |

The renderer builds the binary, prepares `/tmp/c8volt-vhs/bin/config.yaml`, and runs the selected tape. Private config stays outside the repository. GIFs are written under `docs/assets/screencasts/`. See [recording.example.yml](config/recording.example.yml) for the generated config shape.
