FROM alpine:latest@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM gcr.io/distroless/static-debian13:debug@sha256:a97e59debfbb00f104068c1de7d9c6d4163abc117d437d97797ba31b14e9ce9f

SHELL ["/busybox/sh", "-c"]

ARG TARGETARCH
COPY bin/linux/tempo-cli-${TARGETARCH} /tempo-cli
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/tempo-cli"]
