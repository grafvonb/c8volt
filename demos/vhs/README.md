# Documentation recordings

VHS tapes show Camunda 8.10 workflows with current v5 syntax. They are source files; checked-in GIFs are recordings of earlier builds and must be regenerated before reuse in v5 documentation.

Recording contacts a real cluster and changes data. Run it only as a separately requested recording workflow against a chosen target; it is not implementation validation.

## Setup

Install `vhs`, `ttyd`, `ffmpeg`, Go, and Python 3.8+. Recordings reuse your normal c8volt configuration, including its active profile, authentication, tenant, and environment overrides. All supported auth modes work: none, cookie, and OAuth2.

To select a different config or profile:

```bash
export C8VOLT_CONFIG="$HOME/.config/c8volt/config.yaml"
export C8VOLT_ACTIVE_PROFILE="c810"
```

Without these overrides, normal config discovery and the configured active profile apply. The effective config must use `app.camunda_version: "8.10"`. There are no separate VHS credentials; see [recording.env.example](config/recording.env.example).

Before starting VHS, the renderer validates the config and runs `config test-connection --json`. It requires a reachable Camunda 8.10 gateway, matching configuration, 8.10 brokers, and healthy partitions. Invalid config, authentication failures, version mismatches, and unhealthy or incomplete topology stop the recording before the tape runs. This check confirms topology access; individual demo commands still require their own permissions.

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

Direct selection is also available:

```bash
./demos/vhs/scripts/render.sh fast-start --config "$HOME/.config/c8volt/config.yaml" --profile c810
```

The renderer builds c8volt and copies the selected config into a private temporary directory. Preflight and recording use that same copy and profile. Authentication stays with c8volt; the renderer sets human logging and interactive defaults for the tapes. The temporary directory is removed on exit, and the original config is unchanged. GIFs are written under `docs/assets/screencasts/`.

`make demo-vhs-check` checks local tools only. Isolated preflight tests build c8volt and use test-owned HTTP servers without starting VHS:

```bash
python3 -B -m unittest discover -s demos/vhs/scripts -p 'test_*.py'
```
