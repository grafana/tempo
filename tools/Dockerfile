# Some tools are only supported on specific versions of Go.  The image version here tracks the main go.mod of the project.
FROM golang:1.21-alpine

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN apk --update add --no-cache make git bash

WORKDIR /tools
COPY tools /tools
RUN /tools/install.sh
COPY tools/entrypoint.sh /bin/entrypoint.sh
ENTRYPOINT [ "/bin/entrypoint.sh" ]
