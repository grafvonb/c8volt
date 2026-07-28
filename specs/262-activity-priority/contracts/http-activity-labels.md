# Contract: HTTP Fallback Activity Labels

## Purpose

Known Camunda request paths used by c8volt should produce resource-aware fallback activity when no higher-level activity is visible.

## Label Rules

- Labels MUST be short, lower-case action phrases.
- Labels MUST describe the resource action rather than the transport mechanism.
- Labels MUST NOT include host names, URLs, query strings, credentials, request bodies, or tenant secrets.
- Labels SHOULD work for both version-prefixed and legacy paths where c8volt uses both forms.
- Unknown paths MAY use generic fallback wording.

## Known Labels

| Method | Path Pattern | Fallback Activity |
|--------|--------------|-------------------|
| GET | `/v2/topology` | checking cluster topology |
| GET | `/v2/license` | loading license |
| POST | `/v2/deployments` | deploying resources |
| GET | `/v2/resources` | loading resources |
| GET | `/v2/resources/{key}` | loading resource |
| POST | `/v2/resources/{key}/deletion` | submitting resource deletion |
| POST | `/v2/process-instances/search` | searching process instances |
| POST | `/v2/process-instances` | creating process instance |
| GET | `/v2/process-instances/{key}` | loading process instance |
| POST | `/v2/process-instances/{key}/incidents/search` | searching process-instance incidents |
| POST | `/v2/process-instances/{key}/deletion` | submitting process-instance deletion |
| POST | `/v2/process-instances/{key}/cancellation` | submitting process-instance cancellation |
| POST | `/v2/process-definitions/search` | searching process definitions |
| GET | `/v2/process-definitions/{key}` | loading process definition |
| GET | `/v2/process-definitions/{key}/xml` | loading process-definition XML |
| POST | `/v2/process-definitions/{key}/deletion` | submitting process-definition deletion |
| POST | `/v2/incidents/search` | searching incidents |
| GET | `/v2/incidents/{key}` | loading incident |
| POST | `/v2/incidents/{key}/resolution` | submitting incident resolution |
| POST | `/v2/jobs/search` | searching jobs |
| GET | `/v2/jobs/{key}` | loading job |
| PATCH | `/v2/jobs/{key}` | updating job |
| POST | `/v2/batch-operations/search` | searching batch operations |
| GET | `/v2/batch-operations/{key}` | loading batch operation |
| POST | `/v2/batch-operations/cancellation` | submitting batch-operation cancellation |
| POST | `/v2/element-instances/search` | searching element instances |
| GET | `/v2/element-instances/{key}` | loading element instance |
| POST | `/v2/element-instances/{key}/incidents/search` | searching element-instance incidents |
| PUT | `/v2/element-instances/{key}/variables` | setting element variables |
| POST | `/v2/variables/search` | searching variables |
| GET | `/v2/variables/{key}` | loading variable |
| POST | `/v2/user-tasks/search` | searching user tasks |
| GET | `/v2/user-tasks/{key}` | loading user task |
| PATCH | `/v2/user-tasks/{key}` | updating user task |
| POST | `/v2/tenants/search` | searching tenants |
| GET | `/v2/tenants/{id}` | loading tenant |

## Legacy Path Equivalents

These non-`/v2` paths are considered equivalent where c8volt uses legacy Camunda 8.7 or Operate clients:

| Method | Path Pattern | Fallback Activity |
|--------|--------------|-------------------|
| POST | `/deployments` | deploying resources |
| POST | `/resources/{key}/deletion` | submitting resource deletion |
| POST | `/process-instances/search` | searching process instances |
| POST | `/process-definitions/search` | searching process definitions |
| POST | `/variables/search` | searching variables |

## Generic Fallbacks

| Method | Unknown Path Fallback |
|--------|-----------------------|
| GET | loading Camunda API data |
| POST | submitting Camunda API request |
| PATCH or PUT | updating Camunda API resource |
| DELETE | deleting Camunda API resource |
| Other | calling Camunda API |
