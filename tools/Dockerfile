FROM golang:1.26.2-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN apk --update add --no-cache make git bash protoc

# Install tools with @version to avoid dependency cross-contamination
# These tools have transitive deps that conflict with the shared tools/go.mod graph
# Ref: https://buf.build/docs/cli/installation/
RUN GOBIN=/usr/local/bin go install github.com/bufbuild/buf/cmd/buf@v1.63.0
RUN GOBIN=/usr/local/bin go install github.com/gogo/protobuf/protoc-gen-gogofaster@v1.3.2
RUN GOBIN=/usr/local/bin go install github.com/grafana/tanka/cmd/tk@v0.36.3
RUN GOBIN=/usr/local/bin go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@v0.6.0

WORKDIR /tools
COPY tools /tools
RUN /tools/install.sh
COPY tools/entrypoint.sh /bin/entrypoint.sh
ENTRYPOINT [ "/bin/entrypoint.sh" ]
