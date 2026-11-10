#!/usr/bin/env sh
set -e

ROOT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)

case "$(uname -s)" in
  Linux)
    exec sh "${ROOT_DIR}/.github/build-daemonless.sh" "$@"
    ;;
  Darwin)
    exec sh "${ROOT_DIR}/.github/build-buildx.sh" "$@"
    ;;
  *)
    printf 'Error: Unsupported build host: %s\n' "$(uname -s)" >&2
    exit 1
    ;;
esac
