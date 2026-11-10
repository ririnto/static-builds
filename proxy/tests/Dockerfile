ARG FUNCTIONAL_BASE_IMAGE
FROM ${FUNCTIONAL_BASE_IMAGE} AS functional

RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf -y install bash coreutils-single curl dnsmasq grep openssl sed socat util-linux
RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf clean all

COPY proxy/tests/run.sh /usr/local/libexec/healthcheck-suite/run.sh
COPY proxy/tests/fixtures/ /usr/local/libexec/healthcheck-suite/fixtures/
ENTRYPOINT ["/usr/local/libexec/healthcheck-suite/run.sh"]
