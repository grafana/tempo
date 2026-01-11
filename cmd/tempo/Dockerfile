FROM alpine:latest@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62 AS ca-certificates
RUN apk add --update --no-cache ca-certificates

# Use the same image with busybox for the setup to ensure the /etc files are compatible.
FROM gcr.io/distroless/static-debian12:debug@sha256:20d9c135406d8029d30d59eaaa9d62d2edcd4ec5915dbcda324243c40460e8df AS tempo-setup

RUN ["/busybox/addgroup", "-g", "10001", "-S", "tempo"]
RUN ["/busybox/adduser", "-u", "10001", "-S", "tempo", "-G", "tempo"]
RUN ["/busybox/mkdir", "-p", "/var/tempo"]

FROM gcr.io/distroless/static-debian12@sha256:cd64bec9cec257044ce3a8dd3620cf83b387920100332f2b041f19c4d2febf93

ARG TARGETARCH
COPY bin/linux/tempo-${TARGETARCH} /tempo
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=tempo-setup /etc/passwd /etc/passwd
COPY --from=tempo-setup /etc/group /etc/group
COPY --from=tempo-setup --chown=10001:10001 --chmod=0700 /var/tempo /var/tempo

USER 10001:10001

ENTRYPOINT ["/tempo"]
