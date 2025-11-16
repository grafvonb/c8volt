import yaml
import sys
import os

# Fix the OpenAPI - "items" is part of the generated result
#
# In the spec, ProcessDefinitionSearchQueryResult looks like this:
# ProcessDefinitionSearchQueryResult:
#   type: object
#   allOf:
#     - $ref: "#/components/schemas/SearchQueryResponse"
#   properties:
#     items:
#       description: The matching process definitions.
#       type: array
#       items:
#         $ref: "#/components/schemas/ProcessDefinitionResult"
# :contentReference[oaicite:1]{index=1}
#
# Because of the type: object + allOf with a single $ref, oapi-codegen again generates a type alias instead of a struct, so items is dropped.
# We need to change the schema to:
# ProcessDefinitionSearchQueryResult:
#   allOf:
#     - $ref: "#/components/schemas/SearchQueryResponse"
#     - type: object
#       properties:
#         items:
#           description: The matching process definitions.
#           type: array
#           items:
#             $ref: "#/components/schemas/ProcessDefinitionResult"
#       required:
#         - items
#       additionalProperties: false

def patch_search_result_schemas(input_file: str):
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
        # - have properties with "items" (our *SearchQueryResult types)
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

        if "items" not in props:
            # Likely not one of the *SearchQueryResult types we care about
            continue

        # Optional: keep/propagate "required" keys into the inner object
        required = schema.get("required")
        description = schema.get("description")

        # Preserve other top-level keys that are not structural, if any
        other_top_level = {
            k: v
            for k, v in schema.items()
            if k
               not in {
                   "type",
                   "allOf",
                   "properties",
                   "additionalProperties",
                   "required",
                   "description",
               }
        }

        # Inner object with properties + required
        inner_obj = {
            "type": "object",
            "properties": props,
            "additionalProperties": False,
        }
        if isinstance(required, list) and required:
            inner_obj["required"] = required

        # New schema shape:
        # <Result>:
        #   allOf:
        #     - $ref: "#/components/schemas/SearchQueryResponse"
        #     - type: object
        #       properties:
        #         items: ...
        #       required: [...]
        #       additionalProperties: false
        new_schema = {
            "allOf": [
                {"$ref": base_ref_obj["$ref"]},
                inner_obj,
            ]
        }

        if description is not None:
            new_schema["description"] = description

        new_schema.update(other_top_level)

        schemas[name] = new_schema

    # Write out patched spec
    folder, filename = os.path.split(input_file)
    name, ext = os.path.splitext(filename)
    output_file = os.path.join(folder, f"{name}-search-result-patched{ext}")

    with open(output_file, "w", encoding="utf-8") as f:
        yaml.dump(spec, f, sort_keys=False, allow_unicode=True)

    print(f"Wrote patched spec to: {output_file}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python mutate-search-results.py <inputfile.yaml>")
        sys.exit(1)

    patch_search_result_schemas(sys.argv[1])
