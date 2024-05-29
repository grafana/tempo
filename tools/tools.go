//go:build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/google/go-jsonnet/cmd/jsonnet"
	_ "github.com/google/go-jsonnet/cmd/jsonnetfmt"
	_ "github.com/grafana/tanka/cmd/tk"
	_ "github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb"
	_ "github.com/psampaz/go-mod-outdated"
	_ "golang.org/x/tools/cmd/goimports"
	_ "gotest.tools/gotestsum"
	_ "mvdan.cc/gofumpt"
)
