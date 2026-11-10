#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
. "${ROOT_DIR}/build-env.sh"

if [ "$#" -lt 1 ]; then
  printf 'Usage: %s <target> [buildctl build arguments...]\n' "$0" >&2
  exit 1
fi
TARGET=$1
shift
case "$TARGET" in
  '' | *[!A-Za-z0-9._-]*)
    printf 'Error: Invalid target: %s\n' "$TARGET" >&2
    exit 1
    ;;
esac
if [ ! -d "${ROOT_DIR}/${TARGET}" ] || [ ! -f "${ROOT_DIR}/${TARGET}/Dockerfile" ]; then
  printf 'Error: Target Dockerfile not found: %s\n' "$TARGET" >&2
  exit 1
fi
METADATA_ENV=$(sh "${ROOT_DIR}/metadata.sh" get-env "$TARGET")
if [ "$(id -u)" -eq 0 ]; then
  printf 'Error: Native Linux builds require a non-root user.\n' >&2
  exit 1
fi
for prerequisite in rootlesskit setsid timeout; do
  command -v "$prerequisite"
done
if command -v runc; then
  OCI_RUNTIME=runc
elif command -v crun; then
  OCI_RUNTIME=crun
else
  printf 'Error: Install runc or crun for the rootless OCI worker.\n' >&2
  exit 1
fi
buildkitd_executable=$(go tool -n buildkitd)
if [ ! -f "$buildkitd_executable" ] || [ ! -x "$buildkitd_executable" ]; then
  printf 'Error: go tool -n buildkitd must resolve one executable file.\n' >&2
  exit 1
fi
go tool buildctl --version
BUILDKIT_PLATFORM=${BUILDKIT_PLATFORM:-linux/amd64}
BUILDKIT_NETWORK=${BUILDKIT_NETWORK:-default}
BUILDKIT_CACHE_BACKEND=${BUILDKIT_CACHE_BACKEND:-local}
BUILD_OUTPUT_DEST=${BUILD_OUTPUT_DEST:-${ROOT_DIR}/.out/${TARGET}}
CACHE_DIR=${CACHE_DIR:-${ROOT_DIR}/.cache/${TARGET}}
case "$BUILDKIT_CACHE_BACKEND" in
  local)
    CACHE_FROM="type=local,src=${CACHE_DIR}"
    CACHE_TO="type=local,dest=${CACHE_DIR},mode=max"
    ;;
  gha)
    CACHE_FROM="type=gha,scope=${TARGET}"
    CACHE_TO="type=gha,mode=max,scope=${TARGET}"
    ;;
  *)
    printf 'Error: Invalid BUILDKIT_CACHE_BACKEND: %s\n' "$BUILDKIT_CACHE_BACKEND" >&2
    exit 1
    ;;
esac
case "$BUILDKIT_NETWORK" in
  default) frontend_network=sandbox ;;
  none) frontend_network=none ;;
  host) frontend_network=host ;;
  *)
    printf 'Error: Invalid BUILDKIT_NETWORK: %s\n' "$BUILDKIT_NETWORK" >&2
    exit 1
    ;;
esac
while IFS='=' read -r key value || [ -n "$key" ]; do
  if normalize_build_env "$key" "$value"; then
    set -- --opt "build-arg:${key}=$(build_env_value)" "$@"
  fi
done <<EOF
${METADATA_ENV}
EOF
set -- build --frontend dockerfile.v0 \
  --local "context=${ROOT_DIR}" --local "dockerfile=${ROOT_DIR}" \
  --opt "filename=${TARGET}/Dockerfile" --opt "platform=${BUILDKIT_PLATFORM}" \
  --opt "force-network-mode=${frontend_network}" \
  --output "type=local,dest=${BUILD_OUTPUT_DEST}" \
  --import-cache "$CACHE_FROM" --export-cache "$CACHE_TO" "$@"
if [ "$BUILDKIT_NETWORK" = host ]; then
  set -- "$@" --allow network.host
fi

client_pid=
daemon_pid=
logger_pid=
temporary_directory=
namespace_pid=

# Terminate an invocation-owned session, escalate after five seconds, and reap its leader.
# @param $1 Session leader PID, or an empty string before launch.
# @return Zero after the session no longer contains live processes.
# @return-type Integer
stop_session() {
  if [ -n "$1" ]; then
    kill -TERM "-$1" 2>/dev/null || :
    kill -TERM "$1" 2>/dev/null || :
    remaining=5
    while { kill -0 "-$1" 2>/dev/null || kill -0 "$1" 2>/dev/null; } && [ "$remaining" -gt 0 ]; do
      sleep 1
      remaining=$((remaining - 1))
    done
    kill -KILL "-$1" 2>/dev/null || :
    kill -KILL "$1" 2>/dev/null || :
    wait "$1" 2>/dev/null || :
    if kill -0 "-$1" 2>/dev/null; then
      printf 'Error: Owned process group remains; retaining private state.\n' >&2
      return 1
    fi
  fi
}

# Stop owned clients, the PID-namespace daemon tree, and the log drainer before deleting state.
# @return Preserve the triggering status, or fail if owned processes prevent cleanup.
# @return-type Integer
cleanup() {
  status=$?
  trap '' HUP INT TERM
  trap - 0
  cleanup_status=0
  if [ -f "$temporary_directory/rootlesskit/child_pid" ]; then
    namespace_pid=$(cat "$temporary_directory/rootlesskit/child_pid")
    case "$namespace_pid" in
      '' | *[!0-9]*) namespace_pid= ;;
    esac
  fi
  for owned_pid in "$client_pid" "$daemon_pid" "$logger_pid"; do
    if ! stop_session "$owned_pid"; then
      cleanup_status=1
    fi
  done
  if [ -n "$namespace_pid" ]; then
    remaining=5
    while kill -0 "$namespace_pid" 2>/dev/null && [ "$remaining" -gt 0 ]; do
      sleep 1
      remaining=$((remaining - 1))
    done
    if kill -0 "$namespace_pid" 2>/dev/null; then
      printf 'Error: Owned PID namespace remains; retaining private state.\n' >&2
      cleanup_status=1
    fi
  fi
  if [ "$cleanup_status" -eq 0 ]; then
    if [ "$status" -ne 0 ] && [ -f "$temporary_directory/daemon.log" ]; then
      printf 'Rootless BuildKit failed (exit %s); final daemon diagnostics:\n' "$status" >&2
      tail -c 8192 "$temporary_directory/daemon.log" >&2
    fi
    if [ -n "$temporary_directory" ]; then
      rm -rf -- "$temporary_directory"
    fi
  else
    status=1
  fi
  exit "$status"
}
trap 'cleanup' EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
umask 077
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/static-buildkit.XXXXXXXX")
daemon_address="unix://${temporary_directory}/buildkit.sock"
mkfifo "$temporary_directory/diagnostics"
diagnostics_program="{ if (size < 65536) { line = substr(\$0, 1, 65536 - size); print line; size += length(line) + 1 } }"
setsid awk "$diagnostics_program" \
  "$temporary_directory/diagnostics" >"$temporary_directory/daemon.log" &
logger_pid=$!
set -- --addr="$daemon_address" "$@"
if [ "$BUILDKIT_CACHE_BACKEND" = local ]; then
  mkdir -p "$CACHE_DIR"
fi
mkdir -p "$BUILD_OUTPUT_DEST"
(
  set -- "$buildkitd_executable" --addr="$daemon_address" \
    --root="$temporary_directory/state" --containerd-worker=false \
    --oci-worker=true --oci-worker-binary="$OCI_RUNTIME"
  if [ "$BUILDKIT_NETWORK" = host ]; then
    set -- "$@" --allow-insecure-entitlement network.host
  fi
  exec setsid rootlesskit --pidns --reaper=true \
    --state-dir="$temporary_directory/rootlesskit" "$@"
) >"$temporary_directory/diagnostics" 2>&1 &
daemon_pid=$!
deadline=$(($(date +%s) + 60))
attempt=0
ready=0
while [ "$ready" -eq 0 ] && [ "$attempt" -lt 30 ] && [ "$(date +%s)" -lt "$deadline" ] && kill -0 "$daemon_pid"; do
  setsid timeout --kill-after=1 1 go tool buildctl --addr="$daemon_address" debug workers --format '{{range .}}{{.ID}}{{end}}' >"$temporary_directory/workers" 2>&1 &
  client_pid=$!
  if wait "$client_pid" && grep -q '[^[:space:]]' "$temporary_directory/workers"; then
    ready=1
  else
    sleep 1
  fi
  client_pid=
  attempt=$((attempt + 1))
done
if [ "$ready" -ne 1 ]; then
  printf 'Error: Rootless BuildKit worker readiness deadline expired.\n' >&2
  exit 1
fi
cd "$ROOT_DIR"
setsid go tool buildctl "$@" &
client_pid=$!
if wait "$client_pid"; then
  cleanup
else
  cleanup
fi
