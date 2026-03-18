#!/usr/bin/env sh
set -eu
if [ "$#" -ne 2 ]; then
  printf 'Usage: %s <target> <package-name>\n' "$0" >&2
  exit 1
fi
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
target="$1"
package_name="${2}.tar.gz"
release_files="$(sh "${ROOT_DIR}/scripts/metadata.sh" get-release-files "${target}")"
if [ -z "${release_files}" ]; then
  printf 'Error: No release files configured for %s\n' "$target" >&2
  exit 1
fi
rm -f "${package_name}"
for release_file in ${release_files}; do
  release_path="${ROOT_DIR}/.out/${target}/${release_file}"
  if [ ! -f "${release_path}" ] && [ ! -d "${release_path}" ]; then
    printf 'Error: Release path not found at %s\n' "${release_path}" >&2
    exit 1
  fi
done
COPYFILE_DISABLE=1 tar -czvf "${package_name}" -C "${ROOT_DIR}/.out/${target}" ${release_files}
