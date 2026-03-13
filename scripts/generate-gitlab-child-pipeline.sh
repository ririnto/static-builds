#!/usr/bin/env sh
set -eu
if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  printf 'Usage: %s <target> [package-version]\n' "$0" >&2
  exit 1
fi
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
target="$1"
package_version="${2:-}"
sh "${ROOT_DIR}/scripts/metadata.sh" get-tag-prefix "$target" >/dev/null
if [ -z "$package_version" ]; then
  package_version="$(sh "${ROOT_DIR}/scripts/metadata.sh" get-official-version "$target")"
fi
package_name="${target}-${package_version}"
printf '%s\n' 'include:'
printf '%s\n' '  - component: $CI_SERVER_FQDN/$CI_PROJECT_PATH/static-release@$CI_COMMIT_SHA'
printf '%s\n' '    inputs:'
printf '%s\n' '      stage: release'
printf '      target: %s\n' "$target"
printf '      package_name: %s\n' "$package_name"
