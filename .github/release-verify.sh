#!/usr/bin/env sh
set -e
ROOT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
TARGET="$1"
RELEASE_FILES="$(sh "${ROOT_DIR}/metadata.sh" get-release-files "$TARGET")"
if [ -z "${RELEASE_FILES}" ]; then
  echo "Error: No release files for ${TARGET}" >&2
  exit 1
fi
for f in ${RELEASE_FILES}; do
  if [ ! -f ".out/${TARGET}/${f}" ] && [ ! -d ".out/${TARGET}/${f}" ]; then
    echo "Error: ${f} not found in .out/${TARGET}" >&2
    exit 1
  fi
done
