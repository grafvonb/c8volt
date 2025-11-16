#!/bin/bash

# auth
./api/generate-client.sh ./api/auth/oauth2-openapi.json ./internal/clients/auth/oauth2/client.gen.go oauth2

# v89
./api/generate-client.sh ./api/camunda-docs/api/administration-sm/administration-sm-openapi.yaml ./internal/clients/camunda/v89/administrationsm/client.gen.go administrationsm
./api/generate-client.sh ./api/camunda-docs/api/camunda/camunda-openapi.yaml ./internal/clients/camunda/v89/camunda/client.gen.go camunda

python ./api/mutate-operation-ids.py ./api/camunda-docs/api/operate/operate-openapi.yaml
python ./api/mutate-remove-sort-values.py ./api/camunda-docs/api/operate/operate-openapi-oids-updated.yaml
./api/generate-client.sh ./api/camunda-docs/api/operate/operate-openapi-oids-updated-sortvalues-removed.yaml ./internal/clients/camunda/v89/operate/client.gen.go operate

./api/generate-client.sh ./api/camunda-docs/api/tasklist/tasklist-openapi.yaml ./internal/clients/camunda/v89/tasklist/client.gen.go tasklist

# v88
./api/generate-client.sh ./api/camunda-docs/api/administration-sm/administration-sm-openapi.yaml ./internal/clients/camunda/v88/administrationsm/client.gen.go administrationsm

python ./api/mutate-search-query-schemas.py ./api/camunda-docs/api/camunda/version-8.8/camunda-openapi.yaml
python ./api/mutate-search-result-schemas.py ./api/camunda-docs/api/camunda/version-8.8/camunda-openapi-search-query-patched.yaml
./api/generate-client.sh ./api/camunda-docs/api/camunda/version-8.8/camunda-openapi-search-query-patched-search-result-patched.yaml ./internal/clients/camunda/v88/camunda/client.gen.go camunda

python ./api/mutate-operation-ids.py ./api/camunda-docs/api/operate/operate-openapi.yaml
python ./api/mutate-remove-sort-values.py ./api/camunda-docs/api/operate/operate-openapi-oids-updated.yaml
./api/generate-client.sh ./api/camunda-docs/api/operate/operate-openapi-oids-updated-sortvalues-removed.yaml ./internal/clients/camunda/v88/operate/client.gen.go operate

./api/generate-client.sh ./api/camunda-docs/api/tasklist/tasklist-openapi.yaml ./internal/clients/camunda/v88/tasklist/client.gen.go tasklist

# v87
./api/generate-client.sh ./api/camunda-docs/api/administration-sm/version-8.7/administration-sm-openapi.yaml ./internal/clients/camunda/v87/administrationsm/client.gen.go administrationsm
./api/generate-client.sh ./api/camunda-docs/api/camunda/version-8.7/camunda-openapi.yaml ./internal/clients/camunda/v87/camunda/client.gen.go camunda

python ./api/mutate-operation-ids.py ./api/camunda-docs/api/operate/version-8.7/operate-openapi.yaml
python ./api/mutate-remove-sort-values.py ./api/camunda-docs/api/operate/version-8.7/operate-openapi-oids-updated.yaml
./api/generate-client.sh ./api/camunda-docs/api/operate/version-8.7/operate-openapi-oids-updated-sortvalues-removed.yaml ./internal/clients/camunda/v87/operate/client.gen.go operate

./api/generate-client.sh ./api/camunda-docs/api/tasklist/version-8.7/tasklist-openapi.yaml ./internal/clients/camunda/v87/tasklist/client.gen.go tasklist

# v86
./api/generate-client.sh ./api/camunda-docs/api/administration-sm/version-8.6/administration-sm-openapi.yaml ./internal/clients/camunda/v86/administrationsm/client.gen.go administrationsm
./api/generate-client.sh ./api/camunda-docs/api/camunda/version-8.6/camunda-openapi.yaml ./internal/clients/camunda/v86/camunda/client.gen.go camunda
./api/generate-client.sh ./api/camunda-docs/api/operate/version-8.6/operate-openapi.yaml ./internal/clients/camunda/v86/operate/client.gen.go operate
./api/generate-client.sh ./api/camunda-docs/api/tasklist/version-8.6/tasklist-openapi.yaml ./internal/clients/camunda/v86/tasklist/client.gen.go tasklist
./api/generate-client.sh ./api/camunda-docs/api/zeebe/version-8.6/zeebe-openapi.yaml ./internal/clients/camunda/v86/zeebe/client.gen.go zeebe
