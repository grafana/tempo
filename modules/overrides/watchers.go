package overrides

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/pkg/traceql"
)

// WatchAttributeCompileOptions builds the traceql compile options that install
// the tenant's configured report-attribute watchers. It is the single place
// query paths (search and metrics, in the querier and live-store) turn the
// per-tenant WatchAttributes override into engine watchers, so a new path only
// needs to append the result to its compile options.
//
// Specs that fail to build are skipped and logged (operator-controlled config).
// Returns nil when there is nothing to install.
func WatchAttributeCompileOptions(specs []WatchAttribute, logger log.Logger) []traceql.CompileOption {
	var watchers []traceql.SpanWatcher
	for _, s := range specs {
		o, err := traceql.NewWatcher(traceql.WatcherSpec{Attribute: s.Attribute, Type: s.Type})
		if err != nil {
			level.Warn(logger).Log("msg", "skipping invalid watch_attribute watcher", "attribute", s.Attribute, "type", s.Type, "err", err)
			continue
		}
		watchers = append(watchers, o)
	}
	if len(watchers) == 0 {
		return nil
	}
	return []traceql.CompileOption{traceql.WithWatchers(watchers...)}
}
