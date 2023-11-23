# Span Event Context

The Span Event Context is a Context implementation for [pdata SpanEvents](https://github.com/open-telemetry/opentelemetry-collector/blob/main/pdata/ptrace/generated_spanevent.go), the Collector's internal representation for OTLP Span Event data.  This Context should be used when interacting with individual OTLP Span Events.

## Paths
In general, the Span Event Context supports accessing pdata using the field names from the [traces proto](https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto).  All integers are returned and set via `int64`.  All doubles are returned and set via `float64`.

The following paths are supported.

| path                                   | field accessed                                                                                                                                                                | type                                                                    |
|----------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------|
| cache                                  | the value of the current transform context's temporary cache. cache can be used as a temporary placeholder for data during complex transformations                            | pcommon.Map                                                             |
| cache\[""\]                            | the value of an item in cache. Supports multiple indexes to access nested fields.                                                                                             | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| resource                               | resource of the span event being processed                                                                                                                                    | pcommon.Resource                                                        |
| resource.attributes                    | resource attributes of the span event being processed                                                                                                                         | pcommon.Map                                                             |
| resource.attributes\[""\]              | the value of the resource attribute of the span event being processed. Supports multiple indexes to access nested fields.                                                     | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| instrumentation_scope                  | instrumentation scope of the span event being processed                                                                                                                       | pcommon.InstrumentationScope                                            |
| instrumentation_scope.name             | name of the instrumentation scope of the span event being processed                                                                                                           | string                                                                  |
| instrumentation_scope.version          | version of the instrumentation scope of the span event being processed                                                                                                        | string                                                                  |
| instrumentation_scope.attributes       | instrumentation scope attributes of the span event being processed                                                                                                            | pcommon.Map                                                             |
| instrumentation_scope.attributes\[""\] | the value of the instrumentation scope attribute of the span event being processed. Supports multiple indexes to access nested fields.                                        | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| span                                   | span of the span event being processed                                                                                                                                        | ptrace.Span                                                             |
| span.*                                 | All fields exposed by the [ottlspan context](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlspan) can accessed via `span.` | varies                                                                  |
| attributes                             | attributes of the span event being processed                                                                                                                                  | pcommon.Map                                                             |
| attributes\[""\]                       | the value of the attribute of the span event being processed. Supports multiple indexes to access nested fields.                                                              | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| time_unix_nano                         | time_unix_nano of the span event being processed                                                                                                                              | int64                                                                   |
| time                                   | time of the span event being processed                                                                                                                                        | `time.Time`                                                             |
| name                                   | name of the span event being processed                                                                                                                                        | string                                                                  |
| dropped_attributes_count               | dropped_attributes_count of the span event being processed                                                                                                                    | int64                                                                   |

## Enums

The Span Event Context supports the enum names from the [traces proto](https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto).

| Enum Symbol           | Value |
|-----------------------|-------|
| SPAN_KIND_UNSPECIFIED | 0     |
| SPAN_KIND_INTERNAL    | 1     |
| SPAN_KIND_SERVER      | 2     |
| 	SPAN_KIND_CLIENT     | 3     |
| 	SPAN_KIND_PRODUCER   | 4     |
| 	SPAN_KIND_CONSUMER   | 5     |
| 	STATUS_CODE_UNSET    | 0     |
| 	STATUS_CODE_OK       | 1     |
| 	STATUS_CODE_ERROR    | 2     |
