FROM alpine:latest@sha256:3f4d6b32e26703c6082839234f43769d5046b3ac7c910033911d1e71d8cd630a AS ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM gcr.io/distroless/static-debian13:debug@sha256:63b4c0065ff7bd68753510400b2d5cedd5d5571d15eac4be0b6b0dd236a2d148

SHELL ["/busybox/sh", "-c"]

ARG TARGETARCH
COPY bin/linux/tempo-cli-${TARGETARCH} /tempo-cli
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/tempo-cli"]
