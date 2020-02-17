#!/usr/bin/env bash

set -e

DIR="${DIR:-/tmp}"
cd ${DIR}

wget https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v1.9.6/protoc-gen-swagger-v1.9.6-linux-x86_64 \
    && chmod +x protoc-gen-swagger-v1.9.6-linux-x86_64 \
    && sudo ln -s ${DIR}/protoc-gen-swagger-v1.9.6-linux-x86_64 /usr/bin/protoc-gen-swagger

wget https://github.com/protocolbuffers/protobuf/releases/download/v3.9.1/protoc-3.9.1-linux-x86_64.zip \
    && unzip protoc-3.9.1-linux-x86_64.zip \
    && sudo ln -s ${DIR}/bin/protoc /usr/bin/protoc

GIT_TAG="v1.3.2"
go get -d -u github.com/golang/protobuf/protoc-gen-go
git -C "$(go env GOPATH)"/src/github.com/golang/protobuf checkout $GIT_TAG
go install github.com/golang/protobuf/protoc-gen-go