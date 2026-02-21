#!/usr/bin/env sh
set -eu

BUILDCTL_PROGRESS=""
BUILDCTL_NO_CACHE=""
TARGET=""

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --plain)
            BUILDCTL_PROGRESS="plain"
            ;;
        --no-cache)
            BUILDCTL_NO_CACHE="1"
            ;;
        -*)
            echo "Unknown option: $arg"
            exit 1
            ;;
        *)
            if [ -z "$TARGET" ]; then
                TARGET="$arg"
            else
                echo "Error: Multiple targets specified"
                exit 1
            fi
            ;;
    esac
done

if [ -z "$TARGET" ]; then
    echo "Usage: $0 [options] <target>"
    echo "       $0 <target> [options]"
    echo "Options:"
    echo "  --plain      Output docker build logs in plain text"
    echo "  --no-cache   Build without using cache"
    echo "Example:"
    echo "  $0 nginx"
    echo "  $0 --plain haproxy"
    echo "  $0 apache-httpd --plain"
    echo "  $0 --no-cache nginx"
    exit 1
fi

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

TARGET="${TARGET}" BUILDCTL_PROGRESS="${BUILDCTL_PROGRESS}" BUILDCTL_NO_CACHE="${BUILDCTL_NO_CACHE}" docker compose -f .github/docker-compose.yaml --project-directory . run --rm build
