FROM alpine:3.19 as certs
RUN apk --update add ca-certificates
ARG TARGETARCH
COPY bin/linux/tempo-${TARGETARCH} /tempo
ENTRYPOINT ["/tempo"]
