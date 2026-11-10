#!/bin/sh
set -eu
case "$(uname -m)" in
  x86_64) architecture=x86_64 ;;
  aarch64 | arm64) architecture=aarch64 ;;
  *)
    printf 'Unsupported architecture: %s\n' "$(uname -m)" >&2
    exit 1
    ;;
esac
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT
mkdir "$temporary/rumdl" "$temporary/lychee"
curl --fail --location --silent --show-error "https://github.com/rvben/rumdl/releases/download/v0.3.1/rumdl-v0.3.1-${architecture}-unknown-linux-musl.tar.gz" --output "$temporary/rumdl.tar.gz"
tar -xzf "$temporary/rumdl.tar.gz" -C "$temporary/rumdl"
install -m 0755 "$(find "$temporary/rumdl" -type f -name rumdl -print -quit)" /usr/local/bin/rumdl
curl --fail --location --silent --show-error "https://github.com/lycheeverse/lychee/releases/download/lychee-v0.24.2/lychee-${architecture}-unknown-linux-musl.tar.gz" --output "$temporary/lychee.tar.gz"
tar -xzf "$temporary/lychee.tar.gz" -C "$temporary/lychee"
install -m 0755 "$(find "$temporary/lychee" -type f -name lychee -print -quit)" /usr/local/bin/lychee
