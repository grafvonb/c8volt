#!/usr/bin/env python3
import yaml, sys, os

def fix(node, parent=None):
    if isinstance(node, dict):
        if parent == "sortValues" and node.get("type") == "array":
            items = node.get("items")
            if isinstance(items, dict) and items.get("type") == "object":
                node["items"] = {"type": "string"}
        for k, v in node.items():
            fix(v, k)
    elif isinstance(node, list):
        for v in node:
            fix(v, parent)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python mutate-sort-values.py <inputfile.yaml>")
        sys.exit(1)

    path = sys.argv[1]
    with open(path, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f)

    fix(data)

    name, ext = os.path.splitext(path)
    out = f"{name}-sortvalues-fixed{ext}"
    with open(out, "w", encoding="utf-8") as f:
        yaml.dump(data, f, sort_keys=False, allow_unicode=True)
