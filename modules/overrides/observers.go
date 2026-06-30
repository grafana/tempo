package overrides

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/pkg/traceql"
)

// ReportAttributeCompileOptions builds the traceql compile options that install
// the tenant's configured report-attribute observers. It is the single place
// query paths (search and metrics, in the querier and live-store) turn the
// per-tenant ReportAttributes override into engine observers, so a new path only
// needs to append the result to its compile options.
//
// Specs that fail to build are skipped and logged (operator-controlled config).
// Returns nil when there is nothing to install.
func ReportAttributeCompileOptions(specs []ReportAttribute, logger log.Logger) []traceql.CompileOption {
	var observers []traceql.SpanObserver
	for _, s := range specs {
		o, err := traceql.NewObserver(traceql.ObserverSpec{Attribute: s.Attribute, Type: s.Type})
		if err != nil {
			level.Warn(logger).Log("msg", "skipping invalid report_attribute observer", "attribute", s.Attribute, "type", s.Type, "err", err)
			continue
		}
		observers = append(observers, o)
	}
	if len(observers) == 0 {
		return nil
	}
	return []traceql.CompileOption{traceql.WithObservers(observers...)}
}
