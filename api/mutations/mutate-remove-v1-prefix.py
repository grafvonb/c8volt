#!/usr/bin/env python3
import os
import sys

import yaml


def strip_v1_prefix(path: str) -> str:
    if path == "/v1":
        return "/"
    if path.startswith("/v1/"):
        return path[3:]
    return path


if len(sys.argv) < 2:
    print("Usage: python mutations/mutate-remove-v1-prefix.py <inputfile.yaml>")
    sys.exit(1)

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as f:
    doc = yaml.safe_load(f)

if not isinstance(doc, dict):
    raise ValueError(f"{path} did not parse as an OpenAPI mapping document")

paths = doc.get("paths", {})
if not isinstance(paths, dict):
    raise ValueError(f"{path} does not contain a paths mapping")

updated_paths = {}
for original_path, operations in paths.items():
    new_path = strip_v1_prefix(original_path)
    if new_path in updated_paths:
        raise ValueError(f"{path} would create duplicate path after /v1 removal: {new_path}")
    updated_paths[new_path] = operations

doc["paths"] = updated_paths

name, ext = os.path.splitext(path)
out = f"{name}-v1-prefix-removed{ext}"
with open(out, "w", encoding="utf-8") as f:
    yaml.dump(doc, f, sort_keys=False, allow_unicode=True)

print(f"Wrote mutated spec to: {out}")
