# CLI Contract: Slow Process Timeline Readability

## Command Surface

Existing command spellings remain supported:

```text
c8volt ops analyse slow-process-instances [flags]
c8volt ops analyze slow-process-instances [flags]
```

Aliases such as `spi` and `slow-pi` remain compatible with the existing command contract.

## New Flag

```text
--with-full-timeline
```

Rules:

- Applies to human output only.
- Restores complete chronological element and transition detail in human output.
- Does not change process-instance selection, root sorting, duration calculation, existing filters, JSON output, or keys-only output.
- Is available through both British and American command spellings and existing aliases.

## Default Human Output Contract

Default human output renders each process instance as an unindented root row followed by a compact hotspot summary when timeline rows are available.

Representative shape:

```text
6755399474030835 clmdirect-dev04-gke KycMonitoringProcess v1 COMPLETED s:2026-05-20T10:54:40.644 e:2026-05-20T12:52:04.887 p:6755399474030811 dur:1h57m24.243s
└─ slowest elements:
   ├─ [██████████] 97% USER_TASK UserTask_ConductQA COMPLETED dur:1h53m32.288s s:10:58:32.084 e:12:52:04.372
   ├─ [░░░░░░░░░░] 3% USER_TASK UserTask_ProcessOrder COMPLETED dur:3m27.242s s:10:55:04.842 e:10:58:32.084
   └─ hidden: 18 instant/fast timeline rows; use --with-full-timeline
process instances: 1
```

Rules:

- The section label is `slowest elements:`.
- Completed element contributors with process-duration share at or above 1% are visible and ordered largest to smallest.
- Active element rows are visible even when their current share is below 1%.
- Incident-bearing element rows are visible even when their share is below 1%.
- Rows matching multiple visibility reasons appear once.
- Default rows show enough context to identify the BPMN element: type, element ID, state, duration when available, and compact start or end timing when useful.
- Default rows do not show element instance keys unless needed to identify an incident-bearing row.
- When analyzed timeline rows are omitted, output includes a hidden-row summary with the omitted count and `--with-full-timeline` guidance.
- When no analyzed timeline rows are omitted, no hidden-row summary is rendered.
- The process-instance final count remains present.
- Detail filters such as `--element-id`, `--type`, `--element-state`, and `--dur-element-longer` keep only process instances with matching analyzed element or transition rows, then render those matching rows under the root.

## Full-Timeline Human Output Contract

When `--with-full-timeline` is set, human output keeps the root row and renders complete chronological detail using the existing timeline style.

Representative shape:

```text
c8volt ops analyse slow-process-instances -k 6755399474030835 --with-full-timeline
```

Expected behavior:

- Complete analyzed timeline rows are rendered under the root.
- Zero-duration gateways, transitions, and sub-1% rows remain visible when included by existing detail filters.
- Element instance keys remain visible as in the current full timeline style.
- Existing detail filters keep their meaning by excluding roots with no matching detail rows and do not create synthetic transitions.
- Hidden-row summaries are not rendered in full-timeline mode.

## JSON Output Contract

JSON output remains unchanged.

Rules:

- `--with-full-timeline` does not add, remove, rename, or reorder JSON fields.
- JSON continues to expose the complete ordered analysis payload for process instances that pass selection and detail filters.
- Summary-only fields such as hidden-row counts are not emitted in JSON.
- Values may still vary only for reasons that already existed, such as captured analysis time or live process state.

## Keys-Only Output Contract

Keys-only output remains unchanged.

Rules:

- Only unique process-instance keys are printed.
- One key appears per line.
- No hidden-row, summary, or full-timeline text appears.
- Detail filters emit only root keys for process instances with matching analyzed detail rows.
- `--with-full-timeline` has no effect on keys-only output.

## Documentation Contract

Command help, examples, README content, and generated CLI docs must include:

- The compact default hotspot summary behavior.
- The `--with-full-timeline` flag.
- A short example or explanation for using full-timeline mode when chronological audit/debug detail is needed.
