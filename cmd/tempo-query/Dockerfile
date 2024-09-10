FROM alpine:3.20 as certs
RUN apk --update add ca-certificates
ARG TARGETARCH
COPY bin/linux/tempo-query-${TARGETARCH} /tempo-query

RUN addgroup -g 10001 -S tempo && \
    adduser -u 10001 -S tempo -G tempo

USER 10001:10001

COPY bin/linux/tempo-query-${TARGETARCH} /tempo-query

ENTRYPOINT ["/tempo-query"]
