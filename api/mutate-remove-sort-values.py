#!/usr/bin/env python3
import yaml, sys, os

def remove_sortvalues(node):
    if isinstance(node, dict):
        # delete key if present
        if "sortValues" in node:
            del node["sortValues"]
        # recurse
        for v in list(node.values()):
            remove_sortvalues(v)
    elif isinstance(node, list):
        for v in node:
            remove_sortvalues(v)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python mutate-remove-sort-values.py <inputfile.yaml>")
        sys.exit(1)

    path = sys.argv[1]
    with open(path, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f)

    remove_sortvalues(data)

    name, ext = os.path.splitext(path)
    out = f"{name}-sortvalues-removed{ext}"
    with open(out, "w", encoding="utf-8") as f:
        yaml.dump(data, f, sort_keys=False, allow_unicode=True)
