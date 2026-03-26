# Camunda Client Generation

This directory contains the scripts and patch helpers used to generate Go clients from Camunda OpenAPI specifications.

## Upstream sources

- `/v2` Camunda API: fetched from `camunda/camunda`
- legacy component APIs such as Operate, Tasklist, and Administration SM: fetched from `camunda/camunda-docs`

The local fetch scripts pin to a fetched Git ref so generation stays reproducible.

## Main scripts

- `1-fetch-camunda-product-v2-spec.sh`
  Fetches the `/v2` OpenAPI source from the Camunda product repository.
- `1-fetch-camunda-docs-api-specs.sh`
  Fetches the legacy component API specs from `camunda-docs`.
- `2-bundle-camunda-docs-v2-spec.sh`
  Bundles the legacy docs-repo `/v2` spec into a single YAML file.
- `3-generate-clients-from-fetched-specs.sh`
  Regenerates the checked-in Go clients under `internal/clients/` from already fetched sources.
- `refresh-clients.sh`
  Runs the full workflow: fetch upstream specs and regenerate the checked-in Go clients.
- `generate-go-client.sh`
  Runs `oapi-codegen` for a single spec file and output path.

These wrappers still point to the docs-backed fetch flow.

## Mutation helpers

Some generated `/v2` types still need local spec patching before `oapi-codegen` produces usable Go types.

The Python mutation helpers are stored in `api/mutations/`.

Currently used for the product-repo `/v2` flow:

- `mutations/mutate-search-query-schemas.py`
  Preserves `filter` and `sort` fields on generated search request types.
- `mutations/mutate-search-result-schemas.py`
  Preserves `items` fields on generated search result types.
- `mutations/mutate-fix-process-instance-filter-fields.py`
  Preserves process-instance-specific filter fields that would otherwise collapse into the base filter type.
- `mutations/mutate-fix-jobresult-discriminator.py`
  Restores discriminator-related `type` fields needed for correct generated models.

Other patch helpers in this directory may still be used for older or component-specific generation paths.

## Typical workflow

1. Fetch upstream specs:
   - `bash api/1-fetch-camunda-product-v2-spec.sh`
   - `bash api/1-fetch-camunda-docs-api-specs.sh`
2. Regenerate clients:
   - `bash api/3-generate-clients-from-fetched-specs.sh`
3. Validate:
   - `make test`

Full workflow:

1. Fetch and regenerate in one step:
   - `bash api/refresh-clients.sh`
2. Validate:
   - `make test`

## Commit mode

`api/refresh-clients.sh` and `api/3-generate-clients-from-fetched-specs.sh` support `--commit`.

Example:

- `bash api/refresh-clients.sh --commit`

When enabled, the script:

- stages the generated client files under `internal/clients/`
- creates a Conventional Commit
- includes the fetched `camunda` and `camunda-docs` refs in the commit body

Optional source pinning:

- `bash api/refresh-clients.sh --camunda-tag 8.8.19 --camunda-docs-tag 8.8.196`
