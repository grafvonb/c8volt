#!/bin/bash

# determine latest tag
repo="git@github.com:camunda/camunda-docs.git"
tag=$(git ls-remote --tags --refs "$repo" | awk -F/ '{print $3}' | sort -V | tail -n1)

# checkout latest tagged version with sparse checkout, meaning only the /api folder
git clone --depth 1 --filter=blob:none --branch "$tag" "$repo" camunda-docs
cd camunda-docs
git sparse-checkout init --no-cone
git sparse-checkout set '/api/*'

# cleanup
rm -rf ./camunda-docs/.git
find . -type f -name '*.js' -delete