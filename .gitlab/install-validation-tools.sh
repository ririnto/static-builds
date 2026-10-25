#!/usr/bin/env sh
set -eu
case "$(uname -m)" in
  x86_64)
    archive_arch=x86_64
    shellcheck_arch=x86_64
    shfmt_arch=amd64
    ;;
  aarch64 | arm64)
    archive_arch=aarch64
    shellcheck_arch=aarch64
    shfmt_arch=arm64
    ;;
  *)
    echo "Unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac
tools_dir=$(mktemp -d)
trap 'rm -rf "$tools_dir"' EXIT
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install --yes --no-install-recommends ca-certificates curl make xz-utils
mkdir "$tools_dir/rumdl" "$tools_dir/lychee" "$tools_dir/shellcheck"
curl --fail --location --silent --show-error "https://github.com/rvben/rumdl/releases/download/v0.3.0/rumdl-v0.3.0-${archive_arch}-unknown-linux-gnu.tar.gz" --output "$tools_dir/rumdl.tar.gz"
tar --extract --gzip --file "$tools_dir/rumdl.tar.gz" --directory "$tools_dir/rumdl"
install --mode=0755 "$(find "$tools_dir/rumdl" -type f -name rumdl -print -quit)" /usr/local/bin/rumdl
curl --fail --location --silent --show-error "https://github.com/lycheeverse/lychee/releases/download/lychee-v0.24.2/lychee-${archive_arch}-unknown-linux-gnu.tar.gz" --output "$tools_dir/lychee.tar.gz"
tar --extract --gzip --file "$tools_dir/lychee.tar.gz" --directory "$tools_dir/lychee"
install --mode=0755 "$(find "$tools_dir/lychee" -type f -name lychee -print -quit)" /usr/local/bin/lychee
curl --fail --location --silent --show-error "https://github.com/koalaman/shellcheck/releases/download/v0.11.0/shellcheck-v0.11.0.linux.${shellcheck_arch}.tar.xz" --output "$tools_dir/shellcheck.tar.xz"
tar --extract --xz --file "$tools_dir/shellcheck.tar.xz" --directory "$tools_dir/shellcheck"
install --mode=0755 "$(find "$tools_dir/shellcheck" -type f -name shellcheck -print -quit)" /usr/local/bin/shellcheck
curl --fail --location --silent --show-error "https://github.com/mvdan/sh/releases/download/v3.14.1/shfmt_v3.14.1_linux_${shfmt_arch}" --output /usr/local/bin/shfmt
chmod 0755 /usr/local/bin/shfmt
