FROM jaegertracing/jaeger-query:1.17

ENV SPAN_STORAGE_TYPE=grpc-plugin \
    GRPC_STORAGE_PLUGIN_BINARY=/tmp/tempo-query

# This is silly, but it's important that tempo-query gets copied into /tmp
#  b/c it forces a /tmp dir to exist which hashicorp plugins depend on
COPY tempo-query /tmp/tempo-query