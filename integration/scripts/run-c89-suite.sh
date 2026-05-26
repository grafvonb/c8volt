#!/usr/bin/env bash

set -u

export IT_SUITE_NAME="c89"
export IT_TARGET_VERSION="8.9"
export IT_FIXTURE_PREFIX="C89"
export C8VOLT_IT_PROFILE="${C8VOLT_IT_PROFILE:-kind-camunda-platform-local-c89}"

exec "$(dirname "$0")/run-suite.sh"

