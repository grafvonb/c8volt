#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="${ROOT_DIR}/docs"

cd "${DOCS_DIR}"
bundle exec jekyll serve --livereload --host 127.0.0.1 --baseurl /c8volt
