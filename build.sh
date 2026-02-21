#!/usr/bin/env sh
set -eu

WORKDIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <target> [buildctl args...]"
    echo "Example:"
    echo "  $0 nginx"
    echo "  $0 haproxy --progress=plain"
    exit 1
fi

TARGET="$1"
shift

case "${TARGET}" in
    *[!A-Za-z0-9._-]*)
        echo "Error: Invalid target '${TARGET}'"
        echo "Allowed characters: letters, numbers, dot, underscore, hyphen"
        exit 1
        ;;
esac

if [ ! -d "${WORKDIR}/${TARGET}" ]; then
    echo "Error: Target directory '${TARGET}' not found"
    exit 1
fi

if [ ! -f "${WORKDIR}/${TARGET}/.env" ]; then
    echo "Error: .env file not found in '${TARGET}'"
    exit 1
fi

if [ ! -f "${WORKDIR}/${TARGET}/Dockerfile" ]; then
    echo "Error: Dockerfile not found in '${TARGET}'"
    exit 1
fi

mkdir -p "${WORKDIR}/${TARGET}/.cache"
chmod -R 0777 "${WORKDIR}/${TARGET}/.cache"

if [ "${CI:-}" = "true" ] || [ "${GITHUB_ACTIONS:-}" = "true" ]; then
    BUILD_OUTPUT_DEST="."
else
    BUILD_OUTPUT_DEST="${BUILD_OUTPUT_DEST:-./out}"
fi

if [ "$#" -gt 0 ]; then
    echo "Error: Additional buildctl args are not supported in compose mode"
    exit 1
fi

TARGET="${TARGET}" BUILD_OUTPUT_DEST="${BUILD_OUTPUT_DEST}" exec docker compose -f "${WORKDIR}/.github/docker-compose.yaml" --project-directory "${WORKDIR}" run --rm build
