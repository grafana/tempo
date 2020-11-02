# Custom build for jaeger-query
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.15.3-alpine AS build-env-jaeger-query

RUN apk --update add ca-certificates make git

ARG JAEGER_VERSION=1.20.0
RUN git clone --depth=1 -b v${JAEGER_VERSION} https://github.com/jaegertracing/jaeger.git /go/src/
WORKDIR /go/src/

RUN git submodule update --init

# build ui
FROM --platform=${BUILDPLATFORM:-linux/amd64} node AS ui-jaeger-query

COPY --from=build-env-jaeger-query /go/src/jaeger-ui /go/src/jaeger-ui
WORKDIR /go/src/jaeger-ui
RUN yarn install --frozen-lockfile && cd packages/jaeger-ui && yarn build

# build jaeger-query
FROM --platform=${BUILDPLATFORM:-linux/amd64} build-env-jaeger-query AS builder-jaeger-query

COPY --from=ui-jaeger-query /go/src/jaeger-ui/packages/jaeger-ui/build /go/src/jaeger-ui/packages/jaeger-ui/build

RUN go get -u github.com/mjibson/esc

RUN esc -pkg assets -o cmd/query/app/ui/actual/gen_assets.go -prefix jaeger-ui/packages/jaeger-ui/build jaeger-ui/packages/jaeger-ui/build
RUN esc -pkg assets -o cmd/query/app/ui/placeholder/gen_assets.go -prefix cmd/query/app/ui/placeholder/public cmd/query/app/ui/placeholder/public

ARG TARGETARCH
RUN make build-query GOARCH=${TARGETARCH}

FROM alpine:3.9 AS jaeger-query

ARG TARGETARCH
COPY --from=builder-jaeger-query /go/src/cmd/query/query-linux-${TARGETARCH} /go/bin/query-linux
COPY --from=builder-jaeger-query /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 16686/tcp
ENTRYPOINT ["/go/bin/query-linux"]

# TODO using back jaegertracing/jaeger-query, when upstream support multi-arch, and remove all above
# FROM jaegertracing/jaeger-query:1.20.0 AS jaeger-query

FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.15.3-alpine AS builder

RUN apk --update add make git

WORKDIR /go/src
COPY ./ ./

ARG TARGETARCH
RUN GOARCH=${TARGETARCH} make tempo-query

FROM jaeger-query

ENV SPAN_STORAGE_TYPE=grpc-plugin \
    GRPC_STORAGE_PLUGIN_BINARY=/tmp/tempo-query

# This is silly, but it's important that tempo-query gets copied into /tmp
#  b/c it forces a /tmp dir to exist which hashicorp plugins depend on
ARG TARGETARCH
COPY --from=builder /go/src/bin/linux/tempo-query-${TARGETARCH} /tmp/tempo-query