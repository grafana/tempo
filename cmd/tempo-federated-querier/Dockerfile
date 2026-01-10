FROM alpine:latest@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62 AS ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM gcr.io/distroless/static-debian12:debug@sha256:4ca65de9063e5b25e07b8585cc9d70d4aca9e28dfc6dcdcfcc9bb79acda55678

SHELL ["/busybox/sh", "-c"]

ARG TARGETARCH
COPY bin/linux/tempo-federated-querier-${TARGETARCH} /tempo-federated-querier
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 3200

ENTRYPOINT ["/tempo-federated-querier"]
