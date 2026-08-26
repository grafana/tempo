package overrides

import (
	"github.com/grafana/tempo/pkg/traceql"
)

// SpanPruningAwarenessCompileOptions builds a CompileOption that install the span-pruning awareness watcher.
func SpanPruningAwarenessCompileOptions(enabled bool) []traceql.CompileOption {
	if !enabled {
		return nil
	}
	return []traceql.CompileOption{traceql.WithWatchers(traceql.NewSpanPruningWatcher())}
}
