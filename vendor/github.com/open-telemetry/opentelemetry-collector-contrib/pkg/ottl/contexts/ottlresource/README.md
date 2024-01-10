# Resource Context

The Resource Context is a Context implementation for [pdata Resources](https://github.com/open-telemetry/opentelemetry-collector/blob/main/pdata/pcommon/generated_resource.go), the Collector's internal representation for an OTLP Resource.  This Context should be used when interacting only with OTLP resources.

## Paths
In general, the Resource Context supports accessing pdata using the field names from the [resource proto](https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/resource/v1/resource.proto).  All integers are returned and set via `int64`.  All doubles are returned and set via `float64`.

The following paths are supported.

| path                     | field accessed                                                                                                                                     | type                                                                    |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------|
| cache                    | the value of the current transform context's temporary cache. cache can be used as a temporary placeholder for data during complex transformations | pcommon.Map                                                             |
| cache\[""\]              | the value of an item in cache. Supports multiple indexes to access nested fields.                                                                  | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| attributes               | attributes of the resource being processed                                                                                                         | pcommon.Map                                                             |
| attributes\[""\]         | the value of the attribute of the resource being processed. Supports multiple indexes to access nested fields.                                     | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil |
| dropped_attributes_count | number of dropped attributes of the resource being processed                                                                                       | int64                                                                   |

## Enums

The Resource Context does not define any Enums at this time.
