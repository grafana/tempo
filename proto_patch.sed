# Update package and types from opentelemetry.proto.* -> tempopb.*
# giving final types like "tempopb.common.v1.InstrumentationLibrary" which
# will not conflict with other usages of opentelemetry proto in downstream apps.
s+ opentelemetry.proto+ tempopb+g

# Update go_package
s+go.opentelemetry.io/proto/otlp+github.com/grafana/tempo/pkg/tempopb+g

# Update import paths
s+import "opentelemetry/proto/+import "+g

# Import gogoproto
s+package tempopb\.\(.*\)\.v\(.*\);+package tempopb.\1.v\2;\
\
import "github.com/gogo/protobuf/gogoproto/gogo.proto";+g

# Custom type for Trace ID
s+bytes trace_id = \(.*\);+bytes trace_id = \1\
  [\
  // Use custom TraceId data type for this field.\
  (gogoproto.nullable) = false,\
  (gogoproto.customtype) = "TraceID"\
  ];+g

# Custom type for Span ID
s+bytes \(.*span_id\) = \(.*\);+bytes \1 = \2\
  [\
  // Use custom SpanId data type for this field.\
  (gogoproto.nullable) = false,\
  (gogoproto.customtype) = "SpanID"\
  ];+g
