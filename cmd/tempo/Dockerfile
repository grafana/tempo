FROM alpine:latest@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659 AS ca-certificates
RUN apk add --update --no-cache ca-certificates

# Use the same image with busybox for the setup to ensure the /etc files are compatible.
FROM gcr.io/distroless/static-debian12:debug@sha256:7985579713fb1171e707d74659c67af3605642d1c9db305304c2998a99032615 AS tempo-setup

RUN ["/busybox/addgroup", "-g", "10001", "-S", "tempo"]
RUN ["/busybox/adduser", "-u", "10001", "-S", "tempo", "-G", "tempo"]
RUN ["/busybox/mkdir", "-p", "/var/tempo"]

FROM gcr.io/distroless/static-debian12@sha256:20bc6c0bc4d625a22a8fde3e55f6515709b32055ef8fb9cfbddaa06d1760f838

ARG TARGETARCH
COPY bin/linux/tempo-${TARGETARCH} /tempo
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=tempo-setup /etc/passwd /etc/passwd
COPY --from=tempo-setup /etc/group /etc/group
COPY --from=tempo-setup --chown=10001:10001 --chmod=0700 /var/tempo /var/tempo

USER 10001:10001

ENTRYPOINT ["/tempo"]
