#!/bin/sh
set -eu
nginx_bin="${TARGET_PREFIX}/sbin/nginx"
nginx_pid=
probe_pid=
response_file=

# Signals and reaps an owned timeout supervisor with its bounded kill grace period.
# @param child_id Direct child timeout process ID, or an empty string.
# @param child_signal Signal forwarded to the supervised process group.
# @return 0 after the child has been reaped.
# @return-type int
stop_child() {
  child_id=$1
  child_signal=$2
  if [ -n "$child_id" ]; then
    if kill "-$child_signal" "$child_id"; then
      :
    fi
    if wait "$child_id"; then
      :
    fi
  fi
}

# Stops and reaps probes and nginx, removes the response file, and preserves exit status.
# @return The status that caused shell exit.
# @return-type int
cleanup() {
  cleanup_status=$?
  trap '' INT TERM
  trap - EXIT
  stop_child "$probe_pid" TERM
  stop_child "$nginx_pid" QUIT
  if [ -n "$response_file" ]; then
    rm -f "$response_file"
  fi
  exit "$cleanup_status"
}

# Runs an HTTP probe with a total deadline and records its supervisor for signal cleanup.
# @param curl_arguments Curl options and URL to pass to curl.
# @return The timeout or curl status.
# @return-type int
run_http_probe() {
  timeout --signal=TERM --kill-after=1s 15s curl "$@" &
  probe_pid=$!
  if wait "$probe_pid"; then
    probe_status=0
  else
    probe_status=$?
  fi
  probe_pid=
  return "$probe_status"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

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
response_file=$(mktemp /tmp/njs-result.XXXXXX)
timeout --signal=QUIT --kill-after=2s 40s "$nginx_bin" -c conf/scripting.conf -g 'daemon off;' &
nginx_pid=$!
run_http_probe --retry 10 --retry-connrefused --retry-delay 1 --retry-max-time 10 --max-time 2 -fsS -o "$response_file" http://127.0.0.1:18081/njs
grep -Fx "njs ${NJS_VERSION} engine QuickJS" "$response_file"
run_http_probe --max-time 5 -fsS -o "$response_file" http://127.0.0.1:18081/njs
grep -Fx "njs ${NJS_VERSION} engine QuickJS" "$response_file"
timeout 3 /usr/local/bin/nginx-stream-read.sh | grep -Fx "njs ${NJS_VERSION} engine QuickJS"
