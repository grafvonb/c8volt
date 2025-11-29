#!/usr/bin/env python3
import yaml
import re
import sys
import os

def to_camel_case(text: str) -> str:
    """Convert summary string into camelCase suitable for operationId."""
    words = re.split(r'[^a-zA-Z0-9]+', text.strip())
    words = [w for w in words if w]  # remove empty parts
    if not words:
        return ""
    return words[0].lower() + ''.join(word.capitalize() for word in words[1:])

def update_operation_ids(input_file: str):
    with open(input_file, "r", encoding="utf-8") as f:
        spec = yaml.safe_load(f)

    for path, methods in spec.get("paths", {}).items():
        for method, details in methods.items():
            if isinstance(details, dict) and "summary" in details:
                summary = details["summary"]
                new_id = to_camel_case(summary)
                details["operationId"] = new_id

    # Build output filename with suffix
    folder, filename = os.path.split(input_file)
    name, ext = os.path.splitext(filename)
    output_file = os.path.join(folder, f"{name}-oids-updated{ext}")

    with open(output_file, "w", encoding="utf-8") as f:
        yaml.dump(spec, f, sort_keys=False, allow_unicode=True)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python mutate-operation-ids.py <inputfile.yaml>")
        sys.exit(1)

    input_file = sys.argv[1]
    update_operation_ids(input_file)