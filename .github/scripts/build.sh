#!/usr/bin/env sh
set -eu

BUILD_ARGS=""
while IFS='=' read -r key value || [ -n "$key" ]; do
    case "$key" in
        ''|\#*) continue ;;
    esac
    BUILD_ARGS="${BUILD_ARGS} --opt=build-arg:${key}=${value}"
done < .env

CACHE_DIR="./.cache"

CACHE_ARGS=""
if [ -z "${BUILDCTL_NO_CACHE:-}" ] && [ -d "${CACHE_DIR}" ]; then
    CACHE_ARGS="--import-cache type=local,src=${CACHE_DIR}"
fi
CACHE_ARGS="${CACHE_ARGS} --export-cache type=local,dest=${CACHE_DIR},mode=max"

PROGRESS_ARGS=""
if [ -n "${BUILDCTL_PROGRESS:-}" ]; then
    PROGRESS_ARGS="--progress=${BUILDCTL_PROGRESS}"
fi

exec /bin/sh -c "buildctl-daemonless.sh build \
    ${PROGRESS_ARGS} \
    --frontend=dockerfile.v0 \
    --opt platform=linux/amd64 \
    --local=context=. \
    --local=dockerfile=. \
    ${BUILD_ARGS} \
    ${CACHE_ARGS} \
    --output=type=local,dest=."
