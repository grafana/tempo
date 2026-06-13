FROM alpine:latest@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4 AS ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM gcr.io/distroless/static-debian12:debug@sha256:46fcf1fa44d251b0944ba4b98ef4bbd266e33034e46489dbf92f680ca4917451

SHELL ["/busybox/sh", "-c"]

ARG TARGETARCH
COPY bin/linux/tempo-cli-${TARGETARCH} /tempo-cli
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/tempo-cli"]
