#!/usr/bin/env sh
set -e
TAG="$1"
PACKAGE_NAME="${TAG}.tar.gz"
if [ ! -f "${PACKAGE_NAME}" ]; then
  echo "Error: ${PACKAGE_NAME} not found" >&2
  exit 1
fi
gh release view "${TAG}" >/dev/null 2>&1 ||
  gh release create "${TAG}" --title "${TAG}" --notes "Automated static binary release from tag ${TAG}."
gh release upload "${TAG}" "${PACKAGE_NAME}" --clobber
