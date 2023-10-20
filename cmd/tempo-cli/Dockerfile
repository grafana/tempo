FROM alpine:3.18 as certs
RUN apk --update add ca-certificates
ARG TARGETARCH
COPY bin/linux/tempo-cli-${TARGETARCH} /tempo-cli
ENTRYPOINT ["/tempo-cli"]
