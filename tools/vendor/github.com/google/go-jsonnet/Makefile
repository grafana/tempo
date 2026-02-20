all: install.dependencies generate generate.stdlib build.bazel test tidy
.PHONY: all

install.dependencies:
	git submodule init
	git submodule update
.PHONY: install.dependencies

build.bazel:
	bazel build //cmd/jsonnet
.PHONY: build.bazel

_build.bazel.os:
	bazel build --platforms=@io_bazel_rules_go//go/toolchain:$(OS)_amd64 //cmd/jsonnet
.PHONY: build.bazel.os

build.bazel.linux : OS=linux
build.bazel.linux: _build.bazel.os
.PHONY: build.bazel.linux

build.bazel.darwin : OS=darwin
build.bazel.darwin: _build.bazel.os
.PHONY: build.bazel.darwin


build.bazel.windows : OS=windows
build.bazel.windows: _build.bazel.os
.PHONY: build.bazel.windows

build:
	go build ./cmd/jsonnet
.PHONY: build

build.old:
	go build -o jsonnet-old ./cmd/jsonnet
.PHONY: build.old

test:
	./tests.sh
.PHONY: test

benchmark : FILTER ?= Builtin
benchmark: build
	./benchmark.sh ${FILTER}
.PHONY: benchmark

generate:
	go generate
.PHONY: generate

generate.stdlib:
	go run cmd/dumpstdlibast/dumpstdlibast.go cpp-jsonnet/stdlib/std.jsonnet > astgen/stdast.go
.PHONY: generate.stdlib

tidy:
	go mod tidy
	bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=bazel/deps.bzl%jsonnet_go_dependencies
.PHONY: tidy

gazelle:
	bazel run //:gazelle
.PHONY: gazelle
