FROM alpine:3.9 as certs
RUN apk --update add ca-certificates
# RUN apk add --no-cache git make build-base zstd-dev
ARG TARGETARCH
COPY bin/linux/tempo-${TARGETARCH} /tempo
ENTRYPOINT ["/tempo"]


