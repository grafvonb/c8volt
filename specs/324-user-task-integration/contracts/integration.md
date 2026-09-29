# Integration Execution Contract

## Entry points and profiles

- `make integration-cli-get`: existing reads plus `TestGetFamilyUserTaskVariables` standalone and called-process scenarios.
- `make integration-cli-get-volume`: existing get volume coverage plus filtered variable paging cases.
- `make integration-test-all`: includes both through existing dependencies; no runner change.
- Select existing default-local profiles through `C8VOLT_IT_PROFILES`. Each supported selected profile runs matching model cases. 8.7 runs explicit unsupported assertions for search and effective display.
- No new config source, authentication override, dependencies, direct setup API, model edits or public flags.

## Baseline filter table

Every command is scoped to a ready known owning PI. The target has local numeric `incident=1`; its process has `incident=99` and `customer="alice"`.

| Argument fragment | Expected target membership |
| --- | --- |
| `--var 'incident=1'` | Included |
| `--var 'incident=99'` | Excluded |
| `--var-exists incident` | Included |
| `--var 'incident.$exists=true'` | Included |
| `--var 'incident.$exists=false'` | Excluded (known-present variable) |
| `--var 'incident.$neq=99'` | Included |
| `--var 'incident.$neq=1'` | Excluded |
| `--var 'incident.$in=["1","99"]'` | Included |
| `--var 'incident.$in=["99"]'` | Excluded |
| `--var-like 'incident=1*'` | Included, basic pattern over serialized local value |
| `--var-like 'incident=9*'` | Excluded |
| `--var 'customer="alice"'` | Excluded |

Do not expand this table into every output mode or repeat the full operator matrix on the child scenario. Child coverage reuses equality/exclusion/display checks. If the backend rejects a required operator or exhibits different serialization, preserve failed evidence for that version; do not rewrite the product or silently waive the assertion in this issue.

## Display and output

Compare identities before/after `--with-vars`. Check local incident shadows the process value and process-only customer/payload are included. Check owner/scope keys, not merely variable names. Human shortening uses a deliberately long known value and existing labels; JSON retains the received value and backend truncation metadata. Keys-only and total continue their existing retrieval/output contracts; do not invent network-count proof from stdout.

Volume uses batch size one, at least three seeded matches, an unrestricted traversal, keys-only, total, and a limit of two. Compare sets independent of ordering; assert no duplicates before set conversion. Empty human output is `found: 0\n`; empty keys-only output is zero bytes; JSON is exactly one successful envelope with the existing initialized empty collection shape.

## Evidence and failure boundaries

Use exact deployed definition/PI scopes and label preexisting matches distinctly from owned seeds. Readiness must observe the expected variables before exclusions. Timeouts, wrong scope, wrong child, malformed result, command errors and assertion errors are failures. Preserve existing skip behavior for unavailable prerequisites with a precise reason, never a verified result. Record version-by-version status. No new runtime backlog/proposal generation; any discovered capability gap belongs in spec-owned documentation.

## Live coverage exclusion

`$notIn` is excluded for every Camunda version at user request after the observed C89 HTTP 500. Similar backend behavior elsewhere is an unverified assumption; no other-version `$notIn` behavior is claimed. Existing unit encoding tests are retained. No skip or automatic HTTP-500 suppression is introduced.
