FROM alpine:latest@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412 AS ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM gcr.io/distroless/static-debian12:debug@sha256:7dc183cc0aea6abd9d105135e49d37b7474a79391ebea7eb55557cd4486d2225

SHELL ["/busybox/sh", "-c"]

ARG TARGETARCH
COPY bin/linux/tempo-cli-${TARGETARCH} /tempo-cli
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/tempo-cli"]
