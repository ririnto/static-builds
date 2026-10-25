#!/usr/bin/env sh
set -e
ROOT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
. "${ROOT_DIR}/build-env.sh"
METADATA_SCRIPT="${ROOT_DIR}/metadata.sh"
if [ "$#" -lt 1 ]; then
  printf 'Usage: %s <target>\n' "$0" >&2
  exit 1
fi
TARGET="$1"
case "${TARGET}" in
  *[!A-Za-z0-9._-]*)
    printf 'Error: Invalid target %s\n' "${TARGET}" >&2
    exit 1
    ;;
esac
if [ ! -f "${ROOT_DIR}/${TARGET}/Dockerfile" ]; then
  printf 'Error: Dockerfile not found in %s\n' "${TARGET}" >&2
  exit 1
fi
if [ "$(id -u)" -eq 0 ]; then
  echo 'Error: Rootless BuildKit must run as a non-root user' >&2
  exit 1
fi
METADATA_ENV=$(sh "${METADATA_SCRIPT}" get-env "${TARGET}")
if ! command -v buildctl-daemonless.sh; then
  echo 'Error: buildctl-daemonless.sh not found in PATH' >&2
  exit 1
fi
BUILDKIT_PLATFORM="${BUILDKIT_PLATFORM:-linux/amd64}"
BUILDKIT_CACHE_BACKEND="${BUILDKIT_CACHE_BACKEND:-local}"
BUILDKIT_NETWORK="${BUILDKIT_NETWORK:-default}"
BUILD_OUTPUT_DEST="${BUILD_OUTPUT_DEST:-${ROOT_DIR}/.out/${TARGET}}"
case "${BUILD_OUTPUT_DEST}" in
  "${ROOT_DIR}/.out/${TARGET}" | "${ROOT_DIR}/.out/${TARGET}/"* | ".out/${TARGET}" | ".out/${TARGET}/"* | "./.out/${TARGET}" | "./.out/${TARGET}/"*)
    case "/${BUILD_OUTPUT_DEST}/" in
      */../*)
        echo "Error: Build output must be under .out/${TARGET}" >&2
        exit 1
        ;;
    esac
    ;;
  *)
    echo "Error: Build output must be under .out/${TARGET}" >&2
    exit 1
    ;;
esac
mkdir -p "${BUILD_OUTPUT_DEST}"
case "${BUILDKIT_CACHE_BACKEND}" in
  local)
    mkdir -p "${ROOT_DIR}/.cache/${TARGET}"
    CACHE_FROM="type=local,src=${ROOT_DIR}/.cache/${TARGET}"
    CACHE_TO="type=local,dest=${ROOT_DIR}/.cache/${TARGET},mode=max"
    ;;
  *)
    printf 'Error: Invalid BUILDKIT_CACHE_BACKEND %s\n' "${BUILDKIT_CACHE_BACKEND}" >&2
    echo 'Allowed values: local' >&2
    exit 1
    ;;
esac
case "${BUILDKIT_NETWORK}" in
  host | default) ;;
  *)
    echo 'Error: Rootless BuildKit only supports host/default network semantics in this repository' >&2
    exit 1
    ;;
esac
set -- buildctl-daemonless.sh build \
  --frontend dockerfile.v0 \
  --local "context=${ROOT_DIR}" \
  --local "dockerfile=${ROOT_DIR}" \
  --opt "filename=${TARGET}/Dockerfile" \
  --opt "platform=${BUILDKIT_PLATFORM}" \
  --output "type=local,dest=${BUILD_OUTPUT_DEST}" \
  --import-cache "${CACHE_FROM}" \
  --export-cache "${CACHE_TO}"
while IFS='=' read -r key value || [ -n "${key}" ]; do
  if normalize_build_env "${key}" "${value}"; then
    set -- "$@" --opt "build-arg:${key}=$(build_env_value)"
  fi
done <<EOF2
${METADATA_ENV}
EOF2
"$@"
