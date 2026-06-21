FROM golang:1.26.4-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN apk --update add --no-cache make git bash

WORKDIR /tools
COPY tools /tools
RUN /tools/install.sh
COPY tools/entrypoint.sh /bin/entrypoint.sh
ENTRYPOINT [ "/bin/entrypoint.sh" ]
