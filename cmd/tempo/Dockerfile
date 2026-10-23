FROM alpine:latest@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS ca-certificates
RUN apk add --update --no-cache ca-certificates

# Use the same image with busybox for the setup to ensure the /etc files are compatible.
FROM gcr.io/distroless/static-debian13:debug@sha256:11c839f18a0d7b91b093cda53a5d4cee84f23ec450a6f6e1bf17ab12bf296333 AS tempo-setup

RUN ["/busybox/addgroup", "-g", "10001", "-S", "tempo"]
RUN ["/busybox/adduser", "-u", "10001", "-S", "tempo", "-G", "tempo"]
RUN ["/busybox/mkdir", "-p", "/var/tempo"]

FROM gcr.io/distroless/static-debian13@sha256:034df5c1737e8241f6ed6423a12e720ee0f6d6f088609ee172f98f6f2166ef5d

ARG TARGETARCH
COPY bin/linux/tempo-${TARGETARCH} /tempo
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=tempo-setup /etc/passwd /etc/passwd
COPY --from=tempo-setup /etc/group /etc/group
COPY --from=tempo-setup --chown=10001:10001 --chmod=0700 /var/tempo /var/tempo

USER 10001:10001

ENTRYPOINT ["/tempo"]
