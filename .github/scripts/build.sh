#!/usr/bin/env sh
set -eu

mkdir -p "./.cache"
exec buildctl-daemonless.sh build \
    --frontend=dockerfile.v0 \
    --opt platform=linux/amd64 \
    --local=context=. \
    --local=dockerfile=. \
    $(
        while IFS='=' read -r key value || [ -n "$key" ]; do
            case "$key" in
                ''|\#*) continue ;;
            esac
            printf '%s\n' "--opt=build-arg:${key}=${value}"
        done < .env
    ) \
    --output=type=local,dest=. \
    --import-cache type=local,src=./.cache \
    --export-cache type=local,dest=./.cache,mode=max \
    "$@"
