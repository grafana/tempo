# Metric Context

> [!NOTE]
> This documentation applies only to version `0.120.0` and later. For information on earlier versions, please refer to the previous [documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/release/0.119.x/pkg/ottl/contexts/ottlmetric/README.md).

The Metric Context is a Context implementation for [pdata Metric](https://github.com/open-telemetry/opentelemetry-collector/tree/main/pdata/pmetric), the collector's internal representation for OTLP metrics.  This Context should be used when interacting with individual OTLP metrics.

## Paths
In general, the Metric Context supports accessing pdata using the field names from the [metrics proto](https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/metrics/v1/metrics.proto).  All integers are returned and set via `int64`.  All doubles are returned and set via `float64`.

The following paths are supported.

| path                                   | field accessed                                                                                                                                     | type                                                                                                                                        |
|----------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------|
| metric.cache                           | the value of the current transform context's temporary cache. cache can be used as a temporary placeholder for data during complex transformations | pcommon.Map                                                                                                                                 |
| metric.cache\[""\]                     | the value of an item in cache. Supports multiple indexes to access nested fields.                                                                  | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil                                                                     |
| resource                               | resource of the metric being processed                                                                                                             | pcommon.Resource                                                                                                                            |
| resource.attributes                    | resource attributes of the metric being processed                                                                                                  | pcommon.Map                                                                                                                                 |
| resource.attributes\[""\]              | the value of the resource attribute of the metric being processed. Supports multiple indexes to access nested fields.                              | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil                                                                     |
| instrumentation_scope                  | instrumentation scope of the metric being processed                                                                                                | pcommon.InstrumentationScope                                                                                                                |
| instrumentation_scope.name             | name of the instrumentation scope of the metric being processed                                                                                    | string                                                                                                                                      |
| instrumentation_scope.version          | version of the instrumentation scope of the metric being processed                                                                                 | string                                                                                                                                      |
| instrumentation_scope.attributes       | instrumentation scope attributes of the metric being processed                                                                                     | pcommon.Map                                                                                                                                 |
| instrumentation_scope.attributes\[""\] | the value of the instrumentation scope attribute of the metric being processed. Supports multiple indexes to access nested fields.                 | string, bool, int64, float64, pcommon.Map, pcommon.Slice, []byte or nil                                                                     |
| metric.name                            | the name of the metric                                                                                                                             | string                                                                                                                                      |
| metric.description                     | the description of the metric                                                                                                                      | string                                                                                                                                      |
| metric.unit                            | the unit of the metric                                                                                                                             | string                                                                                                                                      |
| metric.type                            | the data type of the metric                                                                                                                        | int64                                                                                                                                       |
| metric.metadata                        | metadata associated with the metric                                                                                                                | pcommon.Map                                                                                                                                       |
| metric.aggregation_temporality         | the aggregation temporality of the metric                                                                                                          | int64                                                                                                                                       |
| metric.is_monotonic                    | the monotonicity of the metric                                                                                                                     | bool                                                                                                                                        |
| metric.data_points                     | the data points of the metric                                                                                                                      | pmetric.NumberDataPointSlice, pmetric.HistogramDataPointSlice, pmetric.ExponentialHistogramDataPointSlice, or pmetric.SummaryDataPointSlice | 

## Enums

The Metrics Context supports the enum names from the [metrics proto](https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/metrics/v1/metrics.proto).

In addition, it also supports an enum for metrics data type, with the numeric value being [defined by pdata](https://github.com/open-telemetry/opentelemetry-collector/blob/main/pdata/pmetric/metrics.go).

| Enum Symbol                            | Value |
|----------------------------------------|-------|
| AGGREGATION_TEMPORALITY_UNSPECIFIED    | 0     |
| AGGREGATION_TEMPORALITY_DELTA          | 1     |
| AGGREGATION_TEMPORALITY_CUMULATIVE     | 2     |
| METRIC_DATA_TYPE_NONE                  | 0     |
| METRIC_DATA_TYPE_GAUGE                 | 1     |
| METRIC_DATA_TYPE_SUM                   | 2     |
| METRIC_DATA_TYPE_HISTOGRAM             | 3     |
| METRIC_DATA_TYPE_EXPONENTIAL_HISTOGRAM | 4     |
| METRIC_DATA_TYPE_SUMMARY               | 5     |