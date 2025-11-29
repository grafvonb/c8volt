#!/usr/bin/env python3
import sys, os, yaml

if len(sys.argv) < 2:
    print("Usage: python mutate-fix-jobresult-discriminator.py <inputfile.yaml>")
    sys.exit(1)

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as f:
    doc = yaml.safe_load(f)

schemas = doc.get("components", {}).get("schemas", {})

mapping = {
    "JobResultUserTask": "userTask",
    "JobResultAdHocSubProcess": "adHocSubProcess",
    "ProcessInstanceCreationTerminateInstruction": "ProcessInstanceCreationTerminateInstruction",
}

for name, const in mapping.items():
    schema = schemas.get(name)
    if not isinstance(schema, dict):
        continue

    props = schema.setdefault("properties", {})
    type_prop = props.setdefault("type", {})

    type_prop["type"] = "string"
    type_prop.pop("nullable", None)
    type_prop["enum"] = [const]

    req = schema.setdefault("required", [])
    if "type" not in req:
        req.append("type")

name, ext = os.path.splitext(path)
out = f"{name}-jobresult-fixed{ext}"
with open(out, "w", encoding="utf-8") as f:
    yaml.dump(doc, f, sort_keys=False, allow_unicode=True)

print(f"Wrote mutated spec to: {out}")