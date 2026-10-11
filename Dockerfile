FROM --platform=$BUILDPLATFORM busybox:1.38.0@sha256:dc2d74b28e4cf8984fa52af1f39bc7c3d9c73760b41a74d629f5d11b1ab28616 AS builder

ARG TARGETPLATFORM

COPY ./dist ./dist

RUN case ${TARGETPLATFORM} in \
        "linux/amd64")    ARCH="amd64"    ;; \
        "linux/arm64")    ARCH="arm64"    ;; \
        "linux/riscv64")  ARCH="riscv64"  ;; \
        "linux/ppc64le")  ARCH="ppc64le"  ;; \
        "linux/s390x")    ARCH="s390x"    ;; \
        "linux/386")      ARCH="386"      ;; \
        "linux/arm/v7")   ARCH="armv7"   ;; \
        "linux/arm/v6")   ARCH="armv6"   ;; \
    esac && \
    mv ./dist/headless-broadcaster_linux_${ARCH}/headless-broadcaster /usr/bin/headless-broadcaster

FROM scratch AS final

COPY --from=builder /usr/bin/headless-broadcaster /usr/bin/headless-broadcaster

ENTRYPOINT ["/usr/bin/headless-broadcaster"]

LABEL org.opencontainers.image.source="https://github.com/zekihan/headless-broadcaster" \
    org.opencontainers.image.licenses="AGPL-3.0" \
    org.opencontainers.image.title="Headless Broadcaster" \
    org.opencontainers.image.description="Broadcast HTTP requests to every IP address resolved for a hostname."
