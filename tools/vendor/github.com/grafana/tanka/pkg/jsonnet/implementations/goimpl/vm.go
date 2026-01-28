package goimpl

import (
	"github.com/google/go-jsonnet"
	"github.com/grafana/tanka/pkg/jsonnet/native"
)

// MakeRawVM returns a Jsonnet VM with some extensions of Tanka, including:
// - extended importer
// - extCode and tlaCode applied
// - native functions registered
// This is exposed because Go is used for advanced use cases, like finding transitive imports or linting.
func MakeRawVM(importPaths []string, extCode map[string]string, tlaCode map[string]string, maxStack int) *jsonnet.VM {
	vm := jsonnet.MakeVM()
	vm.Importer(newExtendedImporter(importPaths))

	for k, v := range extCode {
		vm.ExtCode(k, v)
	}
	for k, v := range tlaCode {
		vm.TLACode(k, v)
	}

	for _, nf := range native.Funcs() {
		vm.NativeFunction(nf)
	}

	if maxStack > 0 {
		vm.MaxStack = maxStack
	}

	return vm
}
