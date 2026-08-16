#!/usr/bin/env python3
"""Verify that each V810 OpenAPI mutation made its intended semantic change."""

import re
import sys
from pathlib import Path

import yaml


def single_ref_object_with(schema: object, property_name: str) -> bool:
    return (
        isinstance(schema, dict)
        and schema.get("type") == "object"
        and isinstance(schema.get("allOf"), list)
        and len(schema["allOf"]) == 1
        and isinstance(schema["allOf"][0], dict)
        and isinstance(schema["allOf"][0].get("$ref"), str)
        and isinstance(schema.get("properties"), dict)
        and property_name in schema["properties"]
    )


def assert_wrapped_schema(
    before_schemas: dict, after_schemas: dict, name: str, property_name: str
) -> None:
    old = before_schemas[name]
    new = after_schemas.get(name)
    if not isinstance(new, dict) or old == new:
        raise ValueError(f"schema {name} was not changed")
    all_of = new.get("allOf")
    if not isinstance(all_of, list) or len(all_of) != 2:
        raise ValueError(f"schema {name} does not have the expected allOf wrapper")
    if all_of[0] != old["allOf"][0]:
        raise ValueError(f"schema {name} changed its base reference")
    inner = all_of[1]
    if (
        not isinstance(inner, dict)
        or inner.get("type") != "object"
        or inner.get("additionalProperties") is not False
        or property_name not in inner.get("properties", {})
    ):
        raise ValueError(f"schema {name} does not preserve {property_name}")
    expected_properties = dict(old["properties"])
    filter_schema = expected_properties.get("filter")
    if (
        isinstance(filter_schema, dict)
        and isinstance(filter_schema.get("allOf"), list)
        and len(filter_schema["allOf"]) == 1
        and isinstance(filter_schema["allOf"][0], dict)
        and "$ref" in filter_schema["allOf"][0]
    ):
        expected_filter = {
            key: value for key, value in filter_schema.items() if key != "allOf"
        }
        expected_filter["$ref"] = filter_schema["allOf"][0]["$ref"]
        expected_properties["filter"] = expected_filter
    if inner["properties"] != expected_properties:
        raise ValueError(f"schema {name} changed its wrapped properties")
    required = old.get("required")
    if isinstance(required, list) and required and inner.get("required") != required:
        raise ValueError(f"schema {name} changed its required properties")


def pascal(value: str) -> str:
    return "".join(
        part[:1].upper() + part[1:]
        for part in re.split(r"[^a-zA-Z0-9]+", value)
        if part
    )


def validate(before: dict, after: dict, suffix: str) -> int:
    before_schemas = before.get("components", {}).get("schemas", {})
    after_schemas = after.get("components", {}).get("schemas", {})
    if not isinstance(before_schemas, dict) or not isinstance(after_schemas, dict):
        raise ValueError("components.schemas is not a mapping")

    changed = 0
    if suffix == "search-query-patched":
        targets = [
            name
            for name, schema in before_schemas.items()
            if single_ref_object_with(schema, "sort")
            or single_ref_object_with(schema, "filter")
        ]
        for name in targets:
            property_name = (
                "sort" if "sort" in before_schemas[name]["properties"] else "filter"
            )
            assert_wrapped_schema(
                before_schemas, after_schemas, name, property_name
            )
            changed += 1
    elif suffix == "search-result-patched":
        targets = [
            name
            for name, schema in before_schemas.items()
            if single_ref_object_with(schema, "items")
        ]
        for name in targets:
            assert_wrapped_schema(before_schemas, after_schemas, name, "items")
            changed += 1
    elif suffix == "process-instance-filter-fields-fixed":
        name = "ProcessInstanceFilterFields"
        old = before_schemas.get(name)
        new = after_schemas.get(name)
        movable = (
            "type",
            "properties",
            "required",
            "additionalProperties",
            "minProperties",
            "maxProperties",
        )
        moved = {
            key: old[key]
            for key in movable
            if isinstance(old, dict) and key in old
        }
        if moved and isinstance(new, dict):
            all_of = new.get("allOf")
            appended = all_of[-1] if isinstance(all_of, list) and all_of else None
            if (
                isinstance(appended, dict)
                and all(appended.get(key) == value for key, value in moved.items())
                and all(key not in new for key in moved)
            ):
                changed = 1
    elif suffix == "jobresult-fixed":
        mapping = {
            "JobResultUserTask": "userTask",
            "JobResultAdHocSubProcess": "adHocSubProcess",
            "ProcessInstanceCreationTerminateInstruction": "TERMINATE_PROCESS_INSTANCE",
        }
        for name, expected in mapping.items():
            old = before_schemas.get(name)
            new = after_schemas.get(name)
            if not isinstance(old, dict):
                continue
            if not isinstance(new, dict):
                raise ValueError(f"schema {name} disappeared")
            type_property = new.get("properties", {}).get("type", {})
            if not (
                type_property.get("type") == "string"
                and type_property.get("enum") == [expected]
                and "type" in new.get("required", [])
            ):
                raise ValueError(f"schema {name} has the wrong discriminator")
            if old != new:
                changed += 1
    elif suffix == "opids-fixed":
        schema_names = set(before_schemas)
        before_paths = before.get("paths", {})
        after_paths = after.get("paths", {})
        collisions = 0
        for path, methods in before_paths.items():
            if not isinstance(methods, dict):
                continue
            for method, operation in methods.items():
                if not isinstance(operation, dict):
                    continue
                old_id = operation.get("operationId")
                new_id = after_paths.get(path, {}).get(method, {}).get("operationId")
                if (
                    isinstance(old_id, str)
                    and f"{pascal(old_id)}Response" in schema_names
                ):
                    collisions += 1
                    if isinstance(new_id, str) and new_id != old_id:
                        changed += 1
                    else:
                        raise ValueError(
                            f"operationId collision {old_id} was not fixed"
                        )
        if changed != collisions:
            raise ValueError("not every operationId collision was fixed")
    else:
        raise ValueError(f"unknown mutation suffix: {suffix}")

    if changed < 1:
        raise ValueError(f"mutation {suffix} changed no matching schema")
    return changed


def main() -> int:
    if len(sys.argv) != 4:
        print(
            "Usage: validate-v810-mutation-effects.py <before.yaml> <after.yaml> <suffix>",
            file=sys.stderr,
        )
        return 2
    before_path, after_path, suffix = sys.argv[1:]
    before = yaml.safe_load(Path(before_path).read_text(encoding="utf-8"))
    after = yaml.safe_load(Path(after_path).read_text(encoding="utf-8"))
    if not isinstance(before, dict) or not isinstance(after, dict):
        print("mutation inputs must be OpenAPI mapping documents", file=sys.stderr)
        return 1
    try:
        validate(before, after, suffix)
    except ValueError as exc:
        print(exc, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
