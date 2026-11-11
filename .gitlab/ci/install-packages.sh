#!/bin/sh
set -eu

cache_dir="${CI_PROJECT_DIR:-$(pwd)}/.cache/apk/$(cat /etc/alpine-release)/$(apk --print-arch)"
mkdir -p "$cache_dir"
apk --cache-dir "$cache_dir" update
apk --cache-dir "$cache_dir" add "$@"
