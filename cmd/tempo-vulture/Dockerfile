FROM alpine:latest@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS ca-certificates
RUN apk add --update --no-cache ca-certificates

FROM gcr.io/distroless/static-debian12:debug@sha256:7985579713fb1171e707d74659c67af3605642d1c9db305304c2998a99032615 AS tempo-setup

RUN ["/busybox/addgroup", "-g", "10001", "-S", "tempo"]
RUN ["/busybox/adduser", "-u", "10001", "-S", "tempo", "-G", "tempo"]

FROM gcr.io/distroless/static-debian12@sha256:20bc6c0bc4d625a22a8fde3e55f6515709b32055ef8fb9cfbddaa06d1760f838

ARG TARGETARCH
COPY bin/linux/tempo-vulture-${TARGETARCH} /tempo-vulture
COPY --from=ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=tempo-setup /etc/passwd /etc/passwd
COPY --from=tempo-setup /etc/group /etc/group

USER 10001:10001

ENTRYPOINT ["/tempo-vulture"]
