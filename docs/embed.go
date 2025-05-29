package docs

import _ "embed"

//go:embed sources/tempo/shared/traceql-main.md
var TraceQLMain string

//go:embed sources/tempo/traceql/metrics-queries/functions.md
var TraceQLMetrics string
