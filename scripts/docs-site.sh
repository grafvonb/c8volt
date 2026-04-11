#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="${ROOT_DIR}/docs"
RUBY_VERSION_FILE="${ROOT_DIR}/.ruby-version"

# Current docs gem stack resolves through ffi, which rejects Ruby >= 3.5.
RUBY_MIN="2.5.0"
RUBY_MAX_EXCLUSIVE="3.5.0"

cmd="${1:-help}"

ruby_runner=(ruby)
bundle_runner=(bundle)

configure_runners() {
  if command -v rbenv >/dev/null 2>&1 && [[ -f "${RUBY_VERSION_FILE}" ]]; then
    local requested
    requested="$(<"${RUBY_VERSION_FILE}")"
    ruby_runner=(rbenv exec ruby)
    bundle_runner=(rbenv exec bundle)
    export RBENV_VERSION="${requested}"
  fi
}

run_ruby() {
  "${ruby_runner[@]}" "$@"
}

run_bundle() {
  "${bundle_runner[@]}" "$@"
}

configure_bundler_env() {
  # Keep docs gems isolated from ~/.gem so native extensions are rebuilt per Ruby version.
  unset GEM_HOME
  unset GEM_PATH
  export BUNDLE_PATH="${DOCS_DIR}/vendor/bundle"
  export BUNDLE_DISABLE_SHARED_GEMS="true"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command '$1' is not available in PATH" >&2
    exit 1
  fi
}

check_ruby_version() {
  if [[ "${ruby_runner[*]}" == "ruby" ]]; then
    require_cmd ruby
  fi

  if ! run_ruby -e '
min = Gem::Version.new(ARGV[0])
max = Gem::Version.new(ARGV[1])
cur = Gem::Version.new(RUBY_VERSION)
exit(cur >= min && cur < max ? 0 : 1)
' "${RUBY_MIN}" "${RUBY_MAX_EXCLUSIVE}"; then
    local current
    current="$(run_ruby -e 'print RUBY_VERSION')"
    echo "error: incompatible Ruby version for docs: ${current}" >&2
    echo "expected: Ruby >= ${RUBY_MIN} and < ${RUBY_MAX_EXCLUSIVE}" >&2
    echo "hint: switch to Ruby 3.4.x (for example with rbenv/asdf/mise) and retry" >&2
    exit 2
  fi
}

check_bundler() {
  if [[ "${bundle_runner[*]}" == "bundle" ]]; then
    require_cmd bundle
  fi
  if ! run_bundle --version >/dev/null 2>&1; then
    echo "error: Bundler is not available for the selected Ruby runtime" >&2
    exit 1
  fi
}

bundle_install_if_needed() {
  cd "${DOCS_DIR}"
  if ! run_bundle check >/dev/null 2>&1; then
    run_bundle install
  fi
}

do_install() {
  check_ruby_version
  check_bundler
  configure_bundler_env
  cd "${DOCS_DIR}"
  run_bundle install
}

do_build() {
  check_ruby_version
  check_bundler
  configure_bundler_env
  bundle_install_if_needed
  cd "${DOCS_DIR}"
  run_bundle exec jekyll build
}

do_build_root() {
  check_ruby_version
  check_bundler
  configure_bundler_env
  bundle_install_if_needed
  cd "${DOCS_DIR}"
  run_bundle exec jekyll build --baseurl ""
}

do_serve() {
  check_ruby_version
  check_bundler
  configure_bundler_env
  bundle_install_if_needed
  cd "${DOCS_DIR}"
  run_bundle exec jekyll serve --livereload --host 127.0.0.1 --baseurl /c8volt
}

do_check() {
  check_ruby_version
  check_bundler
  echo "ok: docs Ruby and Bundler checks passed"
}

usage() {
  cat <<'EOF'
Usage: scripts/docs-site.sh <command>

Commands:
  check    verify Ruby and Bundler prerequisites for docs
  install  run bundle install in docs/
  build    run jekyll build in docs/
  build-root run jekyll build in docs/ with baseurl forced to /
  serve    run jekyll serve in docs/
EOF
}

configure_runners

case "${cmd}" in
  check)
    do_check
    ;;
  install)
    do_install
    ;;
  build)
    do_build
    ;;
  build-root)
    do_build_root
    ;;
  serve)
    do_serve
    ;;
  *)
    usage
    exit 1
    ;;
esac




