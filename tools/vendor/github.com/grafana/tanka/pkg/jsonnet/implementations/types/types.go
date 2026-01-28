package types

// JsonnetEvaluator represents a struct that can evaluate Jsonnet code
// It is configured with import paths, external code and top-level arguments
type JsonnetEvaluator interface {
	EvaluateAnonymousSnippet(snippet string) (string, error)
	EvaluateFile(filename string) (string, error)
}

// JsonnetImplementation is a factory for JsonnetEvaluator
type JsonnetImplementation interface {
	MakeEvaluator(importPaths []string, extCode map[string]string, tlaCode map[string]string, maxStack int) JsonnetEvaluator
}
