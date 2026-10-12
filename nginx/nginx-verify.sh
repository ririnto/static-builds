#!/bin/sh
set -eu
nginx_bin="${TARGET_PREFIX}/sbin/nginx"
echo "=== ELF Header ==="
readelf -h "$nginx_bin" | grep -E 'Type:|Machine:'
echo "=== Program Headers ==="
readelf -l "$nginx_bin" | grep -E 'INTERP|DYNAMIC' || true
echo "=== Dynamic Section ==="
readelf -d "$nginx_bin" || true
file_output="$(file "$nginx_bin" 2>&1 || true)"
ldd_output="$(ldd "$nginx_bin" 2>&1 || true)"
echo "=== file output ==="
echo "$file_output"
echo "=== ldd output ==="
echo "$ldd_output"
if ! echo "$file_output" | grep -Eq 'statically linked|static-pie linked'; then
  echo "Error: file(1) did not report a static binary."
  exit 1
fi
if ! echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program'; then
  echo "Error: Dynamic libraries found. This should be a static build!"
  exit 1
fi
strace "$nginx_bin" -V
"$nginx_bin" -t -c conf/scripting.conf
"$nginx_bin" -c conf/scripting.conf -g 'daemon off;' &
nginx_pid=$!
trap '"$nginx_bin" -s quit -c conf/scripting.conf; wait "$nginx_pid"' EXIT
curl --retry 10 --retry-connrefused --retry-delay 1 --retry-max-time 10 --max-time 2 -fsS http://127.0.0.1:18081/njs >/tmp/njs-result
grep -Fx "njs ${NJS_VERSION} engine QuickJS" /tmp/njs-result
curl -fsS http://127.0.0.1:18081/njs | grep -Fx "njs ${NJS_VERSION} engine QuickJS"
timeout 3 /usr/local/bin/nginx-stream-read.sh | grep -Fx "njs ${NJS_VERSION} engine QuickJS"
