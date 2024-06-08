FROM alpine:3.20 as certs
RUN apk --update add ca-certificates
ARG TARGETARCH
COPY bin/linux/tempo-vulture-${TARGETARCH} /tempo-vulture

RUN addgroup -g 10001 -S tempo && \
    adduser -u 10001 -S tempo -G tempo

USER 10001:10001

ENTRYPOINT ["/tempo-vulture"]
