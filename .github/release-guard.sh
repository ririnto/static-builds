#!/usr/bin/env sh
set -e
ROOT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)

# Print an error message to stderr.
#
# @param message Error message to print.
# @return Nothing.
# @return-type None
err() {
  printf '%s\n' "$*" >&2
}

# Parse a version from a tag using the metadata tag prefix.
#
# @param tag Release tag in format <target>-<version>.<revision>.
# @param tag_prefix Target tag prefix from metadata.
# @return Version string without revision printed to stdout.
# @return-type str
parse_tag() {
  tag="$1"
  tag_prefix="$2"
  version_revision="${tag#"${tag_prefix}"-}"
  version="${version_revision%.*}"
  revision="${version_revision##*.}"
  case "$revision" in
    '' | *[!0-9]*)
      err "Error: Revision '$revision' must contain only decimal digits"
      return 1
      ;;
    *) printf '%s' "$version" ;;
  esac
}

# Get the official version from centralized metadata.
#
# @param target Build target name.
# @return Official version string printed to stdout.
# @return-type str
get_official_version() {
  target="$1"
  metadata_script="${ROOT_DIR}/metadata.sh"
  version=$(sh "$metadata_script" get-official-version "$target")
  if [ -z "$version" ]; then
    err "Error: official version not found for $target"
    return 1
  fi
  printf '%s' "$version"
}

# Validate a release tag against target metadata.
#
# @param target Target name (for example nginx, haproxy, apache-httpd).
# @param tag Release tag (for example nginx-1.28.2.18).
# @return Exit code 0 on success, 1 on validation failure.
# @return-type int
main() {
  if [ $# -ne 2 ]; then
    err "Usage: $0 <target> <tag>"
    err "Example: $0 nginx nginx-1.28.2.18"
    exit 1
  fi
  target="$1"
  tag="$2"
  tag_prefix=$(sh "${ROOT_DIR}/metadata.sh" get-tag-prefix "$target") || exit 1
  case "$tag" in
    "${tag_prefix}"-*) ;;
    *)
      err "Error: Invalid tag format '$tag'"
      err "Expected format: ${tag_prefix}-<version>.<revision>"
      exit 1
      ;;
  esac
  tag_version=$(parse_tag "$tag" "$tag_prefix") || exit 1
  official_version=$(get_official_version "$target") || exit 1
  case "$tag_version" in
    "${official_version}") ;;
    "${official_version}"-*)
      prerelease=${tag_version#"${official_version}"-}
      if ! printf '%s\n' "$prerelease" | grep -Eq '^[A-Za-z0-9]+([.-][A-Za-z0-9]+)*$'; then
        err "Error: Invalid prerelease suffix '$prerelease'"
        exit 1
      fi
      ;;
    *)
      err "Error: Version mismatch"
      err "  Tag version:      $tag_version (from tag '$tag')"
      err "  Official version: $official_version (from metadata.json)"
      exit 1
      ;;
  esac
  printf 'Tag validation passed: %s (version %s)\n' "$tag" "$tag_version"
}
main "$@"
