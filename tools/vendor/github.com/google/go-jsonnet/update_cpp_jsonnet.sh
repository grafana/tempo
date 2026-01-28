#!/bin/bash

# Updates cpp-jsonnet repo and regenerates dependent files

set -e
set -x

cd cpp-jsonnet
git checkout master
git pull
hash=$(git rev-parse HEAD)
cd ..
go run cmd/dumpstdlibast/dumpstdlibast.go cpp-jsonnet/stdlib/std.jsonnet > astgen/stdast.go

sha256=$(curl -fL https://github.com/google/jsonnet/archive/$hash.tar.gz | shasum -a 256 | awk '{print $1}')

sed -i.bak \
    -e "s/CPP_JSONNET_SHA256 = .*/CPP_JSONNET_SHA256 = \"$sha256\"/;" \
    -e "s/CPP_JSONNET_GITHASH = .*/CPP_JSONNET_GITHASH = \"$hash\"/;" \
    bazel/repositories.bzl

# NB: macOS sed doesn't support -i without arg. This is the easy workaround.
rm bazel/repositories.bzl.bak

set +x
echo
echo -e "\033[1mUpdate completed. Please check if any tests are broken and fix any encountered issues.\033[0m"
