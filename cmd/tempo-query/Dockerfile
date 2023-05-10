FROM jaegertracing/jaeger-query:1.43.0

ENV SPAN_STORAGE_TYPE=grpc-plugin \
    GRPC_STORAGE_PLUGIN_BINARY=/tempo-query

# This is silly, but it's important b/c it forces a /tmp dir to exist which hashicorp plugins depend on
RUN mkdir -p /tmp

ARG TARGETARCH
COPY bin/linux/tempo-query-${TARGETARCH} /tempo-query
