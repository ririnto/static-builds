# syntax=docker/dockerfile:1.4
ARG ALPINE_VERSION
ARG HTTPD_VERSION
ARG APR_VERSION
ARG APR_UTIL_VERSION
ARG UBI9_MICRO_VERSION


FROM alpine:${ALPINE_VERSION} AS build
ARG HTTPD_VERSION
ARG APR_VERSION
ARG APR_UTIL_VERSION

WORKDIR /tmp
RUN --mount=type=cache,target=/var/cache/apk \
    apk add \
        build-base \
        linux-headers \
        perl \
        openssl-dev \
        openssl-libs-static \
        nghttp2-dev \
        nghttp2-static \
        pcre2-dev \
        pcre2-static \
        zlib-dev \
        zlib-static \
        expat-dev \
        expat-static \
        brotli-dev \
        brotli-static \
    ;
ADD "src/httpd-${HTTPD_VERSION}.tar.gz" /tmp/
ADD "src/apr-${APR_VERSION}.tar.gz" /tmp/
ADD "src/apr-util-${APR_UTIL_VERSION}.tar.gz" /tmp/
RUN mv "/tmp/apr-${APR_VERSION}" "/tmp/httpd-${HTTPD_VERSION}/srclib/apr"
RUN mv "/tmp/apr-util-${APR_UTIL_VERSION}" "/tmp/httpd-${HTTPD_VERSION}/srclib/apr-util"
WORKDIR "/tmp/httpd-${HTTPD_VERSION}"
RUN ./configure \
    # --help \
    # --help=short \
    # --help=recursive \
    # --version \
    # --quiet \
    # --cache-file=FILE \
    # --config-cache \
    # --no-create \
    # --srcdir=DIR \
    --prefix=/usr/local/apache2 \
    # --exec-prefix=EPREFIX \
    # --bindir=DIR \
    # --sbindir=DIR \
    # --libexecdir=DIR \
    # --sysconfdir=DIR \
    # --sharedstatedir=DIR \
    # --localstatedir=DIR \
    # --runstatedir=DIR \
    # --libdir=DIR \
    # --includedir=DIR \
    # --oldincludedir=DIR \
    # --datarootdir=DIR \
    # --datadir=DIR \
    # --infodir=DIR \
    # --localedir=DIR \
    # --mandir=DIR \
    # --docdir=DIR \
    # --htmldir=DIR \
    # --dvidir=DIR \
    # --pdfdir=DIR \
    # --psdir=DIR \
    # --build=BUILD \
    # --host=HOST \
    # --target=TARGET \
    # --disable-option-checking \
    # --disable-FEATURE \
    # --enable-FEATURE[=ARG] \
    # --enable-layout=LAYOUT \
    # --enable-dtrace \
    # --enable-hook-probes \
    # --enable-exception-hook \
    # --enable-load-all-modules \
    # --enable-maintainer-mode \
    # --enable-debugger-mode \
    --enable-pie \
    # --enable-modules=MODULE-LIST \
    # --enable-mods-shared=MODULE-LIST \
    --enable-mods-static=few \
    # --disable-authn-file \
    # --enable-authn-dbm \
    # --enable-authn-anon \
    # --enable-authn-dbd \
    # --enable-authn-socache \
    # --disable-authn-core \
    # --disable-authz-host \
    # --disable-authz-groupfile \
    # --disable-authz-user \
    # --enable-authz-dbm \
    # --enable-authz-owner \
    # --enable-authz-dbd \
    # --disable-authz-core \
    # --enable-authnz-ldap \
    # --enable-authnz-fcgi \
    # --disable-access-compat \
    # --disable-auth-basic \
    # --enable-auth-form \
    # --enable-auth-digest \
    # --enable-allowmethods \
    # --enable-isapi \
    # --enable-file-cache \
    # --enable-cache \
    # --enable-cache-disk \
    # --enable-cache-socache \
    --enable-socache-shmcb \
    # --enable-socache-dbm \
    # --enable-socache-memcache \
    # --enable-socache-redis \
    # --enable-socache-dc \
    # --enable-so \
    --enable-so=no \
    --enable-watchdog \
    # --enable-macro \
    # --enable-dbd \
    # --enable-bucketeer \
    # --enable-dumpio \
    # --enable-echo \
    # --enable-example-hooks \
    # --enable-case-filter \
    # --enable-case-filter-in \
    # --enable-example-ipc \
    # --enable-buffer \
    # --enable-data \
    # --enable-ratelimit \
    # --disable-reqtimeout \
    # --enable-ext-filter \
    # --enable-request \
    # --enable-include \
    # --disable-filter \
    # --enable-reflector \
    # --enable-substitute \
    # --enable-sed \
    # --disable-charset-lite \
    # --enable-charset-lite \
    --enable-deflate \
    # --enable-xml2enc \
    # --enable-proxy-html \
    --enable-brotli \
    --enable-http \
    # --disable-mime \
    # --enable-ldap \
    # --disable-log-config \
    # --enable-log-debug \
    # --enable-log-forensic \
    # --enable-logio \
    # --enable-lua \
    # --enable-luajit \
    # --disable-env \
    # --enable-mime-magic \
    # --enable-cern-meta \
    # --enable-expires \
    # --disable-headers \
    # --enable-ident \
    # --enable-usertrack \
    # --enable-unique-id \
    # --disable-setenvif \
    # --disable-version \
    --enable-remoteip \
    --enable-proxy \
    # --enable-proxy-connect \
    # --enable-proxy-ftp \
    --enable-proxy-http \
    # --enable-proxy-fcgi \
    # --enable-proxy-scgi \
    # --enable-proxy-uwsgi \
    # --enable-proxy-fdpass \
    # --enable-proxy-wstunnel \
    # --enable-proxy-ajp \
    --enable-proxy-balancer \
    # --enable-proxy-express \
    --enable-proxy-hcheck \
    # --enable-session \
    # --enable-session-cookie \
    # --enable-session-crypto \
    # --enable-session-dbd \
    --enable-slotmem-shm \
    # --enable-slotmem-plain \
    --enable-ssl \
    --enable-ssl-staticlib-deps \
    # --enable-optional-hook-export \
    # --enable-optional-hook-import \
    # --enable-optional-fn-import \
    # --enable-optional-fn-export \
    # --enable-dialup \
    # --enable-static-support \
    # --enable-static-htpasswd \
    # --enable-static-htdigest \
    --enable-static-rotatelogs \
    # --enable-static-logresolve \
    # --enable-static-htdbm \
    # --enable-static-ab \
    # --enable-static-checkgid \
    # --enable-static-htcacheclean \
    # --enable-static-httxt2dbm \
    # --enable-static-fcgistarter \
    --enable-http2 \
    --enable-nghttp2-staticlib-deps \
    --enable-proxy-http2 \
    # --enable-md \
    # --enable-jansson-staticlib-deps \
    # --enable-curl-staticlib-deps \
    --enable-lbmethod-byrequests \
    --enable-lbmethod-bytraffic \
    --enable-lbmethod-bybusyness \
    # --enable-lbmethod-heartbeat \
    # --enable-mpms-shared=MPM-LIST \
    # --enable-unixd \
    # --enable-privileges \
    # --enable-systemd \
    # --enable-heartbeat \
    # --enable-heartmonitor \
    # --enable-dav \
    # --disable-status \
    --disable-autoindex \
    --enable-asis \
    # --enable-info \
    # --enable-suexec \
    # --enable-cgid \
    # --enable-cgi \
    # --enable-cgid-fdpassing \
    # --enable-dav-fs \
    # --enable-dav-lock \
    # --enable-vhost-alias \
    # --enable-negotiation \
    # --disable-dir \
    # --enable-imagemap \
    # --enable-actions \
    # --enable-speling \
    # --enable-userdir \
    # --disable-alias \
    --enable-rewrite \
    # --enable-suexec-capabilities \
    # --enable-v4-mapped \
    --disable-proxy-ajp \
    --disable-proxy-connect \
    --disable-proxy-express \
    --disable-proxy-fcgi \
    --disable-proxy-fdpass \
    --disable-proxy-ftp \
    --disable-proxy-scgi \
    --disable-proxy-uwsgi \
    --disable-lbmethod-heartbeat \
    # --with-PACKAGE[=ARG] \
    # --without-PACKAGE \
    --with-included-apr \
    # --with-apr=PATH \
    # --with-apr-util=PATH \
    # --with-pcre=PATH \
    # --with-pcre=/usr \
    # --with-port=PORT \
    # --with-sslport=SSLPORT \
    # --with-distcache=PATH \
    # --with-z=PATH \
    # --with-libxml2=PATH \
    # --with-brotli=PATH \
    # --with-lua=PATH \
    # --with-ssl=PATH \
    # --with-nghttp2=PATH \
    # --with-jansson=PATH \
    # --with-curl=PATH \
    --with-mpm=event \
    # --with-module=module-type:module-file \
    # --with-program-name \
    # --with-suexec-bin \
    # --with-suexec-caller \
    # --with-suexec-userdir \
    # --with-suexec-docroot \
    # --with-suexec-uidmin \
    # --with-suexec-gidmin \
    # --with-suexec-logfile \
    # --with-suexec-syslog \
    # --with-suexec-safepath \
    # --with-suexec-umask \
    # CC \
    CFLAGS="-O2 -pipe -fPIE -fstack-protector-strong -fstack-clash-protection -ffunction-sections -fdata-sections -fno-delete-null-pointer-checks -fno-strict-overflow -fno-strict-aliasing -ftrivial-auto-var-init=zero -Wformat -Wformat=2 -Werror=format-security" \
    LDFLAGS="-static -static-pie -static-libgcc -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack -Wl,-z,separate-code -Wl,--as-needed -Wl,--gc-sections" \
    LIBS="-lbrotlienc -lbrotlidec -lbrotlicommon" \
    # CPPFLAGS \
    # CPP \
    ;
RUN make "-j$(nproc)"
RUN make install


FROM redhat/ubi9-minimal:${UBI9_MICRO_VERSION} AS verify

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build /usr/local/apache2 /usr/local/apache2
RUN set -eu; \
    echo "=== ELF Header ==="; \
    readelf -h /usr/local/apache2/bin/httpd | egrep 'Type:|Machine:'; \
    echo "=== Program Headers ==="; \
    readelf -l /usr/local/apache2/bin/httpd | egrep 'INTERP|DYNAMIC' || true; \
    echo "=== Dynamic Section ==="; \
    readelf -d /usr/local/apache2/bin/httpd || true; \
    file_output="$(file /usr/local/apache2/bin/httpd 2>&1 || true)"; \
    ldd_output="$(ldd /usr/local/apache2/bin/httpd 2>&1 || true)"; \
    echo "=== file output ==="; echo "$file_output"; \
    echo "=== ldd output ==="; echo "$ldd_output"; \
    (echo "$file_output" | grep -Eq 'statically linked|static-pie linked') || (echo "Error: file(1) did not report a static binary." && exit 1); \
    (echo "$ldd_output" | grep -Eqi 'statically linked|static-pie linked|not a dynamic executable|not a valid dynamic program') || (echo "Error: Dynamic libraries found. This should be a static build!" && exit 1); \
    strace /usr/local/apache2/bin/httpd -V && \
    echo "=== Static Modules (-l) ===" && /usr/local/apache2/bin/httpd -l && \
    echo "=== Loaded Modules (-M) ===" && /usr/local/apache2/bin/httpd -M


FROM scratch

COPY --from=verify /usr/local/apache2 /usr/local/apache2
EXPOSE 80 443
ENTRYPOINT ["/usr/local/apache2/bin/httpd"]
