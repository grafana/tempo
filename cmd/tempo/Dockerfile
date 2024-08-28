FROM alpine:3.20 AS certs
RUN apk --update add ca-certificates
ARG TARGETARCH
COPY bin/linux/tempo-${TARGETARCH} /tempo

RUN addgroup -g 10001 -S tempo && \
    adduser -u 10001 -S tempo -G tempo

RUN mkdir -p /var/tempo && \
    chown -R tempo:tempo /var/tempo

USER 10001:10001

ENTRYPOINT ["/tempo"]
