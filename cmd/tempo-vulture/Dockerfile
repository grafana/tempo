FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.15.3-alpine AS builder

RUN apk --update add ca-certificates make git

WORKDIR /go/src
COPY ./ ./

ARG TARGETARCH
RUN GOARCH=${TARGETARCH} make tempo-vulture

FROM alpine:3.9

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ARG TARGETARCH
COPY --from=builder /go/src/bin/linux/tempo-vulture-${TARGETARCH} /tempo-vulture

ENTRYPOINT ["/tempo-vulture"]