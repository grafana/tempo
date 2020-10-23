FROM alpine:3.9 as certs
RUN apk --update add ca-certificates
COPY cmd/tempo/tempo /
ENTRYPOINT ["/tempo"]
