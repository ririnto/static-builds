#!/usr/bin/env sh
set -eu
if [ "$#" -ne 1 ]; then
  printf 'Usage: %s <tag>
' "$0" >&2
  exit 1
fi
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tag="$1"
target="$(sh "${ROOT_DIR}/scripts/metadata.sh" get-target-by-tag "$tag")"
printf '%s
' 'include:'
printf '%s
' '  - component: $CI_SERVER_FQDN/$CI_PROJECT_PATH/static-release@$CI_COMMIT_SHA'
printf '%s
' '    inputs:'
printf '%s
' '      stage: release'
printf '      target: %s
' "$target"
printf '      tag: %s
' "$tag"
printf '%s
' '      upload_release: true'
