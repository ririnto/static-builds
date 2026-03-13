#!/usr/bin/env sh
set -eu

# Print an error message to stderr and exit.
#
# :param str message: Error message body without trailing newline.
# :returns: Does not return.
# :rtype: None
fail() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

# Print usage information and exit.
#
# :returns: Does not return.
# :rtype: None
usage() {
  printf 'Usage: %s <helper command...> [--commands command1,command2]\n' "$0" >&2
  exit 1
}

# Resolve a path to an absolute path when the file exists.
#
# :param str path: Path to resolve.
# :returns: Absolute path printed to stdout when the file exists.
# :rtype: str
resolve_path() {
  path="$1"
  if [ ! -f "$path" ]; then
    return 1
  fi
  dir_path=$(dirname -- "$path")
  base_name=$(basename -- "$path")
  abs_dir=$(CDPATH= cd -- "$dir_path" && pwd)
  printf '%s/%s\n' "$abs_dir" "$base_name"
}

# Return success when a command should be checked.
#
# :param str command_name: Command name to test.
# :returns: 0 when enabled, 1 otherwise.
# :rtype: int
command_enabled() {
  command_name="$1"
  if [ -z "$COMMAND_FILTER" ]; then
    return 0
  fi
  case ",${COMMAND_FILTER}," in
  *,${command_name},*) return 0 ;;
  esac
  return 1
}

# Write expected text into a file.
#
# :param str destination: File path to write.
# :param str content: Exact content to write.
# :returns: 0 on success.
# :rtype: int
write_expected() {
  destination="$1"
  content="$2"
  printf '%s' "$content" >"$destination"
}

# Print a passing check line.
#
# :param str label: Check label.
# :returns: 0 on success.
# :rtype: int
pass_check() {
  label="$1"
  printf 'PASS %s\n' "$label"
}

# Print a failing check line and terminate.
#
# :param str label: Check label.
# :param str message: Failure description.
# :returns: Does not return.
# :rtype: None
fail_check() {
  label="$1"
  message="$2"
  printf 'FAIL %s: %s\n' "$label" "$message" >&2
  exit 1
}

# Execute the helper command and capture stdout, stderr, and exit code.
#
# :param str label: Case label for output files.
# :param str metadata_path: Fixture metadata path, or empty for live metadata.
# :param str ...: Helper command arguments to run after the script path.
# :returns: 0 on success.
# :rtype: int
run_helper() {
  label="$1"
  metadata_path="$2"
  shift 2
  stdout_path="${QA_DIR}/${label}.stdout"
  stderr_path="${QA_DIR}/${label}.stderr"
  sandbox_dir=''
  helper_path="$HELPER_SCRIPT"
  if [ -n "$metadata_path" ]; then
    sandbox_dir=$(mktemp -d "${QA_DIR}/sandbox.${label}.XXXXXX")
    mkdir -p "${sandbox_dir}/scripts"
    cp "$metadata_path" "${sandbox_dir}/metadata.json"
    cp "$HELPER_SCRIPT" "${sandbox_dir}/scripts/${HELPER_BASENAME}"
    helper_path="${sandbox_dir}/scripts/${HELPER_BASENAME}"
  fi

  set +e
  if [ -n "$HELPER_PREFIX" ]; then
    sh -c 'set -- "$@"; exec "$@"' sh ${HELPER_PREFIX} "$helper_path" "$@" >"$stdout_path" 2>"$stderr_path"
  else
    "$helper_path" "$@" >"$stdout_path" 2>"$stderr_path"
  fi
  LAST_STATUS=$?
  set -e

  if [ -n "$sandbox_dir" ]; then
    rm -rf "$sandbox_dir"
  fi
}

# Write a temporary metadata fixture for contract checks.
#
# :param str fixture_name: Fixture identifier.
# :param str destination: Metadata file path to write.
# :returns: 0 on success.
# :rtype: int
write_fixture_metadata() {
  fixture_name="$1"
  destination="$2"
  case "$fixture_name" in
  unknown-placeholder)
    cat >"$destination" <<'EOF'
{
  "nginx": {
    "env": {
      "NGINX_VERSION": "1.28.2"
    },
    "downloads": [
      {
        "url": "https://example.invalid/nginx-{MISSING_VERSION}.tar.gz",
        "name": "nginx-1.28.2.tar.gz"
      }
    ]
  }
}
EOF
    ;;
  missing-version-env-var)
    cat >"$destination" <<'EOF'
{
  "nginx": {
    "env": {
      "NGINX_VERSION": "1.28.2"
    },
    "tag_prefix": "nginx"
  }
}
EOF
    ;;
  no-downloads)
    cat >"$destination" <<'EOF'
{
  "nginx": {
    "env": {
      "NGINX_VERSION": "1.28.2"
    },
    "tag_prefix": "nginx"
  }
}
EOF
    ;;
  *)
    fail "unknown fixture '${fixture_name}'"
    ;;
  esac
}

# Execute the helper with temporary fixture metadata.
#
# :param str label: Case label for output files.
# :param str fixture_name: Fixture identifier.
# :param str ...: Helper command arguments to run after the script path.
# :returns: 0 on success.
# :rtype: int
run_helper_with_fixture() {
  label="$1"
  fixture_name="$2"
  shift 2
  fixture_dir=$(mktemp -d "${QA_DIR}/fixture.${label}.XXXXXX")
  fixture_path="${fixture_dir}/metadata.json"
  write_fixture_metadata "$fixture_name" "$fixture_path"
  run_helper "$label" "$fixture_path" "$@"
  rm -rf "$fixture_dir"
}

# Verify the captured stdout, stderr, and exit code.
#
# :param str label: Case label.
# :param int expected_status: Expected exit code.
# :param str expected_stdout: Exact stdout contents.
# :param str expected_stderr: Exact stderr contents.
# :returns: 0 on success.
# :rtype: int
assert_capture() {
  label="$1"
  expected_status="$2"
  expected_stdout="$3"
  expected_stderr="$4"
  expected_stdout_path="${QA_DIR}/${label}.expected.stdout"
  expected_stderr_path="${QA_DIR}/${label}.expected.stderr"
  write_expected "$expected_stdout_path" "$expected_stdout"
  write_expected "$expected_stderr_path" "$expected_stderr"

  if [ "$LAST_STATUS" -ne "$expected_status" ]; then
    fail_check "$label" "expected exit ${expected_status}, got ${LAST_STATUS}"
  fi
  if ! cmp -s "$expected_stdout_path" "${QA_DIR}/${label}.stdout"; then
    diff -u "$expected_stdout_path" "${QA_DIR}/${label}.stdout" >&2 || true
    fail_check "$label" "stdout mismatch"
  fi
  if ! cmp -s "$expected_stderr_path" "${QA_DIR}/${label}.stderr"; then
    diff -u "$expected_stderr_path" "${QA_DIR}/${label}.stderr" >&2 || true
    fail_check "$label" "stderr mismatch"
  fi
  pass_check "$label"
}

# Verify live-metadata commands against exact contract outputs.
#
# :returns: 0 on success.
# :rtype: int
check_live_contract() {
  if command_enabled 'list-targets'; then
    run_helper 'list-targets' '' 'list-targets'
    assert_capture 'list-targets' 0 'nginx haproxy apache-httpd coredns dnsmasq vector monit
' ''
  fi

  if command_enabled 'get-tag-prefix'; then
    run_helper 'get-tag-prefix' '' 'get-tag-prefix' 'apache-httpd'
    assert_capture 'get-tag-prefix' 0 'httpd
' ''

    run_helper 'get-tag-prefix-unknown-target' '' 'get-tag-prefix' 'no-such-target'
    assert_capture 'get-tag-prefix-unknown-target' 1 '' "Error: unknown target 'no-such-target'
"
  fi

  if command_enabled 'get-official-version'; then
    run_helper 'get-official-version' '' 'get-official-version' 'apache-httpd'
    assert_capture 'get-official-version' 0 '2.4.66
' ''
  fi

  if command_enabled 'get-env'; then
    run_helper 'get-env' '' 'get-env' 'nginx'
    assert_capture 'get-env' 0 'ALPINE_VERSION=3.23
NGINX_LUA_RESTY_CORE_VERSION=0.1.32R1
NGINX_LUA_MODULE_VERSION=0.10.29
NGINX_LUA_RESTY_UPSTREAM_HEALTHCHECK_MODULE_VERSION=0.08
NGINX_LUA_UPSTREAM_MODULE_VERSION=0.07
NGINX_VERSION=1.28.2
NGINX_VTS_MODULE_VERSION=0.2.5
UBI9_MICRO_VERSION=9.5
' ''
  fi

  if command_enabled 'get-release-files'; then
    run_helper 'get-release-files' '' 'get-release-files' 'nginx'
    assert_capture 'get-release-files' 0 'sbin/nginx
lualib/resty/core.lua
lualib/resty/core
lualib/resty/upstream
' ''
  fi

  if command_enabled 'get-downloads'; then
    run_helper 'get-downloads' '' 'get-downloads' 'nginx'
    assert_capture 'get-downloads' 0 'https://nginx.org/download/nginx-1.28.2.tar.gz	nginx-1.28.2.tar.gz
https://github.com/vozlt/nginx-module-vts/archive/refs/tags/v0.2.5.tar.gz	nginx-module-vts-0.2.5.tar.gz
https://github.com/openresty/lua-nginx-module/archive/refs/tags/v0.10.29.tar.gz	lua-nginx-module-0.10.29.tar.gz
https://github.com/openresty/lua-resty-core/archive/refs/tags/v0.1.32R1.tar.gz	lua-resty-core-0.1.32R1.tar.gz
https://github.com/openresty/lua-upstream-nginx-module/archive/refs/tags/v0.07.tar.gz	lua-upstream-nginx-module-0.07.tar.gz
https://github.com/openresty/lua-resty-upstream-healthcheck/archive/refs/tags/v0.08.tar.gz	lua-resty-upstream-healthcheck-0.08.tar.gz
' ''
  fi

  if [ -z "$COMMAND_FILTER" ]; then
    run_helper 'usage' ''
    assert_capture 'usage' 1 '' "Usage: ${HELPER_BASENAME} <command> [args...]

Commands:
  list-targets
  get-env <target>
  get-official-version <target>
  get-tag-prefix <target>
  get-release-files <target>
  get-downloads <target>
"
  fi
}

# Verify fixture-based success and failure cases.
#
# :returns: 0 on success.
# :rtype: int
check_fixture_contract() {
  if command_enabled 'get-official-version'; then
    run_helper_with_fixture 'fixture-missing-version-env-var' 'missing-version-env-var' 'get-official-version' 'nginx'
    assert_capture 'fixture-missing-version-env-var' 1 '' "Error: target 'nginx' is missing version_env_var
"
  fi

  if command_enabled 'get-downloads'; then
    run_helper_with_fixture 'fixture-unknown-placeholder' 'unknown-placeholder' 'get-downloads' 'nginx'
    assert_capture 'fixture-unknown-placeholder' 1 '' "Error: target 'nginx' references unknown placeholder 'MISSING_VERSION' in downloads[1].url
"

    run_helper_with_fixture 'fixture-no-downloads' 'no-downloads' 'get-downloads' 'nginx'
    assert_capture 'fixture-no-downloads' 0 '' ''
  fi
}

# Parse CLI arguments into helper command and optional filters.
#
# :param str ...: Script arguments.
# :returns: 0 on success.
# :rtype: int
parse_args() {
  COMMAND_FILTER=''
  helper_count=0
  HELPER_PREFIX=''
  HELPER_SCRIPT=''
  while [ $# -gt 0 ]; do
    case "$1" in
    --commands)
      shift
      [ $# -gt 0 ] || fail 'missing value for --commands'
      COMMAND_FILTER="$1"
      shift
      [ $# -eq 0 ] || fail 'unexpected arguments after --commands'
      ;;
    *)
      helper_count=$((helper_count + 1))
      if [ $helper_count -eq 1 ]; then
        HELPER_SCRIPT="$1"
      else
        HELPER_PREFIX="${HELPER_PREFIX}${HELPER_PREFIX:+ }${HELPER_SCRIPT}"
        HELPER_SCRIPT="$1"
      fi
      shift
      ;;
    esac
  done
  [ $helper_count -gt 0 ] || usage
}

# Validate the helper command before running contract checks.
#
# :returns: 0 on success.
# :rtype: int
validate_helper() {
  resolved_path=$(resolve_path "$HELPER_SCRIPT") || fail "helper execution failure: script not found: ${HELPER_SCRIPT}"
  HELPER_SCRIPT="$resolved_path"
  HELPER_BASENAME=$(basename -- "$HELPER_SCRIPT")
}

# Run the complete contract harness.
#
# :param str ...: Script arguments.
# :returns: 0 on success.
# :rtype: int
main() {
  ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
  QA_DIR="${ROOT_DIR}/.tmp/qa/metadata-contract"
  mkdir -p "$QA_DIR"
  parse_args "$@"
  validate_helper
  check_live_contract
  check_fixture_contract
}

main "$@"
