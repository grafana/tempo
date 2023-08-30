//go:build tools

package tools

import (
	_ "github.com/google/go-jsonnet/cmd/jsonnetfmt"
	_ "github.com/psampaz/go-mod-outdated"
	_ "golang.org/x/tools/cmd/goimports"
	_ "gotest.tools/gotestsum"
	_ "mvdan.cc/gofumpt"
)
