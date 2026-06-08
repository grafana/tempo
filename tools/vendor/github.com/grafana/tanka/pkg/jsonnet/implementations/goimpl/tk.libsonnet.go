package goimpl

import jsonnet "github.com/google/go-jsonnet"

var tkLibsonnet = jsonnet.MakeContents(`
{
  env: std.extVar("tanka.dev/environment"),
}
`)
