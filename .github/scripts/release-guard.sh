#!/usr/bin/env sh
# release-guard.sh - Validate release tag format and version consistency
#
# Usage: release-guard.sh <target> <tag>
#
# Validates:
# 1. Tag format: <prefix>-<version>.<revision> (e.g., nginx-1.28.2.18)
# 2. Version matches target's .env file
# 3. Revision is numeric
#
# Returns: 0 on success, non-zero on failure

set -eu

# Print error message to stderr
err() {
    printf '%s\n' "$*" >&2
}

# Parse version and revision from tag using last-dot split
# Example: nginx-1.28.2.18 → version=1.28.2, revision=18
parse_tag() {
    tag="$1"

    # Remove prefix to get version-revision part
    version_revision="${tag#*-}"

    # Split on last dot
    version="${version_revision%.*}"
    revision="${version_revision##*.}"

    # Validate revision is numeric
    if ! [ "$revision" -eq "$revision" ] 2>/dev/null; then
        err "Error: Revision '$revision' is not numeric"
        return 1
    fi

    printf '%s' "$version"
}

# Get official version from target's .env file
get_official_version() {
    target="$1"
    env_file="${target}/.env"

    if [ ! -f "$env_file" ]; then
        err "Error: .env file not found: $env_file"
        return 1
    fi

    # Map target to version variable name
    case "$target" in
        apache-httpd)
            var_name="HTTPD_VERSION"
            ;;
        coredns)
            var_name="COREDNS_VERSION"
            ;;
        dnsmasq)
            var_name="DNSMASQ_VERSION"
            ;;
        haproxy)
            var_name="HAPROXY_VERSION"
            ;;
        monit)
            var_name="MONIT_VERSION"
            ;;
        nginx)
            var_name="NGINX_VERSION"
            ;;
        vector)
            var_name="VECTOR_VERSION"
            ;;
        *)
            err "Error: Unknown target '$target'"
            return 1
            ;;
    esac

    # Extract version from .env file
    version=$(grep "^${var_name}=" "$env_file" | cut -d'=' -f2)

    if [ -z "$version" ]; then
        err "Error: ${var_name} not found in $env_file"
        return 1
    fi

    printf '%s' "$version"
}

main() {
    if [ $# -ne 2 ]; then
        err "Usage: $0 <target> <tag>"
        err "Example: $0 nginx nginx-1.28.2.18"
        exit 1
    fi

    target="$1"
    tag="$2"

    # Map target directory to tag prefix
    case "$target" in
        apache-httpd)
            tag_prefix="httpd"
            ;;
        *)
            tag_prefix="$target"
            ;;
    esac

    # Validate tag format: <prefix>-<version>.<revision>
    if ! printf '%s' "$tag" | grep -qE "^${tag_prefix}-[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$"; then
        err "Error: Invalid tag format '$tag'"
        err "Expected format: ${tag_prefix}-<version>.<revision>"
        err "Example: ${tag_prefix}-1.28.2.18"
        exit 1
    fi

    # Parse version from tag
    tag_version=$(parse_tag "$tag") || exit 1

    # Get official version from .env
    official_version=$(get_official_version "$target") || exit 1

    # Validate version match
    if [ "$tag_version" != "$official_version" ]; then
        err "Error: Version mismatch"
        err "  Tag version:      $tag_version"
        err "  Official version: $official_version (from ${target}/.env)"
        exit 1
    fi

    printf '✓ Tag validation passed: %s (version %s)\n' "$tag" "$tag_version"
}

main "$@"
