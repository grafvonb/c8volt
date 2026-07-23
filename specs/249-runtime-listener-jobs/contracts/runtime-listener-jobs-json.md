# JSON Contract: Runtime Listener Jobs Under Elements

## Process Activity Item Shape

When `--with-listeners` is requested with element enrichment, each element object includes a `listeners` array:

```json
{
  "items": [
    {
      "item": {
        "key": "2251799813688001",
        "bpmnProcessId": "OrderProcess"
      },
      "elements": [
        {
          "elementInstanceKey": "2251799813689002",
          "elementId": "ReviewOrder",
          "type": "USER_TASK",
          "state": "ACTIVE",
          "processInstanceKey": "2251799813688001",
          "listeners": [
            {
              "jobKey": "2251799813689101",
              "kind": "TASK_LISTENER",
              "listenerEventType": "COMPLETING",
              "type": "audit-user-task",
              "state": "CREATED",
              "retries": 3,
              "worker": "audit-worker",
              "processInstanceKey": "2251799813688001",
              "elementInstanceKey": "2251799813689002",
              "elementId": "ReviewOrder",
              "tenantId": "tenant-a"
            }
          ]
        }
      ]
    }
  ]
}
```

## Walk Payload Shape

`--json walk pi --with-elements --with-listeners` preserves traversal metadata and adds listener arrays under elements inside each walked item:

```json
{
  "mode": "family",
  "outcome": "complete",
  "rootKey": "2251799813688001",
  "keys": ["2251799813688001"],
  "edges": {},
  "items": [
    {
      "item": {
        "key": "2251799813688001"
      },
      "elements": [
        {
          "elementInstanceKey": "2251799813689002",
          "elementId": "ReviewOrder",
          "type": "USER_TASK",
          "state": "ACTIVE",
          "listeners": []
        }
      ]
    }
  ]
}
```

## Element Command Shape

`--json get element --pi-key <process-instance-key> --with-listeners` returns the existing element collection shape with `listeners` included on each returned element when requested.

```json
{
  "items": [
    {
      "elementInstanceKey": "2251799813689002",
      "elementId": "ReviewOrder",
      "type": "USER_TASK",
      "state": "ACTIVE",
      "listeners": [
        {
          "jobKey": "2251799813689101",
          "kind": "TASK_LISTENER",
          "listenerEventType": "COMPLETING",
          "type": "audit-user-task",
          "state": "CREATED",
          "retries": 3
        }
      ]
    }
  ],
  "limit": 0
}
```

## Slow Analysis Shape

When slow-process analysis listener enrichment is requested, element timeline entries include `listeners`. Transition timeline entries do not include listener fields.

```json
{
  "items": [
    {
      "key": "2251799813688001",
      "timeline": [
        {
          "kind": "element",
          "elementInstanceKey": "2251799813689002",
          "elementId": "ReviewOrder",
          "type": "USER_TASK",
          "state": "ACTIVE",
          "listeners": [
            {
              "jobKey": "2251799813689101",
              "kind": "TASK_LISTENER",
              "listenerEventType": "COMPLETING",
              "type": "audit-user-task",
              "state": "CREATED",
              "retries": 3
            }
          ]
        },
        {
          "kind": "transition",
          "fromElementInstanceKey": "2251799813689002",
          "toElementInstanceKey": "2251799813689003"
        }
      ]
    }
  ]
}
```

## Field Presence Rules

- If `--with-listeners` is not requested, `listeners` fields are omitted.
- If `--with-listeners` is requested and an element has no matched listener jobs, `listeners` is an empty array.
- Listener jobs without a matched element instance key are omitted.
- Existing `variables`, `incidents`, `elements`, traversal metadata, and slow-analysis fields retain their current meanings.
