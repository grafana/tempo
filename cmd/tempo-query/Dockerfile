FROM jaegertracing/jaeger-query:1.58.1

# Ensure /tmp dir exists, hashicorp plugins need a /tmp dir to exist.
RUN mkdir -p /tmp

ARG TARGETARCH
COPY bin/linux/tempo-query-${TARGETARCH} /tempo-query
