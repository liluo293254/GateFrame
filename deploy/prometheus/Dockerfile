FROM debian:bookworm-slim

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG PROMETHEUS_VERSION=2.54.1
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
    && wget -q "https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/prometheus-${PROMETHEUS_VERSION}.linux-${ARCH}.tar.gz" -O /tmp/prometheus.tar.gz \
    && tar -xzf /tmp/prometheus.tar.gz -C /tmp \
    && mv "/tmp/prometheus-${PROMETHEUS_VERSION}.linux-${ARCH}/prometheus" /usr/local/bin/prometheus \
    && mv "/tmp/prometheus-${PROMETHEUS_VERSION}.linux-${ARCH}/promtool" /usr/local/bin/promtool \
    && rm -rf /tmp/prometheus* \
    && apt-get purge -y wget \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --no-create-home --shell /usr/sbin/nologin prometheus
USER prometheus
WORKDIR /prometheus
EXPOSE 9090
VOLUME ["/prometheus"]

ENTRYPOINT ["/usr/local/bin/prometheus"]
CMD ["--config.file=/etc/prometheus/prometheus.yml", "--storage.tsdb.path=/prometheus", "--web.enable-lifecycle"]
