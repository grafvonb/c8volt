#!/bin/bash

repo="git@github.com:camunda/camunda-docs.git"
tag=$(git ls-remote --tags --refs "$repo" | awk -F/ '{print $3}' | sort -V | tail -n1)

git clone --depth 1 --filter=blob:none --branch "$tag" "$repo" camunda-docs
cd camunda-docs
git sparse-checkout init --no-cone
git sparse-checkout set '/api/*'