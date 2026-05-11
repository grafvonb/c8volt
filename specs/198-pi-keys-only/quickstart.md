# Quickstart: get incident process-instance key output

## Keyed Lookup

```sh
c8volt get incident --key 2251799813685249 --pi-keys-only
```

Expected result:

```text
2251799813711967
```

## Search Pipeline

```sh
c8volt get incident --state active --error-type job_no_retries --pi-keys-only | c8volt cancel pi --auto-confirm -
```

Expected result:

- `get incident` emits process instance keys only.
- Duplicate process instance keys may appear when multiple incidents match the same process instance.
- The downstream process-instance command handles its normal stdin validation and dedupe behavior.

## Validation Examples

These combinations should fail locally:

```sh
c8volt --json get incident --pi-keys-only
c8volt --keys-only get incident --pi-keys-only
c8volt get incident --pi-keys-only --total
c8volt get incident --pi-keys-only --error-message-limit 20
c8volt get incident --pi-keys-only --with-no-error-message
```

## Suggested Verification

```sh
GOCACHE=/tmp/c8volt-gocache go test ./cmd -run 'TestGetIncidentCommand_.*PIKeysOnly|TestListIncidentsView_.*PIKeysOnly|TestDeleteProcessInstanceCommand_.*Duplicate' -count=1
make test
```
