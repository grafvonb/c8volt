#!/usr/bin/env bash

set -u

export IT_SUITE_NAME="c88"
export IT_TARGET_VERSION="8.8"
export IT_FIXTURE_PREFIX="C88"
export C8VOLT_IT_PROFILE="${C8VOLT_IT_PROFILE:-kind-camunda-platform-local-c88}"

exec "$(dirname "$0")/run-suite.sh"

