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
RUN cmake -S "/build/quickjs-${QUICKJS_VERSION}" -B /build/quickjs-build \
        -DBUILD_SHARED_LIBS=OFF -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DCMAKE_BUILD_TYPE=Release && \
    cmake --build /build/quickjs-build "-j$(nproc)" && \
    cmake --install /build/quickjs-build
ENV NJS_QUICKJS=YES NJS_LIBXSLT=NO
COPY nginx-upstream-healthcheck/ngx_http_upstream_healthcheck_module "/build/nginx-${NGINX_VERSION}/ngx_http_upstream_healthcheck_module"
WORKDIR "/build/nginx-${NGINX_VERSION}"
RUN ./configure \
    --prefix=${TARGET_PREFIX} \
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
    --add-module="nginx-module-vts-${NGINX_VTS_MODULE_VERSION}" \
    --add-module="ngx_http_upstream_healthcheck_module" \
    --add-module="/build/njs-${NJS_VERSION}/nginx" \
    --with-cc-opt="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    --with-ld-opt="-L/usr/local/lib -static -static-pie -static-libgcc -Wl,-E -rdynamic -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed" \
    --with-libatomic \
    ;
RUN make "-j$(nproc)" && make install
COPY nginx/scripting.conf nginx/scripting.js ${TARGET_PREFIX}/conf/
COPY nginx-upstream-healthcheck/upstream-healthcheck.conf ${TARGET_PREFIX}/conf/



FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG NJS_VERSION
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace openssl dnsmasq socat
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
RUN set -eu; \
    echo "=== ELF Header ==="; \
    readelf -h ${TARGET_PREFIX}/sbin/nginx | egrep 'Type:|Machine:'; \
    echo "=== Program Headers ==="; \
    readelf -l ${TARGET_PREFIX}/sbin/nginx | egrep 'INTERP|DYNAMIC' || true; \
    echo "=== Dynamic Section ==="; \
    readelf -d ${TARGET_PREFIX}/sbin/nginx || true; \
    file_output="$(file ${TARGET_PREFIX}/sbin/nginx 2>&1 || true)"; \
    ldd_output="$(ldd ${TARGET_PREFIX}/sbin/nginx 2>&1 || true)"; \
    echo "=== file output ==="; echo "$file_output"; \
    echo "=== ldd output ==="; echo "$ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace ${TARGET_PREFIX}/sbin/nginx -V; \
    test ! -d ${TARGET_PREFIX}/lualib; \
    test ! -d ${TARGET_PREFIX}/lib/lua; \
    test ! -d ${TARGET_PREFIX}/share/lua; \
    ${TARGET_PREFIX}/sbin/nginx -t -c conf/scripting.conf
RUN <<'EOF'
set -eu
nginx_bin="${TARGET_PREFIX}/sbin/nginx"
"$nginx_bin" -c conf/scripting.conf -g 'daemon off;' &
nginx_pid=$!
trap '"$nginx_bin" -s quit -c conf/scripting.conf; wait "$nginx_pid"' EXIT
curl --retry 10 --retry-connrefused --retry-delay 1 --retry-max-time 10 --max-time 2 -fsS http://127.0.0.1:18081/njs > /tmp/njs-result
grep -Fx "njs ${NJS_VERSION}" /tmp/njs-result
curl -fsS http://127.0.0.1:18081/njs | grep -Fx "njs ${NJS_VERSION}"
timeout 3 bash -c 'cat < /dev/tcp/127.0.0.1/18082' | grep -Fx "njs ${NJS_VERSION}"
EOF

RUN <<'RUNTIME'
set -eu
cd "${TARGET_PREFIX}"
mkdir -p /tmp/healthcheck
openssl req -x509 -newkey rsa:2048 -nodes -keyout /tmp/healthcheck/key.pem -out /tmp/healthcheck/ca.pem -days 1 -subj /CN=backend.local -addext subjectAltName=DNS:backend.local
openssl req -x509 -newkey rsa:2048 -nodes -keyout /tmp/healthcheck/wrong-key.pem -out /tmp/healthcheck/wrong-ca.pem -days 1 -subj /CN=untrusted.local
openssl s_server -accept 127.0.0.1:19445 -cert /tmp/healthcheck/wrong-ca.pem -key /tmp/healthcheck/wrong-key.pem -quiet -ign_eof < /dev/null &
recover_pid=$!
printf '127.0.0.2 backend.local\n127.0.0.3 backend.local\n' > /tmp/healthcheck/hosts
dnsmasq --keep-in-foreground --no-resolv --no-hosts --addn-hosts=/tmp/healthcheck/hosts --port=15353 --listen-address=127.0.0.1 --bind-interfaces --pid-file=/tmp/healthcheck/dns.pid &
dns_pid=$!
socat TCP-LISTEN:19443,bind=127.0.0.1,reuseaddr,fork EXEC:'sleep 2' &
stall_pid=$!
cat > conf/healthcheck-runtime.conf <<'CONF'
worker_processes 2;
error_log stderr notice;
events {}
http {
    resolver 127.0.0.1:15353 valid=1s ipv6=off;
    vhost_traffic_status_zone;
    upstream dynamic_http {
        zone dynamic_http 256k;
        server backend.local:19080 resolve;
        server 127.0.0.1:19999 backup;
        healthcheck type=http interval=100ms timeout=200ms fall=1 rise=2;
    }
    upstream tls_only {
        zone tls_only 256k;
        server 127.0.0.1:19444;
        healthcheck type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    upstream wrong_name {
        zone wrong_name 256k;
        server 127.0.0.1:19444;
        healthcheck type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name wrong.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    upstream handshake_failure {
        zone handshake_failure 256k;
        server 127.0.0.2:19080;
        healthcheck type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    upstream handshake_timeout {
        zone handshake_timeout 256k;
        server 127.0.0.1:19443;
        healthcheck type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    upstream wrong_ca {
        zone wrong_ca 256k;
        server 127.0.0.1:19444;
        healthcheck type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/wrong-ca.pem;
    }
    upstream tls_recovery {
        zone tls_recovery 256k;
        server 127.0.0.1:19445;
        healthcheck type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    upstream ipv6_tcp {
        zone ipv6_tcp 256k;
        server [::1]:19084;
        healthcheck type=tcp interval=100ms timeout=200ms fall=1 rise=2;
    }
    upstream l4_failure {
        zone l4_failure 256k;
        server 127.0.0.1:19999;
        healthcheck type=tcp interval=100ms timeout=200ms fall=1 rise=2;
    }
    upstream l4 {
        zone l4 256k;
        server 127.0.0.2:19080;
        healthcheck type=tcp interval=100ms timeout=200ms fall=1 rise=2;
    }
    server {
        listen 127.0.0.2:19080;
        listen 127.0.0.4:19080;
        location / { return 200 "$server_addr\n"; }
    }
    server {
        listen 127.0.0.3:19080;
        location / {
            if (-f /tmp/healthcheck/recovered) { return 200 "$server_addr\n"; }
            return 503;
        }
    }
    server {
        listen 127.0.0.1:19081;
        location /route { proxy_pass http://dynamic_http; }
        location /metrics { healthcheck_metrics; }
        location /traffic { vhost_traffic_status_display; vhost_traffic_status_display_format prometheus; }
    }
}
stream {
    resolver 127.0.0.1:15353 valid=1s ipv6=off;
    upstream dynamic_tcp {
        zone dynamic_tcp 256k;
        server backend.local:19082 resolve;
        server 127.0.0.1:19999 backup;
        healthcheck_tcp interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_send "PING\r\n";
        healthcheck_expect "PONG" min_recv=4;
    }
    upstream tls_stream {
        zone tls_stream 256k;
        server 127.0.0.1:19444;
        healthcheck_tcp type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    upstream stream_wrong_name {
        zone stream_wrong_name 256k;
        server 127.0.0.1:19444;
        healthcheck_tcp type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name wrong.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    upstream stream_wrong_ca {
        zone stream_wrong_ca 256k;
        server 127.0.0.1:19444;
        healthcheck_tcp type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/wrong-ca.pem;
    }
    upstream stream_handshake_failure {
        zone stream_handshake_failure 256k;
        server 127.0.0.2:19080;
        healthcheck_tcp type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    upstream stream_handshake_timeout {
        zone stream_handshake_timeout 256k;
        server 127.0.0.1:19443;
        healthcheck_tcp type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    upstream stream_tls_recovery {
        zone stream_tls_recovery 256k;
        server 127.0.0.1:19445;
        healthcheck_tcp type=tls interval=100ms timeout=200ms fall=1 rise=2;
        healthcheck_tls_name backend.local;
        healthcheck_trusted_certificate /tmp/healthcheck/ca.pem;
    }
    log_format application_bytes '$bytes_received';
    server {
        listen 127.0.0.1:19444 ssl;
        ssl_certificate /tmp/healthcheck/ca.pem;
        ssl_certificate_key /tmp/healthcheck/key.pem;
        ssl_handshake_timeout 1s;
        access_log /tmp/healthcheck/tls-application.log application_bytes;
        return "TLS ready\n";
    }
    server { listen 127.0.0.2:19082; listen 127.0.0.4:19082; return "$server_addr PONG\n"; }
    server { listen 127.0.0.3:19082; return "FAIL\n"; }
    server { listen 127.0.0.1:19083; proxy_pass dynamic_tcp; }
    server { listen [::1]:19084; return "IPv6 ready\n"; }
}
CONF
sbin/nginx -t -c conf/healthcheck-runtime.conf
sbin/nginx -c conf/healthcheck-runtime.conf -g 'daemon off;' &
nginx_pid=$!
trap 'sbin/nginx -s quit -c conf/healthcheck-runtime.conf; wait "$nginx_pid"; kill "$dns_pid" "$stall_pid" "$recover_pid"; wait "$dns_pid" || test "$?" = 143; wait "$stall_pid" || test "$?" = 143; wait "$recover_pid" || test "$?" = 143; rm conf/healthcheck-runtime.conf' EXIT
sleep 3
curl -fsS http://127.0.0.1:19081/route | grep -Fx '127.0.0.2'
timeout 3 bash -c 'cat < /dev/tcp/127.0.0.1/19083' | grep -Fx '127.0.0.2 PONG'
curl -fsS http://127.0.0.1:19081/metrics > /tmp/healthcheck/metrics
for group in tls_only l4 ipv6_tcp; do grep -E "^nginx_healthcheck_peer_up\{upstream=\"$group\".* 1$" /tmp/healthcheck/metrics; done
for group in wrong_name wrong_ca handshake_failure handshake_timeout tls_recovery l4_failure; do grep -E "^nginx_healthcheck_check_status\{upstream=\"$group\".* 0$" /tmp/healthcheck/metrics; done
for group in stream_wrong_name stream_wrong_ca stream_handshake_failure stream_handshake_timeout stream_tls_recovery; do grep -E "^nginx_stream_healthcheck_check_status\{upstream=\"$group\".* 0$" /tmp/healthcheck/metrics; done
grep -E '^nginx_stream_healthcheck_peer_up\{upstream="tls_stream".* 1$' /tmp/healthcheck/metrics
grep -E '^nginx_healthcheck_check_code\{upstream="tls_only".* 0$' /tmp/healthcheck/metrics
grep -Fx 0 /tmp/healthcheck/tls-application.log
if grep -Ev '^0$' /tmp/healthcheck/tls-application.log; then exit 1; fi
timeout 3 bash -c 'cat < /dev/tcp/::1/19084' | grep -Fx 'IPv6 ready'
curl -fsS http://127.0.0.1:19081/traffic | grep '^nginx_vts_'
printf '127.0.0.4 backend.local\n127.0.0.3 backend.local\n' > /tmp/healthcheck/hosts
kill -HUP "$dns_pid"
sleep 3
curl -fsS http://127.0.0.1:19081/route | grep -Fx '127.0.0.4'
timeout 3 bash -c 'cat < /dev/tcp/127.0.0.1/19083' | grep -Fx '127.0.0.4 PONG'
curl -fsS http://127.0.0.1:19081/metrics > /tmp/healthcheck/metrics
if grep -E 'upstream="dynamic_(http|tcp)".*peer="127.0.0.2:' /tmp/healthcheck/metrics; then exit 1; fi
grep -E '^nginx_healthcheck_peer_up\{upstream="dynamic_http".*peer="127.0.0.4:19080".* 1$' /tmp/healthcheck/metrics
grep -E '^nginx_stream_healthcheck_peer_up\{upstream="dynamic_tcp".*peer="127.0.0.4:19082".* 1$' /tmp/healthcheck/metrics
kill "$recover_pid"
wait "$recover_pid" || test "$?" = 143
openssl s_server -accept 127.0.0.1:19445 -cert /tmp/healthcheck/ca.pem -key /tmp/healthcheck/key.pem -quiet -ign_eof < /dev/null &
recover_pid=$!
touch /tmp/healthcheck/recovered
sleep 1
curl -fsS http://127.0.0.1:19081/metrics | grep -E '^nginx_healthcheck_peer_up\{upstream="dynamic_http".*peer="127.0.0.3:19080".* 1$'
curl -fsS http://127.0.0.1:19081/metrics > /tmp/healthcheck/metrics
grep -E '^nginx_healthcheck_peer_up\{upstream="tls_recovery".* 1$' /tmp/healthcheck/metrics
grep -E '^nginx_stream_healthcheck_peer_up\{upstream="stream_tls_recovery".* 1$' /tmp/healthcheck/metrics
printf 'L4, L6, application, DNS routing, recovery and metrics OK\n'
RUNTIME


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/nginx"]
