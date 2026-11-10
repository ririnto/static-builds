ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS script-tools
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/nginx-build-quickjs/ ./cmd/nginx-build-quickjs/
COPY cmd/nginx-configure/ ./cmd/nginx-configure/
COPY cmd/nginx-build/ ./cmd/nginx-build/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/nginx-build-quickjs ./cmd/nginx-build-quickjs \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/nginx-configure ./cmd/nginx-configure \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/nginx-build ./cmd/nginx-build

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
COPY --from=script-tools /script-tools/nginx-build /script-tools/nginx-build-quickjs /script-tools/nginx-configure /usr/local/bin/
RUN /usr/local/bin/nginx-build-quickjs
ENV NJS_QUICKJS=YES NJS_LIBXSLT=NO
COPY nginx-upstream-healthcheck/ngx_http_upstream_healthcheck_module "/build/nginx-${NGINX_VERSION}/ngx_http_upstream_healthcheck_module"
WORKDIR "/build/nginx-${NGINX_VERSION}"
RUN /usr/local/bin/nginx-configure \
    "--add-module=nginx-module-vts-${NGINX_VTS_MODULE_VERSION}" \
    "--add-module=ngx_http_upstream_healthcheck_module" \
    "--add-module=/build/njs-${NJS_VERSION}/nginx"
RUN /usr/local/bin/nginx-build
COPY nginx/scripting.conf nginx/scripting.js ${TARGET_PREFIX}/conf/
COPY nginx-upstream-healthcheck/conf/upstream-healthcheck.conf ${TARGET_PREFIX}/conf/
COPY nginx-upstream-healthcheck/README.md ${TARGET_PREFIX}/README.md



FROM registry.access.redhat.com/ubi10/ubi-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG NJS_VERSION
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace openssl
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY nginx/nginx-stream-read.sh nginx/nginx-verify.sh /usr/local/bin/
RUN /usr/local/bin/nginx-verify.sh



FROM scratch AS runtime

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} /
COPY nginx-upstream-healthcheck/passwd /etc/passwd
COPY nginx-upstream-healthcheck/group /etc/group
EXPOSE 80 443
ENTRYPOINT ["/sbin/nginx", "-p", "/", "-g", "daemon off;"]


FROM verify AS verify-runtime
COPY --from=runtime / /runtime/
COPY nginx-upstream-healthcheck/verify-startup.sh /usr/local/bin/
RUN /usr/local/bin/verify-startup.sh


FROM runtime
COPY --from=verify-runtime /runtime/ /
