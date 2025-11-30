#!/usr/bin/env python3
import sys
import os
import yaml

if len(sys.argv) < 2:
    print("Usage: python mutate-fix-process-instance-filter-fields.py <inputfile.yaml>")
    sys.exit(1)

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as f:
    doc = yaml.safe_load(f)

schemas = doc.get("components", {}).get("schemas", {})

target_name = "ProcessInstanceFilterFields"
schema = schemas.get(target_name)

if isinstance(schema, dict):
    all_of = schema.get("allOf")
    if isinstance(all_of, list) and len(all_of) >= 1:
        # Collect keys that should move into a separate allOf element
        extra = {}
        for key in ["type", "properties", "required", "additionalProperties",
                    "minProperties", "maxProperties"]:
            if key in schema:
                extra[key] = schema.pop(key)

        # Only append if we actually moved something
        if extra:
            # Ensure it's explicitly marked as object if not already
            extra.setdefault("type", "object")
            all_of.append(extra)
            schema["allOf"] = all_of

name, ext = os.path.splitext(path)
out = f"{name}-process-instance-filter-fields-fixed{ext}"
with open(out, "w", encoding="utf-8") as f:
    yaml.dump(doc, f, sort_keys=False, allow_unicode=True)

print(f"Wrote mutated spec to: {out}")
