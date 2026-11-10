#!/usr/bin/env sh
set -eu
echo "=== ELF Header ==="
readelf -h "${TARGET_PREFIX}/bin/monit" | grep -E 'Type:|Machine:'
echo "=== Program Headers ==="
readelf -l "${TARGET_PREFIX}/bin/monit" | grep -E 'INTERP|DYNAMIC' || true
echo "=== Dynamic Section ==="
readelf -d "${TARGET_PREFIX}/bin/monit" || true
file_output="$(file "${TARGET_PREFIX}/bin/monit" 2>&1 || true)"
ldd_output="$(ldd "${TARGET_PREFIX}/bin/monit" 2>&1 || true)"
echo "=== file output ==="
echo "$file_output"
echo "=== ldd output ==="
echo "$ldd_output"
(echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1)
(echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1)
strace "${TARGET_PREFIX}/bin/monit" -V
