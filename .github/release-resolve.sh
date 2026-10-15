#!/usr/bin/env sh
set -e
ROOT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
TAG="$1"
TARGET=$(sh "${ROOT_DIR}/metadata.sh" get-target-from-tag "$TAG")
sh "${ROOT_DIR}/.github/release-guard.sh" "$TARGET" "$TAG"
printf 'target=%s\n' "$TARGET" >>"$GITHUB_OUTPUT"
