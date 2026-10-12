ARG FUNCTIONAL_BASE_IMAGE
FROM ${FUNCTIONAL_BASE_IMAGE} AS functional

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install bash coreutils-single curl dnsmasq grep openssl sed socat util-linux && \
    microdnf clean all

COPY proxy/tests/run.sh /usr/local/bin/healthcheck-suite
ENTRYPOINT ["/usr/local/bin/healthcheck-suite"]
