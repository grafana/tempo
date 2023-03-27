FROM alpine:3.19 as certs
RUN apk --update add ca-certificates
ARG TARGETARCH
COPY bin/linux/tempo-${TARGETARCH} /tempo

RUN addgroup -g 10001 -S tempo && \
    adduser -u 10001 -S tempo -G tempo

RUN chown -R tempo:tempo /var/tempo

USER 10001:10001

ENTRYPOINT ["/tempo"]
