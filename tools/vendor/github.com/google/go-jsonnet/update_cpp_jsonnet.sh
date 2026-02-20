#!/bin/bash

# Updates cpp-jsonnet repo and regenerates dependent files

set -e
set -x

cd cpp-jsonnet
git remote update --prune

if [[ $# -gt 0 ]]; then
    WANT_VERSION_NAME="$1"
    WANT_VERSION_REF=refs/tags/"$WANT_VERSION_NAME"
else
    WANT_VERSION_NAME=
    WANT_VERSION_REF=refs/remotes/origin/master
fi

hash="$(git rev-parse "$WANT_VERSION_REF")"
git checkout "$hash"

if [[ -z "$WANT_VERSION_NAME" ]]; then
    ARCHIVE_URL="https://github.com/google/jsonnet/archive/${hash}.tar.gz"
else
    ARCHIVE_URL="https://github.com/google/jsonnet/releases/download/${WANT_VERSION_NAME}/jsonnet-${WANT_VERSION_NAME}.tar.gz"
fi

cd ..
go run cmd/dumpstdlibast/dumpstdlibast.go cpp-jsonnet/stdlib/std.jsonnet > astgen/stdast.go

sha256=$(curl -fL "${ARCHIVE_URL}" | shasum -a 256 | awk '{print $1}')

sed -i.bak \
    -e "s/CPP_JSONNET_SHA256 = .*/CPP_JSONNET_SHA256 = \"$sha256\"/;" \
    -e "s/CPP_JSONNET_GITHASH = .*/CPP_JSONNET_GITHASH = \"$hash\"/;" \
    -e "s/CPP_JSONNET_RELEASE_VERSION = .*/CPP_JSONNET_RELEASE_VERSION = \"$WANT_VERSION_NAME\"/;" \
    bazel/repositories.bzl MODULE.bazel

# NB: macOS sed doesn't support -i without arg. This is the easy workaround.
rm bazel/repositories.bzl.bak
rm MODULE.bazel.bak

set +x
echo
echo -e "\033[1mUpdate completed. Please check if any tests are broken and fix any encountered issues.\033[0m"
