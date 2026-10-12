#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 3 ] && [ "$1" = --platform ]; then
  PLATFORM=$2
  NGINX_BIN=$3
elif [ "$#" -eq 1 ]; then
  PLATFORM=linux/amd64
  NGINX_BIN=$1
else
  printf 'Usage: %s [--platform linux/amd64] /path/to/linux/nginx\n' "$0" >&2
  exit 2
fi

if ! command -v docker >/dev/null 2>&1; then
  printf 'Required command is missing: docker\n' >&2
  exit 2
fi
if ! docker buildx version >/dev/null 2>&1; then
  printf 'Required Docker Buildx plugin is missing.\n' >&2
  exit 2
fi

if [ ! -x "$NGINX_BIN" ]; then
  printf 'NGINX binary is not executable: %s\n' "$NGINX_BIN" >&2
  exit 2
fi
BIN_DIR=$(cd "$(dirname "$NGINX_BIN")" && pwd -P)
NGINX_BIN="$BIN_DIR/$(basename "$NGINX_BIN")"
REPOSITORY_ROOT=$(cd "$(dirname "$0")/../.." && pwd -P)
METADATA_ENV=$(sh "$REPOSITORY_ROOT/metadata.sh" get-env nginx-upstream-healthcheck)
UBI_VERSION=
while IFS='=' read -r name value; do
  if [ "$name" = UBI10_MINIMAL_VERSION ]; then
    UBI_VERSION=$value
  fi
done <<<"$METADATA_ENV"
if [ -z "$UBI_VERSION" ]; then
  printf 'Target metadata does not define UBI10_MINIMAL_VERSION.\n' >&2
  exit 2
fi

RUN_ID="$(date -u '+%Y%m%dT%H%M%SZ')-$$"
IMAGE_NAME="static-builds-healthcheck-suite:$RUN_ID"
CONTAINER_NAME="static-builds-healthcheck-suite-$RUN_ID"
OWNER_LABEL="static-builds.healthcheck.run=$RUN_ID"
IMAGE_ATTEMPTED=0
CONTAINER_ATTEMPTED=0

# Removes only the uniquely named image and container created by this runner.
# @return The runner status that triggered cleanup.
# @return-type int
cleanup() {
  local runner_status=$?
  local cleanup_status
  local container_id
  local image_id
  local image_owner
  if [ "$#" -gt 0 ]; then
    runner_status=$1
  fi
  trap - EXIT INT TERM
  if [ "$CONTAINER_ATTEMPTED" -eq 1 ]; then
    if container_id=$(docker container ls --all --quiet \
      --filter "name=^/$CONTAINER_NAME$" --filter "label=$OWNER_LABEL"); then
      if [ -n "$container_id" ]; then
        if docker rm -f "$container_id"; then
          CONTAINER_ATTEMPTED=0
        else
          cleanup_status=$?
          if [ "$runner_status" -eq 0 ]; then
            runner_status=$cleanup_status
          fi
        fi
      fi
    else
      cleanup_status=$?
      if [ "$runner_status" -eq 0 ]; then
        runner_status=$cleanup_status
      fi
    fi
  fi
  if [ "$IMAGE_ATTEMPTED" -eq 1 ]; then
    if image_id=$(docker image ls --quiet "$IMAGE_NAME"); then
      if [ -n "$image_id" ]; then
        if image_owner=$(docker image inspect --format \
          '{{ index .Config.Labels "static-builds.healthcheck.run" }}' "$IMAGE_NAME"); then
          if [ "$image_owner" = "$RUN_ID" ]; then
            if docker image rm "$IMAGE_NAME"; then
              IMAGE_ATTEMPTED=0
            else
              cleanup_status=$?
              if [ "$runner_status" -eq 0 ]; then
                runner_status=$cleanup_status
              fi
            fi
          else
            printf 'Refusing to remove an image without this runner ownership label: %s\n' "$IMAGE_NAME" >&2
            if [ "$runner_status" -eq 0 ]; then
              runner_status=1
            fi
          fi
        else
          cleanup_status=$?
          if [ "$runner_status" -eq 0 ]; then
            runner_status=$cleanup_status
          fi
        fi
      fi
    else
      cleanup_status=$?
      if [ "$runner_status" -eq 0 ]; then
        runner_status=$cleanup_status
      fi
    fi
  fi
  exit "$runner_status"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

IMAGE_ATTEMPTED=1
docker buildx build --load \
  --platform "$PLATFORM" \
  --build-arg "FUNCTIONAL_BASE_IMAGE=redhat/ubi10-minimal:$UBI_VERSION" \
  --tag "$IMAGE_NAME" \
  --label "$OWNER_LABEL" \
  --file "$REPOSITORY_ROOT/proxy/tests/Dockerfile" \
  "$REPOSITORY_ROOT"
CONTAINER_ATTEMPTED=1
docker create \
  --platform "$PLATFORM" \
  --name "$CONTAINER_NAME" \
  --label "$OWNER_LABEL" \
  --cpus 4 \
  --memory 8g \
  --mount "type=bind,source=$NGINX_BIN,target=/input/nginx,readonly" \
  "$IMAGE_NAME" /input/nginx
docker start "$CONTAINER_NAME"
CONTAINER_STATUS=$(docker wait "$CONTAINER_NAME")
docker logs "$CONTAINER_NAME"
cleanup "$CONTAINER_STATUS"
