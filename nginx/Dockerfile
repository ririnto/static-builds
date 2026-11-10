# syntax=docker/dockerfile:1
ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM alpine:${ALPINE_VERSION} AS build
ARG NGINX_VERSION
ARG NGINX_VTS_MODULE_VERSION
ARG NJS_VERSION
ARG QUICKJS_VERSION
ARG TARGET_PREFIX

WORKDIR /build
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        cmake \
        pkgconf \
        libatomic_ops-dev \
        libatomic_ops-static \
        linux-headers \
        openssl-dev \
        openssl-libs-static \
        pcre2-dev \
        pcre2-static \
        zlib-dev \
        zlib-static
ADD ".tmp/nginx-${NGINX_VERSION}.tar.gz" /build/
ADD ".tmp/nginx-module-vts-${NGINX_VTS_MODULE_VERSION}.tar.gz" "/build/nginx-${NGINX_VERSION}/"
ADD ".tmp/njs-${NJS_VERSION}.tar.gz" /build/
ADD ".tmp/quickjs-${QUICKJS_VERSION}.tar.gz" /build/
RUN <<'BUILD_QUICKJS'
set -eu
cmake -S "/build/quickjs-${QUICKJS_VERSION}" -B /build/quickjs-build \
  -DBUILD_SHARED_LIBS=OFF -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DCMAKE_BUILD_TYPE=Release
cmake --build /build/quickjs-build "-j$(nproc)"
cmake --install /build/quickjs-build
BUILD_QUICKJS
ENV NJS_QUICKJS=YES NJS_LIBXSLT=NO
WORKDIR "/build/nginx-${NGINX_VERSION}"
RUN <<'CONFIGURE'
set -eu
./configure \
  --prefix="${TARGET_PREFIX}" \
  --with-threads \
  --with-file-aio \
  --with-http_ssl_module \
  --with-http_v2_module \
  --with-http_v3_module \
  --with-http_realip_module \
  --with-http_gzip_static_module \
  --with-http_stub_status_module \
  --without-http_ssi_module \
  --without-http_userid_module \
  --without-http_autoindex_module \
  --without-http_split_clients_module \
  --without-http_fastcgi_module \
  --without-http_uwsgi_module \
  --without-http_scgi_module \
  --without-http_memcached_module \
  --without-http_empty_gif_module \
  --without-http_browser_module \
  --with-stream \
  --with-stream_ssl_module \
  --with-stream_realip_module \
  --with-stream_ssl_preread_module \
  "--add-module=nginx-module-vts-${NGINX_VTS_MODULE_VERSION}" \
  "--add-module=/build/njs-${NJS_VERSION}/nginx" \
  --with-cc-opt="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
  --with-ld-opt="-L/usr/local/lib -static -static-pie -static-libgcc -Wl,-E -rdynamic -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed" \
  --with-libatomic
CONFIGURE
RUN make "-j$(nproc)" && make install
COPY nginx/scripting.conf nginx/scripting.js ${TARGET_PREFIX}/conf/



FROM docker.io/redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG NJS_VERSION
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace openssl
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN <<'VERIFY'
set -eu
nginx_bin="${TARGET_PREFIX}/sbin/nginx"
nginx_pid=
probe_pid=
response_file=

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
timeout 3 bash -c 'exec 3<>/dev/tcp/127.0.0.1/18082; IFS= read -r response <&3; printf "%s\n" "$response"' | grep -Fx "njs ${NJS_VERSION} engine QuickJS"
VERIFY


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/nginx"]
