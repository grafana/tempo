package docs

import _ "embed"

//go:embed sources/tempo/traceql/construct-traceql-queries.md
var TraceQLMain string

//go:embed sources/tempo/traceql/metrics-queries/functions.md
var TraceQLMetrics string
