FROM debian:bookworm-slim

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG MINIO_VERSION=RELEASE.2024-12-18T13-15-44Z
ARG TARGETARCH

ENV http_proxy=${HTTP_PROXY} \
    https_proxy=${HTTPS_PROXY}

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates wget \
    && case "${TARGETARCH}" in \
         amd64) ARCH=amd64 ;; \
         arm64) ARCH=arm64 ;; \
         *) ARCH=amd64 ;; \
       esac \
    && wget -q "https://dl.min.io/server/minio/release/linux-${ARCH}/archive/minio.${MINIO_VERSION}" -O /usr/local/bin/minio \
    && chmod +x /usr/local/bin/minio \
    && apt-get purge -y wget \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --no-create-home --shell /usr/sbin/nologin minio
USER minio
WORKDIR /data
VOLUME ["/data"]

EXPOSE 9000 9001

ENTRYPOINT ["/usr/local/bin/minio"]
CMD ["server", "/data", "--console-address", ":9001"]
