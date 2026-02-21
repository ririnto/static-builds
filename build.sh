#!/usr/bin/env sh
set -eu

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <target> [buildctl args...]"
    echo "Example:"
    echo "  $0 nginx"
    echo "  $0 haproxy --progress=plain"
    exit 1
fi

TARGET="$1"
shift

case "$TARGET" in
    *[!A-Za-z0-9._-]*)
        echo "Error: Invalid target '${TARGET}'"
        echo "Allowed characters: letters, numbers, dot, underscore, hyphen"
        exit 1
        ;;
esac

if [ ! -d "${TARGET}" ]; then
    echo "Error: Target directory '${TARGET}' not found"
    exit 1
fi

if [ ! -f "${TARGET}/.env" ]; then
    echo "Error: .env file not found in '${TARGET}'"
    exit 1
fi

if [ ! -f "${TARGET}/Dockerfile" ]; then
    echo "Error: Dockerfile not found in '${TARGET}'"
    exit 1
fi

# Create cache directory if it doesn't exist (buildkit container may not have permission to create it)
mkdir -p "${TARGET}/.cache"
chmod 0777 "${TARGET}" "${TARGET}/.cache"

# Download source files if pre-download.sh exists
if [ -f "${TARGET}/pre-download.sh" ]; then
    echo "Downloading source files for ${TARGET}..."
    sh "${TARGET}/pre-download.sh"
fi

TARGET="${TARGET}" docker compose -f .github/docker-compose.yaml --project-directory . run --rm build "$@"
