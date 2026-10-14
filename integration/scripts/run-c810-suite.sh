#!/usr/bin/env bash

set -u

export IT_SUITE_NAME="c810"
export IT_TARGET_VERSION="8.10"
export IT_FIXTURE_PREFIX="C810"
export C8VOLT_IT_PROFILE="${C8VOLT_IT_PROFILE:-kind-camunda-platform-local-c810}"

exec "$(dirname "$0")/run-suite.sh"
