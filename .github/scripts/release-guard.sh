#!/usr/bin/env sh
set -eu
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
# Get official version from target's .env file.
# :param str target: Build target name.
# :returns: Official version string printed to stdout.
# :rtype: str
get_official_version() {
  target="$1"
  env_file="${target}/.env"
  if [ ! -f "$env_file" ]; then
    err "Error: .env file not found: $env_file"
    return 1
  fi

  case "$target" in
  apache-httpd)
    var_name="HTTPD_VERSION"
    ;;
  *)
    var_name="$(printf '%s' "$target" | tr '[:lower:]' '[:upper:]')_VERSION"
    ;;
  esac

  version=$(grep "^${var_name}=" "$env_file" | cut -d'=' -f2)
  if [ -z "$version" ]; then
    err "Error: ${var_name} not found in $env_file"
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
  tag_prefix="$target"
  if [ "$target" = "apache-httpd" ]; then
    tag_prefix="httpd"
  fi
  if ! printf '%s' "$tag" | grep -qE "^${tag_prefix}-"; then
    err "Error: Invalid tag format '$tag'"
    err "Expected format: ${tag_prefix}-<version>.<revision>"
    err "Example: ${tag_prefix}-1.28.2.18"
    exit 1
  fi
  tag_version=$(parse_tag "$tag") || exit 1
  official_version=$(get_official_version "$target") || exit 1
  if [ "$tag_version" != "$official_version" ]; then
    err "Error: Version mismatch"
    err "  Tag version:      $tag_version"
    err "  Official version: $official_version (from ${target}/.env)"
    exit 1
  fi
  printf '✓ Tag validation passed: %s (version %s)\n' "$tag" "$tag_version"
}
main "$@"
