ARG ALPINE_VERSION
ARG UBI10_MINIMAL_VERSION
ARG TARGET_PREFIX=/home/nobody


FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS script-tools
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/build-quickjs/ ./cmd/build-quickjs/
COPY cmd/configure-nginx/ ./cmd/configure-nginx/
COPY cmd/build-nginx/ ./cmd/build-nginx/
COPY cmd/verify-nginx/ ./cmd/verify-nginx/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/build-quickjs ./cmd/build-quickjs \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/configure-nginx ./cmd/configure-nginx \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/build-nginx ./cmd/build-nginx \
    && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /script-tools/verify-nginx ./cmd/verify-nginx


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
COPY --from=script-tools /script-tools/build-nginx /script-tools/build-quickjs /script-tools/configure-nginx /usr/local/bin/
RUN /usr/local/bin/build-quickjs
ENV NJS_QUICKJS=YES NJS_LIBXSLT=NO
WORKDIR "/build/nginx-${NGINX_VERSION}"
RUN /usr/local/bin/configure-nginx \
    "--add-module=nginx-module-vts-${NGINX_VTS_MODULE_VERSION}" \
    "--add-module=/build/njs-${NJS_VERSION}/nginx"
RUN /usr/local/bin/build-nginx
COPY nginx/scripting.conf nginx/scripting.js ${TARGET_PREFIX}/conf/



FROM registry.access.redhat.com/ubi10/ubi-minimal:${UBI10_MINIMAL_VERSION} AS verify
ARG NJS_VERSION
ARG TARGET_PREFIX

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install file binutils strace openssl
COPY --from=build ${TARGET_PREFIX} ${TARGET_PREFIX}
COPY --from=script-tools /script-tools/verify-nginx /usr/local/bin/
RUN /usr/local/bin/verify-nginx


FROM scratch

ARG TARGET_PREFIX

COPY --from=verify ${TARGET_PREFIX} .
EXPOSE 80 443
ENTRYPOINT ["./sbin/nginx"]
