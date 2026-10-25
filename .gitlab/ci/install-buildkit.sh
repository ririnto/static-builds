#!/usr/bin/env sh
set -eu
ROOT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
if [ "$#" -ne 1 ]; then
  printf 'Usage: %s <target>\n' "$0" >&2
  exit 1
fi
TARGET="$1"
sh "${ROOT_DIR}/metadata.sh" get-env "${TARGET}" >/dev/null
if [ ! -f "${ROOT_DIR}/${TARGET}/Dockerfile" ]; then
  printf 'Error: Dockerfile not found in %s\n' "${TARGET}" >&2
  exit 1
fi
BUILD_OUTPUT_DEST="${BUILD_OUTPUT_DEST:-${ROOT_DIR}/.out/${TARGET}}"
case "${BUILD_OUTPUT_DEST}" in
  "${ROOT_DIR}/.out/${TARGET}" | "${ROOT_DIR}/.out/${TARGET}/"* | ".out/${TARGET}" | ".out/${TARGET}/"* | "./.out/${TARGET}" | "./.out/${TARGET}/"*)
    case "/${BUILD_OUTPUT_DEST}/" in
      */../*)
        printf 'Error: Build output must be under .out/%s\n' "${TARGET}" >&2
        exit 1
        ;;
    esac
    ;;
  *)
    printf 'Error: Build output must be under .out/%s\n' "${TARGET}" >&2
    exit 1
    ;;
esac
ALPINE_ARCH=$(apk --print-arch)
case "${ALPINE_ARCH}" in
  x86_64)
    BUILDKIT_ARCH=amd64
    ROOTLESSKIT_ARCH=x86_64
    ;;
  aarch64)
    BUILDKIT_ARCH=arm64
    ROOTLESSKIT_ARCH=aarch64
    ;;
  *)
    printf 'Unsupported Alpine architecture for rootless BuildKit: %s\n' "${ALPINE_ARCH}" >&2
    exit 1
    ;;
esac
apk add --no-cache fuse3 fuse-overlayfs git openssh openssl pigz shadow-uidmap su-exec xz
adduser -D -u 1000 user
mkdir -p /run/user/1000
chmod 0700 /run/user/1000
mkdir -p /home/user/.local/tmp /home/user/.local/share/buildkit
chown -R 1000:1000 /run/user/1000 /home/user
printf 'user:100000:65536\n' >/etc/subuid
printf 'user:100000:65536\n' >/etc/subgid
mkdir -p "${ROOT_DIR}/.cache/${TARGET}" "${BUILD_OUTPUT_DEST}"
chown -R 1000:1000 "${ROOT_DIR}/.cache/${TARGET}" "${BUILD_OUTPUT_DEST}"
BUILDKIT_VERSION=0.34.1
ROOTLESSKIT_VERSION=3.3.0
temporary_dir=$(mktemp -d)
trap 'rm -rf "$temporary_dir"' EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
curl --fail --location --silent --show-error \
  --output "${temporary_dir}/buildkit.tar.gz" \
  "https://github.com/moby/buildkit/releases/download/v${BUILDKIT_VERSION}/buildkit-v${BUILDKIT_VERSION}.linux-${BUILDKIT_ARCH}.tar.gz"
curl --fail --location --silent --show-error \
  --output "${temporary_dir}/rootlesskit.tar.gz" \
  "https://github.com/rootless-containers/rootlesskit/releases/download/v${ROOTLESSKIT_VERSION}/rootlesskit-${ROOTLESSKIT_ARCH}.tar.gz"
curl --fail --location --silent --show-error \
  --output "${temporary_dir}/buildctl-daemonless.sh" \
  "https://raw.githubusercontent.com/moby/buildkit/v${BUILDKIT_VERSION}/examples/buildctl-daemonless/buildctl-daemonless.sh"
mkdir -p /usr/local/bin
tar -xzf "${temporary_dir}/buildkit.tar.gz" -C /usr/local
tar -xzf "${temporary_dir}/rootlesskit.tar.gz" -C /usr/local/bin
cp "${temporary_dir}/buildctl-daemonless.sh" /usr/local/bin/buildctl-daemonless.sh
chmod 0755 /usr/local/bin/buildctl-daemonless.sh
