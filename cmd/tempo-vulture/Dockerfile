FROM alpine:3.9 as certs
RUN apk --update add ca-certificates
COPY tempo-vulture /
ENTRYPOINT ["/tempo-vulture"]