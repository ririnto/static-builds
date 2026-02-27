#!/usr/bin/env sh
set -eu

WORKDIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

usage() {
    echo "Usage: $0 <target> [buildx args...]"
    echo ""
    echo "Arguments:"
    echo "  target          Build target (e.g., nginx, haproxy, apache-httpd)"
    echo "  buildx args     Additional arguments passed to docker buildx build"
    echo ""
    echo "Environment variables:"
    echo "  BUILDKIT_PLATFORM     Target platform (default: linux/amd64)"
    echo "  BUILD_OUTPUT_DEST     Output destination (default: current directory)"
    echo ""
    echo "Examples:"
    echo "  $0 nginx"
    echo "  $0 haproxy --progress=plain"
    exit 1
}

if [ "$#" -lt 1 ]; then
    usage
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

# Configuration with defaults
BUILDKIT_PLATFORM="${BUILDKIT_PLATFORM:-linux/amd64}"

# Prepare cache directory
mkdir -p "${WORKDIR}/${TARGET}/.cache"

# Set output destination based on environment
if [ "${CI:-}" = "true" ] || [ "${GITHUB_ACTIONS:-}" = "true" ]; then
    BUILD_OUTPUT_DEST="."
else
    BUILD_OUTPUT_DEST="${BUILD_OUTPUT_DEST:-.}"
fi
# Build the docker buildx command
DOCKER_CMD="docker buildx build"
DOCKER_CMD="${DOCKER_CMD} --network=host"
DOCKER_CMD="${DOCKER_CMD} --platform=${BUILDKIT_PLATFORM}"
DOCKER_CMD="${DOCKER_CMD} --output=type=local,dest=${BUILD_OUTPUT_DEST}"

# Add cache configuration
DOCKER_CMD="${DOCKER_CMD} --cache-from=type=local,src=${WORKDIR}/${TARGET}/.cache"
DOCKER_CMD="${DOCKER_CMD} --cache-to=type=local,dest=${WORKDIR}/${TARGET}/.cache,mode=max"

# Read .env file and add build args
while IFS='=' read -r key value || [ -n "${key}" ]; do
    case "${key}" in
        ''|\#*) continue ;;
    esac
    DOCKER_CMD="${DOCKER_CMD} --build-arg=${key}=${value}"
done < "${WORKDIR}/${TARGET}/.env"

# Add any additional args passed by user
if [ "$#" -gt 0 ]; then
    for arg in "$@"; do
        DOCKER_CMD="${DOCKER_CMD} ${arg}"
    done
fi

# Add the context path
DOCKER_CMD="${DOCKER_CMD} ${WORKDIR}/${TARGET}"

# Execute the build
cd "${WORKDIR}/${TARGET}"
eval "${DOCKER_CMD}"

