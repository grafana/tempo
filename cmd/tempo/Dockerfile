FROM alpine:latest@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS ca-certificates
RUN apk add --update --no-cache ca-certificates

# Use the same image with busybox for the setup to ensure the /etc files are compatible.
FROM gcr.io/distroless/static-debian12:debug@sha256:46fcf1fa44d251b0944ba4b98ef4bbd266e33034e46489dbf92f680ca4917451 AS tempo-setup

RUN ["/busybox/addgroup", "-g", "10001", "-S", "tempo"]
RUN ["/busybox/adduser", "-u", "10001", "-S", "tempo", "-G", "tempo"]
RUN ["/busybox/mkdir", "-p", "/var/tempo"]

FROM gcr.io/distroless/static-debian12@sha256:9c346e4be81b5ca7ff31a0d89eaeade58b0f95cfd3baed1f36083ddb47ca3160

ARG TARGETARCH
COPY bin/linux/tempo-${TARGETARCH} /tempo
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=tempo-setup /etc/passwd /etc/passwd
COPY --from=tempo-setup /etc/group /etc/group
COPY --from=tempo-setup --chown=10001:10001 --chmod=0700 /var/tempo /var/tempo

USER 10001:10001

ENTRYPOINT ["/tempo"]
