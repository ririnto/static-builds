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
COPY nginx/nginx-build.sh nginx/nginx-build-quickjs.sh nginx/nginx-configure.sh /usr/local/bin/
RUN /usr/local/bin/nginx-build-quickjs.sh
ENV NJS_QUICKJS=YES NJS_LIBXSLT=NO
COPY nginx-upstream-healthcheck/ngx_http_upstream_healthcheck_module "/build/nginx-${NGINX_VERSION}/ngx_http_upstream_healthcheck_module"
WORKDIR "/build/nginx-${NGINX_VERSION}"
RUN /usr/local/bin/nginx-configure.sh \
    "--add-module=nginx-module-vts-${NGINX_VTS_MODULE_VERSION}" \
    "--add-module=ngx_http_upstream_healthcheck_module" \
    "--add-module=/build/njs-${NJS_VERSION}/nginx"
RUN /usr/local/bin/nginx-build.sh
COPY nginx/scripting.conf nginx/scripting.js ${TARGET_PREFIX}/conf/
COPY nginx-upstream-healthcheck/upstream-healthcheck.conf ${TARGET_PREFIX}/conf/
COPY nginx-upstream-healthcheck/README.md ${TARGET_PREFIX}/README.md



FROM redhat/ubi10-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG NJS_VERSION
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY nginx/nginx-stream-read.sh nginx/nginx-verify.sh /usr/local/bin/
RUN /usr/local/bin/nginx-verify.sh



FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/nginx"]
