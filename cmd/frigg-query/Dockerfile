FROM jaegertracing/all-in-one:1.16

ENV SPAN_STORAGE_TYPE=grpc-plugin \
    GRPC_STORAGE_PLUGIN_BINARY=/plugin/frigg-query

COPY frigg-query /plugin/frigg-query