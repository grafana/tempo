FROM alpine:latest@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62 AS ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM gcr.io/distroless/static-debian12:debug@sha256:20d9c135406d8029d30d59eaaa9d62d2edcd4ec5915dbcda324243c40460e8df

SHELL ["/busybox/sh", "-c"]

RUN ["/busybox/addgroup", "-g", "10001", "-S", "tempo"]
RUN ["/busybox/adduser", "-u", "10001", "-S", "tempo", "-G", "tempo"]

ARG TARGETARCH
COPY bin/linux/tempo-query-${TARGETARCH} /tempo-query
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER 10001:10001

ENTRYPOINT ["/tempo-query"]
