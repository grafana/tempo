FROM alpine:latest@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM gcr.io/distroless/static-debian13:debug@sha256:6a5e9a4921c51d613ab8128402badf19d27b16627f92f97146f4597d2c593104 AS tempo-setup

RUN ["/busybox/addgroup", "-g", "10001", "-S", "tempo"]
RUN ["/busybox/adduser", "-u", "10001", "-S", "tempo", "-G", "tempo"]

FROM gcr.io/distroless/static-debian13@sha256:15f8637509792e05b97d83451b0b6ccfa4abd548f997e406ac1e72c87fe1fa78

ARG TARGETARCH
COPY bin/linux/tempo-vulture-${TARGETARCH} /tempo-vulture
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=tempo-setup /etc/passwd /etc/passwd
COPY --from=tempo-setup /etc/group /etc/group

USER 10001:10001

ENTRYPOINT ["/tempo-vulture"]
