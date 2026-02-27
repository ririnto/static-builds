#!/usr/bin/env sh
set -eu

WORKDIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

usage() {
    echo "Usage: $0 <target> [buildctl args...]"
    echo ""
    echo "Arguments:"
    echo "  target          Build target (e.g., nginx, haproxy, apache-httpd)"
    echo "  buildctl args   Additional arguments passed to buildctl"
    echo ""
    echo "Environment variables:"
    echo "  BUILDKIT_IMAGE        BuildKit image (default: moby/buildkit:rootless)"
    echo "  BUILDKIT_PLATFORM     Target platform (default: linux/amd64)"
    echo "  BUILDKIT_INSECURE     Set to '1' to enable relaxed security (seccomp/apparmor unconfined)"
    echo "  BUILD_OUTPUT_DEST     Output destination (default: ./out, CI: .)"
    echo ""
    echo "Examples:"
    echo "  $0 nginx"
    echo "  $0 haproxy --progress=plain"
    echo "  BUILDKIT_INSECURE=1 $0 nginx"
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
BUILDKIT_IMAGE="${BUILDKIT_IMAGE:-moby/buildkit:rootless}"
BUILDKIT_PLATFORM="${BUILDKIT_PLATFORM:-linux/amd64}"
BUILDKIT_INSECURE="${BUILDKIT_INSECURE:-0}"

# Prepare cache directory
# chmod 0777 is required for rootless BuildKit container to write to host cache
mkdir -p "${WORKDIR}/${TARGET}/.cache"
chmod -R 0777 "${WORKDIR}/${TARGET}/.cache"

# Set output destination based on environment
if [ "${CI:-}" = "true" ] || [ "${GITHUB_ACTIONS:-}" = "true" ]; then
    BUILD_OUTPUT_DEST="."
    # chmod 0777 is required for rootless BuildKit container to write to host directory
    chmod 0777 "${WORKDIR}/${TARGET}"
else
    BUILD_OUTPUT_DEST="${BUILD_OUTPUT_DEST:-./out}"
fi

# Build the buildctl command arguments
BUILDCTL_CMD="buildctl-daemonless.sh build"
BUILDCTL_CMD="${BUILDCTL_CMD} --frontend=dockerfile.v0"
BUILDCTL_CMD="${BUILDCTL_CMD} --opt=platform=${BUILDKIT_PLATFORM}"
BUILDCTL_CMD="${BUILDCTL_CMD} --local=context=."
BUILDCTL_CMD="${BUILDCTL_CMD} --local=dockerfile=."

# Read .env file and add build args
while IFS='=' read -r key value || [ -n "${key}" ]; do
    case "${key}" in
        ''|\#*) continue ;;
    esac
    BUILDCTL_CMD="${BUILDCTL_CMD} --opt=build-arg:${key}=${value}"
done < "${WORKDIR}/${TARGET}/.env"

BUILDCTL_CMD="${BUILDCTL_CMD} --output=type=local,dest=${BUILD_OUTPUT_DEST}"
BUILDCTL_CMD="${BUILDCTL_CMD} --import-cache type=local,src=./.cache"
BUILDCTL_CMD="${BUILDCTL_CMD} --export-cache type=local,dest=./.cache,mode=max"

# Add any additional buildctl args passed by user
if [ "$#" -gt 0 ]; then
    for arg in "$@"; do
        BUILDCTL_CMD="${BUILDCTL_CMD} ${arg}"
    done
fi

# Build the docker run command
DOCKER_CMD="docker run --rm -v \"${WORKDIR}:/workspace\" --workdir \"/workspace/${TARGET}\" -e \"BUILD_OUTPUT_DEST=${BUILD_OUTPUT_DEST}\""

# Add security settings based on BUILDKIT_INSECURE flag
if [ "${BUILDKIT_INSECURE}" = "1" ]; then
    DOCKER_CMD="${DOCKER_CMD} --security-opt seccomp=unconfined --security-opt apparmor=unconfined -e BUILDKITD_FLAGS=--oci-worker-no-process-sandbox"
fi

DOCKER_CMD="${DOCKER_CMD} ${BUILDKIT_IMAGE} /bin/sh -c \"mkdir -p './.cache' && exec ${BUILDCTL_CMD}\""

# Execute the build
eval "${DOCKER_CMD}"
