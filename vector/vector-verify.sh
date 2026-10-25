#!/usr/bin/env sh
set -eu
readelf -h "${TARGET_PREFIX}/bin/vector" | grep -E 'Type:|Machine:'
readelf -l "${TARGET_PREFIX}/bin/vector" | grep -E 'INTERP|DYNAMIC' || true
readelf -d "${TARGET_PREFIX}/bin/vector" || true
file_output="$(file "${TARGET_PREFIX}/bin/vector" 2>&1 || true)"
ldd_output="$(ldd "${TARGET_PREFIX}/bin/vector" 2>&1 || true)"
echo "file output: $file_output"
echo "ldd output: $ldd_output"
(echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary (allowing musl static PIE)." && exit 1)
(echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1)
strace "${TARGET_PREFIX}/bin/vector" --version &&
  echo "=== Available Components ===" &&
  "${TARGET_PREFIX}/bin/vector" list
