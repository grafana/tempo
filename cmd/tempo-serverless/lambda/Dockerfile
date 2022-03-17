FROM alpine:3.9 as certs
RUN apk --update add ca-certificates
ARG TARGETARCH
COPY bin/linux/tempo-serverless-lambda-${TARGETARCH} /main
ENTRYPOINT ["/main"]
