#!/usr/bin/env sh
set -eu

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <target>"
    exit 1
fi

TARGET="$1"
shift

case "${TARGET}" in
    *[!A-Za-z0-9._-]*)
        echo "Error: Invalid target '${TARGET}'"
        exit 1
        ;;
esac

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
TARGET_DOWNLOAD_SH="${ROOT_DIR}/${TARGET}/download.sh"

if [ ! -f "${TARGET_DOWNLOAD_SH}" ]; then
    echo "Error: download script not found for target '${TARGET}'"
    exit 1
fi

exec sh "${TARGET_DOWNLOAD_SH}" "$@"
