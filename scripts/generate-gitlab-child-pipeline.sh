#!/usr/bin/env sh
set -eu
if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  printf 'Usage: %s <target> [package-version]
' "$0" >&2
  exit 1
fi
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
target="$1"
package_version="${2:-}"
if [ -z "$package_version" ]; then
  package_version="$(sh "${ROOT_DIR}/scripts/metadata.sh" get-official-version "$target")"
fi
package_name="${target}-${package_version}"
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
printf '      package_name: %s
' "$package_name"
