#!/bin/bash

# auth
./generate-client.sh ./auth/oauth2-openapi.json ../internal/clients/auth/oauth2/client.gen.go oauth2

# v89
./generate-client.sh ./camunda-docs/api/administration-sm/administration-sm-openapi.yaml ../internal/clients/camunda/v89/administrationsm/client.gen.go administrationsm
./mutate-fix-jobresult-discriminator.py ./camunda-docs/api/camunda/v2/camunda-openapi-bundled.yaml
./generate-client.sh ./camunda-docs/api/camunda/v2/camunda-openapi-bundled-jobresult-fixed.yaml ../internal/clients/camunda/v89/camunda/client.gen.go camunda

./mutate-operation-ids.py ./camunda-docs/api/operate/operate-openapi.yaml
./mutate-remove-sort-values.py ./camunda-docs/api/operate/operate-openapi-oids-updated.yaml
./generate-client.sh ./camunda-docs/api/operate/operate-openapi-oids-updated-sortvalues-removed.yaml ../internal/clients/camunda/v89/operate/client.gen.go operate

./generate-client.sh ./camunda-docs/api/tasklist/tasklist-openapi.yaml ../internal/clients/camunda/v89/tasklist/client.gen.go tasklist

# v88
./generate-client.sh ./camunda-docs/api/administration-sm/administration-sm-openapi.yaml ../internal/clients/camunda/v88/administrationsm/client.gen.go administrationsm

./mutate-search-query-schemas.py ./camunda-docs/api/camunda/version-8.8/camunda-openapi.yaml
./mutate-search-result-schemas.py ./camunda-docs/api/camunda/version-8.8/camunda-openapi-search-query-patched.yaml
./generate-client.sh ./camunda-docs/api/camunda/version-8.8/camunda-openapi-search-query-patched-search-result-patched.yaml ../internal/clients/camunda/v88/camunda/client.gen.go camunda

./mutate-operation-ids.py ./camunda-docs/api/operate/operate-openapi.yaml
./mutate-remove-sort-values.py ./camunda-docs/api/operate/operate-openapi-oids-updated.yaml
./generate-client.sh ./camunda-docs/api/operate/operate-openapi-oids-updated-sortvalues-removed.yaml ../internal/clients/camunda/v88/operate/client.gen.go operate

./generate-client.sh ./camunda-docs/api/tasklist/tasklist-openapi.yaml ../internal/clients/camunda/v88/tasklist/client.gen.go tasklist

# v87
./generate-client.sh ./camunda-docs/api/administration-sm/version-8.7/administration-sm-openapi.yaml ../internal/clients/camunda/v87/administrationsm/client.gen.go administrationsm
./generate-client.sh ./camunda-docs/api/camunda/version-8.7/camunda-openapi.yaml ../internal/clients/camunda/v87/camunda/client.gen.go camunda

./mutate-operation-ids.py ./camunda-docs/api/operate/version-8.7/operate-openapi.yaml
./mutate-remove-sort-values.py ./camunda-docs/api/operate/version-8.7/operate-openapi-oids-updated.yaml
./generate-client.sh ./camunda-docs/api/operate/version-8.7/operate-openapi-oids-updated-sortvalues-removed.yaml ../internal/clients/camunda/v87/operate/client.gen.go operate

./generate-client.sh ./camunda-docs/api/tasklist/version-8.7/tasklist-openapi.yaml ../internal/clients/camunda/v87/tasklist/client.gen.go tasklist

# v86
./generate-client.sh ./camunda-docs/api/administration-sm/version-8.6/administration-sm-openapi.yaml ../internal/clients/camunda/v86/administrationsm/client.gen.go administrationsm
./generate-client.sh ./camunda-docs/api/camunda/version-8.6/camunda-openapi.yaml ../internal/clients/camunda/v86/camunda/client.gen.go camunda
./generate-client.sh ./camunda-docs/api/operate/version-8.6/operate-openapi.yaml ../internal/clients/camunda/v86/operate/client.gen.go operate
./generate-client.sh ./camunda-docs/api/tasklist/version-8.6/tasklist-openapi.yaml ../internal/clients/camunda/v86/tasklist/client.gen.go tasklist
./generate-client.sh ./camunda-docs/api/zeebe/version-8.6/zeebe-openapi.yaml ../internal/clients/camunda/v86/zeebe/client.gen.go zeebe
