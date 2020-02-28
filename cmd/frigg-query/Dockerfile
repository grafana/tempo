FROM jaegertracing/jaeger-query:1.17

ENV SPAN_STORAGE_TYPE=grpc-plugin \
    GRPC_STORAGE_PLUGIN_BINARY=/tmp/frigg-query

# This is silly, but it's important that frigg-query gets copied into /tmp
#  b/c it forces a /tmp dir to exist which hashicorp plugins depend on
COPY frigg-query /tmp/frigg-query