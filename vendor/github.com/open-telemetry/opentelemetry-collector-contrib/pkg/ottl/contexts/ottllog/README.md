# Log Context

The Log Context is a Context implementation for [pdata Logs](https://github.com/open-telemetry/opentelemetry-collector/tree/main/pdata/plog), the collector's internal representation for OTLP log data.  This Context should be used when interacted with OTLP logs.

## Paths
In general, the Log Context supports accessing pdata using the field names from the [logs proto](https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/logs/v1/logs.proto).  All integers are returned and set via `int64`.  All doubles are returned and set via `float64`.

All TraceIDs and SpanIDs are returned as pdata [SpanID](https://github.com/open-telemetry/opentelemetry-collector/blob/main/pdata/pcommon/spanid.go) and [TraceID](https://github.com/open-telemetry/opentelemetry-collector/blob/main/pdata/pcommon/traceid.go) types.  Use the [SpanID function](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#spanid) and [TraceID function](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#traceid) when interacting with pdata representations of SpanID and TraceID.  When checking for nil, instead check against an empty byte slice (`SpanID(0x0000000000000000)` and `TraceID(0x00000000000000000000000000000000)`).

The following paths are supported.

| path                                           | field accessed                                                                                                                                     | type                                                                    |
|------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------|
| cache                                          | the value of the current transform context's temporary cache. cache can be used as a temporary placeholder for data during complex transformations | pcommon.Map                                                             |
| cache\[""\]                                    | the value of an item in cache. Supports multiple indexes to access nested fields.                                                                  | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| resource                                       | resource of the log being processed                                                                                                                | pcommon.Resource                                                        |
| resource.attributes                            | resource attributes of the log being processed                                                                                                     | pcommon.Map                                                             |
| resource.attributes\[""\]                      | the value of the resource attribute of the log being processed. Supports multiple indexes to access nested fields.                                 | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| resource.dropped_attributes_count              | number of dropped attributes of the resource of the log being processed                                                                            | int64                                                                   |
| instrumentation_scope                          | instrumentation scope of the log being processed                                                                                                   | pcommon.InstrumentationScope                                            |
| instrumentation_scope.name                     | name of the instrumentation scope of the log being processed                                                                                       | string                                                                  |
| instrumentation_scope.version                  | version of the instrumentation scope of the log being processed                                                                                    | string                                                                  |
| instrumentation_scope.dropped_attributes_count | number of dropped attributes of the instrumentation scope of the log being processed                                                               | int64                                                                   |
| instrumentation_scope.attributes               | instrumentation scope attributes of the data point being processed                                                                                 | pcommon.Map                                                             |
| instrumentation_scope.attributes\[""\]         | the value of the instrumentation scope attribute of the data point being processed. Supports multiple indexes to access nested fields.             | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| attributes                                     | attributes of the log being processed                                                                                                              | pcommon.Map                                                             |
| attributes\[""\]                               | the value of the attribute of the log being processed. Supports multiple indexes to access nested fields.                                          | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| trace_id                                       | a byte slice representation of the trace id                                                                                                        | pcommon.TraceID                                                         |
| trace_id.string                                | a string representation of the trace id                                                                                                            | string                                                                  |
| span_id                                        | a byte slice representation of the span id                                                                                                         | pcommon.SpanID                                                          |
| span_id.string                                 | a string representation of the span id                                                                                                             | string                                                                  |
| time_unix_nano                                 | the time in unix nano of the log being processed                                                                                                   | int64                                                                   |
| observed_time_unix_nano                        | the observed time in unix nano of the log being processed                                                                                          | int64                                                                   |
| time                                           | the time in `time.Time` of the log being processed                                                                                                 | `time.Time`                                                             |
| observed_time                                  | the observed time in `time.Time` of the log being processed                                                                                        | `time.Time`                                                             |
| severity_number                                | the severity numbner of the log being processed                                                                                                    | int64                                                                   |
| severity_text                                  | the severity text of the log being processed                                                                                                       | string                                                                  |
| body                                           | the body of the log being processed                                                                                                                | any                                                                     |
| body\[""\]                                     | a value in a map body of the log being processed. Supports multiple indexes to access nested fields.                                               | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| body\[\]                                       | a value in a slice body of the log being processed. Supports multiple indexes to access nested fields.                                             | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| body.string                                    | the body of the log being processed represented as a string.  When setting must pass a string.                                                     | string                                                                  |
| dropped_attributes_count                       | the number of dropped attributes of the log being processed                                                                                        | int64                                                                   |
| flags                                          | the flags of the log being processed                                                                                                               | int64                                                                   |

## Enums

The Log Context supports the enum names from the [logs proto](https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/logs/v1/logs.proto).

| Enum Symbol                 | Value |
|-----------------------------|-------|
| SEVERITY_NUMBER_UNSPECIFIED | 0     |
| SEVERITY_NUMBER_TRACE       | 1     |
| 	SEVERITY_NUMBER_TRACE2     | 2     |
| 	SEVERITY_NUMBER_TRACE3     | 3     |
| 	SEVERITY_NUMBER_TRACE4     | 4     |
| 	SEVERITY_NUMBER_DEBUG      | 5     |
| 	SEVERITY_NUMBER_DEBUG2     | 6     |
| 	SEVERITY_NUMBER_DEBUG3     | 7     |
| 	SEVERITY_NUMBER_DEBUG4     | 8     |
| 	SEVERITY_NUMBER_INFO       | 9     |
| 	SEVERITY_NUMBER_INFO2      | 10    |
| 	SEVERITY_NUMBER_INFO3      | 11    |
| 	SEVERITY_NUMBER_INFO4      | 12    |
| 	SEVERITY_NUMBER_WARN       | 13    |
| 	SEVERITY_NUMBER_WARN2      | 14    |
| 	SEVERITY_NUMBER_WARN3      | 15    |
| 	SEVERITY_NUMBER_WARN4      | 16    |
| 	SEVERITY_NUMBER_ERROR      | 17    |
| 	SEVERITY_NUMBER_ERROR2     | 18    |
| 	SEVERITY_NUMBER_ERROR3     | 19    |
| 	SEVERITY_NUMBER_ERROR4     | 20    |
| 	SEVERITY_NUMBER_FATAL      | 21    |
| 	SEVERITY_NUMBER_FATAL2     | 22    |
| 	SEVERITY_NUMBER_FATAL3     | 23    |
| 	SEVERITY_NUMBER_FATAL4     | 24    |