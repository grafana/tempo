# Instrumentation Scope Context

The Instrumentation Scope Context is a Context implementation for [pdata Instrumentation Scope](https://github.com/open-telemetry/opentelemetry-collector/blob/main/pdata/pcommon/generated_instrumentationscope.go), the Collector's internal representation for OTLP instrumentation scope data.  This Context should be used when interacting only with OTLP instrumentation scope.

## Paths
In general, the Instrumentation Scope Context supports accessing pdata using the field names from the instrumentation section in the [common proto](https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/common/v1/common.proto).  All integers are returned and set via `int64`.  All doubles are returned and set via `float64`.

The following paths are supported.

| path                              | field accessed                                                                                                                                     | type                                                                    |
|-----------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------|
| cache                             | the value of the current transform context's temporary cache. cache can be used as a temporary placeholder for data during complex transformations | pcommon.Map                                                             |
| cache\[""\]                       | the value of an item in cache. Supports multiple indexes to access nested fields.                                                                  | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| resource                          | resource of the instrumentation scope being processed                                                                                              | pcommon.Resource                                                        |
| resource.attributes               | resource attributes of the instrumentation scope being processed                                                                                   | pcommon.Map                                                             |
| resource.attributes\[""\]         | the value of the resource attribute of the instrumentation scope being processed. Supports multiple indexes to access nested fields.               | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| resource.dropped_attributes_count | number of dropped attributes of the resource of the instrumentation scope being processed                                                          | int64                                                                   |
| name                              | name of the instrumentation scope of the scope being processed                                                                                     | string                                                                  |
| version                           | version of the instrumentation scope of the scope being processed                                                                                  | string                                                                  |
| dropped_attributes_count          | number of dropped attributes of the instrumentation scope of the scope being processed                                                             | int64                                                                   |
| attributes                        | instrumentation scope attributes of the scope being processed                                                                                      | pcommon.Map                                                             |
| attributes\[""\]                  | the value of the instrumentation scope attribute of the scope being processed. Supports multiple indexes to access nested fields.                  | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |


## Enums

The Instrumentation Scope Context does not define any Enums at this time.
