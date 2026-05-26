#!/usr/bin/env bash

set -u

IT_TOTAL=0
IT_FAILS=0
IT_EXPECTED_FAIL_MISSES=0
IT_SKIPS=0
IT_CURRENT_GROUP=""

it_timestamp() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

it_slug() {
  printf "%s" "$1" | tr -c "A-Za-z0-9_.-" "_"
}

it_shell_quote() {
  printf "%q" "$1"
}

it_command_line() {
  local out=""
  local arg
  for arg in "$@"; do
    if [ -z "$out" ]; then
      out="$(it_shell_quote "$arg")"
    else
      out="$out $(it_shell_quote "$arg")"
    fi
  done
  printf "%s" "$out"
}

it_init_report() {
  mkdir -p "$IT_WORKDIR/logs" "$IT_WORKDIR/data" "$IT_WORKDIR/reports"
  : > "$IT_PROGRESS"
  cat > "$IT_REPORT" <<EOF
# c8volt ${IT_SUITE_NAME} Release Integration Report

- started: $(it_timestamp)
- suite: ${IT_SUITE_NAME}
- target Camunda minor: ${IT_TARGET_VERSION}
- fixture prefix: ${IT_FIXTURE_PREFIX}
- config: ${IT_CONFIG}
- profile: ${IT_PROFILE:-<config default>}
- workdir: ${IT_WORKDIR}

EOF
}

it_report_section() {
  local section="$1"
  if [ "$IT_CURRENT_GROUP" != "$section" ]; then
    IT_CURRENT_GROUP="$section"
    {
      printf "\n## %s\n\n" "$section"
      printf "| Status | Case | Impact | Evidence |\n"
      printf "| --- | --- | --- | --- |\n"
    } >> "$IT_REPORT"
  fi
}

it_append_progress() {
  local status="$1"
  local group="$2"
  local case_id="$3"
  local impact="$4"
  local exit_code="$5"
  local note="$6"
  printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\n" "$(it_timestamp)" "$status" "$group" "$case_id" "$impact" "$exit_code" "$note" >> "$IT_PROGRESS"
}

it_append_case_report() {
  local status="$1"
  local case_id="$2"
  local impact="$3"
  local evidence="$4"
  printf -- '| %s | `%s` | %s | %s |\n' "$status" "$case_id" "$impact" "$evidence" >> "$IT_REPORT"
}

it_run_case() {
  local group="$1"
  local case_id="$2"
  local impact="$3"
  local expected="$4"
  shift 4

  IT_TOTAL=$((IT_TOTAL + 1))
  it_report_section "$group"

  local slug stdout stderr command_line exit_code status evidence
  slug="$(it_slug "${group}_${case_id}")"
  stdout="$IT_WORKDIR/logs/${slug}.stdout"
  stderr="$IT_WORKDIR/logs/${slug}.stderr"
  command_line="$(it_command_line "$@")"

  {
    printf "$ %s\n\n" "$command_line"
  } > "$stdout"
  {
    printf "$ %s\n\n" "$command_line"
  } > "$stderr"

  "$@" >> "$stdout" 2>> "$stderr"
  exit_code=$?

  case "$expected" in
    pass)
      if [ "$exit_code" -eq 0 ]; then
        status="PASS"
      else
        status="FAIL"
        IT_FAILS=$((IT_FAILS + 1))
      fi
      ;;
    fail)
      if [ "$exit_code" -ne 0 ]; then
        status="EXPECTED_FAIL"
      else
        status="UNEXPECTED_PASS"
        IT_EXPECTED_FAIL_MISSES=$((IT_EXPECTED_FAIL_MISSES + 1))
      fi
      ;;
    optional)
      if [ "$exit_code" -eq 0 ]; then
        status="PASS"
      else
        status="WARN"
      fi
      ;;
    *)
      status="FAIL"
      IT_FAILS=$((IT_FAILS + 1))
      ;;
  esac

  evidence="stdout: \`$stdout\`, stderr: \`$stderr\`, exit: \`$exit_code\`"
  it_append_case_report "$status" "$case_id" "$impact" "$evidence"
  it_append_progress "$status" "$group" "$case_id" "$impact" "$exit_code" "$command_line"
  printf "%-15s %s/%s\n" "$status" "$group" "$case_id"
  return "$exit_code"
}

it_run_capture() {
  local group="$1"
  local case_id="$2"
  local impact="$3"
  local expected="$4"
  local capture_file="$5"
  shift 5

  it_run_case "$group" "$case_id" "$impact" "$expected" "$@"
  local exit_code=$?
  local slug stdout
  slug="$(it_slug "${group}_${case_id}")"
  stdout="$IT_WORKDIR/logs/${slug}.stdout"
  if [ -f "$stdout" ]; then
    awk 'NR > 2 { print }' "$stdout" > "$capture_file"
  fi
  return "$exit_code"
}

it_skip_case() {
  local group="$1"
  local case_id="$2"
  local impact="$3"
  local reason="$4"
  IT_TOTAL=$((IT_TOTAL + 1))
  IT_SKIPS=$((IT_SKIPS + 1))
  it_report_section "$group"
  it_append_case_report "SKIP" "$case_id" "$impact" "$reason"
  it_append_progress "SKIP" "$group" "$case_id" "$impact" "n/a" "$reason"
  printf "%-15s %s/%s (%s)\n" "SKIP" "$group" "$case_id" "$reason"
}

it_require_output_contains() {
  local group="$1"
  local case_id="$2"
  local pattern="$3"
  local slug stdout stderr
  slug="$(it_slug "${group}_${case_id}")"
  stdout="$IT_WORKDIR/logs/${slug}.stdout"
  stderr="$IT_WORKDIR/logs/${slug}.stderr"
  if grep -E "$pattern" "$stdout" "$stderr" >/dev/null 2>&1; then
    return 0
  fi
  {
    printf "\n## Gate Failure\n\n"
    printf -- 'Required pattern `%s` was not found in `%s` or `%s`.\n' "$pattern" "$stdout" "$stderr"
  } >> "$IT_REPORT"
  return 1
}

it_first_key() {
  local file="$1"
  awk '/^[0-9]+$/ { print; exit }' "$file"
}

it_key_args_from_file() {
  local file="$1"
  local limit="$2"
  awk '/^[0-9]+$/ { print "--key"; print; count++; if (count >= limit) exit }' "$file"
}

it_write_summary() {
  cat > "$IT_WORKDIR/summary.env" <<EOF
IT_SUITE_NAME=$(printf "%q" "$IT_SUITE_NAME")
IT_TARGET_VERSION=$(printf "%q" "$IT_TARGET_VERSION")
IT_FIXTURE_PREFIX=$(printf "%q" "$IT_FIXTURE_PREFIX")
IT_CONFIG=$(printf "%q" "$IT_CONFIG")
IT_PROFILE=$(printf "%q" "$IT_PROFILE")
IT_BIN=$(printf "%q" "$IT_BIN")
IT_WORKDIR=$(printf "%q" "$IT_WORKDIR")
IT_RUN_ID=$(printf "%q" "$IT_RUN_ID")
IT_FULL_FORCE=$IT_FULL_FORCE
IT_TOTAL=$IT_TOTAL
IT_FAILS=$IT_FAILS
IT_EXPECTED_FAIL_MISSES=$IT_EXPECTED_FAIL_MISSES
IT_SKIPS=$IT_SKIPS
EOF
}

it_finish_report() {
  {
    printf "\n## Summary\n\n"
    printf -- "- finished: %s\n" "$(it_timestamp)"
    printf -- "- total cases: %s\n" "$IT_TOTAL"
    printf -- "- failures: %s\n" "$IT_FAILS"
    printf -- "- expected-failure misses: %s\n" "$IT_EXPECTED_FAIL_MISSES"
    printf -- "- skipped: %s\n" "$IT_SKIPS"
    printf -- '- progress: `%s`\n' "$IT_PROGRESS"
    printf -- '- summary env: `%s/summary.env`\n' "$IT_WORKDIR"
    printf "\n## UX Review Notes\n\n"
    printf -- 'Review command evidence with `integration/assets/ux-review-checklist.md`.\n'
    printf "Record findings here or in a follow-up issue with command log paths.\n"
  } >> "$IT_REPORT"
  it_write_summary
}
