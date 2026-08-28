FROM golang:1.27.0-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN apk --update add --no-cache make git bash protoc

# Install tools with @version to avoid dependency cross-contamination
# These tools have transitive deps that conflict with the shared tools/go.mod graph
# Ref: https://buf.build/docs/cli/installation/
RUN GOBIN=/usr/local/bin go install github.com/bufbuild/buf/cmd/buf@v1.63.0
RUN GOBIN=/usr/local/bin go install github.com/gogo/protobuf/protoc-gen-gogofaster@v1.3.2
RUN GOBIN=/usr/local/bin go install github.com/grafana/wiresmith/cmd/wiresmith@v0.0.0-20260618160438-f15959a1e4e7
RUN GOBIN=/usr/local/bin go install github.com/grafana/tanka/cmd/tk@v0.36.3
RUN GOBIN=/usr/local/bin go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@v0.6.0

WORKDIR /tools
COPY tools /tools
RUN /tools/install.sh
COPY tools/entrypoint.sh /bin/entrypoint.sh
ENTRYPOINT [ "/bin/entrypoint.sh" ]
