FROM golang:1.23.0-bookworm AS drone
ARG TARGETARCH
RUN curl -L "https://github.com/drone/drone-cli/releases/download/v1.7.0/drone_linux_$TARGETARCH".tar.gz | tar zx && \
    install -t /usr/local/bin drone

FROM golang:alpine

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN apk --update add --no-cache make git bash

WORKDIR /tools
COPY tools /tools
COPY --from=drone /usr/local/bin/drone /usr/bin/drone
RUN /tools/install.sh
COPY tools/entrypoint.sh /bin/entrypoint.sh
ENTRYPOINT [ "/bin/entrypoint.sh" ]
