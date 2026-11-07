#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_DIR="$(mktemp -d "${TMPDIR:-/tmp}/c8volt-generation-tools.XXXXXX")"
trap 'rm -rf "$TEST_DIR"' EXIT
mkdir "$TEST_DIR/bin"
export GENERATION_TEST_LOG="$TEST_DIR/calls"
export GENERATION_TEST_REDOCLY_VERSION="2.76.0"
cat >"$TEST_DIR/bin/go" <<'EOF'
#!/usr/bin/env bash
test "${GOWORK:-}" = off || exit 1
printf '%s\n' "$@" >"$GENERATION_TEST_LOG"
EOF
cat >"$TEST_DIR/bin/oapi-codegen" <<'EOF'
#!/usr/bin/env bash
echo 'unexpected PATH generator' >&2
exit 77
EOF
cat >"$TEST_DIR/bin/redocly" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "--version" ]; then
  printf '%s\n' "$GENERATION_TEST_REDOCLY_VERSION"
else
  printf '%s\n' "$@" >"$GENERATION_TEST_LOG"
fi
EOF
chmod +x "$TEST_DIR/bin/"*
export PATH="$TEST_DIR/bin:$PATH"
source "$REPO_ROOT/api/generation-tools.sh"

# Running from another directory must still select the repository's Go tool.
cd "$TEST_DIR"
bash "$REPO_ROOT/api/generate-go-client.sh" spec.yaml client.go camunda
printf '%s\n' -C "$REPO_ROOT" tool oapi-codegen -config "$REPO_ROOT/api/oapi-codegen.yaml" -generate types,client -package camunda -o "$PWD/client.go" "$PWD/spec.yaml" >expected
cmp expected "$GENERATION_TEST_LOG"

generation_redocly bundle spec.yaml -o bundled.yaml
printf '%s\n' bundle spec.yaml -o bundled.yaml >expected
cmp expected "$GENERATION_TEST_LOG"

# A wrong bundler version must fail before bundling or modifying any output.
rm "$GENERATION_TEST_LOG"
export GENERATION_TEST_REDOCLY_VERSION="0.0.0"
if generation_redocly bundle spec.yaml -o bundled.yaml >stdout 2>stderr; then
  echo 'expected Redocly version mismatch' >&2
  exit 1
fi
grep -Fq 'redocly version mismatch: need 2.76.0, got 0.0.0' stderr
test ! -e "$GENERATION_TEST_LOG"
if bash "$REPO_ROOT/api/generate-v810-client.sh" --target v89 --camunda-tag 8.9.28 >stdout 2>stderr; then
  echo 'the 8.10 compatibility entry point must reject retargeting' >&2
  exit 1
fi
grep -Fq 'Unknown argument: --target' stderr
echo 'ok api/tests/generation_tools_test.sh'
