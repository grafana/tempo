FROM alpine:3.9 as certs
RUN apk --update add ca-certificates
COPY cmd/tempo-vulture/tempo-vulture /
ENTRYPOINT ["/tempo-vulture"]