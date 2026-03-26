#!/usr/bin/env python3
# Fix operationId values in the bundled Camunda v2 OpenAPI spec when
# oapi-codegen response wrapper names would collide with schema model names.
#
# Usage:
#   python3 api/mutate-fix-camunda-v2-operation-id-collisions.py <inputfile.yaml>
#
# Output:
#   Writes <inputfile>-opids-fixed.yaml in the same directory.
import os
import re
import sys

import yaml


def to_pascal_case(text: str) -> str:
    parts = [p for p in re.split(r"[^a-zA-Z0-9]+", text) if p]
    if not parts:
        return ""
    return "".join(part[0].upper() + part[1:] for part in parts)


def make_unique_operation_id(base: str, used_ids: set[str], schema_names: set[str]) -> str:
    # Keep names stable and readable while avoiding response-type collisions.
    candidate = f"{base}Op"
    i = 2
    while candidate in used_ids or f"{to_pascal_case(candidate)}Response" in schema_names:
        candidate = f"{base}Op{i}"
        i += 1
    return candidate


def main() -> int:
    if len(sys.argv) < 2:
        print("Usage: python mutate-fix-camunda-v2-operation-id-collisions.py <inputfile.yaml>")
        return 1

    path = sys.argv[1]
    with open(path, "r", encoding="utf-8") as f:
        doc = yaml.safe_load(f)

    schemas = doc.get("components", {}).get("schemas", {})
    schema_names = set(schemas.keys()) if isinstance(schemas, dict) else set()

    paths = doc.get("paths", {})
    used_ids = set()
    for methods in paths.values():
        if not isinstance(methods, dict):
            continue
        for op in methods.values():
            if isinstance(op, dict) and isinstance(op.get("operationId"), str):
                used_ids.add(op["operationId"])

    changed = []
    for methods in paths.values():
        if not isinstance(methods, dict):
            continue
        for op in methods.values():
            if not isinstance(op, dict):
                continue
            operation_id = op.get("operationId")
            if not isinstance(operation_id, str):
                continue
            response_type = f"{to_pascal_case(operation_id)}Response"
            if response_type in schema_names:
                new_operation_id = make_unique_operation_id(operation_id, used_ids, schema_names)
                op["operationId"] = new_operation_id
                used_ids.add(new_operation_id)
                changed.append((operation_id, new_operation_id))

    name, ext = os.path.splitext(path)
    out = f"{name}-opids-fixed{ext}"
    with open(out, "w", encoding="utf-8") as f:
        yaml.dump(doc, f, sort_keys=False, allow_unicode=True)

    if changed:
        for old, new in changed:
            print(f"Updated operationId: {old} -> {new}")
    else:
        print("No operationId collisions found")
    print(f"Wrote patched spec to: {out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())



