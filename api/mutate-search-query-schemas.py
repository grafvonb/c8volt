#!/usr/bin/env python3
import yaml
import sys
import os

# Patch OpenAPI schemas for SearchQuery types to correct structure.
# The intention is:
# ProcessDefinitionSearchQuery = SearchQueryRequest + { sort, filter }.
#
# oapi-codegen looks at this pattern (allOf with a single $ref plus sibling properties)
# and generates a type alias to the referenced type, dropping the extra properties.
# That’s why all your *SearchQuery types are aliases to SearchQueryRequest and only carry Page.
# ProcessDefinitionSearchQuery:
#   type: object
#   allOf:
#     - $ref: "#/components/schemas/SearchQueryRequest"
#   properties:
#     sort:
#       description: Sort field criteria.
#       type: array
#       items:
#         $ref: "#/components/schemas/ProcessDefinitionSearchQuerySortRequest"
#     filter:
#       description: The process definition search filters.
#       allOf:
#         - $ref: "#/components/schemas/ProcessDefinitionFilter"
# to:
# Key changes:
# Move type: object and properties into a second allOf item.
# Make filter a direct $ref instead of allOf with a single $ref (not strictly required but cleaner).
# ProcessDefinitionSearchQuery:
#   allOf:
#     - $ref: "#/components/schemas/SearchQueryRequest"
#     - type: object
#       properties:
#         sort:
#           description: Sort field criteria.
#           type: array
#           items:
#             $ref: "#/components/schemas/ProcessDefinitionSearchQuerySortRequest"
#         filter:
#           description: The process definition search filters.
#           $ref: "#/components/schemas/ProcessDefinitionFilter"
#       additionalProperties: false

def patch_search_query_schemas(input_file: str):
    with open(input_file, "r", encoding="utf-8") as f:
        spec = yaml.safe_load(f)

    components = spec.get("components", {})
    schemas = components.get("schemas", {})

    if not isinstance(schemas, dict):
        raise ValueError("components.schemas is not a mapping")

    for name, schema in schemas.items():
        if not isinstance(schema, dict):
            continue

        # Only patch schemas that:
        # - are declared as type: object
        # - have allOf with a single $ref
        # - have properties with either sort or filter (our search query types)
        if schema.get("type") != "object":
            continue

        all_of = schema.get("allOf")
        if not isinstance(all_of, list) or len(all_of) != 1:
            continue

        base_ref_obj = all_of[0]
        if not (isinstance(base_ref_obj, dict) and "$ref" in base_ref_obj):
            continue

        props = schema.get("properties")
        if not isinstance(props, dict):
            continue

        if "sort" not in props and "filter" not in props:
            # Likely not one of the *SearchQuery types we care about
            continue

        # Fix the "filter" property if it is using allOf: [{$ref: ...}]
        if "filter" in props:
            filter_schema = props["filter"]
            if (
                    isinstance(filter_schema, dict)
                    and "allOf" in filter_schema
                    and isinstance(filter_schema["allOf"], list)
                    and len(filter_schema["allOf"]) == 1
                    and isinstance(filter_schema["allOf"][0], dict)
                    and "$ref" in filter_schema["allOf"][0]
            ):
                base_filter_ref = filter_schema["allOf"][0]["$ref"]
                # Preserve other fields like description, etc.
                new_filter_schema = {
                    k: v for k, v in filter_schema.items() if k != "allOf"
                }
                new_filter_schema["$ref"] = base_filter_ref
                props["filter"] = new_filter_schema

        # Build new allOf shape:
        # ProcessDefinitionSearchQuery:
        #   allOf:
        #     - $ref: "#/components/schemas/SearchQueryRequest"
        #     - type: object
        #       properties: { ... }
        #       additionalProperties: false
        #
        # Keep description (and any other top-level metadata) on the top level.
        description = schema.get("description")
        other_top_level = {
            k: v
            for k, v in schema.items()
            if k not in {"type", "allOf", "properties", "additionalProperties", "description"}
        }

        # Inner object with properties
        inner_obj = {
            "type": "object",
            "properties": props,
            # Explicitly disallow additional properties by default
            "additionalProperties": False,
        }

        new_schema = {
            "allOf": [
                {"$ref": base_ref_obj["$ref"]},
                inner_obj,
            ]
        }

        if description is not None:
            new_schema["description"] = description

        # Preserve any other top-level keys that are not structural (if present)
        new_schema.update(other_top_level)

        schemas[name] = new_schema

    # Write out patched spec
    folder, filename = os.path.split(input_file)
    name, ext = os.path.splitext(filename)
    output_file = os.path.join(folder, f"{name}-search-query-patched{ext}")

    with open(output_file, "w", encoding="utf-8") as f:
        yaml.dump(spec, f, sort_keys=False, allow_unicode=True)

    print(f"Wrote patched spec to: {output_file}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python mutate-search-schemas.py <inputfile.yaml>")
        sys.exit(1)

    patch_search_query_schemas(sys.argv[1])
