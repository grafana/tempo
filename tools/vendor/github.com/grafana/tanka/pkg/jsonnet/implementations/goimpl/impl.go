package goimpl

import (
	"github.com/google/go-jsonnet"
	"github.com/grafana/tanka/pkg/jsonnet/implementations/types"
)

type JsonnetGoVM struct {
	vm *jsonnet.VM

	path string
}

func (vm *JsonnetGoVM) EvaluateAnonymousSnippet(snippet string) (string, error) {
	return vm.vm.EvaluateAnonymousSnippet(vm.path, snippet)
}

func (vm *JsonnetGoVM) EvaluateFile(filename string) (string, error) {
	return vm.vm.EvaluateFile(filename)
}

type JsonnetGoImplementation struct {
	Path string
}

func (i *JsonnetGoImplementation) MakeEvaluator(importPaths []string, extCode map[string]string, tlaCode map[string]string, maxStack int) types.JsonnetEvaluator {
	return &JsonnetGoVM{
		vm: MakeRawVM(importPaths, extCode, tlaCode, maxStack),

		path: i.Path,
	}
}
