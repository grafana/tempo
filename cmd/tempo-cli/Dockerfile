FROM alpine:latest@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM gcr.io/distroless/static-debian12:debug@sha256:ecbc4ac563b95a132fd370065553857dcc38943066cb0662d09999011f3d3597

SHELL ["/busybox/sh", "-c"]

ARG TARGETARCH
COPY bin/linux/tempo-cli-${TARGETARCH} /tempo-cli
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/tempo-cli"]
