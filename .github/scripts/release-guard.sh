#!/usr/bin/env sh
set -eu
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)

# Print error message to stderr.
# :param str message: Error message to print.
# :returns: Nothing (prints to stderr).
# :rtype: None
err() {
  printf '%s\n' "$*" >&2
}

# Parse version and revision from tag using last-dot split.
# :param str tag: Release tag in format <target>-<version>.<revision>.
# :returns: Version string (without revision) printed to stdout.
# :rtype: str
parse_tag() {
  tag="$1"
  version_revision="${tag#*-}"
  version="${version_revision%.*}"
  revision="${version_revision##*.}"
  if ! [ "$revision" -eq "$revision" ] 2>/dev/null; then
    err "Error: Revision '$revision' is not numeric"
    return 1
  fi
  printf '%s' "$version"
}

# Get official version from centralized metadata.
# :param str target: Build target name.
# :returns: Official version string printed to stdout.
# :rtype: str
get_official_version() {
  target="$1"
  metadata_script="${ROOT_DIR}/scripts/metadata.sh"
  version=$(sh "$metadata_script" get-official-version "$target")
  if [ -z "$version" ]; then
    err "Error: official version not found for $target"
    return 1
  fi
  printf '%s' "$version"
}

# Main entry point for tag validation.
# :param str target: Target name (e.g., nginx, haproxy, apache-httpd).
# :param str tag: Release tag (e.g., nginx-1.28.2.18).
# :returns: Exit code 0 on success, 1 on validation failure.
# :rtype: int
main() {
  if [ $# -ne 2 ]; then
    err "Usage: $0 <target> <tag>"
    err "Example: $0 nginx nginx-1.28.2.18"
    exit 1
  fi
  target="$1"
  tag="$2"
  tag_prefix=$(sh "${ROOT_DIR}/scripts/metadata.sh" get-tag-prefix "$target") || exit 1
  if ! printf '%s' "$tag" | grep -qE "^${tag_prefix}-"; then
    err "Error: Invalid tag format '$tag'"
    err "Expected format: ${tag_prefix}-<version>.<revision>"
    err "Example: ${tag_prefix}-1.28.2.18"
    exit 1
  fi
  tag_version=$(parse_tag "$tag") || exit 1
  official_version=$(get_official_version "$target") || exit 1
  case "$tag_version" in
    "${official_version}" | "${official_version}"-*) ;;
    *)
      err "Error: Version mismatch"
      err "  Tag version:      $tag_version (from tag '$tag')"
      err "  Official version: $official_version (from metadata.json)"
      exit 1
      ;;
  esac
  printf '✓ Tag validation passed: %s (version %s)\n' "$tag" "$tag_version"
}
main "$@"
