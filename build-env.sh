#!/usr/bin/env sh

# Normalize one nonempty build variable and check its key.
# @param key Nonempty variable key that does not start with a hash.
# @param value Nonempty unnormalized variable value.
# @return 0 for an allowed key, 1 otherwise, with BUILD_ENV_VALUE normalized.
# @return-type int
normalize_build_env_value() {
  BUILD_ENV_VALUE=$(printf '%s\n' "$2" | tr -d '\r') || exit "$?"
  case "$1" in
    *[!A-Z0-9_]*) false ;;
    *) true ;;
  esac
}

# Select one build variable while preserving normalization order.
# @param key Variable key, or an empty or commented line.
# @param value Unnormalized variable value.
# @return 0 for a build argument, 1 for a skipped record.
# @return-type int
normalize_build_env() {
  case "$1" in
    '' | \#*) false ;;
    *) [ -n "$2" ] && normalize_build_env_value "$1" "$2" ;;
  esac
}

# Read the value selected by normalize_build_env.
# @return The normalized build argument value on stdout.
# @return-type string
build_env_value() {
  printf '%s' "$BUILD_ENV_VALUE"
}
